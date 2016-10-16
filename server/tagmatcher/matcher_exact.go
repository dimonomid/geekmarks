package tagmatcher

import "strings"

type MatcherExact struct {
}

func (m *MatcherExact) Filter(tags []TagPather, pattern string) []*Result {
	results := []*Result{}

	patternItems := strings.Split(pattern, "/")

	for patIdx := len(patternItems) - 1; patIdx >= 0; patIdx-- {
		patNum := len(patternItems) - patIdx - 1
		patPart := patternItems[patIdx]
		res := NewResult()

		for idx, tag := range tags {
			pitems := tag.PathItems()
			for i := len(pitems) - 1 - patNum; i >= 0; i-- {
				for _, tagName := range pitems[i] {
					if tagName == patPart {
						res.Add(tag.PathAllNames(), idx, ExactMatch)
					} else if strings.HasPrefix(tagName, patPart) {
						res.Add(tag.PathAllNames(), idx, BeginMatch)
					} else if strings.HasSuffix(tagName, patPart) {
						res.Add(tag.PathAllNames(), idx, EndMatch)
					} else if strings.Contains(tagName, patPart) {
						res.Add(tag.PathAllNames(), idx, MiddleMatch)
					}
				}
			}
		}

		if res.Len() > 0 {
			results = append(results, res)
		}
	}

	return results
}
