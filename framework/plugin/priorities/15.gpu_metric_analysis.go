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

type GPUMerticAnalysis struct{}

func (pl GPUMerticAnalysis) Name() string {
	return "GPUMerticAnalysis"
}

func (pl GPUMerticAnalysis) Debugg(nodeInfoCache *r.NodeCache) {
	r.KETI_LOG_L2(fmt.Sprintf("S#15. %s", pl.Name()))
	for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			for _, gpu := range nodeInfo.PluginResult.GPUScores {
				if !gpu.IsFiltered {
					r.KETI_LOG_L1(fmt.Sprintf("-node {%s} gpu {%s} score: %d\n", nodeName, gpu.UUID, gpu.GPUScore))
				}
			}
		}
	}
}

func (pl GPUMerticAnalysis) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			for j, _ := range nodeinfo.GPUMetrics {
				if !nodeinfo.PluginResult.GPUScores[j].IsFiltered {
					gpuScore := float64(100)
					nodeinfo.PluginResult.GPUScores[j].GPUScore += int(math.Round(gpuScore / float64(r.Gs)))
				}
			}
		}
	}
}
