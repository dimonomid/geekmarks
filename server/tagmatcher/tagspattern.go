package tagmatcher

type TagPather interface {
	Path() string
	// Should return path like "/|foo|/|bar|bar_alias|/|baz|"
	PathItems() [][]string
	SetMatchDetails(details *MatchDetails)
}
