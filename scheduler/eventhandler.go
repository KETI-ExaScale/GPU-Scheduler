package scheduler

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"

	r "gpu-scheduler/resourceinfo"
)

/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// addAllEventHandlers is a helper function used in tests and in Scheduler
// to add event handlers for various informers.
func AddAllEventHandlers(
	sched *GPUScheduler,
	informerFactory informers.SharedInformerFactory,
) {
	// scheduled pod -> cache
	informerFactory.Core().V1().Pods().Informer().AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch t := obj.(type) {
				case *v1.Pod:
					return assignedPod(t) && sched.nodeInfoExist(t)
				case cache.DeletedFinalStateUnknown:
					if _, ok := t.Obj.(*v1.Pod); ok {
						// The carried object may be stale, so we don't use it to check if
						// it's assigned or not. Attempting to cleanup anyways.
						return true
					}
					r.KETI_LOG_L3(fmt.Sprintf("[error] unable to convert object %T to *v1.Pod in %T\n", obj, sched))
					return false
				default:
					r.KETI_LOG_L3(fmt.Sprintf("[error] unable to handle object in %T: %T\n", sched, obj))
					return false
				}
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc:    sched.addPodToCache,
				UpdateFunc: sched.updatePodInCache,
				DeleteFunc: sched.deletePodFromCache,
			},
		},
	)

	// unscheduled pod -> scheduling queue
	informerFactory.Core().V1().Pods().Informer().AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch t := obj.(type) {
				case *v1.Pod:
					return !assignedPod(t) && responsibleForPod(t)
				case cache.DeletedFinalStateUnknown:
					return false
				default:
					r.KETI_LOG_L3(fmt.Sprintf("[error] unable to handle object in %T: %T\n", sched, obj))
					return false
				}
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc:    sched.addPodToSchedulingQueue,
				UpdateFunc: sched.updatePodInSchedulingQueue,
				DeleteFunc: sched.deletePodFromSchedulingQueue,
			},
		},
	)

	// node
	informerFactory.Core().V1().Nodes().Informer().AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch obj.(type) {
				case *v1.Node:
					return true //!r.IsMasterNode(t)
				case cache.DeletedFinalStateUnknown:
					return false
				default:
					r.KETI_LOG_L3(fmt.Sprintf("[error] unable to handle object in %T: %T\n", sched, obj))
					return false
				}
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc:    sched.addNodeToCache,
				UpdateFunc: sched.updateNodeInCache,
				DeleteFunc: sched.deleteNodeFromCache,
			},
		},
	)

	// policy
	informerFactory.Core().V1().ConfigMaps().Informer().AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch t := obj.(type) {
				case *v1.ConfigMap:
					return (t.ObjectMeta.Name == "gpu-scheduler-configmap")
				case cache.DeletedFinalStateUnknown:
					return false
				default:
					r.KETI_LOG_L3(fmt.Sprintf("[error] unable to handle object in %T: %T\n", sched, obj))
					return false
				}
			},
			Handler: cache.ResourceEventHandlerFuncs{
				AddFunc:    sched.addPolicyToCache,
				UpdateFunc: sched.updatePolicyInCache,
			},
		},
	)
}

func (sched *GPUScheduler) nodeInfoExist(pod *v1.Pod) bool {
	if _, ok := sched.NodeInfoCache.NodeInfoList[pod.Spec.NodeName]; ok {
		return true
	}
	return false
}

func (sched *GPUScheduler) addNodeToCache(obj interface{}) {
	node, ok := obj.(*v1.Node)
	if !ok {
		r.KETI_LOG_L3(fmt.Sprintf("[error] cannot convert to *v1.Node -> %+v", obj))
		return
	}

	if _, ok := sched.NodeInfoCache.NodeInfoList[node.Name]; ok {
		return
	}

	r.KETI_LOG_L2(fmt.Sprintf("[event] add new node {%s} to cache\n", node.Name))

	err := sched.NodeInfoCache.AddNode(node)
	if err != nil {
		klog.ErrorS(nil, "cannot add node [", node.Name, "]")
	}

	// klog.V(3).InfoS("Add event for node", "node", klog.KObj(node))
	sched.SchedulingQueue.MoveAllToActiveOrBackoffQueue(true)
}

