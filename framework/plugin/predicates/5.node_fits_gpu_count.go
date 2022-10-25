package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
)

type NodeFitsGPUCount struct{}

func (pl NodeFitsGPUCount) Name() string {
	return "NodeFitsGPUCount"
}

func (pl NodeFitsGPUCount) Debugg() {
	r.KETI_LOG_L2(fmt.Sprintf("F#5. %s", pl.Name()))
}

func (pl NodeFitsGPUCount) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			if nodeinfo.PluginResult.AvailableGPUCount < newPod.RequestedResource.GPUCount {
				fmt.Println("<test> ", nodeinfo.PluginResult.AvailableGPUCount, newPod.RequestedResource.GPUCount)
				nodeinfo.PluginResult.FilterNode(nodeName, pl.Name())
				nodeInfoCache.NodeCountDown()
				newPod.FilterNode(pl.Name())
			}
		}
	}
}
