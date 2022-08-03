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
	fmt.Println("#6. ", pl.Name())
}

func (pl PodFitsNodeResources) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	fmt.Print("- nodes: {")
	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			if nodeinfo.NodeMetric.MilliCPUUsed+newPod.RequestedResource.MilliCPU > nodeinfo.NodeMetric.MilliCPUTotal {
				fmt.Println("//cpu", nodeinfo.NodeMetric.MilliCPUUsed, newPod.RequestedResource.MilliCPU)
				nodeinfo.PluginResult.FilterNode(pl.Name())
				nodeInfoCache.NodeCountDown()
				continue
			}
			if nodeinfo.NodeMetric.MemoryUsed+newPod.RequestedResource.Memory > nodeinfo.NodeMetric.MemoryTotal {
				fmt.Println("//memory", nodeinfo.NodeMetric.MemoryUsed, newPod.RequestedResource.Memory)
				nodeinfo.PluginResult.FilterNode(pl.Name())
				nodeInfoCache.NodeCountDown()
				continue
			}
			if nodeinfo.NodeMetric.StorageUsed+newPod.RequestedResource.EphemeralStorage > nodeinfo.NodeMetric.StorageTotal {
				fmt.Println("//storage", nodeinfo.NodeMetric.StorageUsed, newPod.RequestedResource.EphemeralStorage)
				nodeinfo.PluginResult.FilterNode(pl.Name())
				nodeInfoCache.NodeCountDown()
				continue
			}
		}
		if !nodeinfo.PluginResult.IsFiltered {
			fmt.Print(nodeName, ", ")
		}
	}
	fmt.Println("}")
}
