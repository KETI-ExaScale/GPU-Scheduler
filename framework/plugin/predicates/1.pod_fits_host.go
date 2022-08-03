package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
)

type PodFitsHost struct{}

func (pl PodFitsHost) Name() string {
	return "PodFitsHost"
}

func (pl PodFitsHost) Debugg() {
	fmt.Println("#1. ", pl.Name())
}

func (pl PodFitsHost) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	if len(newPod.Pod.Spec.NodeName) == 0 {
		return
	}

	fmt.Print("- nodes: {")
	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			if newPod.Pod.Spec.NodeName != nodeName {
				nodeinfo.PluginResult.FilterNode(pl.Name())
				nodeInfoCache.NodeCountDown()
			}
		}
		if !nodeinfo.PluginResult.IsFiltered {
			fmt.Print(nodeName, ", ")
		}
	}
	fmt.Println("}")
}
