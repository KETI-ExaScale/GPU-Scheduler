package scheduler

import (
	"fmt"
	"reflect"
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
	fmt.Println("Add All Event Handlers")
	// scheduled pod -> cache
	informerFactory.Core().V1().Pods().Informer().AddEventHandler(
		cache.FilteringResourceEventHandler{
			FilterFunc: func(obj interface{}) bool {
				switch t := obj.(type) {
				case *v1.Pod:
					if assignedPod(t) && sched.nodeInfoExist(t) {
						return true
					}
					return false
				case cache.DeletedFinalStateUnknown:
					if _, ok := t.Obj.(*v1.Pod); ok {
						// The carried object may be stale, so we don't use it to check if
						// it's assigned or not. Attempting to cleanup anyways.
						return true
					}
					fmt.Errorf("unable to convert object %T to *v1.Pod in %T", obj, sched)
					return false
				default:
					fmt.Errorf("unable to handle object in %T: %T", sched, obj)
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
					fmt.Errorf("unable to handle object in %T: %T", sched, obj)
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
				switch t := obj.(type) {
				case *v1.Node:
					return !r.IsMasterNode(t)
				case cache.DeletedFinalStateUnknown:
					return false
				default:
					fmt.Errorf("unable to handle object in %T: %T", sched, obj)
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
}

func (sched *GPUScheduler) nodeInfoExist(pod *v1.Pod) bool {
	if _, ok := sched.NodeInfoCache.NodeInfoList[pod.Spec.NodeName]; ok {
		return true
	}
	return false
}

func (sched *GPUScheduler) addNodeToCache(obj interface{}) {
	fmt.Println("addNodeToCache")
	node, ok := obj.(v1.Node)
	if !ok {
		klog.ErrorS(nil, "Cannot convert to *v1.Node", "obj", obj)
		return
	}

	err := sched.NodeInfoCache.AddNode(node)
	if err != nil {
		klog.ErrorS(nil, "Cannot Add Node [", node.Name, "]")
	}

	// klog.V(3).InfoS("Add event for node", "node", klog.KObj(node))
	sched.SchedulingQueue.FlushBackoffQCompleted()
}

func (sched *GPUScheduler) updateNodeInCache(oldObj, newObj interface{}) {
	fmt.Println("updateNodeInCache")
	oldNode, ok := oldObj.(*v1.Node)
	if !ok {
		klog.ErrorS(nil, "Cannot convert oldObj to *v1.Node", "oldObj", oldObj)
		return
	}

	newNode, ok := newObj.(*v1.Node)
	if !ok {
		klog.ErrorS(nil, "Cannot convert newObj to *v1.Node", "newObj", newObj)
		return
	}

	err := sched.NodeInfoCache.UpdateNode(oldNode, newNode)
	if err != nil {
		klog.ErrorS(nil, "Cannot Update Node [", newNode.Name, "]")
	}

	// Only requeue unschedulable pods if the node became more schedulable.
	if event := nodeSchedulingPropertiesChange(newNode, oldNode); event != nil {
		sched.SchedulingQueue.FlushBackoffQCompleted()
	}
}

func (sched *GPUScheduler) deleteNodeFromCache(obj interface{}) {
	fmt.Println("deleteNodeFromCache")
	var node *v1.Node
	switch t := obj.(type) {
	case *v1.Node:
		node = t
	case cache.DeletedFinalStateUnknown:
		var ok bool
		node, ok = t.Obj.(*v1.Node)
		if !ok {
			klog.ErrorS(nil, "Cannot convert to *v1.Node", "obj", t.Obj)
			return
		}
	default:
		klog.ErrorS(nil, "Cannot convert to *v1.Node", "obj", t)
		return
	}
	klog.V(3).InfoS("Delete event for node", "node", klog.KObj(node))
	if err := sched.NodeInfoCache.RemoveNode(node); err != nil {
		klog.ErrorS(err, "Scheduler cache RemoveNode failed")
	}
}

func (sched *GPUScheduler) addPodToSchedulingQueue(obj interface{}) {
	pod := obj.(*v1.Pod)
	fmt.Println("sched.addPodToScheduligQueue: ", pod.Name)
	klog.V(3).InfoS("Add event for unscheduled pod", "pod", klog.KObj(pod))
	if err := sched.SchedulingQueue.Add_AvtiveQ(pod); err != nil {
		//podstates에 추가?
		fmt.Errorf("unable to queue %T: %v", obj, err)
	} else {
		sched.NodeInfoCache.AddPodState(*pod, r.Pending)
	}
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
		fmt.Errorf("unable to update %T: %v", newObj, err)
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
			fmt.Errorf("unable to convert object %T to *v1.Pod in %T", obj, sched)
			return
		}
	default:
		fmt.Errorf("unable to handle object in %T: %T", sched, obj)
		return
	}

	if ok, state := sched.NodeInfoCache.CheckPodStateExist(pod); ok {
		if state != r.Pending {
			return
		}
	}

	klog.V(3).InfoS("Delete event for unscheduled pod", "pod", klog.KObj(pod))
	if err := sched.SchedulingQueue.Delete(pod); err != nil {
		fmt.Errorf("unable to dequeue %T: %v", obj, err)
	}
}

func (sched *GPUScheduler) addPodToCache(obj interface{}) {
	pod, _ := obj.(*v1.Pod)

	if ok, state := sched.NodeInfoCache.CheckPodStateExist(pod); ok {
		if state == r.BindingFinished {
			return

		} else if state == r.Pending {
			pod := obj.(*v1.Pod)
			fmt.Println("<error> Pod {", pod.Name, "} State is Pending")
			//what?

		} else { //Assumed
			sched.NodeInfoCache.AddPod(pod)
			sched.NodeInfoCache.UpdatePodState(pod, r.BindingFinished)
		}

	} else {
		fmt.Println("# Pod {", pod.Name, "} NonExist")
		sched.NodeInfoCache.AddPod(pod)
		sched.NodeInfoCache.AddPodState(*pod, r.BindingFinished)

		if strings.HasPrefix(pod.Name, "keti-gpu-metric-collector") {
			fmt.Println("add node {", pod.Spec.NodeName, "} gpu metric collector")
			sched.NodeInfoCache.NodeInfoList[pod.Spec.NodeName].GRPCHost = pod.Status.PodIP
		}
	}
}

func (sched *GPUScheduler) updatePodInCache(oldObj, newObj interface{}) {
	oldPod, ok := oldObj.(*v1.Pod)
	if !ok {
		klog.ErrorS(nil, "Cannot convert oldObj to *v1.Pod", "oldObj", oldObj)
		return
	}
	// fmt.Println("updatePodInCache: ", oldPod.Name)
	newPod, ok := newObj.(*v1.Pod)
	if !ok {
		klog.ErrorS(nil, "Cannot convert newObj to *v1.Pod", "newObj", newObj)
		return
	}
	klog.V(4).InfoS("Update event for scheduled pod", "pod", klog.KObj(oldPod))

	// ok, state := sched.NodeInfoCache.CheckPodStateExist(oldPod)
	// if !ok || state != r.BindingFinished {
	// 	fmt.Println("cannot update. there isn't pod {", newPod.Name, "} state")
	// 	return
	// }

	if err := sched.NodeInfoCache.UpdatePod(oldPod, newPod); err != nil {
		klog.ErrorS(err, "Scheduler cache UpdatePod failed", "pod", klog.KObj(oldPod))
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
			klog.ErrorS(nil, "Cannot convert to *v1.Pod", "obj", t.Obj)
			return
		}
	default:
		klog.ErrorS(nil, "Cannot convert to *v1.Pod", "obj", t)
		return
	}

	if ok, _ := sched.NodeInfoCache.CheckPodStateExist(pod); !ok {
		fmt.Println("cannot delete. there isn't pod {", pod.Name, "} state")
		return
	}

	if strings.HasPrefix(pod.Name, "keti-gpu-metric-collector") {
		fmt.Println("remove node {", pod.Spec.NodeName, "} gpu metric collector")
		sched.NodeInfoCache.NodeInfoList[pod.Spec.NodeName].GRPCHost = ""
	}

	klog.V(3).InfoS("Delete event for scheduled pod", "pod", klog.KObj(pod))
	if err := sched.NodeInfoCache.RemovePod(pod); err != nil {
		klog.ErrorS(err, "Scheduler cache RemovePod failed", "pod", klog.KObj(pod))
	}

	// sched.SchedulingQueue.MoveAllToActiveOrBackoffQueue(queue.AssignedPodDelete, nil)
}

// assignedPod selects pods that are assigned (scheduled and running).
func assignedPod(pod *v1.Pod) bool {
	return len(pod.Spec.NodeName) != 0
}

// responsibleForPod returns true if the pod has asked to be scheduled by the given scheduler.
func responsibleForPod(pod *v1.Pod) bool {
	responsibleForPod := (pod.Spec.SchedulerName == "gpu-scheduler")
	fmt.Println("responsibleForPod- ", pod.Spec.SchedulerName)
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
