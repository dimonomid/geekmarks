// +build all_tests unit_tests

package taghier

import (
	"reflect"
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

func (tr *tmpRegistry) GetParent(id int) int {
	switch id {
	case 0:
		panic("zero id is illegal")
	case 1:
		return 0
	case 2:
		return 0
	case 3:
		return 0

	case 4:
		return 1
	case 5:
		return 1
	case 6:
		return 1

	case 7:
		return 4

	case 8:
		return 7

	case 9:
		return 5

	case 10:
		return 6

	case 11:
		return 10
	case 12:
		return 10

	case 13:
		return 2

	case 14:
		return 13
	case 15:
		return 13

	case 16:
		return 3
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

	return nil
}
