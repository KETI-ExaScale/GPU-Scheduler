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

type PodTopologySpread struct{}

func (pl PodTopologySpread) Name() string {
	return "PodTopologySpread"
}

func (pl PodTopologySpread) Debugg(nodeInfoCache *r.NodeCache) {
	fmt.Println("#5. ", pl.Name())
	// for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
	// 	if !nodeInfo.PluginResult.IsFiltered {
	// 		fmt.Printf("-node {%s} score: %f\n", nodeName, nodeInfo.PluginResult.NodeScore)
	// 	}
	// }
}

func (pl PodTopologySpread) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			nodeScore, weight := float64(0), int64(0)
			// var topologyScore topologyPairToScore
			// if priorityMeta, ok := meta.(*priorityMetadata); ok {
			// 	topologyScore = priorityMeta.topologyScore
			// }

			// for tpKey, tpValues := range topologyScore {
			// 	if v, exist := nodeinfo.Node.Labels[tpKey]; exist {
			// 		weight += tpValues[v]
			// 	}
			// }
			nodeScore = float64(weight)
			nodeinfo.PluginResult.NodeScore += math.Round(nodeScore * (1 / r.Ns))
		}
	}

}
