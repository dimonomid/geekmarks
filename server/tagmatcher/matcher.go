package tagmatcher

type Matcher interface {
	Filter(tags []TagPather, pattern string) []*Result
}

func SetTagMatchDetails(
	tag TagPather, pathComponentIdx, matchedNameIdx int, prio Priority,
	det *MatchDetails,
) {
	tag.SetMatchDetails(pathComponentIdx, matchedNameIdx, prio, &MatchDetails{})

	if pathComponentIdx > tag.GetMaxPathItemIdx() {
		tag.SetMaxPathItemIdx(pathComponentIdx, prio)
	}
}
