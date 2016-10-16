package tagmatcher

type SubResult struct {
	PResult  *Result
	ItemsMap map[string]mapItem
	ItemsArr []string
}

func (sr *SubResult) Add(item string, idx int) {
	if _, ok := sr.PResult.ItemsMap[item]; !ok {
		mi := mapItem{idx: idx}
		sr.PResult.ItemsMap[item] = mi
		sr.ItemsMap[item] = mi
		sr.ItemsArr = append(sr.ItemsArr, item)
	}
}
