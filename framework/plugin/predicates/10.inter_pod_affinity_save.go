package predicates

// import (
// 	"fmt"
// 	r "gpu-scheduler/resourceinfo"
// 	"sync/atomic"

// 	v1 "k8s.io/api/core/v1"
// 	"k8s.io/apimachinery/pkg/labels"
// 	listersv1 "k8s.io/client-go/listers/core/v1"
// 	"k8s.io/klog/v2"
// )

// type InterPodAffinity struct {
// 	args         config.InterPodAffinityArgs
// 	sharedLister framework.SharedLister
// 	nsLister     listersv1.NamespaceLister
// }

// const (
// 	ErrReasonExistingAntiAffinityRulesNotMatch = "node(s) didn't satisfy existing pods anti-affinity rules"
// 	ErrReasonAffinityRulesNotMatch             = "node(s) didn't match pod affinity rules"
// 	ErrReasonAntiAffinityRulesNotMatch         = "node(s) didn't match pod anti-affinity rules"
// )

// type topologyToMatchedTermCount map[topologyPair]int64

// func (pl InterPodAffinity) Name() string {
// 	return "InterPodAffinity"
// }

// func (pl InterPodAffinity) Debugg() {
// 	fmt.Println("F#10.", pl.Name())
// }

// // PreFilter invoked at the prefilter extension point.
// func (pl *InterPodAffinity) PreFilter(podInfo *r.PodInfo, nodeInfoCache *r.NodeCache) (*preFilterState10, error) {
// 	var nodesWithRequiredAntiAffinityPods []*r.NodeInfo
// 	var err error
// 	if nodesWithRequiredAntiAffinityPods, err = pl.sharedLister.NodeInfos().HavePodsWithRequiredAntiAffinityList(); err != nil {
// 		return nil, fmt.Errorf("failed to list NodeInfos with pods with affinity: %w", err)
// 	}

// 	s := &preFilterState10{}

// 	s.podInfo = podInfo
// 	if s.podInfo.ParseError != nil {
// 		return nil, fmt.Errorf("parsing pod: %+v", s.podInfo.ParseError)
// 	}

// 	for i := range s.podInfo.RequiredAffinityTerms {
// 		if err := pl.mergeAffinityTermNamespacesIfNotEmpty(&s.podInfo.RequiredAffinityTerms[i]); err != nil {
// 			return nil, err
// 		}
// 	}
// 	for i := range s.podInfo.RequiredAntiAffinityTerms {
// 		if err := pl.mergeAffinityTermNamespacesIfNotEmpty(&s.podInfo.RequiredAntiAffinityTerms[i]); err != nil {
// 			return nil, err
// 		}
// 	}
// 	s.namespaceLabels = GetNamespaceLabelsSnapshot(podInfo.Pod.Namespace, pl.nsLister)

// 	s.existingAntiAffinityCounts = pl.getExistingAntiAffinityCounts(podInfo.Pod, s.namespaceLabels, nodesWithRequiredAntiAffinityPods)
// 	s.affinityCounts, s.antiAffinityCounts = pl.getIncomingAffinityAntiAffinityCounts(s.podInfo, nodeInfoCache)

// 	return s, nil
// }

// func (pl InterPodAffinity) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
// 	fmt.Print("- nodes: {")
// 	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
// 		if !nodeinfo.PluginResult.IsFiltered {

// 			state, err := pl.PreFilter(newPod.PodInfo, nodeInfoCache)
// 			if err != nil {
// 				fmt.Println("inter_pod_affinity prefilter error - ", err)
// 				return
// 			}

// 			if !satisfyPodAffinity(state, nodeinfo) {
// 				nodeinfo.PluginResult.FilterNode(pl.Name() + ErrReasonAffinityRulesNotMatch)
// 				nodeInfoCache.NodeCountDown()
//              newPod.FilterNode(pl.Name())
// 				continue
// 			}

// 			if !satisfyPodAntiAffinity(state, nodeinfo) {
// 				nodeinfo.PluginResult.FilterNode(pl.Name() + ErrReasonAntiAffinityRulesNotMatch)
// 				nodeInfoCache.NodeCountDown()
//              newPod.FilterNode(pl.Name())
// 				continue
// 			}

