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
	if config.Debugg {
		fmt.Println("[step 2-1] Scoring > LeastRequestedResource")
	}

	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			allocable := nodeinfo.AvailableResource
			requested := newPod.RequestedResource
			nodeScore := float64(0)

			if (allocable.MilliCPU == 0) || (allocable.MilliCPU > requested.MilliCPU) {
				continue
			} else {
				nodeScore += float64(allocable.MilliCPU-requested.MilliCPU) / float64(allocable.MilliCPU) * 40
			}
			if (allocable.Memory == 0) || (allocable.Memory > requested.Memory) {
				continue
			} else {
				nodeScore += float64(allocable.Memory-requested.Memory) / float64(allocable.Memory) * 40
			}

			if (allocable.EphemeralStorage == 0) || (allocable.EphemeralStorage > requested.EphemeralStorage) {
				continue
			} else {
				nodeScore += float64(allocable.EphemeralStorage-requested.EphemeralStorage) / float64(allocable.EphemeralStorage) * 20
			}

			nodeinfo.NodeScore = int(math.Round(nodeScore * float64(1/config.N)))
		}
	}

	return nil
}
