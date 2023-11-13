package predicates

import (
	"fmt"
	r "gpu-scheduler/resourceinfo"
)

type CheckGPUTemperature struct{}

func (pl CheckGPUTemperature) Name() string {
	return "NodeFitsGPUMemory"
}

func (pl CheckGPUTemperature) Debugg() {
	r.KETI_LOG_L2(fmt.Sprintf("[stage] F#12. %s", pl.Name()))
}

func (pl CheckGPUTemperature) Filter(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	// for nodeName, nodeinfo := range nodeInfoCache.NodeInfoList {
	// 	if !nodeinfo.PluginResult.IsFiltered {
	// 		for gpuName, gpu := range nodeinfo.GPUMetrics {
	// 			if !nodeinfo.PluginResult.GPUScores[gpuName].IsFiltered {
	// 				if gpu.GPUTemperature > gpu.GPUSlowdownTemp {
	// 					reason := fmt.Sprintf("GPUTemperature=%d > GPUSlowdownTemp=%d", gpu.GPUTemperature, gpu.GPUSlowdownTemp)
	// 					filterState := r.FilterStatus{r.Unschedulable, pl.Name(), reason, nil}
	// 					nodeinfo.PluginResult.GPUScores[gpuName].FilterGPU(nodeName, gpuName, filterState)
	// 					nodeinfo.PluginResult.GPUCountDown()
	// 				}
	// 			}
	// 		}
	// 	}
	// }
}
