package predicates

import (
	"fmt"
	"gpu-scheduler/postevent"
	resource "gpu-scheduler/resourceinfo"
	"log"
)

func CheckGPUAvailable(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	fmt.Println("[step 1-2] Filtering > CheckGPUAvailable")

	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			if nodeinfo.AvailableGPUCount < newPod.RequestedResource.GPUMPS {
				nodeinfo.FilterNode()
				continue
			}
			for _, gpu := range nodeinfo.GPUMetrics {
				if !GPUFiltering(gpu, newPod.ExpectedResource) {
					gpu.FilterGPU(nodeinfo)
				}
			}
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

//GPU Filtering by {GPU Memory, Temperature} +a
func GPUFiltering(gpu *resource.GPUMetric, exGPURequest *resource.ExResource) bool {
	//Temperature >= 95, expectationMemory(현재0) > freeMemory
	if gpu.GPUTemperature >= 95 || exGPURequest.ExGPUMemory > gpu.GPUMemoryFree {
		return false
	}
	return true
}
