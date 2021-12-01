package predicates

import (
	"errors"
	"fmt"
	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"
)

func CheckGPUAvailable(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	if config.Filtering {
		fmt.Println("[step 1-6] Filtering > CheckGPUAvailable")
	}

	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			if nodeinfo.AvailableGPUCount < newPod.RequestedResource.GPUMPS {
				nodeinfo.FilterNode()
				continue
			}
			// for _, gpu := range nodeinfo.GPUMetrics {
			// 	if !GPUFiltering(gpu, newPod.ExpectedResource) {
			// 		gpu.FilterGPU(nodeinfo)
			// 	}
			// }
		}
	}

	//no node to allocate
	if !resource.IsThereAnyNode(newPod) {
		return errors.New("<Failed Stage> check_gpu_available")
	}

	return nil
}

//GPU Filtering by {GPU Memory, Temperature} +a
func GPUFiltering(gpu *resource.GPUMetric) bool {
	//Temperature >= 95, expectationMemory(현재0) > freeMemory
	if gpu.GPUTemperature >= 95 || 0 > gpu.GPUMemoryFree {
		return false
	}
	return true
}
