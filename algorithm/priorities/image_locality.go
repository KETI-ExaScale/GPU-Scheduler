// // Copyright 2016 Google Inc. All Rights Reserved.
// // Licensed under the Apache License, Version 2.0 (the "License");
// // you may not use this file except in compliance with the License.
// // You may obtain a copy of the License at
// //
// //     http://www.apache.org/licenses/LICENSE-2.0
// //
// // Unless required by applicable law or agreed to in writing, software
// // distributed under the License is distributed on an "AS IS" BASIS,
// // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// // See the License for the specific language governing permissions and
// // limitations under the License.

package priorities

// import (
// 	"fmt"
// 	"strings"

// 	"gpu-scheduler/config"
// 	resource "gpu-scheduler/resourceinfo"

// 	corev1 "k8s.io/api/core/v1"
// )

// const (
// 	mb                    int64 = 1024 * 1024
// 	minThreshold          int64 = 23 * mb
// 	maxContainerThreshold int64 = 1000 * mb
// )

// func ImageLocality() error {
// 	if config.Scoring {
// 		fmt.Println("[step 2-3] Scoring > ImageLocality")
// 	}

// 	for _, nodeinfo := range resource.NodeInfoList {
// 		if !nodeinfo.IsFiltered {
// 			// allocable := nodeinfo.AvailableResource
// 			// requested := resource.NewPod.RequestedResource
// 			nodeScore := calculatePriority(sumImageScores(nodeinfo, resource.NewPod.Pod.Spec.Containers, len(nodeInfoList)), len(newPod.Pod.Spec.Containers))

// 		}
// 	}

// 	return nil
// }

// func calculatePriority(sumScores int64, numContainers int) int64 {
// 	maxThreshold := maxContainerThreshold * int64(numContainers)
// 	if sumScores < minThreshold {
// 		sumScores = minThreshold
// 	} else if sumScores > maxThreshold {
// 		sumScores = maxThreshold
// 	}

// 	return int64(100) * (sumScores - minThreshold) / (maxThreshold - minThreshold)
// }

// func sumImageScores(nodeinfo *resource.NodeInfo, containers []corev1.Container, totalNumNodes int) int64 {
// 	var sum int64
// 	for _, container := range containers {
// 		if ok := nodeinfo.Node.Status.Images[normalizedImageName(container.Image)]; ok {
// 			sum += scaledImageScore(state, totalNumNodes)
// 		}
// 	}
// 	return sum
// }

// func scaledImageScore(imageState *framework.ImageStateSummary, totalNumNodes int) int64 {
// 	spread := float64(imageState.NumNodes) / float64(totalNumNodes)
// 	return int64(float64(imageState.Size) * spread)
// }

// func normalizedImageName(name string) string {
// 	if strings.LastIndex(name, ":") <= strings.LastIndex(name, "/") {
// 		name = name + ":latest"
// 	}
// 	return name
// }
