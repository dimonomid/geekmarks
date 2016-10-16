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
	ItemsMap   map[int]struct{}
	SubResults []SubResult
}

func NewResult() *Result {
	r := Result{
		SubResults: make([]SubResult, PrioritiesCnt),
		ItemsMap:   make(map[int]struct{}),
	}

	for i, _ := range r.SubResults {
		r.SubResults[i].PResult = &r
		r.SubResults[i].ItemsMap = make(map[int]struct{})
	}

	return &r
}

func (r *Result) Add(idx int, prio Priority) {
	r.SubResults[prio].Add(idx)
}

func (r *Result) Len() int {
	return len(r.ItemsMap)
}

func (r *Result) GetPrio(idx int) Priority {
	if _, ok := r.ItemsMap[idx]; !ok {
		return NoMatch
	}

	for prio, sr := range r.SubResults {
		if _, ok := sr.ItemsMap[idx]; ok {
			return Priority(prio)
		}
	}
	panic("no subresults include item")
}
