package predicates

import (
	"fmt"
	resource "gpu-scheduler/resourceinfo"
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
				if !gpu.IsFiltered {
					if !GPUFiltering(gpu, newPod.ExpectedResource) {
						gpu.FilterGPU(nodeinfo)
					}
				}
			}
		}
	}

	return nil
}

//GPU Filtering by {GPU Memory, Temperature}
func GPUFiltering(gpu *resource.GPUMetric, exGPURequest *resource.ExResource) bool {
	//온도 >= 95, 예측메모리 > 가용메모리
	if gpu.GPUTemperature >= 95 || exGPURequest.ExGPUMemory > gpu.GPUMemoryFree {
		return false
	}
	return true
}
