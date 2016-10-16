package server

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"

	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/interror"
	"dmitryfrank.com/geekmarks/server/storage"
	"dmitryfrank.com/geekmarks/server/tagspattern"

	"github.com/juju/errors"
)

const (
	TagsShape     = "shape"
	TagsShapeTree = "tree"
	TagsShapeFlat = "flat"

	TagsPattern = "pattern"
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
	Path        string `json:"path"`
	ID          int    `json:"id"`
	Description string `json:"description"`
}

type tagDataFlatInternal struct {
	pathAllNames string
	id           int
	description  string
	matchDetails *tagspattern.MatchDetails
}

func (t *tagDataFlatInternal) PathAllNames() string {
	return t.pathAllNames
}

func (t *tagDataFlatInternal) Path() string {
	//TODO: use matchDetails
	parts := strings.Split(t.pathAllNames, "/")
	for k, part := range parts {
		if len(part) > 0 {
			names := strings.Split(part, "|")
			parts[k] = names[1]
		}
	}
	return strings.Join(parts, "/")
}

func (t *tagDataFlatInternal) SetMatchDetails(details *tagspattern.MatchDetails) {
	t.matchDetails = details
}

type userTagsPostArgs struct {
	Names       []string `json:"names"`
	Description string   `json:"description"`
}

type userTagsPostResp struct {
	TagID int `json:"tagID"`
}

func (gm *GMServer) getTagIDFromPath(gmr *GMRequest, tx *sql.Tx, ownerID int) (int, error) {
	parentTagID := 0

	if len(gmr.Path) > 0 {
		if parentID, err := strconv.Atoi(gmr.Path[1:]); err == nil {
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
		parentTagID, err = gm.si.GetTagIDByPath(tx, ownerID, gmr.Path)
		if err != nil {
			return 0, errors.Trace(err)
		}
	}

	return parentTagID, nil
}

func (gm *GMServer) userTagsGet(gmr *GMRequest) (resp interface{}, err error) {
	err = gm.authorizeOperation(gmr.Caller, &authzArgs{OwnerID: gmr.SubjUser.ID})
	if err != nil {
		return nil, errors.Trace(err)
	}

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

		parentTagID, err = gm.getTagIDFromPath(gmr, tx, gmr.SubjUser.ID)
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
		tagsFlat := gm.createTagDataFlatInternal(tagData, nil, "")

		if pattern != "" {
			// Convert a slice to a slice of needed interface (tagspattern.TagPather)
			tp := make([]tagspattern.TagPather, len(tagsFlat))
			for i, v := range tagsFlat {
				tp[i] = v
			}

			// Match against the pattern
			matcher := &tagspattern.TagMatcher{}
			tp = matcher.Filter(tp, pattern)

			// Convert resulting slice back to a slice of tagDataFlatInternal
			tagsFlat = make([]*tagDataFlatInternal, len(tp))
			for i, v := range tp {
				tagsFlat[i] = v.(*tagDataFlatInternal)
			}
		}

		// Convert internal slice to a public slice
		userTagsFlat := make([]userTagDataFlat, len(tagsFlat))
		for i, v := range tagsFlat {
			userTagsFlat[i] = userTagDataFlat{
				Path:        v.Path(),
				ID:          v.id,
				Description: v.description,
			}
		}
		resp = userTagsFlat

	default:
		return nil, hh.MakeInternalServerError(errors.Errorf("should never be here"))
	}

	return resp, nil
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
	path string,
) []*tagDataFlatInternal {
	if in == nil {
		return result
	}

	newPath := path + "|" + strings.Join(in.Names, "|") + "|/"
	if newPath == "||/" {
		newPath = "/"
	}
	item := tagDataFlatInternal{
		pathAllNames: newPath[:(len(newPath) - 1)],
		id:           in.ID,
		description:  in.Description,
	}

	result = append(result, &item)

	for _, td := range in.Subtags {
		result = gm.createTagDataFlatInternal(&td, result, newPath)
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
		parentTagID, err := gm.getTagIDFromPath(gmr, tx, gmr.SubjUser.ID)
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
