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

	r "gpu-scheduler/resourceinfo"
)

type GPUPower struct{}

func (pl GPUPower) Name() string {
	return "GPUPower"
}

func (pl GPUPower) Debugg(nodeInfoCache *r.NodeCache) {
	fmt.Println("#17. ", pl.Name())
	for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			for _, gpu := range nodeInfo.PluginResult.GPUScores {
				if !gpu.IsFiltered {
<<<<<<< HEAD
					fmt.Printf("-node {%s} gpu {%s} score: %d\n", nodeName, gpu.UUID, gpu.GPUScore)
=======
					fmt.Printf("-node {%s} gpu {%s} score: %f\n", nodeName, gpu.UUID, gpu.GPUScore)
>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85
				}
			}
		}
	}
}

func (pl GPUPower) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			for j, gpu := range nodeinfo.GPUMetrics {
				if !nodeinfo.PluginResult.GPUScores[j].IsFiltered {
					gpuScore := float64((gpu.GPUPowerTotal - gpu.GPUPowerUsed) / gpu.GPUPowerTotal)
					if gpuScore < 10000 {
						temp := float64(nodeinfo.PluginResult.GPUScores[j].GPUScore) * 0.9
<<<<<<< HEAD
						nodeinfo.PluginResult.GPUScores[j].GPUScore = int(temp)
=======
						nodeinfo.PluginResult.GPUScores[j].GPUScore = temp
>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85
					}
				}
			}
		}
	}
}
