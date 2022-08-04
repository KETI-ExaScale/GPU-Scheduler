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

type NodeMetricBasedScoring struct{}

func (pl NodeMetricBasedScoring) Name() string {
	return "NodeMetricBasedScoring"
}

func (pl NodeMetricBasedScoring) Debugg(nodeInfoCache *r.NodeCache) {
	fmt.Println("#10. ", pl.Name())
	for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
<<<<<<< HEAD
			fmt.Printf("-node {%s} score: %d\n", nodeName, nodeInfo.PluginResult.NodeScore)
=======
			fmt.Printf("-node {%s} score: %f\n", nodeName, nodeInfo.PluginResult.NodeScore)
>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85
		}
	}
}

func (pl NodeMetricBasedScoring) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			nodeScore := float64(100)
<<<<<<< HEAD
			nodeinfo.PluginResult.NodeScore += int(math.Round(nodeScore / float64(r.Ns)))
=======
			nodeinfo.PluginResult.NodeScore += math.Round(nodeScore * (1 / r.Ns))
>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85
		}
	}

}
