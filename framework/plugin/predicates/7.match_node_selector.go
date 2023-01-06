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
				nodeinfo.PluginResult.FilterNode(nodeName, pl.Name())
				nodeInfoCache.NodeCountDown()
				newPod.FilterNode(nodeName, pl.Name(), "")
				break
			}
		}
	}
}
