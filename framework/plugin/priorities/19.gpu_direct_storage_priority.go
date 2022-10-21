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

type GPUDirectStoragePriority struct{}

func (pl GPUDirectStoragePriority) Name() string {
	return "GPUDirectStoragePriority"
}

func (pl GPUDirectStoragePriority) Debugg(nodeInfoCache *r.NodeCache) {
	fmt.Println("S#19.", pl.Name())
	for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			for _, gpu := range nodeInfo.PluginResult.GPUScores {
				if !gpu.IsFiltered {
					fmt.Printf("-node {%s} gpu {%s} score: %d\n", nodeName, gpu.UUID, gpu.GPUScore)
				}
			}
		}
	}
}

func (pl GPUDirectStoragePriority) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {

			// nodeinfo.PluginResult.NodeScore += int(math.Round(nodeScore * float64(1/r.N)))
		}
	}

}