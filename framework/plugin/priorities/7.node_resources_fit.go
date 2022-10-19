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

type NodeResourcesFit struct{}

func (pl NodeResourcesFit) Name() string {
	return "NodeResourcesFit"
}

func (pl NodeResourcesFit) Debugg(nodeInfoCache *r.NodeCache) {
	fmt.Println("S#7. ", pl.Name())
	for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			fmt.Printf("-node {%s} score: %d\n", nodeName, nodeInfo.PluginResult.NodeScore)
		}
	}
}

func (pl NodeResourcesFit) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {

	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			// allocatable := nodeinfo.Allocatable
			requested := newPod.RequestedResource
			nodeScore := float64(0)

			// if (allocatable.MilliCPU == 0) || (allocatable.MilliCPU < requested.MilliCPU) {
			// 	continue
			// } else {
			// 	nodeScore += float64(allocatable.MilliCPU-requested.MilliCPU) / float64(allocatable.MilliCPU) * 40
			// }
			// if (allocatable.Memory == 0) || (allocatable.Memory < requested.Memory) {
			// 	continue
			// } else {
			// 	nodeScore += float64(allocatable.Memory-requested.Memory) / float64(allocatable.Memory) * 40
			// }
			// if (allocatable.EphemeralStorage == 0) || (allocatable.EphemeralStorage < requested.EphemeralStorage) {
			// 	continue
			// } else {
			// 	nodeScore += float64(allocatable.EphemeralStorage-requested.EphemeralStorage) / float64(allocatable.EphemeralStorage) * 20
			// }
			// nodeinfo.PluginResult.NodeScore += int(math.Round(nodeScore * 0.3))

			milliCPUFree := nodeinfo.NodeMetric.MilliCPUTotal - nodeinfo.NodeMetric.MilliCPUUsed
			memoryFree := nodeinfo.NodeMetric.MemoryTotal - nodeinfo.NodeMetric.MemoryUsed
			storageFree := nodeinfo.NodeMetric.StorageTotal - nodeinfo.NodeMetric.StorageUsed
			nodeScore += float64(milliCPUFree-requested.MilliCPU) / float64(milliCPUFree) * 40
			// fmt.Println("cpu: ", float64(milliCPUFree-requested.MilliCPU), float64(milliCPUFree), float64(milliCPUFree-requested.MilliCPU)/float64(milliCPUFree), float64(milliCPUFree-requested.MilliCPU)/float64(milliCPUFree)*40)
			nodeScore += float64(memoryFree-requested.Memory) / float64(memoryFree) * 40
			// fmt.Println("memory: ", float64(memoryFree-requested.Memory), float64(memoryFree), float64(memoryFree-requested.Memory)/float64(memoryFree), float64(memoryFree-requested.Memory)/float64(memoryFree)*40)
			nodeScore += float64(storageFree-requested.EphemeralStorage) / float64(storageFree) * 20
			// fmt.Println("storage: ", float64(storageFree-requested.EphemeralStorage), float64(storageFree), float64(storageFree-requested.EphemeralStorage)/float64(storageFree), float64(storageFree-requested.EphemeralStorage)/float64(storageFree)*20)

			nodeinfo.PluginResult.NodeScore += int(math.Round(nodeScore * 0.2))

		}
	}
}
