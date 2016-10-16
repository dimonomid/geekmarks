package tagmatcher

type MatchDetails struct {
	Path string
}

type Matcher interface {
	Filter(tags []TagPather, pattern string) []*Result
}
