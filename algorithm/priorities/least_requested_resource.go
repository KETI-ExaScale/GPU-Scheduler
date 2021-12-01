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

func LeastRequestedResource(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	if config.Scoring {
		fmt.Println("[step 2-1] Scoring > LeastRequestedResource")
	}

	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			allocatable := nodeinfo.AllocatableResource
			requested := newPod.RequestedResource
			nodeScore := float64(0)

			if (allocatable.MilliCPU == 0) || (allocatable.MilliCPU < requested.MilliCPU) {
				continue
			} else {
				nodeScore += float64(allocatable.MilliCPU-requested.MilliCPU) / float64(allocatable.MilliCPU) * 40
			}
			if (allocatable.Memory == 0) || (allocatable.Memory < requested.Memory) {
				continue
			} else {
				nodeScore += float64(allocatable.Memory-requested.Memory) / float64(allocatable.Memory) * 40
			}
			if (allocatable.EphemeralStorage == 0) || (allocatable.EphemeralStorage < requested.EphemeralStorage) {
				continue
			} else {
				nodeScore += float64(allocatable.EphemeralStorage-requested.EphemeralStorage) / float64(allocatable.EphemeralStorage) * 20
			}
			nodeinfo.NodeScore = int(math.Round(nodeScore * float64(1/config.N)))

			if config.Score {
				fmt.Println("nodeinfo.NodeScore: ", nodeinfo.NodeScore)
			}
		}
	}

	return nil
}
