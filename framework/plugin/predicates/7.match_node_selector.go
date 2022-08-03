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
	fmt.Println("#7. ", pl.Name())
}

func (pl MatchNodeSelector) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	if len(newPod.Pod.Spec.NodeSelector) == 0 {
		return
	}

	fmt.Print("- nodes: {")
	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			for key, pod_value := range newPod.Pod.Spec.NodeSelector {
				if node_value, ok := nodeinfo.Node().Labels[key]; ok {
					if pod_value == node_value {
						continue
					}
				}
				nodeinfo.PluginResult.FilterNode(pl.Name())
				nodeInfoCache.NodeCountDown()
				break
			}
		}
		if !nodeinfo.PluginResult.IsFiltered {
			fmt.Print(nodeName, ", ")
		}
	}
	fmt.Println("}")
}
