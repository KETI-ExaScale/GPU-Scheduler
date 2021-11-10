package predicates

import (
	"errors"
	"fmt"
	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"
)

func PodFitsResources(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	if config.Filtering {
		fmt.Println("[step 1-7] Filtering > PodFitsResources")
	}

	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			// fmt.Println(" |1#GPU|", nodeinfo.AvailableGPUCount, " | ", newPod.RequestedResource.GPUMPS)
			// fmt.Println(" |2#CPU|#", nodeinfo.AvailableResource.MilliCPU, " | ", newPod.RequestedResource.MilliCPU, " | ", newPod.ExpectedResource.ExMilliCPU)
			// fmt.Println(" |3#Memory|", nodeinfo.AvailableResource.Memory, " | ", newPod.RequestedResource.Memory, " | ", newPod.ExpectedResource.ExMemory)
			// fmt.Println(" |4#Storage|", nodeinfo.AvailableResource.EphemeralStorage, " | ", newPod.RequestedResource.EphemeralStorage)

			if nodeinfo.AvailableGPUCount < newPod.RequestedResource.GPUMPS {
				nodeinfo.FilterNode()
				continue
			}
			if nodeinfo.AvailableResource.MilliCPU < newPod.RequestedResource.MilliCPU ||
				nodeinfo.AvailableResource.MilliCPU < newPod.ExpectedResource.ExMilliCPU {
				nodeinfo.FilterNode()
				continue
			}
			if nodeinfo.AvailableResource.Memory < newPod.RequestedResource.Memory ||
				nodeinfo.AvailableResource.Memory < newPod.ExpectedResource.ExMemory {
				nodeinfo.FilterNode()
				continue
			}
			// if int(nodeinfo.AvailableResource.EphemeralStorage) < newPod.RequestedResource.EphemeralStorage {
			// 	nodeinfo.FilterNode()
			// 	continue
			// }

		}
	}

	//no node to allocate
	if !resource.IsThereAnyNode(newPod) {
		return errors.New("<Failed Stage> pod_fits_resources")
	}

	return nil
}
