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
	"strings"

	r "gpu-scheduler/resourceinfo"

	corev1 "k8s.io/api/core/v1"
)

const (
	mb                    int64 = 1024 * 1024
	minThreshold          int64 = 23 * mb
	maxContainerThreshold int64 = 1000 * mb
)

type ImageLocality struct{}

func (pl ImageLocality) Name() string {
	return "ImageLocality"
}

func (pl ImageLocality) Debugg(nodeInfoCache *r.NodeCache) {
	fmt.Println("#6. ", pl.Name())
	for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			fmt.Printf("-node {%s} score: %d\n", nodeName, nodeInfo.PluginResult.NodeScore)
		}
	}
}

func (pl ImageLocality) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {

	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			nodeScore := int(0)
			nodeScore = int(calculatePriority(sumImageScores(nodeinfo, newPod.Pod.Spec.Containers, nodeInfoCache.AvailableNodeCount), len(newPod.Pod.Spec.Containers)))
			nodeinfo.PluginResult.NodeScore += int(math.Round(float64(nodeScore / r.Ns)))
		}
	}
}

// sumImageScores returns the sum of image scores of all the containers that are already on the node.
// Each image receives a raw score of its size, scaled by scaledImageScore. The raw scores are later used to calculate
// the final score. Note that the init containers are not considered for it's rare for users to deploy huge init containers.
func sumImageScores(nodeInfo *r.NodeInfo, containers []corev1.Container, totalNumNodes int) int64 {
	var sum int64
	for _, container := range containers {
		if state, ok := nodeInfo.ImageStates[normalizedImageName(container.Image)]; ok {
			sum += scaledImageScore(state, totalNumNodes)
		}
	}
	return sum
}

// calculatePriority returns the priority of a node. Given the sumScores of requested images on the node, the node's
// priority is obtained by scaling the maximum priority value with a ratio proportional to the sumScores.
func calculatePriority(sumScores int64, numContainers int) int64 {
	maxThreshold := maxContainerThreshold * int64(numContainers)
	if sumScores < minThreshold {
		sumScores = minThreshold
	} else if sumScores > maxThreshold {
		sumScores = maxThreshold
	}

	return int64(r.MaxScore) * (sumScores - minThreshold) / (maxThreshold - minThreshold)
}

// normalizedImageName returns the CRI compliant name for a given image.
// TODO: cover the corner cases of missed matches, e.g,
// 1. Using Docker as runtime and docker.io/library/test:tag in pod spec, but only test:tag will present in node status
// 2. Using the implicit registry, i.e., test:tag or library/test:tag in pod spec but only docker.io/library/test:tag
// in node status; note that if users consistently use one registry format, this should not happen.
func normalizedImageName(name string) string {
	if strings.LastIndex(name, ":") <= strings.LastIndex(name, "/") {
		name = name + ":latest"
	}
	return name
}

// scaledImageScore returns an adaptively scaled score for the given state of an image.
// The size of the image is used as the base score, scaled by a factor which considers how much nodes the image has "spread" to.
// This heuristic aims to mitigate the undesirable "node heating problem", i.e., pods get assigned to the same or
// a few nodes due to image locality.
func scaledImageScore(imageState *r.ImageStateSummary, totalNumNodes int) int64 {
	spread := float64(imageState.NumNodes) / float64(totalNumNodes)
	return int64(float64(imageState.Size) * spread)
}
