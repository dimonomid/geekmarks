// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENSE file for details.

// +build all_tests unit_tests

package tagmatcher

import (
	"reflect"
	"strings"
	"testing"
)

// TagPather impl {{{

type matchDetails struct {
	matchedNameIdx int
	prio           Priority
	det            *MatchDetails
}

type tagDataFlatInternal struct {
	pathItems [][]string
	id        int
	matches   map[int]matchDetails

	pathComponentIdxMax int
	lastComponentPrio   Priority
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
			//add = fmt.Sprintf("(%d)", n.det.Prio)
			//}
		}
		parts[k] = add + names[nameIdx]
	}
	return strings.Join(parts, "/")
}

func (t *tagDataFlatInternal) SetMatchDetails(
	pathComponentIdx, matchedNameIdx int, prio Priority,
	det *MatchDetails,
) {
	t.matches[pathComponentIdx] = matchDetails{
		matchedNameIdx: matchedNameIdx,
		det:            det,
		prio:           prio,
	}
}

func (t *tagDataFlatInternal) SetMaxPathItemIdx(
	pathComponentIdx int, prio Priority,
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

func (t *tagDataFlatInternal) GetPrio() Priority {
	return t.lastComponentPrio
}

// }}}

// TagPather helpers {{{

func stringsToTags(flat []string) []TagPather {
	res := []TagPather{}
	for k, t := range flat {
		cur := &tagDataFlatInternal{
			id:                k,
			matches:           make(map[int]matchDetails),
			lastComponentPrio: NoMatch,
		}

		for _, patPart := range strings.Split(t, "/") {
			cur.pathItems = append(cur.pathItems, strings.Split(patPart, "|"))
		}

		res = append(res, cur)
	}
	return res
}

func tagsToStrings(tags []TagPather) []string {
	res := []string{}

	for _, t := range tags {
		t2 := t.(*tagDataFlatInternal)
		res = append(res, t2.Path())
	}

	return res
}

func compareTags(t *testing.T, got, expected []string) {
	if !reflect.DeepEqual(expected, got) {
		t.Errorf("wrong tags: expected: %v, got: %v", expected, got)
	}
}

func filterAndCompare(
	t *testing.T, strTags []string, pattern string, expected []string,
) {
	tp := stringsToTags(strTags)
	matcher := NewTagMatcher()
	tpFiltered, err := matcher.Filter(tp, pattern)
	if err != nil {
		t.Errorf("%s", err)
	}
	compareTags(t, tagsToStrings(tpFiltered), expected)
}

// }}}

// Tag matching should be case-insensitive, and results should preserve original
// case.
func TestFilterCase(t *testing.T) {
	strTags := []string{
		"/FOObar",
		"/FOObar/asdQWE",
		"/FOObar/asdQWE/XXyy",
	}

	filterAndCompare(t, strTags, "oba/xy", []string{
		"/FOObar/asdQWE/XXyy",
	})

	filterAndCompare(t, strTags, "obA/xY", []string{
		"/FOObar/asdQWE/XXyy",
	})

}

// Tag matching should be case-insensitive, and results should preserve original
// case.
func TestFilterCaseUnicode(t *testing.T) {
	strTags := []string{
		"/ПреВед",
		"/ПреВед/Раздва/",
		"/ПреВед/Раздва/ТРи",
	}

	filterAndCompare(t, strTags, "ев/рИ", []string{
		"/ПреВед/Раздва/ТРи",
	})
}

func TestFilter1(t *testing.T) {
	strTags := []string{
		"/computer",
		"/computer/programming",
		"/computer/programming/ruby",
		"/computer/programming/python",
		"/computer/programming/c++",
		"/computer/programming/c",
		"/computer/programming/go|golang",
		"/computer/programming/javascript",
		"/computer/linux",
		"/computer/linux/udev",
		"/computer/linux/systemd",
		"/computer/linux/kernel",
		"/life",
		"/life/sport",
		"/life/sport/bike|bicycle",
		"/life/sport/kayak",
	}

	filterAndCompare(t, strTags, "c", []string{
		"/computer/programming/c",
		"/computer/programming/c++",
		"/computer",
		"/computer/linux",
		"/computer/programming",
		"/computer/linux/kernel",
		"/computer/linux/systemd",
		"/computer/linux/udev",
		"/computer/programming/go",
		"/computer/programming/python",
		"/computer/programming/ruby",
		"/computer/programming/javascript",
		"/life/sport/bicycle",
	})

	filterAndCompare(t, strTags, "go", []string{
		"/computer/programming/go",
	})

	filterAndCompare(t, strTags, "gol", []string{
		"/computer/programming/golang",
	})

	filterAndCompare(t, strTags, "p", []string{
		"/computer/programming/python",
		"/computer/programming",
		"/computer/programming/c",
		"/computer/programming/c++",
		"/computer/programming/go",
		"/computer/programming/ruby",
		"/computer/programming/javascript",
		"/life/sport",
		"/computer",
		"/life/sport/bike",
		"/life/sport/kayak",
		"/computer/linux",
		"/computer/linux/kernel",
		"/computer/linux/systemd",
		"/computer/linux/udev",
	})

	filterAndCompare(t, strTags, "prog/p", []string{
		"/computer/programming/python",
		"/computer/programming/javascript",
	})

	filterAndCompare(t, strTags, "///k", []string{
		"/computer/linux/kernel",
		"/life/sport/kayak",
		"/life/sport/bike",
	})

	filterAndCompare(t, strTags, "li/ke", []string{
		"/computer/linux/kernel",
		"/life/sport/bike",
	})

	filterAndCompare(t, strTags, "/li/ke", []string{
		"/computer/linux/kernel",
		"/life/sport/bike",
	})

	filterAndCompare(t, strTags, "li com", []string{
		"/computer/linux",
		"/computer/linux/kernel",
		"/computer/linux/systemd",
		"/computer/linux/udev",
	})
}

func TestFilterSame(t *testing.T) {
	strTags := []string{
		"/foo",
		"/foo/foo",
		"/foo/foo/foo",
	}

	filterAndCompare(t, strTags, "f", []string{
		"/foo/foo/foo",
		"/foo/foo",
		"/foo",
	})

	filterAndCompare(t, strTags, "foo", []string{
		"/foo/foo/foo",
		"/foo/foo",
		"/foo",
	})
}
