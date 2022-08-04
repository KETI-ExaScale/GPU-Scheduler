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
<<<<<<< HEAD:framework/plugin/priorities/8.balanced_node_resource_allocation.go

func (pl BalancedNodeResourceAllocation) Debugg(nodeInfoCache *r.NodeCache) {
	fmt.Println("#8. ", pl.Name())
	for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			fmt.Printf("-node {%s} score: %d\n", nodeName, nodeInfo.PluginResult.NodeScore)
		}
	}
}

=======

func (pl BalancedNodeResourceAllocation) Debugg(nodeInfoCache *r.NodeCache) {
	fmt.Println("#8. ", pl.Name())
	// for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
	// 	if !nodeInfo.PluginResult.IsFiltered {
	// 		fmt.Printf("-node {%s} score: %f\n", nodeName, nodeInfo.PluginResult.NodeScore)
	// 	}
	// }
}

>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85:algorithm/priorities/balanced_resource_allocation.go
func (pl BalancedNodeResourceAllocation) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			allocable := nodeinfo.Allocatable
			requested := newPod.RequestedResource
			nodeScore := float64(0)

			cpuFraction := fractionOfCapacity(requested.MilliCPU, allocable.MilliCPU)
			memoryFraction := fractionOfCapacity(requested.Memory, allocable.Memory)
			volumeFraction := fractionOfCapacity(requested.EphemeralStorage, allocable.EphemeralStorage)
			if cpuFraction >= 1 || memoryFraction >= 1 || volumeFraction >= 1 {
				nodeScore = 0
			} else {
				mean := (cpuFraction + memoryFraction + volumeFraction) / float64(3)
				variance := float64((((cpuFraction - mean) * (cpuFraction - mean)) + ((memoryFraction - mean) * (memoryFraction - mean)) + ((volumeFraction - mean) * (volumeFraction - mean))) / float64(3))
				nodeScore = (1 - variance) * 100
			}

<<<<<<< HEAD:framework/plugin/priorities/8.balanced_node_resource_allocation.go
			nodeinfo.PluginResult.NodeScore += int(math.Round(nodeScore / float64(r.Ns)))
=======
			nodeinfo.PluginResult.NodeScore += math.Round(nodeScore * (1 / r.Ns))
>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85:algorithm/priorities/balanced_resource_allocation.go

		}
	}

}

func fractionOfCapacity(req, cap int64) float64 {
	if cap == 0 {
		return 1
	}
	return float64(req) / float64(cap)
}