func (sched *GPUScheduler) updateNodeInCache(oldObj, newObj interface{}) {

	oldNode, ok := oldObj.(*v1.Node)
	if !ok {
		klog.ErrorS(nil, "cannot convert oldObj to *v1.Node", "oldObj", oldObj)
		return
	}

	newNode, ok := newObj.(*v1.Node)
	if !ok {
		klog.ErrorS(nil, "cannot convert newObj to *v1.Node", "newObj", newObj)
		return
	}

	if _, ok := sched.NodeInfoCache.NodeInfoList[newNode.Name]; !ok {
		return
	}

	err := sched.NodeInfoCache.UpdateNode(oldNode, newNode)
	if err != nil {
		klog.ErrorS(nil, "cannot Update Node [", newNode.Name, "]")
	}

	// Only requeue unschedulable pods if the node became more schedulable.
	event := nodeSchedulingPropertiesChange(newNode, oldNode)
	if event != nil {
		if event == &NodeAllocatableChange {
			sched.SchedulingQueue.MoveAllToActiveOrBackoffQueue(false) //flush only unschedulable pod
		} else {
			sched.SchedulingQueue.MoveAllToActiveOrBackoffQueue(true) //flush all pods
		}
	}
}

func (sched *GPUScheduler) deleteNodeFromCache(obj interface{}) {
	node, ok := obj.(*v1.Node)
	if !ok {
		klog.ErrorS(nil, "cannot convert newObj to *v1.Node", "newObj", obj)
		return
	}

	if _, ok := sched.NodeInfoCache.NodeInfoList[node.Name]; !ok {
		return
	}

	if err := sched.NodeInfoCache.RemoveNode(node); err != nil {
		klog.ErrorS(err, "scheduler cache remove node failed")
	}
}

func (sched *GPUScheduler) addPodToSchedulingQueue(obj interface{}) {
	pod := obj.(*v1.Pod)
	sched.SchedulingQueue.Add(pod)
	sched.NodeInfoCache.AddPodState(*pod, r.Pending)
}

func (sched *GPUScheduler) updatePodInSchedulingQueue(oldObj, newObj interface{}) {
	oldPod, newPod := oldObj.(*v1.Pod), newObj.(*v1.Pod)
	if oldPod.ResourceVersion == newPod.ResourceVersion {
		return
	}

	if ok, state := sched.NodeInfoCache.CheckPodStateExist(newPod); ok {
		if state != r.Pending {
			return
		}
	}

	if err := sched.SchedulingQueue.Update(oldPod, newPod); err != nil {
		r.KETI_LOG_L3(fmt.Sprintf("[error] unable to update %T: %v\n", newObj, err))
	}
}

func (sched *GPUScheduler) deletePodFromSchedulingQueue(obj interface{}) {
	var pod *v1.Pod
	switch t := obj.(type) {
	case *v1.Pod:
		pod = obj.(*v1.Pod)
		// fmt.Println("deletePodFromSchedulingQueue: ", pod.Name)
	case cache.DeletedFinalStateUnknown:
		var ok bool
		pod, ok = t.Obj.(*v1.Pod)
		if !ok {
			r.KETI_LOG_L3(fmt.Sprintf("[error] unable to convert object %T to *v1.Pod in %T\n", obj, sched))
			return
		}
	default:
		r.KETI_LOG_L3(fmt.Sprintf("[error] unable to handle object in %T: %T\n", sched, obj))
		return
	}

	if ok, state := sched.NodeInfoCache.CheckPodStateExist(pod); ok {
		if state != r.Pending {
			return
		}
	}

	if err := sched.SchedulingQueue.Delete(pod); err != nil {
		r.KETI_LOG_L3(fmt.Sprintf("[error] unable to dequeue %T: %v\n", obj, err))
	}
}

func (sched *GPUScheduler) addPodToCache(obj interface{}) {
	pod, _ := obj.(*v1.Pod)

	if ok, state := sched.NodeInfoCache.CheckPodStateExist(pod); ok { // scheduled pod already in cache
		if state == r.BindingFinished {
			return

		} else if state == r.Pending {
			pod := obj.(*v1.Pod)
			r.KETI_LOG_L3(fmt.Sprintf("[error] pod {%s} state is pending", pod.Name)) // penging pod cannot add to cache
			return
		}
	}

	// Assumed Pod or scehduled other scheduler
	// fmt.Printf("- add pod {%s} to cache\n", pod.Name)
	sched.NodeInfoCache.AddPod(pod, r.BindingFinished)
}

