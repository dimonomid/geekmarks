package tagspattern

type MatchDetails struct {
	Path string
}

type TagPather interface {
	Path() string
	// Should return path like "/|foo|/|bar|bar_alias|/|baz|"
	PathAllNames() string
	SetMatchDetails(details *MatchDetails)
}
