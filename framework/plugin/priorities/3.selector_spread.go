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

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
)

type SelectorSpread struct {
	services               corelisters.ServiceLister
	replicationControllers corelisters.ReplicationControllerLister
	replicaSets            appslisters.ReplicaSetLister
	statefulSets           appslisters.StatefulSetLister
}

var (
	rcKind = v1.SchemeGroupVersion.WithKind("ReplicationController")
	rsKind = appsv1.SchemeGroupVersion.WithKind("ReplicaSet")
	ssKind = appsv1.SchemeGroupVersion.WithKind("StatefulSet")
)

func (pl SelectorSpread) Name() string {
	return "SelectorSpread"
}

func (pl SelectorSpread) Debugg(nodeInfoCache *r.NodeCache) {
	r.KETI_LOG_L2(fmt.Sprintf("S#3. %s", pl.Name()))
	for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			r.KETI_LOG_L1(fmt.Sprintf("-node {%s} score: %d\n", nodeName, nodeInfo.PluginResult.NodeScore))
		}
	}
}

func (pl SelectorSpread) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	// if skipSelectorSpread(newPod.Pod) {
	// 	return
	// }
	// selector := DefaultSelector(
	// 	newPod.Pod,
	// 	pl.services,
	// 	pl.replicationControllers,
	// 	pl.replicaSets,
	// 	pl.statefulSets,
	// )
	// state := &preScoreState3{
	// 	selector: selector,
	// }

	// for _, nodeinfo := range nodeInfoCache.NodeInfoList {
	// 	if !nodeinfo.PluginResult.IsFiltered {
	// 		count := countMatchingPods(newPod.Pod.Namespace, state.selector, nodeinfo)
	//		fmt.Println("scoring>selector_spread: ", count, "scode, node: ", nodeinfo.Node().Name)
	// 		nodeinfo.PluginResult.NodeScore += int(math.Round(float64(count)))
	// 	}
	// }
}

func skipSelectorSpread(pod *v1.Pod) bool {
	return len(pod.Spec.TopologySpreadConstraints) != 0
}

func GetPodServices(sl corelisters.ServiceLister, pod *v1.Pod) ([]*v1.Service, error) {
	allServices, err := sl.Services(pod.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	var services []*v1.Service
	for i := range allServices {
		service := allServices[i]
		if service.Spec.Selector == nil {
			// services with nil selectors match nothing, not everything.
			continue
		}
		selector := labels.Set(service.Spec.Selector).AsSelectorPreValidated()
		if selector.Matches(labels.Set(pod.Labels)) {
			services = append(services, service)
		}
	}

	return services, nil
}

func countMatchingPods(namespace string, selector labels.Selector, nodeInfo *r.NodeInfo) int {
	if len(nodeInfo.Pods) == 0 || selector.Empty() {
		return 0
	}
	count := 0
	for _, p := range nodeInfo.Pods {
		// Ignore pods being deleted for spreading purposes
		// Similar to how it is done for SelectorSpreadPriority
		if namespace == p.Pod.Namespace && p.Pod.DeletionTimestamp == nil {
			if selector.Matches(labels.Set(p.Pod.Labels)) {
				count++
			}
		}
	}
	return count
}