func (sched *GPUScheduler) updatePodInCache(oldObj, newObj interface{}) {
	oldPod, ok := oldObj.(*v1.Pod)
	if !ok {
		r.KETI_LOG_L3(fmt.Sprintf("[error] cannot update pod %s in cache\n", oldPod.Name))
		return
	}

	newPod, ok := newObj.(*v1.Pod)
	if !ok {
		r.KETI_LOG_L3(fmt.Sprintf("[error] cannot update pod %s in cache\n", oldPod.Name))
		return
	}

	// //nodeinfocache's podCount update
	// if oldPod.Status.Phase == "Running" && newPod.Status.Phase == "Succeeded" {
	// 	if newPod.ObjectMeta.Annotations["UUID"] != "" {
	// 		// fmt.Println("- complete pod ", newPod.Name, newPod.Spec.NodeName, newPod.ObjectMeta.Annotations["UUID"])
	// 		sched.NodeInfoCache.GPUPodCountDown(newPod)
	// 	}
	// }

	//metric-collector랑 cluster-manager는 ip가 바뀌었는지가 중요
	/*if strings.HasPrefix(newPod.Name, "keti-gpu-metric-collector") {
		if oldPod.Status.PodIP != newPod.Status.PodIP {
			r.KETI_LOG_L2(fmt.Sprintf("# add node {%s} gpu metric collector", newPod.Spec.NodeName))
			sched.NodeInfoCache.NodeInfoList[newPod.Spec.NodeName].MetricCollectorIP = newPod.Status.PodIP
		}
	} else*/if strings.HasPrefix(newPod.Name, "keti-cluster-manager") {
		// if sched.ClusterManagerHost == "" || !sched.AvailableClusterManager {
		if oldPod.Status.PodIP != newPod.Status.PodIP {
			r.KETI_LOG_L2(fmt.Sprintf("[event] add node {%s cluster manager", newPod.Spec.NodeName))
			sched.ClusterManagerHost = newPod.Status.PodIP
			sched.InitClusterManager()
		}
		// }
	} else if strings.HasPrefix(newPod.Name, "keti-analysis-engine") {
		if oldPod.Status.PodIP != newPod.Status.PodIP {
			r.KETI_LOG_L2(fmt.Sprintf("[event] add node {%s analysis engine", newPod.Spec.NodeName))
			sched.MetricAnalysisModuleIP = newPod.Status.PodIP
			sched.InitClusterManager()
		}
	}

}

func (sched *GPUScheduler) deletePodFromCache(obj interface{}) {
	var pod *v1.Pod
	switch t := obj.(type) {
	case *v1.Pod:
		pod = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		pod, ok = t.Obj.(*v1.Pod)
		if !ok {
			r.KETI_LOG_L3(fmt.Sprintf("cannot convert to *v1.Pod -> %+v", t.Obj))
			return
		}
	default:
		r.KETI_LOG_L3(fmt.Sprintf("cannot convert to *v1.Pod -> %+v", t))
		return
	}

	if ok, _ := sched.NodeInfoCache.CheckPodStateExist(pod); !ok {
		r.KETI_LOG_L3(fmt.Sprintf("[error] cannot delete. there isn't pod {%s} state", pod.Name))
		return
	}

	// if strings.HasPrefix(pod.Name, "keti-gpu-metric-collector") {
	// 	r.KETI_LOG_L2(fmt.Sprintf("# remove gpu metric collector in node {%s}", pod.Spec.NodeName))
	// 	sched.NodeInfoCache.NodeInfoList[pod.Spec.NodeName].MetricCollectorIP = ""
	// }

	r.KETI_LOG_L1(fmt.Sprintf("[event] delete pod {%s} from cache\n", pod.Name))
	if err := sched.NodeInfoCache.RemovePod(pod); err != nil {
		klog.ErrorS(err, "[error] scheduler cache remove pod failed", "pod", klog.KObj(pod))
	}
}

