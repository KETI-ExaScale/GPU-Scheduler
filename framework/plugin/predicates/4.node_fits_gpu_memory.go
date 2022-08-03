package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
)

type NodeFitsGPUMemory struct{}

func (pl NodeFitsGPUMemory) Name() string {
	return "NodeFitsGPUMemory"
}

func (pl NodeFitsGPUMemory) Debugg() {
	fmt.Println("#4. ", pl.Name())
}

func (pl NodeFitsGPUMemory) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			fmt.Print("- nodes: {", nodeName, " : gpu-")
			for gpuName, gpu := range nodeinfo.GPUMetrics {
				if !nodeinfo.PluginResult.GPUScores[gpuName].IsFiltered {
					if gpu.GPUMemoryFree < newPod.RequestedResource.GPUMemoryRequest {
						nodeinfo.PluginResult.GPUScores[gpuName].FilterGPU(pl.Name())
						nodeinfo.PluginResult.GPUCountDown()
					}
				}
				if !nodeinfo.PluginResult.GPUScores[gpuName].IsFiltered {
					fmt.Print(gpuName, ", ")
				}
			}
			fmt.Println("}")
		}
	}
}
