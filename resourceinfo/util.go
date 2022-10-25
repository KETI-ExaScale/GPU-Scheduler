/*
Copyright 2017 The Kubernetes Authors.

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

package resourceinfo

import (
	"context"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1helpers "k8s.io/component-helpers/scheduling/corev1"
)

const GPU_SCHEDUER_DEBUGG_LEVEL = 3

func KETI_LOG_L1(log string) { //자세한 출력, DumpClusterInfo DumpNodeInfo
	if GPU_SCHEDUER_DEBUGG_LEVEL < 2 { //LEVEL = 1,2,3
		fmt.Println(log)
	}
}

func KETI_LOG_L2(log string) { // 기본출력
	if GPU_SCHEDUER_DEBUGG_LEVEL < 3 { //LEVEL = 2,3
		fmt.Println(log)
	}
}

func KETI_LOG_L3(log string) { //필수출력, 정량용, 에러
	if GPU_SCHEDUER_DEBUGG_LEVEL < 4 { //LEVEL = 3
		fmt.Println(log)
	}
}

// GetPodFullName returns a name that uniquely identifies a pod.
func GetPodFullName(pod *v1.Pod) string {
	// Use underscore as the delimiter because it is not allowed in pod name
	// (DNS subdomain format).
	return pod.Name + "_" + pod.Namespace
}

// GetPodStartTime returns start time of the given pod or current timestamp
// if it hasn't started yet.
func GetPodStartTime(pod *v1.Pod) *metav1.Time {
	if pod.Status.StartTime != nil {
		return pod.Status.StartTime
	}
	// Assumed pods and bound pods that haven't started don't have a StartTime yet.
	return &metav1.Time{Time: time.Now()}
}

// // GetEarliestPodStartTime returns the earliest start time of all pods that
// // have the highest priority among all victims.
// func GetEarliestPodStartTime(victims *extenderv1.Victims) *metav1.Time {
// 	if len(victims.Pods) == 0 {
// 		// should not reach here.
// 		klog.ErrorS(fmt.Errorf("victims.Pods is empty. Should not reach here"), "")
// 		return nil
// 	}

// 	earliestPodStartTime := GetPodStartTime(victims.Pods[0])
// 	maxPriority := corev1helpers.PodPriority(victims.Pods[0])

// 	for _, pod := range victims.Pods {
// 		if corev1helpers.PodPriority(pod) == maxPriority {
// 			if GetPodStartTime(pod).Before(earliestPodStartTime) {
// 				earliestPodStartTime = GetPodStartTime(pod)
// 			}
// 		} else if corev1helpers.PodPriority(pod) > maxPriority {
// 			maxPriority = corev1helpers.PodPriority(pod)
// 			earliestPodStartTime = GetPodStartTime(pod)
// 		}
// 	}

// 	return earliestPodStartTime
// }

// MoreImportantPod return true when priority of the first pod is higher than
// the second one. If two pods' priorities are equal, compare their StartTime.
// It takes arguments of the type "interface{}" to be used with SortableList,
// but expects those arguments to be *v1.Pod.
func MoreImportantPod(pod1, pod2 *v1.Pod) bool {
	p1 := corev1helpers.PodPriority(pod1)
	p2 := corev1helpers.PodPriority(pod2)
	if p1 != p2 {
		return p1 > p2
	}
	return GetPodStartTime(pod1).Before(GetPodStartTime(pod2))
}

// // PatchPodStatus calculates the delta bytes change from <old.Status> to <newStatus>,
// // and then submit a request to API server to patch the pod changes.
// func PatchPodStatus(cs kubernetes.Interface, old *v1.Pod, newStatus *v1.PodStatus) error {
// 	if newStatus == nil {
// 		return nil
// 	}

// 	oldData, err := json.Marshal(v1.Pod{Status: old.Status})
// 	if err != nil {
// 		return err
// 	}

// 	newData, err := json.Marshal(v1.Pod{Status: *newStatus})
// 	if err != nil {
// 		return err
// 	}
// 	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(oldData, newData, &v1.Pod{})
// 	if err != nil {
// 		return fmt.Errorf("failed to create merge patch for pod %q/%q: %v", old.Namespace, old.Name, err)
// 	}

// 	if "{}" == string(patchBytes) {
// 		return nil
// 	}

// 	_, err = cs.CoreV1().Pods(old.Namespace).Patch(context.TODO(), old.Name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{}, "status")
// 	return err
// }

// DeletePod deletes the given <pod> from API server
func DeletePod(cs kubernetes.Interface, pod *v1.Pod) error {
	return cs.CoreV1().Pods(pod.Namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
}

// // ClearNominatedNodeName internally submit a patch request to API server
// // to set each pods[*].Status.NominatedNodeName> to "".
// func ClearNominatedNodeName(cs kubernetes.Interface, pods ...*v1.Pod) utilerrors.Aggregate {
// 	var errs []error
// 	for _, p := range pods {
// 		if len(p.Status.NominatedNodeName) == 0 {
// 			continue
// 		}
// 		podStatusCopy := p.Status.DeepCopy()
// 		podStatusCopy.NominatedNodeName = ""
// 		if err := PatchPodStatus(cs, p, podStatusCopy); err != nil {
// 			errs = append(errs, err)
// 		}
// 	}
// 	return utilerrors.NewAggregate(errs)
// }

// // IsScalarResourceName validates the resource for Extended, Hugepages, Native and AttachableVolume resources
// func IsScalarResourceName(name v1.ResourceName) bool {
// 	return v1helper.IsExtendedResourceName(name) || v1helper.IsHugePageResourceName(name) ||
// 		v1helper.IsPrefixedNativeResource(name) || v1helper.IsAttachableVolumeResourceName(name)
// }