// 			if !satisfyExistingPodsAntiAffinity(state, nodeinfo) {
// 				nodeinfo.PluginResult.FilterNode(pl.Name() + ErrReasonExistingAntiAffinityRulesNotMatch)
// 				nodeInfoCache.NodeCountDown()
//              newPod.FilterNode(pl.Name())
// 				continue
// 			}
// 		}
// 		if !nodeinfo.PluginResult.IsFiltered {
// 			fmt.Print(nodeName, ", ")
// 		}
// 	}
// 	fmt.Println("}")
// }

// // Checks if scheduling the pod onto this node would break any anti-affinity
// // terms indicated by the existing pods.
// func satisfyExistingPodsAntiAffinity(state *preFilterState10, nodeInfo *r.NodeInfo) bool {
// 	if len(state.existingAntiAffinityCounts) > 0 {
// 		// Iterate over topology pairs to get any of the pods being affected by
// 		// the scheduled pod anti-affinity terms
// 		for topologyKey, topologyValue := range nodeInfo.Node().Labels {
// 			tp := topologyPair{key: topologyKey, value: topologyValue}
// 			if state.existingAntiAffinityCounts[tp] > 0 {
// 				return false
// 			}
// 		}
// 	}
// 	return true
// }

// //  Checks if the node satisfies the incoming pod's anti-affinity rules.
// func satisfyPodAntiAffinity(state *preFilterState10, nodeInfo *r.NodeInfo) bool {
// 	if len(state.antiAffinityCounts) > 0 {
// 		for _, term := range state.podInfo.RequiredAntiAffinityTerms {
// 			if topologyValue, ok := nodeInfo.Node().Labels[term.TopologyKey]; ok {
// 				tp := topologyPair{key: term.TopologyKey, value: topologyValue}
// 				if state.antiAffinityCounts[tp] > 0 {
// 					return false
// 				}
// 			}
// 		}
// 	}
// 	return true
// }

// // Checks if the node satisfies the incoming pod's affinity rules.
// func satisfyPodAffinity(state *preFilterState10, nodeInfo *r.NodeInfo) bool {
// 	podsExist := true
// 	for _, term := range state.podInfo.RequiredAffinityTerms {
// 		if topologyValue, ok := nodeInfo.Node().Labels[term.TopologyKey]; ok {
// 			tp := topologyPair{key: term.TopologyKey, value: topologyValue}
// 			if state.affinityCounts[tp] <= 0 {
// 				podsExist = false
// 			}
// 		} else {
// 			// All topology labels must exist on the node.
// 			return false
// 		}
// 	}

// 	if !podsExist {
// 		if len(state.affinityCounts) == 0 && podMatchesAllAffinityTerms(state.podInfo.RequiredAffinityTerms, state.podInfo.Pod) {
// 			return true
// 		}
// 		return false
// 	}
// 	return true
// }

// // returns true IFF the given pod matches all the given terms.
// func podMatchesAllAffinityTerms(terms []r.AffinityTerm, pod *v1.Pod) bool {
// 	if len(terms) == 0 {
// 		return false
// 	}
// 	for _, t := range terms {
// 		// The incoming pod NamespaceSelector was merged into the Namespaces set, and so
// 		// we are not explicitly passing in namespace labels.
// 		if !t.Matches(pod, nil) {
// 			return false
// 		}
// 	}
// 	return true
// }

// func (pl *InterPodAffinity) mergeAffinityTermNamespacesIfNotEmpty(at *r.AffinityTerm) error {
// 	if at.NamespaceSelector.Empty() {
// 		return nil
// 	}
// 	ns, err := pl.nsLister.List(at.NamespaceSelector)
// 	if err != nil {
// 		return err
// 	}
// 	for _, n := range ns {
// 		at.Namespaces.Insert(n.Name)
// 	}
// 	at.NamespaceSelector = labels.Nothing()
// 	return nil
// }

// func GetNamespaceLabelsSnapshot(ns string, nsLister listersv1.NamespaceLister) (nsLabels labels.Set) {
// 	podNS, err := nsLister.Get(ns)
// 	if err == nil {
// 		// Create and return snapshot of the labels.
// 		return labels.Merge(podNS.Labels, nil)
// 	}
// 	klog.V(3).InfoS("getting namespace, assuming empty set of namespace labels", "namespace", ns, "err", err)
// 	return
// }

