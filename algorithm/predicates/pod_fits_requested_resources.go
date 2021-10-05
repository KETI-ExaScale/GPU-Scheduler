package predicates

import (
	"fmt"
	"gpu-scheduler/postevent"
	resource "gpu-scheduler/resourceinfo"
	"log"
)

//단위맞추기****
func PodFitsRequestedResources(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	fmt.Println("[step 1-3] Filtering > PodFitsRequestedResources")

	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			//fmt.Println("|1#GPU|", nodeinfo.AvailableGPUCount, " | ", newPod.RequestedResource.GPUMPS)
			//fmt.Println("|2#CPU|#", nodeinfo.NodeMetric.NodeCPU, " | ", newPod.RequestedResource.MilliCPU, " | ", newPod.ExpectedResource.ExMilliCPU)
			//fmt.Println("|3#Memory|", nodeinfo.NodeMetric.NodeMemory, " | ", newPod.RequestedResource.Memory, " | ", newPod.ExpectedResource.ExMemory)
			//fmt.Println("|4#Storage|", newPod.RequestedResource.EphemeralStorage)
			if nodeinfo.AvailableGPUCount < newPod.RequestedResource.GPUMPS {
				nodeinfo.FilterNode()
				continue
			}
			// if nodeinfo.NodeMetric.NodeCPU < newPod.RequestedResource.MilliCPU ||
			// 	nodeinfo.NodeMetric.NodeCPU < newPod.ExpectedResource.ExMilliCPU {
			// 	nodeinfo.FilterNode()
			// 	continue
			// }
			// if nodeinfo.NodeMetric.NodeMemory < newPod.RequestedResource.Memory ||
			// 	nodeinfo.NodeMetric.NodeMemory < newPod.ExpectedResource.ExMemory {
			// 	nodeinfo.FilterNode()
			// 	continue
			// }
			// if nodeinfo.NodeMetric.NodeCPU < newPod.RequestedResource.EphemeralStorage {
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
	}

	return nil
}
