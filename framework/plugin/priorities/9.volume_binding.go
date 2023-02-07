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

type VolumeBinding struct{}

func (pl VolumeBinding) Name() string {
	return "VolumeBinding"
}

func (pl VolumeBinding) Debugg(nodeInfoCache *r.NodeCache) {
	r.KETI_LOG_L2(fmt.Sprintf("S#9. %s", pl.Name()))
	for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			r.KETI_LOG_L1(fmt.Sprintf("-node {%s} score: %d", nodeName, nodeInfo.PluginResult.NodeScore))
		}
	}
}

type UtilizationShapePoint struct {
	// Utilization (x axis). Valid values are 0 to 100. Fully utilized node maps to 100.
	Utilization int32
	// Score assigned to a given utilization (y axis). Valid values are 0 to 10.
	Score int32
}

var Shape []UtilizationShapePoint

func (pl VolumeBinding) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {

	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			nodeScore := float64(0)
			nodeinfo.PluginResult.NodeScore += int(math.Round(nodeScore))
		}
	}

}
