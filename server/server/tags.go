// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENSE file for details.

package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"goji.io/pattern"

	"dmitryfrank.com/geekmarks/server/cptr"
	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"github.com/dimonomid/interrors"
	"dmitryfrank.com/geekmarks/server/storage"
	"dmitryfrank.com/geekmarks/server/tagmatcher"

	"github.com/golang/glog"
	"github.com/juju/errors"
)

const (
	QSArgTagsShape       = "shape"
	QSArgTagsShapeTree   = "tree"
	QSArgTagsShapeFlat   = "flat"
	QSArgTagsShapeSingle = "single"

	QSArgTagsPattern = "pattern"

	QSArgTagsAllowNew = "allow_new"

	QSArgNewLeafPolicy     = "new_leaf_policy"
	QSArgNewLeafPolicyKeep = "keep"
	QSArgNewLeafPolicyDel  = "del"

	// In flat tags response, index at which new tag suggestion gets inserted
	// (if QSArgTagsAllowNew was equal to "1")
	newTagSuggestionIndex = 1

	// Max number of tags which can be returned to the tags GET request
	maxFlatTagsCnt = 20
)

var (
	ErrTagSuggestionFailed = errors.New("tag suggestion failed")
	userIDToTagsTree       = cacheUserIDToTagsTree{
		tagsTree: make(map[int]*cacheTagsTree),
	}
)

type userTagsGetResp struct {
	Tags []userTagData `json:"tags"`
}

type userTagData struct {
	ID          int           `json:"id"`
	Description string        `json:"description,omitempty"`
	Names       []string      `json:"names"`
	Subtags     []userTagData `json:"subtags,omitempty"`
}

type userTagDataFlat struct {
	Path string `json:"path"`
	// ID can be -1 for new tag suggestions
	ID          int    `json:"id"`
	Description string `json:"description,omitempty"`
	// Only for new tags (i.e. when ID is -1): indicates how many tags the Path
	// actually includes
	NewTagsCnt int `json:"newTagsCnt,omitempty"`
}

type matchDetails struct {
	matchedNameIdx int
	prio           tagmatcher.Priority
	det            *tagmatcher.MatchDetails
}

type tagDataFlatInternal struct {
	pathItems   [][]string
	id          int
	description string
	matches     map[int]matchDetails

	pathComponentIdxMax int
	lastComponentPrio   tagmatcher.Priority
}

func (t *tagDataFlatInternal) PathItems() [][]string {
	return t.pathItems
}

func (t *tagDataFlatInternal) Path() string {
	parts := make([]string, len(t.pathItems))
	for k, names := range t.pathItems {
		add := ""
		nameIdx := 0
		if n, ok := t.matches[k]; ok {
			nameIdx = n.matchedNameIdx
			//if n.det != nil {
			//add = fmt.Sprintf("(%d) ", n.prio)
			//}
		}
		parts[k] = add + names[nameIdx]
	}
	return strings.Join(parts, "/")
}

func (t *tagDataFlatInternal) SetMatchDetails(
	pathComponentIdx, matchedNameIdx int, prio tagmatcher.Priority,
	det *tagmatcher.MatchDetails,
) {
	t.matches[pathComponentIdx] = matchDetails{
		matchedNameIdx: matchedNameIdx,
		det:            det,
		prio:           prio,
	}
}

func (t *tagDataFlatInternal) SetMaxPathItemIdx(
	pathComponentIdx int, prio tagmatcher.Priority,
) {
	t.pathComponentIdxMax = pathComponentIdx
	t.lastComponentPrio = prio
}

func (t *tagDataFlatInternal) GetMaxPathItemIdx() int {
	return t.pathComponentIdxMax
}

func (t *tagDataFlatInternal) GetMaxPathItemIdxRev() int {
	return len(t.pathItems) - 1 - t.pathComponentIdxMax
}

func (t *tagDataFlatInternal) GetPrio() tagmatcher.Priority {
	return t.lastComponentPrio
}

type userTagsPostArgs struct {
	Names              []string `json:"names"`
	Description        *string  `json:"description"`
	CreateIntermediary bool     `json:"createIntermediary,omitempty"`
}

type userTagsPostResp struct {
	TagID int `json:"tagID"`
}

