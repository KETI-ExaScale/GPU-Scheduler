package predicates

import (
	"fmt"
	"gpu-scheduler/postevent"
	resource "gpu-scheduler/resourceinfo"
	"log"
)

func PodFitsRequestedResources(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	fmt.Println("[step 1-3] Filtering > PodFitsRequestedResources")

	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			fmt.Println("|1#GPU|", nodeinfo.AvailableGPUCount, " | ", newPod.RequestedResource.GPUMPS)
			fmt.Println("|2#CPU|#", nodeinfo.AvailableResource.MilliCPU, " | ", newPod.RequestedResource.MilliCPU, " | ", newPod.ExpectedResource.ExMilliCPU)
			fmt.Println("|3#Memory|", nodeinfo.AvailableResource.Memory, " | ", newPod.RequestedResource.Memory, " | ", newPod.ExpectedResource.ExMemory)
			fmt.Println("|4#Storage|", nodeinfo.AvailableResource.EphemeralStorage, " | ", newPod.RequestedResource.EphemeralStorage)

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
	if *resource.AvailableNodeCount == 0 {
		message := fmt.Sprintf("pod (%s) failed to fit in any node", newPod.Pod.ObjectMeta.Name)
		log.Println(message)
		event := postevent.MakeNoNodeEvent(newPod, message)
		err := postevent.PostEvent(event)
		if err != nil {
			fmt.Println("PodFitsResourcesAndGPU error: ", err)
			return err
		}
		return err
	}

	return nil
}
