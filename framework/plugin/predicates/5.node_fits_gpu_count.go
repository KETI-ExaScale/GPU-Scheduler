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
	fmt.Println("#5. ", pl.Name())
}

func (pl NodeFitsGPUCount) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	fmt.Print("- nodes: {")
	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			if nodeinfo.PluginResult.AvailableGPUCount < newPod.RequestedResource.GPUCount {
				fmt.Println("//gpu", nodeinfo.PluginResult.AvailableGPUCount, newPod.RequestedResource.GPUCount)
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
