package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
)

// import (
// 	"errors"
// 	"fmt"
// 	r "gpu-scheduler/resourceinfo"

// 	v1 "k8s.io/api/core/v1"
// 	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
// 	"k8s.io/apimachinery/pkg/labels"
// 	"k8s.io/klog/v2"
// )

type InterPodAffinity struct{}

// const (
// 	ErrReasonExistingAntiAffinityRulesNotMatch = "node(s) didn't satisfy existing pods anti-affinity rules"
// 	ErrReasonAffinityRulesNotMatch             = "node(s) didn't match pod affinity rules"
// 	ErrReasonAntiAffinityRulesNotMatch         = "node(s) didn't match pod anti-affinity rules"
// )

// type podSet map[*v1.Pod]struct{}
// type topologyPairSet map[topologyPair]struct{}
// type topologyPairsMaps struct {
// 	topologyPairToPods map[topologyPair]podSet
// 	podToTopologyPairs map[string]topologyPairSet
// }

func (pl InterPodAffinity) Name() string {
	return "InterPodAffinity"
}

func (pl InterPodAffinity) Debugg() {
	fmt.Println("F#10.", pl.Name())
}

func (pl InterPodAffinity) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {

			// 			node := nodeinfo.Node()
			// 			if node == nil {
			// 				fmt.Println("node not found")
			// 				continue
			// 			}
			// 			if failedPredicates, err := pl.satisfiesExistingPodsAntiAffinity(newPod, meta, nodeinfo); failedPredicates != nil {
			// 				fmt.Println(failedPredicates, "-", err)
			// 				nodeinfo.PluginResult.FilterNode(pl.Name())
			// 				nodeInfoCache.NodeCountDown()
			//              newPod.FilterNode(pl.Name())
			// 				continue
			// 			}

			// 			// Now check if <pod> requirements will be satisfied on this node.
			// 			affinity := newPod.Pod.Spec.Affinity
			// 			if affinity == nil || (affinity.PodAffinity == nil && affinity.PodAntiAffinity == nil) {
			// 				continue
			// 			}
			// 			if failedPredicates, err := pl.satisfiesPodsAffinityAntiAffinity(newPod, meta, nodeinfo, affinity); failedPredicates != nil {
			// 				fmt.Println(failedPredicates, "-", err)
			// 				nodeinfo.PluginResult.FilterNode(pl.Name())
			// 				nodeInfoCache.NodeCountDown()
			//              newPod.FilterNode(pl.Name())
			// 				continue
			// 			}

		}
	}
}

// func (pl InterPodAffinity) satisfiesExistingPodsAntiAffinity(pod *v1.Pod, meta Metadata, nodeInfo *r.NodeInfo) (bool, error) {
// 	node := nodeInfo.Node()

// 	var topologyMaps *topologyPairsMaps
// 	if predicateMeta, ok := meta.(*predicateMetadata); ok {
// 		topologyMaps = predicateMeta.podAffinityMetadata.topologyPairsAntiAffinityPodsMap
// 	} else {
// 		// Filter out pods whose nodeName is equal to nodeInfo.node.Name, but are not
// 		// present in nodeInfo. Pods on other nodes pass the filter.
// 		filteredPods, err := pl.podLister.FilteredList(nodeInfo.Filter, labels.Everything())
// 		if err != nil {
// 			errMessage := fmt.Sprintf("Failed to get all pods: %v", err)
// 			klog.Error(errMessage)
// 			return false, errors.New(errMessage)
// 		}
// 		if topologyMaps, err = pl.getMatchingAntiAffinityTopologyPairsOfPods(pod, filteredPods); err != nil {
// 			errMessage := fmt.Sprintf("Failed to get all terms that match pod %s: %v", pod.Name, err)
// 			klog.Error(errMessage)
// 			return false, errors.New(errMessage)
// 		}
// 	}

// 	for topologyKey, topologyValue := range node.Labels {
// 		if topologyMaps.topologyPairToPods[topologyPair{key: topologyKey, value: topologyValue}] != nil {
// 			return false, fmt.Errorf("Cannot schedule pod %s onto node %s", pod.Name, node.Name)
// 		}
// 	}
// 	return true, nil
// }