func (sched *GPUScheduler) addPolicyToCache(obj interface{}) {
	configMap := obj.(*v1.ConfigMap)

	w := strings.Split(configMap.Data["node-gpu-score-weight"], ":")
	nodeWeight, _ := strconv.ParseFloat(w[0], 64)
	gpuWeight, _ := strconv.ParseFloat(w[1], 64)
	nvlinkWeightPercentage, _ := strconv.ParseInt(configMap.Data["nvlink-weight-percentage"], 0, 64)
	gpuAllocatePrefer := configMap.Data["gpu-allocate-prefer"]
	nodeReservetionPermit, _ := strconv.ParseBool(configMap.Data["node-reservation-permit"])
	podRschedulePermit, _ := strconv.ParseBool(configMap.Data["pod-re-schedule-permit"])
	avoidNVLinkOneGPU, _ := strconv.ParseBool(configMap.Data["avoid-nvlink-one-gpu"])
	multiNodeAllocationPermit, _ := strconv.ParseBool(configMap.Data["multi-node-allocation-permit"])
	nonGPUNodePrefer, _ := strconv.ParseBool(configMap.Data["non-gpu-node-prefer"])
	multiGPUNodePrefer, _ := strconv.ParseBool(configMap.Data["multi-gpu-node-prefer"])
	leastScoreNodePrefer, _ := strconv.ParseBool(configMap.Data["least-score-node-prefer"])
	avoidHighScoreNode, _ := strconv.ParseBool(configMap.Data["avoid-high-score-node"])

	sched.SchedulingPolicy.NodeWeight = nodeWeight
	sched.SchedulingPolicy.GPUWeight = gpuWeight
	sched.SchedulingPolicy.NVLinkWeightPercentage = nvlinkWeightPercentage
	sched.SchedulingPolicy.GPUAllocatePrefer = gpuAllocatePrefer
	sched.SchedulingPolicy.NodeReservationPermit = nodeReservetionPermit
	sched.SchedulingPolicy.PodReSchedulePermit = podRschedulePermit
	sched.SchedulingPolicy.AvoidNVLinkOneGPU = avoidNVLinkOneGPU
	sched.SchedulingPolicy.MultiNodeAllocationPermit = multiNodeAllocationPermit
	sched.SchedulingPolicy.NonGPUNodePrefer = nonGPUNodePrefer
	sched.SchedulingPolicy.MultiGPUNodePrefer = multiGPUNodePrefer
	sched.SchedulingPolicy.LeastScoreNodePrefer = leastScoreNodePrefer
	sched.SchedulingPolicy.AvoidHighScoreNode = avoidHighScoreNode

	r.KETI_LOG_L3("\n-----:: GPU Scheduler Policy List ::-----")
	r.KETI_LOG_L3(fmt.Sprintf("[policy 1] %s", r.Policy1))
	r.KETI_LOG_L3(fmt.Sprintf("  - node weight : %.1f", nodeWeight))
	r.KETI_LOG_L3(fmt.Sprintf("  - gpu weight : %.1f", gpuWeight))
	r.KETI_LOG_L3(fmt.Sprintf("[policy 2] %s", r.Policy2))
	r.KETI_LOG_L3(fmt.Sprintf("  - value : %d", nvlinkWeightPercentage))
	r.KETI_LOG_L3(fmt.Sprintf("[policy 3] %s", r.Policy3))
	r.KETI_LOG_L3(fmt.Sprintf("  - value(spread/binpack) : %s", gpuAllocatePrefer))
	r.KETI_LOG_L3(fmt.Sprintf("[policy 4] %s", r.Policy4))
	r.KETI_LOG_L3(fmt.Sprintf("  - value : %v", nodeReservetionPermit))
	r.KETI_LOG_L3(fmt.Sprintf("[policy 5] %s", r.Policy5))
	r.KETI_LOG_L3(fmt.Sprintf("  - value : %v", podRschedulePermit))
	r.KETI_LOG_L3(fmt.Sprintf("[policy 6] %s", r.Policy6))
	r.KETI_LOG_L3(fmt.Sprintf("  - value : %v", avoidNVLinkOneGPU))
	r.KETI_LOG_L3(fmt.Sprintf("[policy 7] %s", r.Policy7))
	r.KETI_LOG_L3(fmt.Sprintf("  - value : %v", multiNodeAllocationPermit))
	r.KETI_LOG_L3(fmt.Sprintf("[policy 8] %s", r.Policy8))
	r.KETI_LOG_L3(fmt.Sprintf("  - value : %v", nonGPUNodePrefer))
	r.KETI_LOG_L3(fmt.Sprintf("[policy 9] %s", r.Policy9))
	r.KETI_LOG_L3(fmt.Sprintf("  - value : %v", multiGPUNodePrefer))
	r.KETI_LOG_L3(fmt.Sprintf("[policy 10] %s", r.Policy10))
	r.KETI_LOG_L3(fmt.Sprintf("  - value : %v", leastScoreNodePrefer))
	r.KETI_LOG_L3(fmt.Sprintf("[policy 11] %s", r.Policy11))
	r.KETI_LOG_L3(fmt.Sprintf("  - value : %v", avoidHighScoreNode))

	//추가 희망 메트릭
	//gpu-filtering-temperature
	//activate-gpu-count-filtering

}

