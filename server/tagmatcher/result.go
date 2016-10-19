package tagmatcher

type Priority int

const (
	ExactMatch Priority = iota
	BeginMatch
	EndMatch
	MiddleMatch
	FuzzyMatch

	PrioritiesCnt
)

var NoMatch = PrioritiesCnt

type Result struct {
	ItemsMap map[int]struct{}
}

func NewResult() *Result {
	r := Result{
		ItemsMap: make(map[int]struct{}),
	}

	return &r
}

func (r *Result) Add(idx int) {
	r.ItemsMap[idx] = struct{}{}
}

func (r *Result) Len() int {
	return len(r.ItemsMap)
}

func (r *Result) Exists(idx int) bool {
	_, ok := r.ItemsMap[idx]
	return ok
}
