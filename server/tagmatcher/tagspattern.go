package tagmatcher

type TagPather interface {
	Path() string
	// Should return path like "/|foo|/|bar|bar_alias|/|baz|"
	PathAllNames() string
	PathItems() [][]string
	SetMatchDetails(details *MatchDetails)
}
