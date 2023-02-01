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

type BalancedNodeResourceAllocation struct{}

func (pl BalancedNodeResourceAllocation) Name() string {
	return "BalancedNodeResourceAllocation"
}

func (pl BalancedNodeResourceAllocation) Debugg(nodeInfoCache *r.NodeCache) {
	r.KETI_LOG_L2(fmt.Sprintf("S#8. %s", pl.Name()))
	for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			r.KETI_LOG_L1(fmt.Sprintf("-node {%s} score: %d", nodeName, nodeInfo.PluginResult.NodeScore))
		}
	}
}

func (pl BalancedNodeResourceAllocation) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			var resourceToFractions []float64
			var totalFraction float64

			allocable := nodeinfo.Allocatable
			requested := newPod.RequestedResource

			fraction := float64(requested.MilliCPU) / float64(allocable.MilliCPU)
			if fraction > 1 {
				fraction = 1
			}
			totalFraction += fraction
			resourceToFractions = append(resourceToFractions, fraction)

			fraction = float64(requested.Memory) / float64(allocable.Memory)
			if fraction > 1 {
				fraction = 1
			}
			totalFraction += fraction
			resourceToFractions = append(resourceToFractions, fraction)

			fraction = float64(requested.EphemeralStorage) / float64(allocable.EphemeralStorage)
			if fraction > 1 {
				fraction = 1
			}
			totalFraction += fraction
			resourceToFractions = append(resourceToFractions, fraction)

			std := 0.0

			// For most cases, resources are limited to cpu and memory, the std could be simplified to std := (fraction1-fraction2)/2
			// len(fractions) > 2: calculate std based on the well-known formula - root square of Σ((fraction(i)-mean)^2)/len(fractions)
			// Otherwise, set the std to zero is enough.
			if len(resourceToFractions) == 2 {
				std = math.Abs((resourceToFractions[0] - resourceToFractions[1]) / 2)

			} else if len(resourceToFractions) > 2 {
				mean := totalFraction / float64(len(resourceToFractions))
				var sum float64
				for _, fraction := range resourceToFractions {
					sum = sum + (fraction-mean)*(fraction-mean)
				}
				std = math.Sqrt(sum / float64(len(resourceToFractions)))
			}

			// STD (standard deviation) is always a positive value. 1-deviation lets the score to be higher for node which has least deviation and
			// multiplying it with `MaxNodeScore` provides the scaling factor needed.
			nodeScore := int((1 - std) * float64(r.MaxScore))
			nodeinfo.PluginResult.NodeScore += nodeScore
		}
	}

}
