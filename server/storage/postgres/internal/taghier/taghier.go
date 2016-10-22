package taghier

import "sort"

type Registry interface {
	// If GetParent returns 0, it means "no parent"
	GetParent(id int) int
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

func New(reg Registry) *TagHier {
	return &TagHier{
		idToItem: make(map[int]tagHierItem),
		leafs:    make(map[int]tagHierItem),
		reg:      reg,
	}
}

func (h *TagHier) Add(id int) {
	h.addInternal(id, true)
}

func (h *TagHier) addInternal(id int, isLeaf bool) {
	// Id 0 is used as a parent id, to indicate the absence of the parent.
	// So, here we just ignore zero id.
	if id == 0 {
		return
	}

	// If hierarchy already contains given item, return
	if _, ok := h.idToItem[id]; ok {
		// And if needed, remove this item from leafs
		if !isLeaf {
			if _, ok := h.leafs[id]; ok {
				delete(h.leafs, id)
			}
		}

		return
	}

	item := tagHierItem{
		id:       id,
		parentID: h.reg.GetParent(id),
	}

	h.idToItem[id] = item

	if isLeaf {
		h.leafs[id] = item
	}

	h.addInternal(item.parentID, false)
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
