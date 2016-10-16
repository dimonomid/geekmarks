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
	ItemsMap   map[string]mapItem
	SubResults []SubResult
}

func NewResult() *Result {
	r := Result{
		SubResults: make([]SubResult, PrioritiesCnt),
		ItemsMap:   make(map[string]mapItem),
	}

	for i, _ := range r.SubResults {
		r.SubResults[i].PResult = &r
		r.SubResults[i].ItemsMap = make(map[string]mapItem)
	}

	return &r
}

func (r *Result) Add(item string, idx int, prio Priority) {
	//fmt.Printf("Add: %q, %q\n", item, func(prio Priority) string {
	//switch prio {
	//case ExactMatch:
	//return "exact"
	//case BeginMatch:
	//return "begin"
	//case EndMatch:
	//return "end"
	//case MiddleMatch:
	//return "middle"
	//case FuzzyMatch:
	//return "fuzzy"
	//}
	//return "--"
	//}(prio))
	r.SubResults[prio].Add(item, idx)
}

func (r *Result) Len() int {
	return len(r.ItemsMap)
}

func (r *Result) GetPrio(item string) Priority {
	if _, ok := r.ItemsMap[item]; !ok {
		return NoMatch
	}

	for prio, sr := range r.SubResults {
		if _, ok := sr.ItemsMap[item]; ok {
			return Priority(prio)
		}
	}
	panic("no subresults include item")
}