type userTagPutArgs struct {
	Names       []string `json:"names"`
	Description *string  `json:"description"`
	// ParentTagID should be provided if only tag needs to be moved to a new
	// parent
	ParentTagID *int `json:"parentTagID"`
	// NewLeafPolicy is used (and required) if only ParentTagID is provided
	NewLeafPolicy *string `json:"newLeafPolicy"`
}

type userTagPutResp struct {
}

type userTagDeleteResp struct {
}

func (gm *GMServer) getTagIDFromPath(
	gmr *GMRequest, tx *sql.Tx, ownerID int, createNonExisting bool,
) (int, error) {
	parentTagID := 0

	tagPath := pattern.Path(gmr.HttpReq.Context())

	// A hack for the Swagger spec path parameter to work. There's no support for
	// wildcard path parameters (and it's not going to be supported soon, see:
	// https://github.com/OAI/OpenAPI-Specification/issues/892#issuecomment-281170254 )
	// so if I have a path /tags{tag_names}, and I pass /foo/bar as tag_names,
	// then the resulting URL becomes: /tags%2Ffoo%2Fbar. We just need to
	// urldecode the path, and it will work.
	//
	// TODO(dfrank) remove it when Swagger spec supports wildcard path parameters
	tagPath, err := url.QueryUnescape(tagPath)
	if err != nil {
		return 0, errors.Annotatef(err, "wrong tag path")
	}

	if len(tagPath) > 0 {
		if parentID, err := strconv.Atoi(tagPath[1:]); err == nil {
			parentTagData, err := gm.si.GetTag(tx, parentID, &storage.GetTagOpts{})
			if err != nil {
				return 0, errors.Trace(err)
			}

			err = gm.authorizeOperation(
				gmr.Caller, &authzArgs{OwnerID: parentTagData.OwnerID},
			)
			if err != nil {
				return 0, errors.Trace(err)
			}

			parentTagID = parentID
		}
	}

	if parentTagID == 0 {
		var err error
		if createNonExisting && tagPath != "" && tagPath != "/" {
			det, err := gm.getNewTagDetails(gmr, tx, tagPath)
			if err != nil {
				return 0, errors.Trace(err)
			}

			// Refuse to create tag if the given name needs to be cleaned up
			// (even though the cleanup was successful)
			if det.CleanPath != tagPath {
				return 0, errors.Errorf("invalid tag tagPath %q (the valid one would be: %q)", tagPath, det.CleanPath)
			}

			curTagID := det.ParentTagID
			for _, curName := range det.NonExistingNames {
				var err error
				curTagID, err = gm.si.CreateTag(tx, &storage.TagData{
					OwnerID:     gmr.SubjUser.ID,
					ParentTagID: cptr.Int(curTagID),
					Names:       []string{curName},
				})
				if err != nil {
					return 0, errors.Trace(err)
				}
			}
			parentTagID = curTagID
		} else {
			parentTagID, err = gm.si.GetTagIDByPath(tx, ownerID, tagPath)
			if err != nil {
				return 0, errors.Trace(err)
			}
		}
	}

	return parentTagID, nil
}

