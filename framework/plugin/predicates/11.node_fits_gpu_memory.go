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
	r.KETI_LOG_L2(fmt.Sprintf("F#11. %s", pl.Name()))
}

func (pl NodeFitsGPUMemory) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	// for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
	// 	if !nodeinfo.PluginResult.IsFiltered {
	// 		for gpuName, gpu := range nodeinfo.GPUMetrics {
	// 			if !nodeinfo.PluginResult.GPUScores[gpuName].IsFiltered {
	// 				if gpu.GPUMemoryFree < newPod.RequestedResource.GPUMemoryRequest {
	// 					reason := fmt.Sprintf("GPUMemoryFree=%d < PodRequestedGPUMemory=%d", gpu.GPUMemoryFree, newPod.RequestedResource.GPUMemoryRequest)
	// 					filterState := r.FilterStatus{r.UnschedulableAndUnresolvable, pl.Name(), reason, nil}
	// 					nodeinfo.PluginResult.GPUScores[gpuName].FilterGPU(nodeName, gpuName, filterState)
	// 					nodeinfo.PluginResult.GPUCountDown()
	// 				}
	// 			}
	// 		}
	// 	}
	// }
}
