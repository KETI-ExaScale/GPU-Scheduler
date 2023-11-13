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

type NodeResourcesLeastAllocated struct{}

func (pl NodeResourcesLeastAllocated) Name() string {
	return "NodeResourcesMostAllocated"
}

func (pl NodeResourcesLeastAllocated) Debugg(nodeInfoCache *r.NodeCache) {
	r.KETI_LOG_L2(fmt.Sprintf("[stage] S#7. %s", pl.Name()))
	for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			r.KETI_LOG_L1(fmt.Sprintf("[debugg] node {%s} score: %d", nodeName, nodeInfo.PluginResult.NodeScore))
		}
	}
}

// leastResourceScorer favors nodes with fewer requested resources.
// It calculates the percentage of memory, CPU and other resources requested by pods scheduled on the node, and
// prioritizes based on the minimum of the average of the fraction of requested to capacity.
// Details:
// (cpu((capacity-requested)*MaxNodeScore*cpuWeight/capacity) + memory((capacity-requested)*MaxNodeScore*memoryWeight/capacity) + ...)/weightSum
func (pl NodeResourcesLeastAllocated) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {

	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			allocatable := nodeinfo.Allocatable
			requested := newPod.RequestedResource
			nodeScore := float64(0)

			// milliCPUFree := nodeinfo.NodeMetric.MilliCPUTotal - nodeinfo.NodeMetric.MilliCPUUsed
			// memoryFree := nodeinfo.NodeMetric.MemoryTotal - nodeinfo.NodeMetric.MemoryUsed
			// storageFree := nodeinfo.NodeMetric.StorageTotal - nodeinfo.NodeMetric.StorageUsed

			milliCPUFree := allocatable.MilliCPU
			memoryFree := allocatable.Memory
			storageFree := allocatable.EphemeralStorage

			nodeScore += float64(requested.MilliCPU) / float64(milliCPUFree) * 40
			nodeScore += float64(requested.Memory) / float64(memoryFree) * 40
			nodeScore += float64(requested.EphemeralStorage) / float64(storageFree) * 20

			nodeinfo.PluginResult.NodeScore += int(math.Round(nodeScore * 0.2))

		}
	}
}
