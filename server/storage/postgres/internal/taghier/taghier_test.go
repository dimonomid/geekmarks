// +build all_tests unit_tests

package taghier

import (
	"reflect"
	"sort"
	"testing"

	"github.com/juju/errors"
)

// Hierarchy is as follows:
//
// ├── 1
// │   ├── 4
// │   │   └── 7
// │   │       └── 8
// │   ├── 5
// │   │   └── 9
// │   └── 6
// │       └── 10
// │           ├── 11
// │           └── 12
// ├── 2
// │   └── 13
// │       ├── 14
// │       └── 15
// └── 3
//     └── 16
type tmpRegistry struct{}

var (
	allLeafs = []int{8, 9, 11, 12, 14, 15, 16}
)

func (tr *tmpRegistry) GetParent(id int) (int, error) {
	switch id {
	case 0:
		panic("zero id is illegal")
	case 1:
		return 0, nil
	case 2:
		return 0, nil
	case 3:
		return 0, nil

	case 4:
		return 1, nil
	case 5:
		return 1, nil
	case 6:
		return 1, nil

	case 7:
		return 4, nil

	case 8:
		return 7, nil

	case 9:
		return 5, nil

	case 10:
		return 6, nil

	case 11:
		return 10, nil
	case 12:
		return 10, nil

	case 13:
		return 2, nil

	case 14:
		return 13, nil
	case 15:
		return 13, nil

	case 16:
		return 3, nil
	}

	panic("no tag")
}

