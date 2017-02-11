// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENCE file for details.

package taghier

import (
	"sort"

	"github.com/juju/errors"
)

type Registry interface {
	// If GetParent returns 0, it means "no parent"
	GetParent(id int) (int, error)
}

type tagHierItem struct {
	id          int
	parentID    int
	childrenIDs map[int]struct{}
}

type TagHier struct {
	reg      Registry
	idToItem map[int]*tagHierItem
	leafs    map[int]*tagHierItem
	roots    map[int]*tagHierItem
}

type Diff struct {
	Add    []int
	Delete []int
}

func New(reg Registry) *TagHier {
	return &TagHier{
		idToItem: make(map[int]*tagHierItem),
		leafs:    make(map[int]*tagHierItem),
		roots:    make(map[int]*tagHierItem),
		reg:      reg,
	}
}

func (h *TagHier) Add(id int) error {
	return h.addInternal(id, true)
}

func (h *TagHier) addInternal(id int, isLeaf bool) error {
	// Id 0 is used as a parent id, to indicate the absence of the parent.
	// So, here we just ignore zero id.
	if id == 0 {
		return nil
	}

	// If hierarchy already contains given item, return
	if _, ok := h.idToItem[id]; ok {
		// And if needed, remove this item from leafs
		if !isLeaf {
			if _, ok := h.leafs[id]; ok {
				delete(h.leafs, id)
			}
		}

		return nil
	}

	parentID, err := h.reg.GetParent(id)
	if err != nil {
		return errors.Trace(err)
	}

	item := &tagHierItem{
		id:          id,
		parentID:    parentID,
		childrenIDs: make(map[int]struct{}),
	}

	h.idToItem[id] = item

	if isLeaf {
		h.leafs[id] = item
	}

	if item.parentID != 0 {
		if err := h.addInternal(item.parentID, false); err != nil {
			return errors.Trace(err)
		}

		h.idToItem[item.parentID].childrenIDs[id] = struct{}{}
	} else {
		// this is a root item
		h.roots[id] = item
	}

	return nil
}

func (h *TagHier) Move(id, newParentID int, removeNewLeafs bool) error {
	if _, ok := h.idToItem[id]; !ok {
		return errors.Errorf("can't move %d to %d: no item with id %d", id, newParentID, id)
	}

	if _, ok := h.idToItem[newParentID]; !ok {
		return errors.Errorf("can't move %d to %d: no item with id %d", id, newParentID, newParentID)
	}

	isSubnode, err := h.IsSubnode(newParentID, id)
	if err != nil {
		return errors.Trace(err)
	}

	if newParentID == id || isSubnode {
		return errors.Errorf(
			"can't move %d to %d because the latter is either equal to or is a subnode of the former",
			id, newParentID,
		)
	}

	oldParentID := h.idToItem[id].parentID

	h.idToItem[id].parentID = newParentID

	// Maintain children of the new parent
	h.idToItem[newParentID].childrenIDs[id] = struct{}{}

	// Maintain children of the old parent
	h.removeChild(oldParentID, id, removeNewLeafs)

	// If new parent was a leaf, record that it's not a leaf anymore
	if _, ok := h.leafs[newParentID]; ok {
		delete(h.leafs, newParentID)
	}

	return nil
}

func (h *TagHier) MakeCopy() *TagHier {
	ret := &TagHier{
		idToItem: make(map[int]*tagHierItem),
		leafs:    make(map[int]*tagHierItem),
		roots:    make(map[int]*tagHierItem),
		reg:      h.reg,
	}
	for k, v := range h.idToItem {
		vCopy := v.makeCopy()
		ret.idToItem[k] = vCopy
		if _, ok := h.leafs[k]; ok {
			ret.leafs[k] = vCopy
		}
		if _, ok := h.roots[k]; ok {
			ret.roots[k] = vCopy
		}
	}
	return ret
}

func (hi *tagHierItem) makeCopy() *tagHierItem {
	ret := &tagHierItem{
		id:          hi.id,
		parentID:    hi.parentID,
		childrenIDs: make(map[int]struct{}),
	}

	for k, _ := range hi.childrenIDs {
		ret.childrenIDs[k] = struct{}{}
	}

	return ret
}

func (h *TagHier) removeChild(parentID, oldChildID int, removeNewLeafs bool) {
	delete(h.idToItem[parentID].childrenIDs, oldChildID)

	if len(h.idToItem[parentID].childrenIDs) == 0 {
		if removeNewLeafs {
			h.removeChild(h.idToItem[parentID].parentID, parentID, true)
			delete(h.idToItem, parentID)
		} else {
			h.leafs[parentID] = h.idToItem[parentID]
		}
	}
}

func (h *TagHier) GetLeafs() []int {
	return getKeys(h.leafs)
}

func (h *TagHier) GetRoots() []int {
	return getKeys(h.roots)
}

func (h *TagHier) GetAll() []int {
	return getKeys(h.idToItem)
}

// Returns true if id1 is a subnode of id2.
func (h *TagHier) IsSubnode(id1, id2 int) (bool, error) {
	if _, ok := h.idToItem[id1]; !ok {
		return false, errors.Errorf("can't check if %d is a subnode of %d: no item with id %d", id1, id2, id1)
	}

	if _, ok := h.idToItem[id2]; !ok {
		return false, errors.Errorf("can't check if %d is a subnode of %d: no item with id %d", id1, id2, id2)
	}

	for {
		parentID := h.idToItem[id1].parentID
		if parentID == id2 {
			return true, nil
		} else if parentID == 0 {
			return false, nil
		}

		id1 = parentID
	}
}

func (h *TagHier) GetPath(n int) []int {
	path := []int{}
	for ; n != 0; n = h.idToItem[n].parentID {
		path = append(path, n)
	}

	// Reverse path so that it goes from the root to the leaf
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return path
}

func (h *TagHier) GetParent(n int) int {
	return h.idToItem[n].parentID
}

func getKeys(m map[int]*tagHierItem) []int {
	keys := make([]int, len(m))

	i := 0
	for k := range m {
		keys[i] = k
		i++
	}

	sort.Ints(keys)

	return keys
}

// TODO: move outside, since it just compares ints and has nothing to do
// with the taghier
func GetDiff(current, desired []int) *Diff {
	diff := Diff{}

	cm := make(map[int]struct{})
	dm := make(map[int]struct{})

	for _, k := range current {
		cm[k] = struct{}{}
	}

	for _, k := range desired {
		dm[k] = struct{}{}
	}

	for k := range dm {
		if _, ok := cm[k]; !ok {
			diff.Add = append(diff.Add, k)
		}
	}

	for k := range cm {
		if _, ok := dm[k]; !ok {
			diff.Delete = append(diff.Delete, k)
		}
	}

	return &diff
}
