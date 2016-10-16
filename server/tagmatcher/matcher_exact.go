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
			for i := fidx[idx] - 1; i >= 0; i-- {
				for _, tagName := range pitems[i] {
					//fmt.Printf("* %s (%s)\n", tagName, patPart)
					if tagName == patPart {
						res.Add(idx, ExactMatch)
						fidx[idx] = i
						continue Tags
					} else if strings.HasPrefix(tagName, patPart) {
						res.Add(idx, BeginMatch)
						fidx[idx] = i
						continue Tags
					} else if strings.HasSuffix(tagName, patPart) {
						res.Add(idx, EndMatch)
						fidx[idx] = i
						continue Tags
					} else if strings.Contains(tagName, patPart) {
						res.Add(idx, MiddleMatch)
						fidx[idx] = i
						continue Tags
					}
				}
			}
		}

		if res.Len() == 0 {
			return []*Result{}
		}

		results = append(results, res)
	}

	return results
}
