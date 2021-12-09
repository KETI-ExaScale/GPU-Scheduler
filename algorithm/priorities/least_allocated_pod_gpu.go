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

	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"
)

func LeastAllocatedPodGPU(newPod *resource.Pod) error {
	if config.Scoring {
		fmt.Println("[step 2-11] Scoring > LeastAllocatedPodGPU")
	}

	for _, nodeinfo := range resource.NodeInfoList {
		if !nodeinfo.IsFiltered {
			for _, gpu := range nodeinfo.GPUMetrics {
				if !gpu.IsFiltered {
					gpuScore := float64(0)
					if config.LeastPod {
						gpuScore = 100 - float64(gpu.PodCount)*10
					} else {
						gpuScore = float64(gpu.PodCount) * 10
					}
					if gpuScore < 0 {
						gpu.GPUScore = 0
					} else {
						gpu.GPUScore += int(gpuScore * float64(1/config.G))
					}
				}
			}
		}
	}

	return nil
}
