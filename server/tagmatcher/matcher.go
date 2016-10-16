package tagmatcher

type mapItem struct {
	// index of the item in the source array
	idx int
}

type MatchDetails struct {
	Path string
}

type Matcher interface {
	Filter(tags []TagPather, pattern string) []*Result
}