// userTagsGet is a GET /tags and /tags/* handler
func (gm *GMServer) userTagsGet(gmr *GMRequest) (resp interface{}, err error) {
	err = gm.authorizeOperation(gmr.Caller, &authzArgs{OwnerID: gmr.SubjUser.ID})
	if err != nil {
		return nil, errors.Trace(err)
	}

	allowNew := gmr.FormValue(QSArgTagsAllowNew) == "1"

	// By default, use shape "tree"
	shape := QSArgTagsShapeTree

	// Determine pattern: by default, use an empty string
	strpattern := ""
	if t := gmr.FormValue(QSArgTagsPattern); t != "" {
		strpattern = t
	}

	// If querytype is "pattern", change the default shape to "flat"
	if strpattern != "" {
		shape = QSArgTagsShapeFlat
	}

	// If shape was given, use it
	if s := gmr.FormValue(QSArgTagsShape); s != "" {
		if s != QSArgTagsShapeTree && s != QSArgTagsShapeFlat && s != QSArgTagsShapeSingle {
			return nil, errors.Errorf(
				"invalid %s: %q; valid values are: %q, %q",
				QSArgTagsShape, shape, QSArgTagsShapeTree, QSArgTagsShapeFlat,
			)
		}
		shape = s
	}

	if shape != QSArgTagsShapeFlat && strpattern != "" {
		return nil, errors.Errorf("pattern and %s %q cannot be used together", QSArgTagsShape, shape)
	}

	// Get tags tree from either cache or database
	var tagData *storage.TagData
	withSubtags := (shape != QSArgTagsShapeSingle)

	// Get a cached struct for the given user. If it does not exist, create one,
	// and add to the cache.
	cache := userIDToTagsTree.GetCacheForUser(gmr.SubjUser.ID)
	if cache == nil {
		glog.V(3).Infof("No cache for user %d, creating", gmr.SubjUser.ID)
		cache = &cacheTagsTree{
			tagIDToTree: make(map[string]*storage.TagData),
		}
		userIDToTagsTree.SetCacheForUser(gmr.SubjUser.ID, cache)
	}

	// Try to get cached tree data from the cache struct. If cache does not
	// contain what we need, we'll need to reach the database, get tree data
	// from there, and put it to the cache.
	tagPath := pattern.Path(gmr.HttpReq.Context())
	tagData = cache.GetTagData(tagPath, withSubtags)
	if tagData == nil {
		glog.V(3).Infof(
			"No tree data cache for user %d, path=%q, withSubtags=%v, creating",
			gmr.SubjUser.ID, tagPath, withSubtags,
		)
		err = gm.si.Tx(func(tx *sql.Tx) error {
			var parentTagID int
			var err error

			parentTagID, err = gm.getTagIDFromPath(gmr, tx, gmr.SubjUser.ID, false)
			if err != nil {
				return errors.Trace(err)
			}

			tagData, err = gm.si.GetTag(tx, parentTagID, &storage.GetTagOpts{
				GetNames:   true,
				GetSubtags: withSubtags,
			})
			if err != nil {
				return errors.Trace(err)
			}

			return nil
		})
		if err != nil {
			return nil, errors.Trace(err)
		}

		cache.SetTagData(tagPath, withSubtags, tagData)
	} else {
		glog.V(3).Infof(
			"Got tree data cache for user %d, path=%q, withSubtags=%v",
			gmr.SubjUser.ID, tagPath, withSubtags,
		)
	}

	// Convert internal tags tree into the requested shape
	switch shape {

	case QSArgTagsShapeTree, QSArgTagsShapeSingle:
		resp = gm.createUserTagData(tagData)

	case QSArgTagsShapeFlat:
		tagsFlat := gm.createTagDataFlatInternal(tagData, nil, nil)

		if strpattern != "" {
			// Convert a slice to a slice of needed interface (tagmatcher.TagPather)
			tp := make([]tagmatcher.TagPather, len(tagsFlat))
			for i, v := range tagsFlat {
				tp[i] = v
			}

			// Match against the strpattern
			matcher := tagmatcher.NewTagMatcher()
			tp, err = matcher.Filter(tp, strpattern)
			if err != nil {
				return nil, errors.Trace(err)
			}

			// Convert resulting slice back to a slice of tagDataFlatInternal
			tagsFlat = make([]*tagDataFlatInternal, len(tp))
			for i, v := range tp {
				tagsFlat[i] = v.(*tagDataFlatInternal)
			}

			// if there are too many tags, leave only first maxFlatTagsCnt
			//
			// TODO: add a query parameter which should override default limit
			//
			// TODO: probably we need to add a trailing special item which
			// indicates that there are N extra tags, and if user clicks on it,
			// the client would request all tags
			if len(tagsFlat) > maxFlatTagsCnt {
				tagsFlat = tagsFlat[:maxFlatTagsCnt]
			}
		}

		// If new tags are allowed, get a suggestion for the new tag, if any
		var newTag *userTagDataFlat
		if allowNew {
			var err error
			newTag, err = gm.getNewTagSuggestion(gmr, strpattern)
			if err != nil {
				return nil, errors.Trace(err)
			}
		}

		// Convert internal slice to a public slice
		// (and if new tags are allowed, add a suggestion as a second item)
		userTagsFlat := []userTagDataFlat{}
		for i, v := range tagsFlat {
			if allowNew && i == newTagSuggestionIndex && newTag != nil {
				userTagsFlat = append(userTagsFlat, *newTag)
			}
			userTagsFlat = append(userTagsFlat, userTagDataFlat{
				Path:        v.Path(),
				ID:          v.id,
				Description: v.description,
			})
		}

		// if new tags are allowed, but we didn't have a chance to insert a
		// suggestion due to the low number of the existing matching tags,
		// then add a new tag suggestion as the last item
		if allowNew && len(userTagsFlat) <= newTagSuggestionIndex && newTag != nil {
			userTagsFlat = append(userTagsFlat, *newTag)
		}

		resp = userTagsFlat

	default:
		return nil, hh.MakeInternalServerError(errors.Errorf("should never be here"))
	}

	return resp, nil
}

