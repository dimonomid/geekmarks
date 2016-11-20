package server

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"

	"goji.io/pattern"

	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/interror"
	"dmitryfrank.com/geekmarks/server/storage"
	"dmitryfrank.com/geekmarks/server/tagmatcher"

	"github.com/juju/errors"
)

const (
	TagsShape     = "shape"
	TagsShapeTree = "tree"
	TagsShapeFlat = "flat"

	TagsPattern = "pattern"

	TagsAllowNew = "allow_new"

	// In flat tags response, index at which new tag suggestion gets inserted
	// (if TagsAllowNew was equal to "1")
	newTagSuggestionIndex = 1
)

var (
	ErrTagSuggestionFailed = errors.New("tag suggestion failed")
)

type userTagsGetResp struct {
	Tags []userTagData `json:"tags"`
}

type userTagData struct {
	ID          int           `json:"id"`
	Description string        `json:"description"`
	Names       []string      `json:"names"`
	Subtags     []userTagData `json:"subtags,omitempty"`
}

type userTagDataFlat struct {
	Path string `json:"path"`
	// ID can be -1 for new tag suggestions
	ID          int    `json:"id"`
	Description string `json:"description"`
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
	Description        string   `json:"description"`
	CreateIntermediary bool     `json:"createIntermediary,omitempty"`
}

type userTagsPostResp struct {
	TagID int `json:"tagID"`
}

func (gm *GMServer) getTagIDFromPath(
	gmr *GMRequest, tx *sql.Tx, ownerID int, createNonExisting bool,
) (int, error) {
	parentTagID := 0

	path := pattern.Path(gmr.HttpReq.Context())

	if len(path) > 0 {
		if parentID, err := strconv.Atoi(path[1:]); err == nil {
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
		if createNonExisting {
			det, err := gm.getNewTagDetails(gmr, tx, path)
			if err != nil {
				return 0, errors.Trace(err)
			}

			// Refuse to create tag if the given name needs to be cleaned up
			// (even though the cleanup was successful)
			if det.CleanPath != path {
				return 0, errors.Errorf("invalid tag path %q (the valid one would be: %q)", path, det.CleanPath)
			}

			curTagID := det.ParentTagID
			for _, curName := range det.NonExistingNames {
				var err error
				curTagID, err = gm.si.CreateTag(tx, &storage.TagData{
					OwnerID:     gmr.SubjUser.ID,
					ParentTagID: curTagID,
					Names:       []string{curName},
				})
				if err != nil {
					return 0, errors.Trace(err)
				}
			}
			parentTagID = curTagID
		} else {
			parentTagID, err = gm.si.GetTagIDByPath(tx, ownerID, path)
			if err != nil {
				return 0, errors.Trace(err)
			}
		}
	}

	return parentTagID, nil
}

func (gm *GMServer) userTagsGet(gmr *GMRequest) (resp interface{}, err error) {
	err = gm.authorizeOperation(gmr.Caller, &authzArgs{OwnerID: gmr.SubjUser.ID})
	if err != nil {
		return nil, errors.Trace(err)
	}

	allowNew := gmr.FormValue(TagsAllowNew) == "1"

	// By default, use shape "tree"
	shape := TagsShapeTree

	// Determine pattern: by default, use an empty string
	pattern := ""
	if t := gmr.FormValue(TagsPattern); t != "" {
		pattern = t
	}

	// If querytype is "pattern", change the default shape to "flat"
	if pattern != "" {
		shape = TagsShapeFlat
	}

	// If shape was given, use it
	if s := gmr.FormValue(TagsShape); s != "" {
		if s != TagsShapeTree && s != TagsShapeFlat {
			return nil, errors.Errorf(
				"invalid %s: %q; valid values are: %q, %q",
				TagsShape, shape, TagsShapeTree, TagsShapeFlat,
			)
		}
		shape = s
	}

	if shape == TagsShapeTree && pattern != "" {
		return nil, errors.Errorf("pattern and %s %q cannot be used together", TagsShape, shape)
	}

	// Get tags tree from storage
	var tagData *storage.TagData
	err = gm.si.Tx(func(tx *sql.Tx) error {
		var parentTagID int
		var err error

		parentTagID, err = gm.getTagIDFromPath(gmr, tx, gmr.SubjUser.ID, false)
		if err != nil {
			return errors.Trace(err)
		}

		tagData, err = gm.si.GetTag(tx, parentTagID, &storage.GetTagOpts{
			GetNames:   true,
			GetSubtags: true,
		})
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Convert internal tags tree into requested shape
	switch shape {

	case TagsShapeTree:
		resp = gm.createUserTagData(tagData)

	case TagsShapeFlat:
		tagsFlat := gm.createTagDataFlatInternal(tagData, nil, nil)

		if pattern != "" {
			// Convert a slice to a slice of needed interface (tagmatcher.TagPather)
			tp := make([]tagmatcher.TagPather, len(tagsFlat))
			for i, v := range tagsFlat {
				tp[i] = v
			}

			// Match against the pattern
			matcher := tagmatcher.NewTagMatcher()
			tp, err = matcher.Filter(tp, pattern)
			if err != nil {
				return nil, errors.Trace(err)
			}

			// Convert resulting slice back to a slice of tagDataFlatInternal
			tagsFlat = make([]*tagDataFlatInternal, len(tp))
			for i, v := range tp {
				tagsFlat[i] = v.(*tagDataFlatInternal)
			}
		}

		var newTag *userTagDataFlat
		if allowNew {
			var err error
			newTag, err = gm.getNewTagSuggestion(gmr, pattern)
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

	for i := 0; i < len(names); i++ {
		var err error
		parentTagID, err = gm.si.GetTagIDByPath(
			tx,
			gmr.SubjUser.ID,
			strings.Join(names[:len(names)-i], "/"),
		)
		if err != nil {
			if errors.Cause(err) == storage.ErrTagDoesNotExist {
				newTagsCnt++
				continue
			} else {
				return nil, errors.Trace(err)
			}
		}

		break
	}

	// If all tags exist, then we'll end up with the zero parentTagID: let's get
	// root tag ID then.
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
		Description: in.Description,
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
		description: in.Description,
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
		return nil, interror.WrapInternalError(
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
			ParentTagID: parentTagID,
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

	resp = userTagsPostResp{
		TagID: tagID,
	}

	return resp, nil
}