// func (pl InterPodAffinity) satisfiesPodsAffinityAntiAffinity(pod *v1.Pod,
// 	meta Metadata, nodeInfo *r.NodeInfo,
// 	affinity *v1.Affinity) (PredicateFailureReason, error) {
// 	node := nodeInfo.Node()
// 	if node == nil {
// 		return ErrPodAffinityRulesNotMatch, fmt.Errorf("node not found")
// 	}
// 	if predicateMeta, ok := meta.(*predicateMetadata); ok {
// 		// Check all affinity terms.
// 		topologyPairsPotentialAffinityPods := predicateMeta.podAffinityMetadata.topologyPairsPotentialAffinityPods
// 		if affinityTerms := GetPodAffinityTerms(affinity.PodAffinity); len(affinityTerms) > 0 {
// 			matchExists := c.nodeMatchesAllTopologyTerms(pod, topologyPairsPotentialAffinityPods, nodeInfo, affinityTerms)
// 			if !matchExists {
// 				// This pod may the first pod in a series that have affinity to themselves. In order
// 				// to not leave such pods in pending state forever, we check that if no other pod
// 				// in the cluster matches the namespace and selector of this pod and the pod matches
// 				// its own terms, then we allow the pod to pass the affinity check.
// 				if !(len(topologyPairsPotentialAffinityPods.topologyPairToPods) == 0 && targetPodMatchesAffinityOfPod(pod, pod)) {
// 					klog.V(10).Infof("Cannot schedule pod %+v onto node %v, because of PodAffinity",
// 						podName(pod), node.Name)
// 					return ErrPodAffinityRulesNotMatch, nil
// 				}
// 			}
// 		}

// 		// Check all anti-affinity terms.
// 		topologyPairsPotentialAntiAffinityPods := predicateMeta.podAffinityMetadata.topologyPairsPotentialAntiAffinityPods
// 		if antiAffinityTerms := GetPodAntiAffinityTerms(affinity.PodAntiAffinity); len(antiAffinityTerms) > 0 {
// 			matchExists := c.nodeMatchesAnyTopologyTerm(pod, topologyPairsPotentialAntiAffinityPods, nodeInfo, antiAffinityTerms)
// 			if matchExists {
// 				klog.V(10).Infof("Cannot schedule pod %+v onto node %v, because of PodAntiAffinity",
// 					podName(pod), node.Name)
// 				return ErrPodAntiAffinityRulesNotMatch, nil
// 			}
// 		}
// 	} else { // We don't have precomputed metadata. We have to follow a slow path to check affinity terms.
// 		filteredPods, err := c.podLister.FilteredList(nodeInfo.Filter, labels.Everything())
// 		if err != nil {
// 			return ErrPodAffinityRulesNotMatch, err
// 		}

// 		affinityTerms := GetPodAffinityTerms(affinity.PodAffinity)
// 		antiAffinityTerms := GetPodAntiAffinityTerms(affinity.PodAntiAffinity)
// 		matchFound, termsSelectorMatchFound := false, false
// 		for _, targetPod := range filteredPods {
// 			// Check all affinity terms.
// 			if !matchFound && len(affinityTerms) > 0 {
// 				affTermsMatch, termsSelectorMatch, err := c.podMatchesPodAffinityTerms(pod, targetPod, nodeInfo, affinityTerms)
// 				if err != nil {
// 					errMessage := fmt.Sprintf("Cannot schedule pod %s onto node %s, because of PodAffinity: %v", podName(pod), node.Name, err)
// 					klog.Error(errMessage)
// 					return ErrPodAffinityRulesNotMatch, errors.New(errMessage)
// 				}
// 				if termsSelectorMatch {
// 					termsSelectorMatchFound = true
// 				}
// 				if affTermsMatch {
// 					matchFound = true
// 				}
// 			}

// 			// Check all anti-affinity terms.
// 			if len(antiAffinityTerms) > 0 {
// 				antiAffTermsMatch, _, err := c.podMatchesPodAffinityTerms(pod, targetPod, nodeInfo, antiAffinityTerms)
// 				if err != nil || antiAffTermsMatch {
// 					klog.V(10).Infof("Cannot schedule pod %+v onto node %v, because of PodAntiAffinityTerm, err: %v",
// 						podName(pod), node.Name, err)
// 					return ErrPodAntiAffinityRulesNotMatch, nil
// 				}
// 			}
// 		}