// func (pl *InterPodAffinity) getExistingAntiAffinityCounts(pod *v1.Pod, nsLabels labels.Set, cache *r.NodeCache) topologyToMatchedTermCount {
// 	topoMaps := make([]topologyToMatchedTermCount, cache.AvailableNodeCount)
// 	index := int32(-1)
// 	for _, nodeInfo := range cache.NodeInfoList {
// 		node := nodeInfo.Node()
// 		if node == nil {
// 			klog.ErrorS(nil, "Node not found")
// 			continue
// 		}
// 		topoMap := make(topologyToMatchedTermCount)
// 		for _, existingPod := range nodeInfo.PodsWithRequiredAntiAffinity {
// 			topoMap.updateWithAntiAffinityTerms(existingPod.RequiredAntiAffinityTerms, pod, nsLabels, node, 1)
// 		}
// 		if len(topoMap) != 0 {
// 			topoMaps[atomic.AddInt32(&index, 1)] = topoMap
// 		}
// 	}

// 	result := make(topologyToMatchedTermCount)
// 	for i := 0; i <= int(index); i++ {
// 		result.append(topoMaps[i])
// 	}

// 	return result
// }

// func (m topologyToMatchedTermCount) append(toAppend topologyToMatchedTermCount) {
// 	for pair := range toAppend {
// 		m[pair] += toAppend[pair]
// 	}
// }

// func (pl *InterPodAffinity) getIncomingAffinityAntiAffinityCounts(podInfo *r.PodInfo, cache *r.NodeCache) (topologyToMatchedTermCount, topologyToMatchedTermCount) {
// 	affinityCounts := make(topologyToMatchedTermCount)
// 	antiAffinityCounts := make(topologyToMatchedTermCount)
// 	if len(podInfo.RequiredAffinityTerms) == 0 && len(podInfo.RequiredAntiAffinityTerms) == 0 {
// 		return affinityCounts, antiAffinityCounts
// 	}

// 	affinityCountsList := make([]topologyToMatchedTermCount, cache.AvailableNodeCount)
// 	antiAffinityCountsList := make([]topologyToMatchedTermCount, cache.AvailableNodeCount)
// 	index := int32(-1)
// 	for _, nodeInfo := range cache.NodeInfoList {
// 		if !nodeInfo.PluginResult.IsFiltered {
// 			node := nodeInfo.Node()
// 			if node == nil {
// 				klog.ErrorS(nil, "Node not found")
// 				continue
// 			}
// 			affinity := make(topologyToMatchedTermCount)
// 			antiAffinity := make(topologyToMatchedTermCount)
// 			for _, existingPod := range nodeInfo.Pods {
// 				affinity.updateWithAffinityTerms(podInfo.RequiredAffinityTerms, existingPod.Pod, node, 1)
// 				// The incoming pod's terms have the namespaceSelector merged into the namespaces, and so
// 				// here we don't lookup the existing pod's namespace labels, hence passing nil for nsLabels.
// 				antiAffinity.updateWithAntiAffinityTerms(podInfo.RequiredAntiAffinityTerms, existingPod.Pod, nil, node, 1)
// 			}

// 			if len(affinity) > 0 || len(antiAffinity) > 0 {
// 				k := atomic.AddInt32(&index, 1)
// 				affinityCountsList[k] = affinity
// 				antiAffinityCountsList[k] = antiAffinity
// 			}
// 		}
// 	}

// 	for i := 0; i <= int(index); i++ {
// 		affinityCounts.append(affinityCountsList[i])
// 		antiAffinityCounts.append(antiAffinityCountsList[i])
// 	}

// 	return affinityCounts, antiAffinityCounts
// }

// func (m topologyToMatchedTermCount) updateWithAntiAffinityTerms(terms []r.AffinityTerm, pod *v1.Pod, nsLabels labels.Set, node *v1.Node, value int64) {
// 	// Check anti-affinity terms.
// 	for _, t := range terms {
// 		if t.Matches(pod, nsLabels) {
// 			m.update(node, t.TopologyKey, value)
// 		}
// 	}
// }

// func (m topologyToMatchedTermCount) update(node *v1.Node, tk string, value int64) {
// 	if tv, ok := node.Labels[tk]; ok {
// 		pair := topologyPair{key: tk, value: tv}
// 		m[pair] += value
// 		// value could be negative, hence we delete the entry if it is down to zero.
// 		if m[pair] == 0 {
// 			delete(m, pair)
// 		}
// 	}
// }

// func (m topologyToMatchedTermCount) updateWithAffinityTerms(
// 	terms []r.AffinityTerm, pod *v1.Pod, node *v1.Node, value int64) {
// 	if podMatchesAllAffinityTerms(terms, pod) {
// 		for _, t := range terms {
// 			m.update(node, t.TopologyKey, value)
// 		}
// 	}
// }
