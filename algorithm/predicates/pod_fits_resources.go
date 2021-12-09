package predicates

import (
	"errors"
	"fmt"
	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"
)

func PodFitsResources() error {
	if config.Filtering {
		fmt.Println("[step 1-7] Filtering > PodFitsResources")
	}

	for _, nodeinfo := range resource.NodeInfoList {
		if !nodeinfo.IsFiltered {
			if resource.NewPod.IsGPUPod && nodeinfo.AvailableGPUCount < resource.NewPod.RequestedResource.GPUCount {
				fmt.Println(nodeinfo.AvailableGPUCount, resource.NewPod.RequestedResource.GPUCount)
				nodeinfo.FilterNode()
				continue
			}
			if nodeinfo.NodeMetric.MilliCPUUsed+resource.NewPod.RequestedResource.MilliCPU > nodeinfo.NodeMetric.MilliCPUTotal {
				fmt.Println(nodeinfo.NodeMetric.MilliCPUUsed, resource.NewPod.RequestedResource.MilliCPU)
				nodeinfo.FilterNode()
				continue
			}
			if nodeinfo.NodeMetric.MemoryUsed+resource.NewPod.RequestedResource.Memory > nodeinfo.NodeMetric.MemoryTotal {
				fmt.Println(nodeinfo.NodeMetric.MemoryUsed, resource.NewPod.RequestedResource.Memory)
				nodeinfo.FilterNode()
				continue
			}
			if nodeinfo.NodeMetric.StorageUsed+resource.NewPod.RequestedResource.Storage > nodeinfo.NodeMetric.StorageTotal {
				fmt.Println(nodeinfo.NodeMetric.StorageUsed, resource.NewPod.RequestedResource.Storage)
				nodeinfo.FilterNode()
				continue
			}
		}
	}

	//no node to allocate
	if !resource.IsThereAnyNode() {
		return errors.New("<Failed Stage> pod_fits_resources")
	}

	return nil
}
