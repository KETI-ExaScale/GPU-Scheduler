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
	r.KETI_LOG_L2(fmt.Sprintf("F#6. %s", pl.Name()))
}

func (pl PodFitsNodeResources) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			if nodeinfo.NodeMetric.MilliCPUUsed+newPod.RequestedResource.MilliCPU > nodeinfo.NodeMetric.MilliCPUTotal {
				fmt.Println("//cpu", nodeinfo.NodeMetric.MilliCPUUsed, newPod.RequestedResource.MilliCPU)
				nodeinfo.PluginResult.FilterNode(nodeName, pl.Name())
				nodeInfoCache.NodeCountDown()
				newPod.FilterNode(pl.Name())
				continue
			}
			if nodeinfo.NodeMetric.MemoryUsed+newPod.RequestedResource.Memory > nodeinfo.NodeMetric.MemoryTotal {
				fmt.Println("//memory", nodeinfo.NodeMetric.MemoryUsed, newPod.RequestedResource.Memory)
				nodeinfo.PluginResult.FilterNode(nodeName, pl.Name())
				nodeInfoCache.NodeCountDown()
				newPod.FilterNode(pl.Name())
				continue
			}
			if nodeinfo.NodeMetric.StorageUsed+newPod.RequestedResource.EphemeralStorage > nodeinfo.NodeMetric.StorageTotal {
				fmt.Println("//storage", nodeinfo.NodeMetric.StorageUsed, newPod.RequestedResource.EphemeralStorage)
				nodeinfo.PluginResult.FilterNode(nodeName, pl.Name())
				nodeInfoCache.NodeCountDown()
				newPod.FilterNode(pl.Name())
				continue
			}
		}
	}
}