func (sched *GPUScheduler) updatePolicyInCache(oldObj, newObj interface{}) {
	configMap := newObj.(*v1.ConfigMap)

	r.KETI_LOG_L3("\n[event] update gpu-scheduler policy")

	w := strings.Split(configMap.Data["node-gpu-score-weight"], ":")
	nodeWeight, _ := strconv.ParseFloat(w[0], 64)
	gpuWeight, _ := strconv.ParseFloat(w[1], 64)
	nvlinkWeightPercentage, _ := strconv.ParseInt(configMap.Data["nvlink-weight-percentage"], 0, 64)
	gpuAllocatePrefer := configMap.Data["gpu-allocate-prefer"]
	nodeReservetionPermit, _ := strconv.ParseBool(configMap.Data["node-reservation-permit"])
	podRschedulePermit, _ := strconv.ParseBool(configMap.Data["pod-re-schedule-permit"])
	avoidNVLinkOneGPU, _ := strconv.ParseBool(configMap.Data["avoid-nvlink-one-gpu"])
	multiNodeAllocationPermit, _ := strconv.ParseBool(configMap.Data["multi-node-allocation-permit"])
	nonGPUNodePrefer, _ := strconv.ParseBool(configMap.Data["non-gpu-node-prefer"])
	multiGPUNodePrefer, _ := strconv.ParseBool(configMap.Data["multi-gpu-node-prefer"])
	leastScoreNodePrefer, _ := strconv.ParseBool(configMap.Data["least-score-node-prefer"])
	avoidHighScoreNode, _ := strconv.ParseBool(configMap.Data["avoid-high-score-node"])

	sched.SchedulingPolicy.NodeWeight = nodeWeight
	sched.SchedulingPolicy.GPUWeight = gpuWeight
	sched.SchedulingPolicy.NVLinkWeightPercentage = nvlinkWeightPercentage
	sched.SchedulingPolicy.GPUAllocatePrefer = gpuAllocatePrefer
	sched.SchedulingPolicy.NodeReservationPermit = nodeReservetionPermit
	sched.SchedulingPolicy.PodReSchedulePermit = podRschedulePermit
	sched.SchedulingPolicy.AvoidNVLinkOneGPU = avoidNVLinkOneGPU
	sched.SchedulingPolicy.MultiNodeAllocationPermit = multiNodeAllocationPermit
	sched.SchedulingPolicy.NonGPUNodePrefer = nonGPUNodePrefer
	sched.SchedulingPolicy.MultiGPUNodePrefer = multiGPUNodePrefer
	sched.SchedulingPolicy.LeastScoreNodePrefer = leastScoreNodePrefer
	sched.SchedulingPolicy.AvoidHighScoreNode = avoidHighScoreNode

	r.KETI_LOG_L3("\n-----:: Updated GPU Scheduler Policy List ::-----")
	r.KETI_LOG_L3(fmt.Sprintf("[policy 1] %s", r.Policy1))
	r.KETI_LOG_L3(fmt.Sprintf("  - node weight : %.1f", nodeWeight))
	r.KETI_LOG_L3(fmt.Sprintf("  - gpu weight : %.1f", gpuWeight))
	r.KETI_LOG_L3(fmt.Sprintf("[policy 2] %s", r.Policy2))
	r.KETI_LOG_L3(fmt.Sprintf("  - value : %d", nvlinkWeightPercentage))
	r.KETI_LOG_L3(fmt.Sprintf("[policy 3] %s", r.Policy3))
	r.KETI_LOG_L3(fmt.Sprintf("  - value(spread/binpack) : %s", gpuAllocatePrefer))
	r.KETI_LOG_L3(fmt.Sprintf("[policy 4] %s", r.Policy4))
	r.KETI_LOG_L3(fmt.Sprintf("  - value : %v", nodeReservetionPermit))
	r.KETI_LOG_L3(fmt.Sprintf("[policy 5] %s", r.Policy5))
	r.KETI_LOG_L3(fmt.Sprintf("  - value : %v", podRschedulePermit))
	r.KETI_LOG_L3(fmt.Sprintf("[policy 6] %s", r.Policy6))
	r.KETI_LOG_L3(fmt.Sprintf("  - value : %v", avoidNVLinkOneGPU))
	r.KETI_LOG_L3(fmt.Sprintf("[policy 7] %s", r.Policy7))
	r.KETI_LOG_L3(fmt.Sprintf("  - value : %v", multiNodeAllocationPermit))
	r.KETI_LOG_L3(fmt.Sprintf("[policy 8] %s", r.Policy8))
	r.KETI_LOG_L3(fmt.Sprintf("  - value : %v", nonGPUNodePrefer))
	r.KETI_LOG_L3(fmt.Sprintf("[policy 9] %s", r.Policy9))
	r.KETI_LOG_L3(fmt.Sprintf("  - value : %v", multiGPUNodePrefer))
	r.KETI_LOG_L3(fmt.Sprintf("[policy 10] %s", r.Policy10))
	r.KETI_LOG_L3(fmt.Sprintf("  - value : %v", leastScoreNodePrefer))
	r.KETI_LOG_L3(fmt.Sprintf("[policy 11] %s", r.Policy11))
	r.KETI_LOG_L3(fmt.Sprintf("  - value : %v", avoidHighScoreNode))
}