// 		if !matchFound && len(affinityTerms) > 0 {
// 			// We have not been able to find any matches for the pod's affinity terms.
// 			// This pod may be the first pod in a series that have affinity to themselves. In order
// 			// to not leave such pods in pending state forever, we check that if no other pod
// 			// in the cluster matches the namespace and selector of this pod and the pod matches
// 			// its own terms, then we allow the pod to pass the affinity check.
// 			if termsSelectorMatchFound {
// 				klog.V(10).Infof("Cannot schedule pod %+v onto node %v, because of PodAffinity",
// 					podName(pod), node.Name)
// 				return ErrPodAffinityRulesNotMatch, nil
// 			}
// 			// Check if pod matches its own affinity properties (namespace and label selector).
// 			if !targetPodMatchesAffinityOfPod(pod, pod) {
// 				klog.V(10).Infof("Cannot schedule pod %+v onto node %v, because of PodAffinity",
// 					podName(pod), node.Name)
// 				return ErrPodAffinityRulesNotMatch, nil
// 			}
// 		}
// 	}

// 	if klog.V(10) {
// 		// We explicitly don't do klog.V(10).Infof() to avoid computing all the parameters if this is
// 		// not logged. There is visible performance gain from it.
// 		klog.Infof("Schedule Pod %+v on Node %+v is allowed, pod affinity/anti-affinity constraints satisfied.",
// 			podName(pod), node.Name)
// 	}
// 	return nil, nil
// }

// func (pl InterPodAffinity) getMatchingAntiAffinityTopologyPairsOfPods(pod *v1.Pod, existingPods []*v1.Pod) (*topologyPairsMaps, error) {
// 	topologyMaps := newTopologyPairsMaps()

// 	for _, existingPod := range existingPods {
// 		existingPodNodeInfo, err := c.nodeInfoLister.Get(existingPod.Spec.NodeName)
// 		if err != nil {
// 			klog.Errorf("Pod %s has NodeName %q but node is not found", podName(existingPod), existingPod.Spec.NodeName)
// 			continue
// 		}
// 		existingPodTopologyMaps, err := getMatchingAntiAffinityTopologyPairsOfPod(pod, existingPod, existingPodNodeInfo.Node())
// 		if err != nil {
// 			return nil, err
// 		}
// 		topologyMaps.appendMaps(existingPodTopologyMaps)
// 	}
// 	return topologyMaps, nil
// }

// // returns a pointer to a new topologyPairsMaps
// func newTopologyPairsMaps() *topologyPairsMaps {
// 	return &topologyPairsMaps{
// 		topologyPairToPods: make(map[topologyPair]podSet),
// 		podToTopologyPairs: make(map[string]topologyPairSet),
// 	}
// }

// func getMatchingAntiAffinityTopologyPairsOfPod(newPod *v1.Pod, existingPod *v1.Pod, node *v1.Node) (*topologyPairsMaps, error) {
// 	affinity := existingPod.Spec.Affinity
// 	if affinity == nil || affinity.PodAntiAffinity == nil {
// 		return nil, nil
// 	}

// 	topologyMaps := newTopologyPairsMaps()
// 	for _, term := range GetPodAntiAffinityTerms(affinity.PodAntiAffinity) {
// 		selector, err := metav1.LabelSelectorAsSelector(term.LabelSelector)
// 		if err != nil {
// 			return nil, err
// 		}
// 		namespaces := priorityutil.GetNamespacesFromPodAffinityTerm(existingPod, &term)
// 		if priorityutil.PodMatchesTermsNamespaceAndSelector(newPod, namespaces, selector) {
// 			if topologyValue, ok := node.Labels[term.TopologyKey]; ok {
// 				pair := topologyPair{key: term.TopologyKey, value: topologyValue}
// 				topologyMaps.addTopologyPair(pair, existingPod)
// 			}
// 		}
// 	}
// 	return topologyMaps, nil
// }
