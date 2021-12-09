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

	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

func NodeAffinity(newPod *resource.Pod) error {
	if config.Scoring {
		fmt.Println("[step 2-4] Scoring > NodeAffinity")
	}

	affinity := newPod.Pod.Spec.Affinity

	for _, nodeinfo := range resource.NodeInfoList {
		if !nodeinfo.IsFiltered {
			nodeScore, count, weight := float64(0), 0, int32(0)
			if affinity != nil && affinity.NodeAffinity != nil && affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution != nil {
				for i := range affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution {

					preferredSchedulingTerm := &affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution[i]
					if preferredSchedulingTerm.Weight == 0 {
						continue
					}

					nodeSelector, err := NodeSelectorRequirementsAsSelector(preferredSchedulingTerm.Preference.MatchExpressions)
					if err != nil {
						return err
					}

					if nodeSelector.Matches(labels.Set(nodeinfo.Node.Labels)) {
						count += 1
						weight += preferredSchedulingTerm.Weight
					}
				}
			}

			if count > 0 {
				nodeScore = float64(weight) * 1 / float64(count)
			}

			nodeinfo.NodeScore += int(math.Round(nodeScore * float64(1/config.N)))
			if config.Score {
				fmt.Println("nodeinfo.NodeScore: ", nodeinfo.NodeScore)
			}
		}
	}

	return nil
}

func NodeSelectorRequirementsAsSelector(nsm []corev1.NodeSelectorRequirement) (labels.Selector, error) {
	if len(nsm) == 0 {
		return labels.Nothing(), nil
	}
	selector := labels.NewSelector()
	for _, expr := range nsm {
		var op selection.Operator
		switch expr.Operator {
		case corev1.NodeSelectorOpIn:
			op = selection.In
		case corev1.NodeSelectorOpNotIn:
			op = selection.NotIn
		case corev1.NodeSelectorOpExists:
			op = selection.Exists
		case corev1.NodeSelectorOpDoesNotExist:
			op = selection.DoesNotExist
		case corev1.NodeSelectorOpGt:
			op = selection.GreaterThan
		case corev1.NodeSelectorOpLt:
			op = selection.LessThan
		default:
			return nil, fmt.Errorf("%q is not a valid node selector operator", expr.Operator)
		}
		r, err := labels.NewRequirement(expr.Key, op, expr.Values)
		if err != nil {
			return nil, err
		}
		selector = selector.Add(*r)
	}
	return selector, nil
}
