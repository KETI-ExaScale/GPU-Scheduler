package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
)

type PodFitsNodeResources struct{}

func (pl PodFitsNodeResources) Name() string {
	return "PodFitsNodeResources"
}

func (pl PodFitsNodeResources) Debugg() {
	r.KETI_LOG_L2(fmt.Sprintf("F#13. %s", pl.Name()))
}

func (pl PodFitsNodeResources) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			allowedPodNumber := nodeinfo.Allocatable.AllowedPodNumber
			if len(nodeinfo.Pods)+1 > allowedPodNumber {
				reason := fmt.Sprintf("nodePodCount=%d+1 < allowedPodNumber=%d", len(nodeinfo.Pods), allowedPodNumber)
				filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), reason, nil}
				nodeinfo.PluginResult.FilterNode(nodeName, filterState)
				nodeInfoCache.NodeCountDown()
				continue
			}

			if newPod.RequestedResource.MilliCPU == 0 &&
				newPod.RequestedResource.Memory == 0 &&
				newPod.RequestedResource.EphemeralStorage == 0 {
				continue
			}

			if newPod.RequestedResource.MilliCPU > (nodeinfo.Allocatable.MilliCPU - nodeinfo.Requested.MilliCPU) {
				reason := fmt.Sprintf("NodeCPUFree=%d < PodRequestedCPU=%d", (nodeinfo.Allocatable.MilliCPU - nodeinfo.Requested.MilliCPU), newPod.RequestedResource.MilliCPU)
				filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), reason, nil}
				nodeinfo.PluginResult.FilterNode(nodeName, filterState)
				nodeInfoCache.NodeCountDown()
				continue
			}
			if newPod.RequestedResource.Memory > (nodeinfo.Allocatable.Memory - nodeinfo.Requested.Memory) {
				reason := fmt.Sprintf("NodeMemoryFree=%d < PodRequestedNodeMemory=%d", (nodeinfo.Allocatable.Memory - nodeinfo.Requested.Memory), newPod.RequestedResource.Memory)
				filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), reason, nil}
				nodeinfo.PluginResult.FilterNode(nodeName, filterState)
				nodeInfoCache.NodeCountDown()
				continue
			}
			if newPod.RequestedResource.EphemeralStorage > (nodeinfo.Allocatable.EphemeralStorage - nodeinfo.Requested.EphemeralStorage) {
				reason := fmt.Sprintf("NodeStorageFree=%d < PodRequestedNodeStorage=%d", (nodeinfo.Allocatable.EphemeralStorage - nodeinfo.Requested.EphemeralStorage), newPod.RequestedResource.EphemeralStorage)
				filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), reason, nil}
				nodeinfo.PluginResult.FilterNode(nodeName, filterState)
				nodeInfoCache.NodeCountDown()
				continue
			}
		}
	}
}
