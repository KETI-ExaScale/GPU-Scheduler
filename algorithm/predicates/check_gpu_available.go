package predicates

import (
	"errors"
	"fmt"
	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"
)

//GPUFiltering By GPUMemory
func CheckGPUAvailable(newPod *resource.Pod) error {
	if config.Filtering {
		fmt.Println("[step 1-6] Filtering > CheckGPUAvailable")
	}

	for _, nodeinfo := range resource.NodeInfoList {
		if !nodeinfo.IsFiltered {
			for _, gpu := range nodeinfo.GPUMetrics {
				if !gpu.IsFiltered {
					if gpu.GPUMemoryFree < newPod.GPUMemoryRequest {
						gpu.FilterGPU(nodeinfo)
					}
				}
			}
		}
	}

	//no node to allocate
	if !resource.IsThereAnyNode(newPod) {
		return errors.New("<Failed Stage> check_gpu_available")
	}

	return nil
}
