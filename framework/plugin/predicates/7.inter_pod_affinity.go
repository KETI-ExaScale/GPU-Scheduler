package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
	"log"
	"sync/atomic"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type InterPodAffinity struct {
	nsLister listersv1.NamespaceLister
}

const (
	// ErrReasonExistingAntiAffinityRulesNotMatch is used for ExistingPodsAntiAffinityRulesNotMatch predicate error.
	ErrReasonExistingAntiAffinityRulesNotMatch = "node(s) didn't satisfy existing pods anti-affinity rules"
	// ErrReasonAffinityRulesNotMatch is used for PodAffinityRulesNotMatch predicate error.
	ErrReasonAffinityRulesNotMatch = "node(s) didn't match pod affinity rules"
	// ErrReasonAntiAffinityRulesNotMatch is used for PodAntiAffinityRulesNotMatch predicate error.
	ErrReasonAntiAffinityRulesNotMatch = "node(s) didn't match pod anti-affinity rules"
)

type preFilterState6 struct {
	// A map of topology pairs to the number of existing pods that has anti-affinity terms that match the "pod".
	existingAntiAffinityCounts topologyToMatchedTermCount
	// A map of topology pairs to the number of existing pods that match the affinity terms of the "pod".
	affinityCounts topologyToMatchedTermCount
	// A map of topology pairs to the number of existing pods that match the anti-affinity terms of the "pod".
	antiAffinityCounts topologyToMatchedTermCount
	// podInfo of the incoming pod.
	podInfo *r.PodInfo
	// A copy of the incoming pod's namespace labels.
	namespaceLabels labels.Set
}

// var nsLister =

type topologyToMatchedTermCount map[topologyPair]int64

func (pl InterPodAffinity) Name() string {
	return "InterPodAffinity"
}

func (pl InterPodAffinity) Debugg() {
	r.KETI_LOG_L2(fmt.Sprintf("[stage] F#6. %s", pl.Name()))
}

func (pl InterPodAffinity) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	state, err := calPreFilterState(newPod, nodeInfoCache)
	if err != nil {
		// run prefilter in interpodaffinity error
	}

	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			if !satisfyPodAffinity(state, nodeinfo) {
				filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), ErrReasonAffinityRulesNotMatch, nil}
				nodeinfo.PluginResult.FilterNode(nodeName, filterState)
				nodeInfoCache.NodeCountDown()
				continue
			}

			if !satisfyPodAntiAffinity(state, nodeinfo) {
				filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), ErrReasonAntiAffinityRulesNotMatch, nil}
				nodeinfo.PluginResult.FilterNode(nodeName, filterState)
				nodeInfoCache.NodeCountDown()
				continue
			}

			if !satisfyExistingPodsAntiAffinity(state, nodeinfo) {
				filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), ErrReasonExistingAntiAffinityRulesNotMatch, nil}
				nodeinfo.PluginResult.FilterNode(nodeName, filterState)
				nodeInfoCache.NodeCountDown()
				continue
			}

		}
	}
}

func calPreFilterState(newPod *r.QueuedPodInfo, nodeInfoCache *r.NodeCache) (*preFilterState6, error) {
	var nodesWithRequiredAntiAffinityPods []*r.NodeInfo
	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			if len(nodeinfo.PodsWithRequiredAntiAffinity) > 0 {
				nodesWithRequiredAntiAffinityPods = append(nodesWithRequiredAntiAffinityPods, nodeinfo)
			}
		}
	}

	s := &preFilterState6{}
	s.podInfo = newPod.PodInfo

	hostConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	hostKubeClient := kubernetes.NewForConfigOrDie(hostConfig)
	informerFactory := informers.NewSharedInformerFactory(hostKubeClient, 0)
	nsLister := informerFactory.Core().V1().Namespaces().Lister()

	for i := range s.podInfo.RequiredAffinityTerms {
		if err := mergeAffinityTermNamespacesIfNotEmpty(&s.podInfo.RequiredAffinityTerms[i], nsLister); err != nil {
			return s, err
		}
	}
	for i := range s.podInfo.RequiredAntiAffinityTerms {
		if err := mergeAffinityTermNamespacesIfNotEmpty(&s.podInfo.RequiredAntiAffinityTerms[i], nsLister); err != nil {
			return s, err
		}
	}
	s.namespaceLabels = GetNamespaceLabelsSnapshot(newPod.Pod.Namespace, nsLister)

	s.existingAntiAffinityCounts = getExistingAntiAffinityCounts(newPod.Pod, s.namespaceLabels, nodesWithRequiredAntiAffinityPods)
	s.affinityCounts, s.antiAffinityCounts = getIncomingAffinityAntiAffinityCounts(s.podInfo, nodeInfoCache)

	return s, nil
}

