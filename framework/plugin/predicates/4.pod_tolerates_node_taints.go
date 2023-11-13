package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
)

type PodToleratesNodeTaints struct{}

func (pl PodToleratesNodeTaints) Name() string {
	return "PodToleratesNodeTaints"
}

func (pl PodToleratesNodeTaints) Debugg() {
	r.KETI_LOG_L2(fmt.Sprintf("[stage] F#4. %s", pl.Name()))
}

func (pl PodToleratesNodeTaints) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			filterPredicate := func(t *corev1.Taint) bool {
				// PodToleratesNodeTaints is only interested in NoSchedule and NoExecute taints.
				return t.Effect == corev1.TaintEffectNoSchedule || t.Effect == corev1.TaintEffectNoExecute
			}

			taint, isUntolerated := FindMatchingUntoleratedTaint(nodeinfo.Node().Spec.Taints, newPod.Pod.Spec.Tolerations, filterPredicate)
			if isUntolerated {
				reason := fmt.Sprintf("node(s) had untolerated taint {%s: %s}", taint.Key, taint.Value)
				filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), reason, nil}
				nodeinfo.PluginResult.FilterNode(nodeName, filterState)
				nodeInfoCache.NodeCountDown()
			}
		}
	}
}

type taintsFilterFunc func(*v1.Taint) bool

func FindMatchingUntoleratedTaint(taints []corev1.Taint, tolerations []corev1.Toleration, inclusionFilter taintsFilterFunc) (corev1.Taint, bool) {
	filteredTaints := getFilteredTaints(taints, inclusionFilter)
	for _, taint := range filteredTaints {
		if !TolerationsTolerateTaint(tolerations, &taint) {
			return taint, true
		}
	}
	return corev1.Taint{}, false
}

// getFilteredTaints returns a list of taints satisfying the filter predicate
func getFilteredTaints(taints []v1.Taint, inclusionFilter taintsFilterFunc) []v1.Taint {
	if inclusionFilter == nil {
		return taints
	}
	filteredTaints := []v1.Taint{}
	for _, taint := range taints {
		if !inclusionFilter(&taint) {
			continue
		}
		filteredTaints = append(filteredTaints, taint)
	}
	return filteredTaints
}

// TolerationsTolerateTaint checks if taint is tolerated by any of the tolerations.
func TolerationsTolerateTaint(tolerations []v1.Toleration, taint *v1.Taint) bool {
	for i := range tolerations {
		if tolerations[i].ToleratesTaint(taint) {
			return true
		}
	}
	return false
}

// func (pl PodToleratesNodeTaints) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
// 	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
// 		if !nodeinfo.PluginResult.IsFiltered {
// 			for _, taint := range nodeinfo.Node().Spec.Taints {
// 				tolerated := false
// 				for _, toleration := range newPod.Pod.Spec.Tolerations {
// 					if toleration.ToleratesTaint(&taint) {
// 						tolerated = true
// 						break
// 					}
// 				}
// 				if !tolerated {
// 					nodeinfo.PluginResult.FilterNode(nodeName, pl.Name())
// 					nodeInfoCache.NodeCountDown()
// 					newPod.FilterNode(nodeName, pl.Name(), fmt.Sprintf("node had taints that the pod didn't tolerate {%s}", taint))
// 					break
// 				}
// 			}
// 		}
// 	}
// }
