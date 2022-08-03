package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
)

type CheckNodeReserved struct{}

func (pl CheckNodeReserved) Name() string {
	return "CheckNodeReserved"
}

func (pl CheckNodeReserved) Debugg() {
	fmt.Println("#14. ", pl.Name())
}

func (pl CheckNodeReserved) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	fmt.Print("- nodes: {")
	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			if nodeinfo.Node().Annotations["reserved"] != "" {
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