// Checks if the node satisfies the incoming pod's affinity rules.
func satisfyPodAffinity(state *preFilterState6, nodeInfo *r.NodeInfo) bool {
	podsExist := true
	for _, term := range state.podInfo.RequiredAffinityTerms {
		if topologyValue, ok := nodeInfo.Node().Labels[term.TopologyKey]; ok {
			tp := topologyPair{key: term.TopologyKey, value: topologyValue}
			if state.affinityCounts[tp] <= 0 {
				podsExist = false
			}
		} else {
			// All topology labels must exist on the node.
			return false
		}
	}

	if !podsExist {
		// This pod may be the first pod in a series that have affinity to themselves. In order
		// to not leave such pods in pending state forever, we check that if no other pod
		// in the cluster matches the namespace and selector of this pod, the pod matches
		// its own terms, and the node has all the requested topologies, then we allow the pod
		// to pass the affinity check.
		if len(state.affinityCounts) == 0 && podMatchesAllAffinityTerms(state.podInfo.RequiredAffinityTerms, state.podInfo.Pod) {
			return true
		}
		return false
	}
	return true
}

// Checks if the node satisfies the incoming pod's anti-affinity rules.
func satisfyPodAntiAffinity(state *preFilterState6, nodeInfo *r.NodeInfo) bool {
	if len(state.antiAffinityCounts) > 0 {
		for _, term := range state.podInfo.RequiredAntiAffinityTerms {
			if topologyValue, ok := nodeInfo.Node().Labels[term.TopologyKey]; ok {
				tp := topologyPair{key: term.TopologyKey, value: topologyValue}
				if state.antiAffinityCounts[tp] > 0 {
					return false
				}
			}
		}
	}
	return true
}

// Checks if scheduling the pod onto this node would break any anti-affinity
// terms indicated by the existing pods.
func satisfyExistingPodsAntiAffinity(state *preFilterState6, nodeInfo *r.NodeInfo) bool {
	if len(state.existingAntiAffinityCounts) > 0 {
		// Iterate over topology pairs to get any of the pods being affected by
		// the scheduled pod anti-affinity terms
		for topologyKey, topologyValue := range nodeInfo.Node().Labels {
			tp := topologyPair{key: topologyKey, value: topologyValue}
			if state.existingAntiAffinityCounts[tp] > 0 {
				return false
			}
		}
	}
	return true
}

// calculates the following for each existing pod on each node:
// (1) Whether it has PodAntiAffinity
// (2) Whether any AffinityTerm matches the incoming pod
func getExistingAntiAffinityCounts(pod *v1.Pod, nsLabels labels.Set, nodes []*r.NodeInfo) topologyToMatchedTermCount {
	topoMaps := make([]topologyToMatchedTermCount, len(nodes))
	index := int32(-1)

	for _, nodeinfo := range nodes {
		if !nodeinfo.PluginResult.IsFiltered {
			topoMap := make(topologyToMatchedTermCount)
			for _, existingPod := range nodeinfo.PodsWithRequiredAntiAffinity {
				topoMap.updateWithAntiAffinityTerms(existingPod.RequiredAntiAffinityTerms, pod, nsLabels, nodeinfo.Node(), 1)
			}
			if len(topoMap) != 0 {
				topoMaps[atomic.AddInt32(&index, 1)] = topoMap
			}
		}
	}

	result := make(topologyToMatchedTermCount)
	for i := 0; i <= int(index); i++ {
		result.append(topoMaps[i])
	}

	return result
}

// finds existing Pods that match affinity terms of the incoming pod's (anti)affinity terms.
// It returns a topologyToMatchedTermCount that are checked later by the affinity
// predicate. With this topologyToMatchedTermCount available, the affinity predicate does not
// need to check all the pods in the cluster.
func getIncomingAffinityAntiAffinityCounts(podInfo *r.PodInfo, nodeInfoCache *r.NodeCache) (topologyToMatchedTermCount, topologyToMatchedTermCount) {
	affinityCounts := make(topologyToMatchedTermCount)
	antiAffinityCounts := make(topologyToMatchedTermCount)
	if len(podInfo.RequiredAffinityTerms) == 0 && len(podInfo.RequiredAntiAffinityTerms) == 0 {
		return affinityCounts, antiAffinityCounts
	}

	affinityCountsList := make([]topologyToMatchedTermCount, len(nodeInfoCache.NodeInfoList))
	antiAffinityCountsList := make([]topologyToMatchedTermCount, len(nodeInfoCache.NodeInfoList))
	index := int32(-1)

	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			affinity := make(topologyToMatchedTermCount)
			antiAffinity := make(topologyToMatchedTermCount)
			for _, existingPod := range nodeinfo.Pods {
				affinity.updateWithAffinityTerms(podInfo.RequiredAffinityTerms, existingPod.Pod, nodeinfo.Node(), 1)
				// The incoming pod's terms have the namespaceSelector merged into the namespaces, and so
				// here we don't lookup the existing pod's namespace labels, hence passing nil for nsLabels.
				antiAffinity.updateWithAntiAffinityTerms(podInfo.RequiredAntiAffinityTerms, existingPod.Pod, nil, nodeinfo.Node(), 1)
			}

			if len(affinity) > 0 || len(antiAffinity) > 0 {
				k := atomic.AddInt32(&index, 1)
				affinityCountsList[k] = affinity
				antiAffinityCountsList[k] = antiAffinity
			}
		}
	}

	for i := 0; i <= int(index); i++ {
		affinityCounts.append(affinityCountsList[i])
		antiAffinityCounts.append(antiAffinityCountsList[i])
	}

	return affinityCounts, antiAffinityCounts
}

