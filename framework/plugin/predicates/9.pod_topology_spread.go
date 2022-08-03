package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
)

type PodTopologySpread struct{}

func (pl PodTopologySpread) Name() string {
	return "PodTopologySpread"
}

func (pl PodTopologySpread) Debugg() {
	fmt.Println("#9. ", pl.Name())
}

func (pl PodTopologySpread) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	fmt.Print("- nodes: {")
	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {

		}
		if !nodeinfo.PluginResult.IsFiltered {
			fmt.Print(nodeName, ", ")
		}
	}
	fmt.Println("}")
}
