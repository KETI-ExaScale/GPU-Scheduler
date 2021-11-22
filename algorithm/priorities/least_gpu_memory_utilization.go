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

func LeastGPUMemoryUtilization(nodeInfoList []*resource.NodeInfo, newPod *resource.Pod) error {
	if config.Scoring {
		fmt.Println("[step 2-10] Scoring > LeastGPUMemoryUtilization")
	}

	for _, nodeinfo := range nodeInfoList {
		if !nodeinfo.IsFiltered {
			for _, gpu := range nodeinfo.GPUMetrics {
				gpuScore := float64(gpu.GPUMemoryFree) / float64(gpu.GPUMemoryTotal) * 100
				gpu.GPUScore += int(math.Round(gpuScore * float64(1/config.G)))

				if config.Score {
					fmt.Printf("{%v : %v} \n", gpu.GPUName, gpu.GPUScore)
				}
			}
		}
	}

	return nil
}
