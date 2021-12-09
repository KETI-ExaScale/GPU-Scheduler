package predicates

import (
	"errors"
	"fmt"
	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"
)

func PodFitsResources(newPod *resource.Pod) error {
	if config.Filtering {
		fmt.Println("[step 1-7] Filtering > PodFitsResources")
	}

	for _, nodeinfo := range resource.NodeInfoList {
		if !nodeinfo.IsFiltered {
			if newPod.IsGPUPod && nodeinfo.AvailableGPUCount < newPod.RequestedResource.GPUCount {
				fmt.Println(nodeinfo.AvailableGPUCount, newPod.RequestedResource.GPUCount)
				nodeinfo.FilterNode()
				continue
			}
			if nodeinfo.NodeMetric.MilliCPUUsed+newPod.RequestedResource.MilliCPU > nodeinfo.NodeMetric.MilliCPUTotal {
				fmt.Println(nodeinfo.NodeMetric.MilliCPUUsed, newPod.RequestedResource.MilliCPU)
				nodeinfo.FilterNode()
				continue
			}
			if nodeinfo.NodeMetric.MemoryUsed+newPod.RequestedResource.Memory > nodeinfo.NodeMetric.MemoryTotal {
				fmt.Println(nodeinfo.NodeMetric.MemoryUsed, newPod.RequestedResource.Memory)
				nodeinfo.FilterNode()
				continue
			}
			if nodeinfo.NodeMetric.StorageUsed+newPod.RequestedResource.Storage > nodeinfo.NodeMetric.StorageTotal {
				fmt.Println(nodeinfo.NodeMetric.StorageUsed, newPod.RequestedResource.Storage)
				nodeinfo.FilterNode()
				continue
			}
		}
	}

	//no node to allocate
	if !resource.IsThereAnyNode(newPod) {
		return errors.New("<Failed Stage> pod_fits_resources")
	}

	return nil
}
