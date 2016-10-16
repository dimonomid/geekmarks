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
		patPart := patternItems[patIdx]
		//fmt.Printf("---------- %s\n", patPart)
		res := NewResult()

	Tags:
		for idx, tag := range tags {
			pitems := tag.PathItems()
			//fmt.Printf("* %d: %v\n", idx, pitems)
			for pathCompIdx := fidx[idx] - 1; pathCompIdx >= 0; pathCompIdx-- {
				for nameIdx, tagName := range pitems[pathCompIdx] {
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
						res.Add(idx, prio)
						fidx[idx] = pathCompIdx
						tag.SetMatchDetails(pathCompIdx, nameIdx)
						continue Tags
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
