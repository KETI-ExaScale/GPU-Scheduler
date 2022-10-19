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
	fmt.Println("F#14.", pl.Name())
}

func (pl CheckNodeReserved) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			if nodeinfo.Node().Annotations["reserved"] != "" {
				nodeinfo.PluginResult.FilterNode(nodeName, pl.Name())
				nodeInfoCache.NodeCountDown()
				newPod.FilterNode(pl.Name())
			}
		}
	}
}
