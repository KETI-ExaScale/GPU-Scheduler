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
	"log"
	"math"

	r "gpu-scheduler/resourceinfo"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	appslisters "k8s.io/client-go/listers/apps/v1"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	utilnode "k8s.io/component-helpers/node/topology"
)

type SelectorSpread struct {
	services               corelisters.ServiceLister
	replicationControllers corelisters.ReplicationControllerLister
	replicaSets            appslisters.ReplicaSetLister
	statefulSets           appslisters.StatefulSetLister
}

const (
	// When zone information is present, give 2/3 of the weighting to zone spreading, 1/3 to node spreading
	// TODO: Any way to justify this weighting?
	zoneWeighting float64 = 2.0 / 3.0
)

var (
	rcKind = v1.SchemeGroupVersion.WithKind("ReplicationController")
	rsKind = appsv1.SchemeGroupVersion.WithKind("ReplicaSet")
	ssKind = appsv1.SchemeGroupVersion.WithKind("StatefulSet")
)

func SelectorSpread_() SelectorSpread {
	hostConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	hostKubeClient := kubernetes.NewForConfigOrDie(hostConfig)
	informerFactory := informers.NewSharedInformerFactory(hostKubeClient, 0)

	services := informerFactory.Core().V1().Services().Lister()
	replicationCtrls := informerFactory.Core().V1().ReplicationControllers().Lister()
	replicaSets := informerFactory.Apps().V1().ReplicaSets().Lister()
	statefulSets := informerFactory.Apps().V1().StatefulSets().Lister()

	return SelectorSpread{
		services:               services,
		replicationControllers: replicationCtrls,
		replicaSets:            replicaSets,
		statefulSets:           statefulSets,
	}
}

func (pl SelectorSpread) Name() string {
	return "SelectorSpread"
}

func (pl SelectorSpread) Debugg(nodeInfoCache *r.NodeCache) {
	r.KETI_LOG_L2(fmt.Sprintf("S#3. %s", pl.Name()))
	for nodeName, nodeInfo := range nodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			r.KETI_LOG_L1(fmt.Sprintf("-node {%s} score: %d", nodeName, nodeInfo.PluginResult.NodeScore))
		}
	}
}

func (pl SelectorSpread) Score(nodeInfoCache *r.NodeCache, newPod *r.QueuedPodInfo) {
	if skipSelectorSpread(newPod.Pod) {
		return
	}

	//PreScore
	selector := DefaultSelector(
		newPod.Pod,
		pl.services,
		pl.replicationControllers,
		pl.replicaSets,
		pl.statefulSets,
	)

	//Score
	for _, nodeinfo := range nodeInfoCache.NodeInfoList {
		if !nodeinfo.PluginResult.IsFiltered {
			count := countMatchingPods(newPod.Pod.Namespace, selector, nodeinfo)
			// fmt.Println("scoring>selector_spread: ", count, "scode, node: ", nodeinfo.Node().Name)
			score := pl.NormalizeScore(count, nodeinfo)
			nodeinfo.PluginResult.NodeScore += int(math.Round(float64(score)))
		}
	}
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

// countMatchingPods counts pods based on namespace and matching all selectors
func countMatchingPods(namespace string, selector labels.Selector, nodeInfo *r.NodeInfo) int64 {
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
	return int64(count)
}

func DefaultSelector(
	pod *v1.Pod,
	sl corelisters.ServiceLister,
	cl corelisters.ReplicationControllerLister,
	rsl appslisters.ReplicaSetLister,
	ssl appslisters.StatefulSetLister,
) labels.Selector {
	labelSet := make(labels.Set)
	// Since services, RCs, RSs and SSs match the pod, they won't have conflicting
	// labels. Merging is safe.

	if services, err := GetPodServices(sl, pod); err == nil {
		for _, service := range services {
			labelSet = labels.Merge(labelSet, service.Spec.Selector)
		}
	}
	selector := labelSet.AsSelector()

	owner := metav1.GetControllerOfNoCopy(pod)
	if owner == nil {
		return selector
	}

	gv, err := schema.ParseGroupVersion(owner.APIVersion)
	if err != nil {
		return selector
	}

	gvk := gv.WithKind(owner.Kind)
	switch gvk {
	case rcKind:
		if rc, err := cl.ReplicationControllers(pod.Namespace).Get(owner.Name); err == nil {
			labelSet = labels.Merge(labelSet, rc.Spec.Selector)
			selector = labelSet.AsSelector()
		}
	case rsKind:
		if rs, err := rsl.ReplicaSets(pod.Namespace).Get(owner.Name); err == nil {
			if other, err := metav1.LabelSelectorAsSelector(rs.Spec.Selector); err == nil {
				if r, ok := other.Requirements(); ok {
					selector = selector.Add(r...)
				}
			}
		}
	case ssKind:
		if ss, err := ssl.StatefulSets(pod.Namespace).Get(owner.Name); err == nil {
			if other, err := metav1.LabelSelectorAsSelector(ss.Spec.Selector); err == nil {
				if r, ok := other.Requirements(); ok {
					selector = selector.Add(r...)
				}
			}
		}
	default:
		// Not owned by a supported controller.
	}

	return selector
}

func (pl *SelectorSpread) NormalizeScore(score int64, nodeInfo *r.NodeInfo) int64 {
	countsByZone := make(map[string]int64, 10)
	maxCountByZone := int64(0)
	maxCountByNodeName := int64(0)

	if score > maxCountByNodeName {
		maxCountByNodeName = score
	}

	zoneID := utilnode.GetZoneKey(nodeInfo.Node())
	if zoneID == "" {
	} else {
		countsByZone[zoneID] += score
	}

	for zoneID := range countsByZone {
		if countsByZone[zoneID] > maxCountByZone {
			maxCountByZone = countsByZone[zoneID]
		}
	}

	haveZones := len(countsByZone) != 0

	maxCountByNodeNameFloat64 := float64(maxCountByNodeName)
	maxCountByZoneFloat64 := float64(maxCountByZone)
	MaxNodeScoreFloat64 := float64(r.MaxScore)

	// initializing to the default/max node score of maxPriority
	fScore := MaxNodeScoreFloat64
	if maxCountByNodeName > 0 {
		fScore = MaxNodeScoreFloat64 * (float64(maxCountByNodeName-score) / maxCountByNodeNameFloat64)
	}
	// If there is zone information present, incorporate it
	if haveZones {
		zoneID := utilnode.GetZoneKey(nodeInfo.Node())
		if zoneID != "" {
			zoneScore := MaxNodeScoreFloat64
			if maxCountByZone > 0 {
				zoneScore = MaxNodeScoreFloat64 * (float64(maxCountByZone-countsByZone[zoneID]) / maxCountByZoneFloat64)
			}
			fScore = (fScore * (1.0 - zoneWeighting)) + (zoneWeighting * zoneScore)
		}
	}
	score = int64(fScore)

	return score
}
