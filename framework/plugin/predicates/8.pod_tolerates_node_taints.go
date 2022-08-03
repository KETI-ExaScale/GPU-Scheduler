package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
)

type PodToleratesNodeTaints struct{}

func (pl PodToleratesNodeTaints) Name() string {
	return "PodToleratesNodeTaints"
}

func (pl PodToleratesNodeTaints) Debugg() {
	fmt.Println("#8. ", pl.Name())
}

func (pl PodToleratesNodeTaints) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	fmt.Print("- nodes: {")
	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			for _, taint := range nodeinfo.Node().Spec.Taints {
				tolerated := false
				for _, toleration := range newPod.Pod.Spec.Tolerations {
					if toleration.ToleratesTaint(&taint) {
						tolerated = true
						break
					}
				}
				if !tolerated {
					nodeinfo.PluginResult.FilterNode(pl.Name())
					nodeInfoCache.NodeCountDown()
					break
				}
			}
		}
		if !nodeinfo.PluginResult.IsFiltered {
			fmt.Print(nodeName, ", ")
		}
	}
	fmt.Println("}")
}
