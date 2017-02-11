// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENCE file for details.

package tagmatcher

import (
	"sort"
	"strings"
)

type ByPathItemIdxAndPrio []TagPather

func (a ByPathItemIdxAndPrio) Len() int {
	return len(a)
}

func (a ByPathItemIdxAndPrio) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a ByPathItemIdxAndPrio) Less(i, j int) bool {
	// First of all, consider priority of the last path component
	if a[i].GetPrio() != a[j].GetPrio() {
		return a[i].GetPrio() < a[j].GetPrio()
	}

	// Given equal priorities, pick the one which is closer to the end of path
	if a[i].GetMaxPathItemIdxRev() != a[j].GetMaxPathItemIdxRev() {
		return a[i].GetMaxPathItemIdxRev() < a[j].GetMaxPathItemIdxRev()
	}

	// Now, pick the more deeply nested one (more deeply nested means more
	// refined)
	if a[i].GetMaxPathItemIdx() != a[j].GetMaxPathItemIdx() {
		return a[i].GetMaxPathItemIdx() > a[j].GetMaxPathItemIdx()
	}

	// As a last resort, sort paths lexicographically
	return strings.Compare(a[i].Path(), a[j].Path()) < 0
}

func CombineResults(results []*Result, tags []TagPather) []TagPather {
	restags := []TagPather{}

Tags:
	for idx, tag := range tags {

		for _, r := range results {
			if !r.Exists(idx) {
				continue Tags
			}
		}

		restags = append(restags, tag)
	}

	sort.Sort(ByPathItemIdxAndPrio(restags))

	return restags
}