func TestHier(t *testing.T) {
	reg := tmpRegistry{}
	hier := New(&reg)
	hier.Add(4)
	if err := check(hier, []int{4}, []int{1, 4}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	hier.Add(7)
	if err := check(hier, []int{7}, []int{1, 4, 7}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	hier.Add(1)
	if err := check(hier, []int{7}, []int{1, 4, 7}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	hier.Add(7)
	if err := check(hier, []int{7}, []int{1, 4, 7}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	hier.Add(5)
	if err := check(hier, []int{5, 7}, []int{1, 4, 5, 7}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	hier.Add(12)
	if err := check(hier, []int{5, 7, 12}, []int{1, 4, 5, 6, 7, 10, 12}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	hier.Add(9)
	if err := check(hier, []int{7, 9, 12}, []int{1, 4, 5, 6, 7, 9, 10, 12}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	hier.Add(3)
	if err := check(hier, []int{3, 7, 9, 12}, []int{1, 3, 4, 5, 6, 7, 9, 10, 12}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}
}

func check(hier *TagHier, leafs, all []int) error {
	if !reflect.DeepEqual(hier.GetLeafs(), leafs) {
		return errors.Errorf("leafs are wrong: expected %v, got %v", leafs, hier.GetLeafs())
	}

	if !reflect.DeepEqual(hier.GetAll(), all) {
		return errors.Errorf("all tags are wrong: expected %v, got %v", all, hier.GetAll())
	}

	if err := checkIntegrity(hier); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func TestDiff(t *testing.T) {
	if err := checkDiff([]int{}, []int{1, 3, 4}, []int{1, 3, 4}, []int{}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkDiff([]int{1, 4}, []int{1, 3, 4}, []int{3}, []int{}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkDiff([]int{1, 4, 7, 9}, []int{1, 3, 4}, []int{3}, []int{7, 9}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkDiff([]int{1, 4, 7, 9}, []int{}, []int{}, []int{1, 4, 7, 9}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}
}

func TestSubnode(t *testing.T) {
	if err := checkSubnode([]int{8}, 4, 8, false, false); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkSubnode([]int{8}, 4, 7, false, false); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkSubnode([]int{8}, 4, 4, false, false); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkSubnode([]int{8}, 7, 4, true, false); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkSubnode([]int{8}, 8, 4, true, false); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkSubnode(allLeafs, 14, 1, false, false); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkSubnode(allLeafs, 14, 2, true, false); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkSubnode(allLeafs, 15, 2, true, false); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkSubnode(allLeafs, 12, 1, true, false); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	// non-existing tag ids
	if err := checkSubnode(allLeafs, 100, 2, false, true); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkSubnode(allLeafs, 2, 100, false, true); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkSubnode(allLeafs, 100, 100, false, true); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

}

func TestPath(t *testing.T) {
	if err := checkPath2(allLeafs, 8, []int{1, 4, 7, 8}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkPath2(allLeafs, 7, []int{1, 4, 7}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkPath2(allLeafs, 11, []int{1, 6, 10, 11}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkPath2(allLeafs, 15, []int{2, 13, 15}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}
}

func TestMoveRemoveNewLeafs(t *testing.T) {
	reg := tmpRegistry{}
	hier := New(&reg)

	hier.Add(8)
	if err := check(hier, []int{8}, []int{1, 4, 7, 8}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	hier.Add(9)
	if err := check(hier, []int{8, 9}, []int{1, 4, 5, 7, 8, 9}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	hier.Move(7, 5, true)
	if err := check(hier, []int{8, 9}, []int{1, 5, 7, 8, 9}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkPath(hier, 8, []int{1, 5, 7, 8}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkPath(hier, 9, []int{1, 5, 9}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	hier.Move(7, 9, true)
	if err := check(hier, []int{8}, []int{1, 5, 7, 8, 9}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkPath(hier, 8, []int{1, 5, 9, 7, 8}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	hier.Move(9, 1, true)
	if err := check(hier, []int{8}, []int{1, 7, 8, 9}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	hier.Move(8, 1, true)
	if err := check(hier, []int{8}, []int{1, 8}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}
}

func TestMoveKeepNewLeafs(t *testing.T) {
	reg := tmpRegistry{}
	hier := New(&reg)

	hier.Add(8)
	if err := check(hier, []int{8}, []int{1, 4, 7, 8}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	hier.Add(9)
	if err := check(hier, []int{8, 9}, []int{1, 4, 5, 7, 8, 9}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	hier.Move(7, 5, false)
	if err := check(hier, []int{4, 8, 9}, []int{1, 4, 5, 7, 8, 9}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkPath(hier, 4, []int{1, 4}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkPath(hier, 8, []int{1, 5, 7, 8}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := checkPath(hier, 9, []int{1, 5, 9}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}
}

func TestCopy(t *testing.T) {
	reg := tmpRegistry{}
	hier := New(&reg)

	hier.Add(7)
	if err := check(hier, []int{7}, []int{1, 4, 7}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	hier.Add(9)
	if err := check(hier, []int{7, 9}, []int{1, 4, 5, 7, 9}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	hier2 := hier.MakeCopy()

	if err := check(hier2, []int{7, 9}, []int{1, 4, 5, 7, 9}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	// Add new item to the original taghier: it should not affect the copy
	hier.Add(16)
	if err := check(hier, []int{7, 9, 16}, []int{1, 3, 4, 5, 7, 9, 16}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := check(hier2, []int{7, 9}, []int{1, 4, 5, 7, 9}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	hier.Add(8)
	if err := check(hier, []int{8, 9, 16}, []int{1, 3, 4, 5, 7, 8, 9, 16}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}

	if err := check(hier2, []int{7, 9}, []int{1, 4, 5, 7, 9}); err != nil {
		t.Errorf("%s", errors.Trace(err))
	}
}

func checkDiff(current, desired, add, delete []int) error {
	diff := GetDiff(current, desired)

	if diff.Add == nil {
		diff.Add = []int{}
	}

	if diff.Delete == nil {
		diff.Delete = []int{}
	}

	sort.Ints(diff.Add)
	sort.Ints(diff.Delete)

	if !reflect.DeepEqual(diff.Add, add) {
		return errors.Errorf("diff.add is wrong: expected %v, got %v", add, diff.Add)
	}

	if !reflect.DeepEqual(diff.Delete, delete) {
		return errors.Errorf("diff.delete is wrong: expected %v, got %v", delete, diff.Delete)
	}

	return nil
}

func checkSubnode(nodesToAdd []int, n1, n2 int, isSubnode, isError bool) error {
	reg := tmpRegistry{}
	hier := New(&reg)

	for _, n := range nodesToAdd {
		hier.Add(n)
	}

	got, err := hier.IsSubnode(n1, n2)
	gotError := (err != nil)

	if got != isSubnode || gotError != isError {
		return errors.Errorf("given nodes %v, IsSubnode(%d, %d) should return %v, got %v (expected error: %v, got error: %v)",
			nodesToAdd, n1, n2, isSubnode, got, isError, gotError,
		)
	}

	if err := checkIntegrity(hier); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func checkPath2(nodesToAdd []int, n int, path []int) error {
	reg := tmpRegistry{}
	hier := New(&reg)

	for _, n := range nodesToAdd {
		hier.Add(n)
	}

	if err := checkPath(hier, n, path); err != nil {
		return errors.Annotatef(err, "given nodes %v", nodesToAdd)
	}

	return nil
}

func checkPath(hier *TagHier, n int, path []int) error {
	if got, want := hier.GetPath(n), path; !reflect.DeepEqual(got, want) {
		return errors.Errorf("GetPath(%d) should return %v, got %v",
			n, want, got,
		)
	}

	if err := checkIntegrity(hier); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func checkIntegrity(hier *TagHier) error {
	for id, item := range hier.idToItem {
		actualChildrenIDs := findActualChildren(hier, id)

		if !reflect.DeepEqual(actualChildrenIDs, item.childrenIDs) {
			return errors.Errorf("actual children IDs: %v, saved childrenIDs: %v",
				actualChildrenIDs, item.childrenIDs,
			)
		}

		_, isLeaf := hier.leafs[id]

		if len(actualChildrenIDs) == 0 && !isLeaf {
			return errors.Errorf("item %d: no actual children, but it is NOT in leafs", id)
		} else if len(actualChildrenIDs) > 0 && isLeaf {
			return errors.Errorf("item %d: actual children: %v, but it is in leafs", id, actualChildrenIDs)
		}
	}

	return nil
}

func findActualChildren(hier *TagHier, id int) map[int]struct{} {
	ret := map[int]struct{}{}
	for itemID, item := range hier.idToItem {
		if item.parentID == id {
			ret[itemID] = struct{}{}
		}
	}
	return ret
}
