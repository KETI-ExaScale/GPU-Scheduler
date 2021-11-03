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

	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"
)

func BalancedResourveAllocation(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	if config.Debugg {
		fmt.Println("[step 2-2] Scoring > BalancedResourveAllocation")
	}

	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			allocable := nodeinfo.AvailableResource
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

			nodeinfo.NodeScore = int(math.Round(nodeScore * float64(1/config.N)))
		}
	}

	return nil
}

func fractionOfCapacity(req, cap int64) float64 {
	if cap == 0 {
		return 1
	}
	return float64(req) / float64(cap)
}
