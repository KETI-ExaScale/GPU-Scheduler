package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
)

type InterPodAffinity struct{}

func (pl InterPodAffinity) Name() string {
	return "InterPodAffinity"
}

func (pl InterPodAffinity) Debugg() {
	fmt.Println("#10. ", pl.Name())
}

func (pl InterPodAffinity) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
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