// assignedPod selects pods that are assigned (scheduled and running).
func assignedPod(pod *v1.Pod) bool {
	return len(pod.Spec.NodeName) != 0
}

// responsibleForPod returns true if the pod has asked to be scheduled by the given scheduler.
func responsibleForPod(pod *v1.Pod) bool {
	responsibleForPod := (pod.Spec.SchedulerName == "gpu-scheduler")
	return responsibleForPod
}

func nodeSchedulingPropertiesChange(newNode *v1.Node, oldNode *v1.Node) *r.ClusterEvent {
	if nodeSpecUnschedulableChanged(newNode, oldNode) {
		return &NodeSpecUnschedulableChange
	}
	if nodeAllocatableChanged(newNode, oldNode) {
		return &NodeAllocatableChange
	}
	if nodeLabelsChanged(newNode, oldNode) {
		return &NodeLabelChange
	}
	if nodeTaintsChanged(newNode, oldNode) {
		return &NodeTaintChange
	}
	if nodeConditionsChanged(newNode, oldNode) {
		return &NodeConditionChange
	}

	return nil
}

func nodeAllocatableChanged(newNode *v1.Node, oldNode *v1.Node) bool {
	return !reflect.DeepEqual(oldNode.Status.Allocatable, newNode.Status.Allocatable)
}

func nodeLabelsChanged(newNode *v1.Node, oldNode *v1.Node) bool {
	return !reflect.DeepEqual(oldNode.GetLabels(), newNode.GetLabels())
}

func nodeTaintsChanged(newNode *v1.Node, oldNode *v1.Node) bool {
	return !reflect.DeepEqual(newNode.Spec.Taints, oldNode.Spec.Taints)
}

func nodeConditionsChanged(newNode *v1.Node, oldNode *v1.Node) bool {
	strip := func(conditions []v1.NodeCondition) map[v1.NodeConditionType]v1.ConditionStatus {
		conditionStatuses := make(map[v1.NodeConditionType]v1.ConditionStatus, len(conditions))
		for i := range conditions {
			conditionStatuses[conditions[i].Type] = conditions[i].Status
		}
		return conditionStatuses
	}
	return !reflect.DeepEqual(strip(oldNode.Status.Conditions), strip(newNode.Status.Conditions))
}

func nodeSpecUnschedulableChanged(newNode *v1.Node, oldNode *v1.Node) bool {
	return newNode.Spec.Unschedulable != oldNode.Spec.Unschedulable && !newNode.Spec.Unschedulable
}
