package tagmatcher

type SubResult struct {
	PResult  *Result
	ItemsMap map[int]struct{}
}

func (sr *SubResult) Add(idx int) {
	if _, ok := sr.PResult.ItemsMap[idx]; !ok {
		sr.PResult.ItemsMap[idx] = struct{}{}
		sr.ItemsMap[idx] = struct{}{}
	}
}
