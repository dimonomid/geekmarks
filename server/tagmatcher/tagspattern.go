package tagmatcher

// TODO: find a better name
type TagPather interface {
	// Should return path like "/|foo|/|bar|bar_alias|/|baz|"
	PathItems() [][]string
	// TODO: add a slice of structs like {MatchBegin, MatchLen int}
	SetMatchDetails(pathComponentIdx, matchedNameIdx int)
}
