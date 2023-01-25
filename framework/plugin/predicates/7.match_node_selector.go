package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
)

type MatchNodeSelector struct{}

func (pl MatchNodeSelector) Name() string {
	return "MatchNodeSelector"
}

func (pl MatchNodeSelector) Debugg() {
	r.KETI_LOG_L2(fmt.Sprintf("F#7. %s", pl.Name()))
}

// func (pl MatchNodeSelector) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
// 	if len(newPod.Pod.Spec.NodeSelector) == 0 {
// 		return
// 	}

// 	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
// 		if !nodeinfo.PluginResult.IsFiltered {
// 			node := nodeinfo.Node
// 			if pl.addedNodeSelector != nil && !pl.addedNodeSelector.Match(node) {
// 				nodeinfo.PluginResult.FilterNode(nodeName, pl.Name())
// 				nodeInfoCache.NodeCountDown()
// 				reason := fmt.Sprintf("node(s) didn't match Pod's node affinity/selector")
// 				newPod.FilterNode(nodeName, pl.Name(), reason)
// 				continue
// 			}

// 			s, err := getPreFilterState(state)
// 			if err != nil {
// 				// Fallback to calculate requiredNodeSelector and requiredNodeAffinity
// 				// here when PreFilter is disabled.
// 				s = &preFilterState{requiredNodeSelectorAndAffinity: nodeaffinity.GetRequiredNodeAffinity(pod)}
// 			}

// 			// Ignore parsing errors for backwards compatibility.
// 			match, _ := s.requiredNodeSelectorAndAffinity.Match(node)
// 			if !match {
// 				nodeinfo.PluginResult.FilterNode(nodeName, pl.Name())
// 				nodeInfoCache.NodeCountDown()
// 				reason := fmt.Sprintf("node(s) didn't match Pod's node affinity/selector")
// 				newPod.FilterNode(nodeName, pl.Name(), reason)
// 				continue
// 			}

// 		}
// 	}
// }

func (pl MatchNodeSelector) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	if len(newPod.Pod.Spec.NodeSelector) == 0 {
		return
	}

	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			for key, pod_value := range newPod.Pod.Spec.NodeSelector {
				if node_value, ok := nodeinfo.Node().Labels[key]; ok {
					if pod_value == node_value {
						continue
					}
				}
				reason := fmt.Sprintf("pod nodeSelector (%s-%s) doesn't match any node labels", key, pod_value)
				filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), reason, nil}
				nodeinfo.PluginResult.FilterNode(nodeName, filterState)
				nodeInfoCache.NodeCountDown()
				break
			}
		}
	}
}
