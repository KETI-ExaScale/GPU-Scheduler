package predicates

type topologyPair struct {
	key   string
	value string
}

type preFilterState9 struct {
	Constraints          []topologySpreadConstraint
	TpKeyToCriticalPaths map[string]*criticalPaths
	TpKeyToDomainsNum    map[string]int
	TpPairToMatchNum     map[topologyPair]int
}

// type preFilterState10 struct {
// 	existingAntiAffinityCounts topologyToMatchedTermCount
// 	affinityCounts             topologyToMatchedTermCount
// 	antiAffinityCounts         topologyToMatchedTermCount
// 	podInfo                    *r.PodInfo
// 	namespaceLabels            labels.Set
// }
