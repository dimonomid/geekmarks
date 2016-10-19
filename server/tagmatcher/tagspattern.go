package tagmatcher

type MatchDetails struct {
	// TODO: add a slice of structs like {MatchBegin, MatchLen int}
}

// TODO: find a better name
type TagPather interface {
	Path() string
	PathItems() [][]string
	SetMatchDetails(
		pathComponentIdx, matchedNameIdx int, prio Priority,
		det *MatchDetails,
	)

	SetMaxPathItemIdx(pathComponentIdx int, prio Priority)
	GetMaxPathItemIdx() int
	GetMaxPathItemIdxRev() int
	GetPrio() Priority
}
