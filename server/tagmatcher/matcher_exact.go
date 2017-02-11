// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENCE file for details.

package tagmatcher

import (
	"strings"
)

type MatcherExact struct {
}

func (m *MatcherExact) Filter(tags []TagPather, pattern string) []*Result {
	results := []*Result{}

	//fmt.Printf("================ %s\n", pattern)
	patternItems := strings.Split(pattern, "/")

	fidx := make([]int, len(tags))
	for idx, tag := range tags {
		pitems := tag.PathItems()
		fidx[idx] = len(pitems)
	}

	for patIdx := len(patternItems) - 1; patIdx >= 0; patIdx-- {
		patPart := strings.ToLower(patternItems[patIdx])
		//fmt.Printf("---------- %s\n", patPart)
		res := NewResult()

	Tags:
		for idx, tag := range tags {
			pitems := tag.PathItems()
			//fmt.Printf("* %d: %v\n", idx, pitems)
			//PathItems:
			for pathCompIdx := fidx[idx] - 1; pathCompIdx >= 0; pathCompIdx-- {
				for nameIdx, tagName := range pitems[pathCompIdx] {
					tagName = strings.ToLower(tagName)
					//fmt.Printf("* %s (%s)\n", tagName, patPart)
					prio := NoMatch
					if tagName == patPart {
						prio = ExactMatch
					} else if strings.HasPrefix(tagName, patPart) {
						prio = BeginMatch
					} else if strings.HasSuffix(tagName, patPart) {
						prio = EndMatch
					} else if strings.Contains(tagName, patPart) {
						prio = MiddleMatch
					}

					if prio != NoMatch {
						res.Add(idx)
						fidx[idx] = pathCompIdx
						SetTagMatchDetails(tag, pathCompIdx, nameIdx, prio, &MatchDetails{})
						continue Tags
						//continue PathItems
					}
				}
			}
		}

		if res.Len() == 0 {
			return []*Result{res}
		}

		results = append(results, res)
	}

	return results
}