type newTagDetails struct {
	CleanPath string
	// ParentTagID is the id of the most deep existing tag
	ParentTagID int
	// NonExistingNames is a slice of names for non-existing tags, which can
	// be added to ParentTagID. If the pattern given to getNewTagDetails() refers
	// to an existing tag, then ParentTagID will contains this id, and
	// NonExistingNames will be empty.
	NonExistingNames []string
}

// getNewTagDetails takes a string pattern and returns details of the matching
// existing tag, and names of the non-existing tags which could be created
// (see newTagDetails)
func (gm *GMServer) getNewTagDetails(
	gmr *GMRequest, tx *sql.Tx, pattern string,
) (*newTagDetails, error) {
	// Sanitize input pattern
	n := strings.Split(pattern, "/")
	names := []string{}
	for _, v := range n {
		if v != "" {
			err, v := storage.CleanupTagName(v, false)
			if err != nil {
				return nil, errors.Annotatef(ErrTagSuggestionFailed, "%s", err)
			}

			names = append(names, v)
		}
	}

	// If the resulting pattern is empty, suggest nothing
	if len(names) == 0 {
		return nil, errors.Annotatef(ErrTagSuggestionFailed, "tag name is empty")
	}

	newTagsCnt := 0
	parentTagID := 0

	// Iterate all tag names in the input pattern, remember the id of the most
	// nested one, and count number of non-existing ones.
	for i := 0; i < len(names); i++ {
		var err error
		// Try to get ID of the current tag
		parentTagID, err = gm.si.GetTagIDByPath(
			tx,
			gmr.SubjUser.ID,
			strings.Join(names[:len(names)-i], "/"),
		)
		if err != nil {
			if errors.Cause(err) == storage.ErrTagDoesNotExist {
				// Tag does not exist: increment newTagsCnt counter and continue
				newTagsCnt++
				continue
			}
			// Some other error, return an error
			return nil, errors.Trace(err)
		}

		break
	}

	// If there are no existing matching tags, then we'll end up with the zero
	// parentTagID: let's get root tag ID then.
	if parentTagID == 0 {
		var err error
		parentTagID, err = gm.si.GetRootTagID(tx, gmr.SubjUser.ID)
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	nonExistingNames := []string{}
	if newTagsCnt > 0 {
		nonExistingNames = names[len(names)-newTagsCnt:]
	}

	return &newTagDetails{
		CleanPath:        "/" + strings.Join(names, "/"),
		ParentTagID:      parentTagID,
		NonExistingNames: nonExistingNames,
	}, nil
}

// getNewTagSuggestion takes a pattern and returns details for the new
// tag suggestion
func (gm *GMServer) getNewTagSuggestion(
	gmr *GMRequest, pattern string,
) (*userTagDataFlat, error) {
	var newTagDetails *newTagDetails

	err := gm.si.Tx(func(tx *sql.Tx) error {
		var err error
		newTagDetails, err = gm.getNewTagDetails(gmr, tx, pattern)
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})
	if err != nil {
		if errors.Cause(err) == ErrTagSuggestionFailed {
			// Tag suggestion failed: just silently suggest nothing
			return nil, nil
		}
		// Some other error: return ann error
		return nil, errors.Trace(err)
	}

	// If there are no new tags, return nil
	if len(newTagDetails.NonExistingNames) == 0 {
		return nil, nil
	}

	return &userTagDataFlat{
		Path:        newTagDetails.CleanPath,
		ID:          -1,
		Description: "Non-existing tag",
		NewTagsCnt:  len(newTagDetails.NonExistingNames),
	}, nil
}

func (gm *GMServer) createUserTagData(in *storage.TagData) *userTagData {
	if in == nil {
		return nil
	}

	res := userTagData{
		ID:          in.ID,
		Description: *in.Description,
		Names:       in.Names,
	}

	for _, td := range in.Subtags {
		res.Subtags = append(res.Subtags, *gm.createUserTagData(&td))
	}

	return &res
}

func (gm *GMServer) createTagDataFlatInternal(
	in *storage.TagData,
	result []*tagDataFlatInternal,
	pathItems [][]string,
) []*tagDataFlatInternal {
	if in == nil {
		return result
	}

	newPathItems := make([][]string, len(pathItems)+1)
	copy(newPathItems, pathItems)
	newPathItems[len(newPathItems)-1] = in.Names

	item := tagDataFlatInternal{
		pathItems:   newPathItems,
		id:          in.ID,
		description: *in.Description,
		matches:     make(map[int]matchDetails),

		lastComponentPrio: tagmatcher.NoMatch,
	}

	result = append(result, &item)

	for _, td := range in.Subtags {
		result = gm.createTagDataFlatInternal(&td, result, newPathItems)
	}

	return result
}

func (gm *GMServer) userTagsPost(gmr *GMRequest) (resp interface{}, err error) {
	err = gm.authorizeOperation(gmr.Caller, &authzArgs{OwnerID: gmr.SubjUser.ID})
	if err != nil {
		return nil, errors.Trace(err)
	}

	decoder := json.NewDecoder(gmr.Body)
	var args userTagsPostArgs
	err = decoder.Decode(&args)
	if err != nil {
		// TODO: provide request data example
		return nil, interrors.WrapInternalError(
			err,
			errors.Errorf("invalid data"),
		)
	}

	tagID := 0

	err = gm.si.Tx(func(tx *sql.Tx) error {
		parentTagID, err := gm.getTagIDFromPath(
			gmr, tx, gmr.SubjUser.ID, args.CreateIntermediary,
		)
		if err != nil {
			return errors.Trace(err)
		}

		tagID, err = gm.si.CreateTag(tx, &storage.TagData{
			OwnerID:     gmr.SubjUser.ID,
			ParentTagID: cptr.Int(parentTagID),
			Names:       args.Names,
			Description: args.Description,
		})
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Invalidate tree cache for the user
	userIDToTagsTree.DeleteCacheForUser(gmr.SubjUser.ID)

	resp = userTagsPostResp{
		TagID: tagID,
	}

	return resp, nil
}

func (gm *GMServer) userTagPut(gmr *GMRequest) (resp interface{}, err error) {
	err = gm.authorizeOperation(gmr.Caller, &authzArgs{OwnerID: gmr.SubjUser.ID})
	if err != nil {
		return nil, errors.Trace(err)
	}

	decoder := json.NewDecoder(gmr.Body)
	var args userTagPutArgs
	err = decoder.Decode(&args)
	if err != nil {
		// TODO: provide request data example
		return nil, interrors.WrapInternalError(
			err,
			errors.Errorf("invalid data"),
		)
	}

	err = gm.si.Tx(func(tx *sql.Tx) error {
		tagID, err := gm.getTagIDFromPath(
			gmr, tx, gmr.SubjUser.ID, false,
		)
		if err != nil {
			return errors.Trace(err)
		}

		var leafPolicy storage.TaggableLeafPolicy

		// If ParentTagID is given (i.e. the tag is going to be moved), make sure
		// the user is authorized to edit the new ParentTagID as well.
		if args.ParentTagID != nil {
			newParentTag, err := gm.si.GetTag(
				tx, *args.ParentTagID, &storage.GetTagOpts{},
			)
			if err != nil {
				return errors.Trace(err)
			}

			err = gm.authorizeOperation(gmr.Caller, &authzArgs{OwnerID: newParentTag.OwnerID})
			if err != nil {
				return errors.Trace(err)
			}

			// Make sure newLeafPolicy is specified and is valid
			if args.NewLeafPolicy == nil {
				return errors.New(getErrorMsgParamRequired(
					"newLeafPolicy", []string{QSArgNewLeafPolicyKeep, QSArgNewLeafPolicyDel},
				))
			}

			leafPolicy, err = getStorageTaggableLeafPolicy(*args.NewLeafPolicy)
			if err != nil {
				return errors.Trace(err)
			}
		}

		err = gm.si.UpdateTag(tx, &storage.TagData{
			ID:          tagID,
			Names:       args.Names,
			Description: args.Description,
			ParentTagID: args.ParentTagID,
		}, leafPolicy)
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Invalidate tree cache for the user
	userIDToTagsTree.DeleteCacheForUser(gmr.SubjUser.ID)

	resp = userTagPutResp{}

	return resp, nil
}

func (gm *GMServer) userTagDelete(gmr *GMRequest) (resp interface{}, err error) {
	err = gm.authorizeOperation(gmr.Caller, &authzArgs{OwnerID: gmr.SubjUser.ID})
	if err != nil {
		return nil, errors.Trace(err)
	}

	leafPolicyStr := gmr.FormValue(QSArgNewLeafPolicy)
	if leafPolicyStr == "" {
		return nil, errors.New(getErrorMsgParamRequired(
			QSArgNewLeafPolicy, []string{QSArgNewLeafPolicyKeep, QSArgNewLeafPolicyDel},
		))
	}

	leafPolicy, err := getStorageTaggableLeafPolicy(leafPolicyStr)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if leafPolicy != storage.TaggableLeafPolicyKeep {
		return nil, errors.Annotatef(
			hh.MakeNotImplementedError(),
			"so far %q can only be %q", QSArgNewLeafPolicy, QSArgNewLeafPolicyKeep,
		)
	}

	err = gm.si.Tx(func(tx *sql.Tx) error {
		tagID, err := gm.getTagIDFromPath(
			gmr, tx, gmr.SubjUser.ID, false,
		)
		if err != nil {
			return errors.Trace(err)
		}

		err = gm.si.DeleteTag(tx, tagID, leafPolicy)
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Invalidate tree cache for the user
	userIDToTagsTree.DeleteCacheForUser(gmr.SubjUser.ID)

	resp = userTagDeleteResp{}

	return resp, nil
}

func getStorageTaggableLeafPolicy(
	newLeafPolicy string,
) (storage.TaggableLeafPolicy, error) {
	switch newLeafPolicy {
	case QSArgNewLeafPolicyKeep:
		return storage.TaggableLeafPolicyKeep, nil
	case QSArgNewLeafPolicyDel:
		return storage.TaggableLeafPolicyDel, nil
	default:
		return storage.TaggableLeafPolicy("invalid:" + newLeafPolicy),
			errors.Errorf("unknown %q: %q", QSArgNewLeafPolicy, newLeafPolicy)
	}
}

// Tags tree cache {{{

type cacheUserIDToTagsTree struct {
	// Global mutex, locked for a very short period of time for each request
	// to tags tree
	mutex    sync.Mutex
	tagsTree map[int]*cacheTagsTree
}

type cacheTagsTree struct {
	// Per-user mutex
	mutex       sync.Mutex
	tagIDToTree map[string]*storage.TagData
}

func getCacheMapKey(path string, withSubtags bool) string {
	return fmt.Sprintf("%s-%v", path, withSubtags)
}

func (c *cacheTagsTree) GetTagData(path string, withSubtags bool) *storage.TagData {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.tagIDToTree[getCacheMapKey(path, withSubtags)]
}

func (c *cacheTagsTree) SetTagData(path string, withSubtags bool, td *storage.TagData) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.tagIDToTree[getCacheMapKey(path, withSubtags)] = td
}

func (c *cacheUserIDToTagsTree) GetCacheForUser(userID int) *cacheTagsTree {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.tagsTree[userID]
}

func (c *cacheUserIDToTagsTree) SetCacheForUser(userID int, cache *cacheTagsTree) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.tagsTree[userID] = cache
}

func (c *cacheUserIDToTagsTree) DeleteCacheForUser(userID int) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.tagsTree, userID)
}

// }}}