func (m topologyToMatchedTermCount) append(toAppend topologyToMatchedTermCount) {
	for pair := range toAppend {
		m[pair] += toAppend[pair]
	}
}

// Updates Namespaces with the set of namespaces identified by NamespaceSelector.
// If successful, NamespaceSelector is set to nil.
// The assumption is that the term is for an incoming pod, in which case
// namespaceSelector is either unrolled into Namespaces (and so the selector
// is set to Nothing()) or is Empty(), which means match everything. Therefore,
// there when matching against this term, there is no need to lookup the existing
// pod's namespace labels to match them against term's namespaceSelector explicitly.
func mergeAffinityTermNamespacesIfNotEmpty(at *r.AffinityTerm, nsLister listersv1.NamespaceLister) error {
	if at.NamespaceSelector.Empty() {
		return nil
	}
	ns, err := nsLister.List(at.NamespaceSelector)
	if err != nil {
		return err
	}
	for _, n := range ns {
		at.Namespaces.Insert(n.Name)
	}
	at.NamespaceSelector = labels.Nothing()
	return nil
}

// GetNamespaceLabelsSnapshot returns a snapshot of the labels associated with
// the namespace.
func GetNamespaceLabelsSnapshot(ns string, nsLister listersv1.NamespaceLister) (nsLabels labels.Set) {
	podNS, err := nsLister.Get(ns)
	if err == nil {
		// Create and return snapshot of the labels.
		return labels.Merge(podNS.Labels, nil)
	}
	klog.V(3).InfoS("getting namespace, assuming empty set of namespace labels", "namespace", ns, "err", err)
	return
}

// updates the topologyToMatchedTermCount map with the specified value
// for each affinity term if "targetPod" matches ALL terms.
func (m topologyToMatchedTermCount) updateWithAffinityTerms(
	terms []r.AffinityTerm, pod *v1.Pod, node *v1.Node, value int64) {
	if podMatchesAllAffinityTerms(terms, pod) {
		for _, t := range terms {
			m.update(node, t.TopologyKey, value)
		}
	}
}

// updates the topologyToMatchedTermCount map with the specified value
// for each anti-affinity term matched the target pod.
func (m topologyToMatchedTermCount) updateWithAntiAffinityTerms(terms []r.AffinityTerm, pod *v1.Pod, nsLabels labels.Set, node *v1.Node, value int64) {
	// Check anti-affinity terms.
	for _, t := range terms {
		if t.Matches(pod, nsLabels) {
			m.update(node, t.TopologyKey, value)
		}
	}
}

func (m topologyToMatchedTermCount) update(node *v1.Node, tk string, value int64) {
	if tv, ok := node.Labels[tk]; ok {
		pair := topologyPair{key: tk, value: tv}
		m[pair] += value
		// value could be negative, hence we delete the entry if it is down to zero.
		if m[pair] == 0 {
			delete(m, pair)
		}
	}
}

// returns true IFF the given pod matches all the given terms.
func podMatchesAllAffinityTerms(terms []r.AffinityTerm, pod *v1.Pod) bool {
	if len(terms) == 0 {
		return false
	}
	for _, t := range terms {
		// The incoming pod NamespaceSelector was merged into the Namespaces set, and so
		// we are not explicitly passing in namespace labels.
		if !t.Matches(pod, nil) {
			return false
		}
	}
	return true
}

// New initializes a new plugin and returns it.
func InterPodAffinity_() InterPodAffinity {
	hostConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	hostKubeClient := kubernetes.NewForConfigOrDie(hostConfig)
	informerFactory := informers.NewSharedInformerFactory(hostKubeClient, 0)
	nsLister := informerFactory.Core().V1().Namespaces().Lister()
	return InterPodAffinity{
		nsLister,
	}
}
