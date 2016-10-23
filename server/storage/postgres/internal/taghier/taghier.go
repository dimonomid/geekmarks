package taghier

import (
	"github.com/juju/errors"
	"sort"
)

type Registry interface {
	// If GetParent returns 0, it means "no parent"
	GetParent(id int) (int, error)
}

type tagHierItem struct {
	id       int
	parentID int
}

type TagHier struct {
	reg      Registry
	idToItem map[int]tagHierItem
	leafs    map[int]tagHierItem
}

type Diff struct {
	Add    []int
	Delete []int
}

func New(reg Registry) *TagHier {
	return &TagHier{
		idToItem: make(map[int]tagHierItem),
		leafs:    make(map[int]tagHierItem),
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

	item := tagHierItem{
		id:       id,
		parentID: parentID,
	}

	h.idToItem[id] = item

	if isLeaf {
		h.leafs[id] = item
	}

	return h.addInternal(item.parentID, false)
}

func (h *TagHier) GetLeafs() []int {
	return getKeys(h.leafs)
}

func (h *TagHier) GetAll() []int {
	return getKeys(h.idToItem)
}

func getKeys(m map[int]tagHierItem) []int {
	keys := make([]int, len(m))

	i := 0
	for k := range m {
		keys[i] = k
		i++
	}

	sort.Ints(keys)

	return keys
}

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
