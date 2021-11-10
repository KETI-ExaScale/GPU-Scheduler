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

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func PatchPodAnnotationUUID(bestGPU string) ([]byte, error) {
	patchAnnotations := map[string]interface{}{
		"metadata": map[string]map[string]string{"annotations": {
			"UUID": bestGPU,
		}}}

	return json.Marshal(patchAnnotations)
}

//write GPUID to annotation
func PatchPodAnnotation(newPod *resource.Pod, bestGPU string) error {
	if config.Debugg {
		fmt.Println("[step 3-1] Write GPU UUID in Pod Annotation")
	}

	patchedAnnotationBytes, err := PatchPodAnnotationUUID(bestGPU)
	if err != nil {
		return fmt.Errorf("failed to generate patched annotations,reason: %v", err)
	}

	_, err = config.Host_kubeClient.CoreV1().Pods(newPod.Pod.Namespace).Patch(context.TODO(), newPod.Pod.Name, types.StrategicMergePatchType, patchedAnnotationBytes, metav1.PatchOptions{})
	if err != nil {
		fmt.Println("patchPodAnnotation error: ", err)
		return err
	}

	patchedAnnotationBytes2 := []byte(`{"spec":{"template":{"spec":{"containers":[{"name":"nbody1","env":[{"name":"TESTNAME","value":"TESTVALUE"}]}]}}}}`)
	_, err = config.Host_kubeClient.CoreV1().Pods(newPod.Pod.Namespace).Patch(context.TODO(), newPod.Pod.Name, types.StrategicMergePatchType, patchedAnnotationBytes2, metav1.PatchOptions{})
	if err != nil {
		fmt.Println("patchPodAnnotation error: ", err)
		return err
	}

	// patchedAnnotationBytes3 := []byte(`{"metadata":{"labels":{"app":"web"}}}`)
	// _, err = config.Host_kubeClient.CoreV1().Pods(newPod.Pod.Namespace).Patch(context.TODO(), newPod.Pod.Name, types.StrategicMergePatchType, patchedAnnotationBytes3, metav1.PatchOptions{})
	// if err != nil {
	// 	fmt.Println("patchPodAnnotation error: ", err)
	// 	return err
	// }

	return nil
}

func Binding(newPod *resource.Pod, schedulingResult resource.SchedulingResult) error {
	if config.Debugg {
		fmt.Println("[step 3] Binding stage")
	}

	//파드 스펙에 GPU 업데이트
	err := PatchPodAnnotation(newPod, schedulingResult.BestGPU)
	if err != nil {
		return fmt.Errorf("failed to generate patched annotations,reason: %v", err)
	}

	binding := &corev1.Binding{
		ObjectMeta: metav1.ObjectMeta{
			Name: newPod.Pod.Name,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Binding",
		},
		Target: corev1.ObjectReference{
			APIVersion: "v1",
			Kind:       "Node",
			Name:       schedulingResult.BestNode,
		},
	}

	err = config.Host_kubeClient.CoreV1().Pods(newPod.Pod.Namespace).Bind(context.TODO(), binding, metav1.CreateOptions{})
	if err != nil {
		fmt.Println("binding error: ", err)
		return err
	}

	fmt.Println("<Result> BestNode:", schedulingResult.BestNode)
	fmt.Println("<Result> BestGPU:", schedulingResult.BestGPU)

	// Emit a Kubernetes event that the Pod was scheduled successfully.
	message := fmt.Sprintf("<Binding Success> Successfully assigned %s", newPod.Pod.ObjectMeta.Name)
	event := resource.MakeBindEvent(newPod, message)
	fmt.Println(time.Now().Format("2006-01-02 15:04:05"), message)
	resource.PostEvent(event)
	fmt.Println("----------------------------------------------------------------------------------------------------------------------------")

	return nil
}
