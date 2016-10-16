package tagmatcher

func CombineResults(results []*Result, tags []TagPather) []TagPather {
	restags := []TagPather{}

	for i := Priority(0); i < PrioritiesCnt; i++ {

	Tags:

		for idx, t := range tags {
			prio := NoMatch
			for _, r := range results {
				prio2 := r.GetPrio(idx)
				if prio2 == NoMatch {
					continue Tags
				}

				if prio2 < prio {
					prio = prio2
				}
			}

			if prio == i {
				restags = append(restags, t)
			}
		}
	}

	return restags
}
