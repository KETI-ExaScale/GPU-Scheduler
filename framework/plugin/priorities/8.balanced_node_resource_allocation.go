// Copyright 2016 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package priorities

import (
	"fmt"
	"math"

	r "gpu-scheduler/resourceinfo"
)

type BalancedNodeResourceAllocation struct{}

func (pl BalancedNodeResourceAllocation) Name() string {
	return "BalancedNodeResourceAllocation"
}

func (pl BalancedNodeResourceAllocation) Debugg(nodeInfoCache *r.NodeCache) {
	r.KETI_LOG_L2(fmt.Sprintf("S#8. %s", pl.Name()))
	for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			r.KETI_LOG_L1(fmt.Sprintf("-node {%s} score: %d", nodeName, nodeInfo.PluginResult.NodeScore))
		}
	}
}

func (pl BalancedNodeResourceAllocation) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			// allocable := nodeinfo.Allocatable
			milliCPUFree := nodeinfo.NodeMetric.MilliCPUTotal - nodeinfo.NodeMetric.MilliCPUUsed
			memoryFree := nodeinfo.NodeMetric.MemoryTotal - nodeinfo.NodeMetric.MemoryUsed
			storageFree := nodeinfo.NodeMetric.StorageTotal - nodeinfo.NodeMetric.StorageUsed
			requested := newPod.RequestedResource
			nodeScore := float64(0)
			std := float64(0)

			// cpuFraction1 := math.Min(fractionOfCapacity(requested.MilliCPU, allocable.MilliCPU), 1)
			// memoryFraction1 := math.Min(fractionOfCapacity(requested.Memory, allocable.Memory), 1)
			// volumeFraction1 := math.Min(fractionOfCapacity(requested.EphemeralStorage, allocable.EphemeralStorage), 1)

			cpuFraction := math.Min(fractionOfCapacity(requested.MilliCPU, milliCPUFree), 1)
			memoryFraction := math.Min(fractionOfCapacity(requested.Memory, memoryFree), 1)
			storageFraction := math.Min(fractionOfCapacity(requested.EphemeralStorage, storageFree), 1)
			mean := (cpuFraction + memoryFraction + storageFraction) / float64(3)
			variance := float64((((cpuFraction - mean) * (cpuFraction - mean)) +
				((memoryFraction - mean) * (memoryFraction - mean)) +
				((storageFraction - mean) * (storageFraction - mean))) / float64(3))
			std = math.Sqrt(variance / float64(3))
			nodeScore = (float64(1) - std) * float64(r.MaxScore/20)
			// fmt.Println(cpuFraction, memoryFraction, storageFraction, mean, variance, std, nodeScore)
			nodeinfo.PluginResult.NodeScore += int(math.Round(nodeScore))
		}
	}

}

func fractionOfCapacity(req, cap int64) float64 {
	if cap == 0 {
		return 1
	}
	return float64(req) / float64(cap)
}
