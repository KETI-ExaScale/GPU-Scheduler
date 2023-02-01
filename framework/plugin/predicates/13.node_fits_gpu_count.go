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
	r.KETI_LOG_L2(fmt.Sprintf("F#12. %s", pl.Name()))
}

func (pl NodeFitsGPUCount) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			if nodeinfo.PluginResult.AvailableGPUCount < newPod.RequestedResource.GPUCount {
				reason := fmt.Sprintf("NodeAvailableGPUCount=%d < PodRequestedGPUCount=%d", nodeinfo.PluginResult.AvailableGPUCount, newPod.RequestedResource.GPUCount)
				filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), reason, nil}
				nodeinfo.PluginResult.FilterNode(nodeName, filterState)
				nodeInfoCache.NodeCountDown()
			}
		}
	}
}
