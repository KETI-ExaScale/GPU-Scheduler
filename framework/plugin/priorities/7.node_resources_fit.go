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

type NodeResourcesFit struct{}

func (pl NodeResourcesFit) Name() string {
	return "NodeResourcesFit"
}

func (pl NodeResourcesFit) Debugg(nodeInfoCache *r.NodeCache) {
	fmt.Println("#7. ", pl.Name())
<<<<<<< HEAD:framework/plugin/priorities/7.node_resources_fit.go
	for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			fmt.Printf("-node {%s} score: %d\n", nodeName, nodeInfo.PluginResult.NodeScore)
		}
	}
=======
	// for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
	// 	if !nodeInfo.PluginResult.IsFiltered {
	// 		fmt.Printf("-node {%s} score: %f\n", nodeName, nodeInfo.PluginResult.NodeScore)
	// 	}
	// }
>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85:algorithm/priorities/least_requested_resource.go
}

func (pl NodeResourcesFit) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {

	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			allocatable := nodeinfo.Allocatable
			requested := newPod.RequestedResource
			nodeScore := float64(0)

			if (allocatable.MilliCPU == 0) || (allocatable.MilliCPU < requested.MilliCPU) {
				continue
			} else {
				nodeScore += float64(allocatable.MilliCPU-requested.MilliCPU) / float64(allocatable.MilliCPU) * 40
			}
			if (allocatable.Memory == 0) || (allocatable.Memory < requested.Memory) {
				continue
			} else {
				nodeScore += float64(allocatable.Memory-requested.Memory) / float64(allocatable.Memory) * 40
			}
			if (allocatable.EphemeralStorage == 0) || (allocatable.EphemeralStorage < requested.EphemeralStorage) {
				continue
			} else {
				nodeScore += float64(allocatable.EphemeralStorage-requested.EphemeralStorage) / float64(allocatable.EphemeralStorage) * 20
			}
<<<<<<< HEAD:framework/plugin/priorities/7.node_resources_fit.go
			nodeinfo.PluginResult.NodeScore += int(math.Round(nodeScore / float64(r.Ns)))
=======
			nodeinfo.PluginResult.NodeScore += math.Round(nodeScore * float64(1/r.Ns))
>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85:algorithm/priorities/least_requested_resource.go

		}
	}
}
