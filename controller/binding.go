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
	"log"

	"gpu-scheduler/postevent"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var isTrue bool = true

//write GPUID to annotation
func PatchPodAnnotationSpec(devId string) ([]byte, error) {
	patchAnnotations := map[string]interface{}{
		"metadata": map[string]map[string]string{"annotations": {
			"UUID": devId,
		}}}

	return json.Marshal(patchAnnotations)
}

func PatchPodAnnotation(pod *corev1.Pod, devId string) error {
	host_config, _ := rest.InClusterConfig()
	host_kubeClient := kubernetes.NewForConfigOrDie(host_config)

	patchedAnnotationBytes, err := PatchPodAnnotationSpec(devId)
	if err != nil {
		return fmt.Errorf("failed to generate patched annotations,reason: %v", err)
	}

	pod, err = host_kubeClient.CoreV1().Pods(pod.Namespace).Patch(context.TODO(), pod.Name, types.StrategicMergePatchType, patchedAnnotationBytes, metav1.PatchOptions{})
	if err != nil {
		fmt.Println("patchPodAnnotation error: ", err)
		return err
	}
	return nil
}

func Binding(pod *corev1.Pod, bestNode corev1.Node) error {
	var devId string

	fmt.Println("3. Binding stage")

	//임의로 지정된 deviceID
	for _, container := range pod.Spec.Containers { //파드의 컨테이너 검사
		GPUreq := container.Resources.Limits["keti.com/mpsgpu"]
		if GPUreq.String() == "2" {
			devId = "GPU-a06cd524-72c4-d6f0-4eda-d64af512dd8b,GPU-f6db4146-092d-146f-0814-8ff90b04f3d2"
		} else {
			if isTrue {
				devId = "GPU-a06cd524-72c4-d6f0-4eda-d64af512dd8b"
				isTrue = false
			} else {
				devId = "GPU-f6db4146-092d-146f-0814-8ff90b04f3d2"
				isTrue = true
			}
		}
	}

	//파드 스펙에 GPU 업데이트
	err := PatchPodAnnotation(pod, devId)
	if err != nil {
		return fmt.Errorf("failed to generate patched annotations,reason: %v", err)
	}

	binding := &corev1.Binding{
		ObjectMeta: metav1.ObjectMeta{
			Name: pod.Name,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Binding",
		},
		Target: corev1.ObjectReference{
			APIVersion: "v1",
			Kind:       "Node",
			Name:       bestNode.Name,
		},
	}

	host_config, _ := rest.InClusterConfig()
	host_kubeClient := kubernetes.NewForConfigOrDie(host_config)

	err = host_kubeClient.CoreV1().Pods(pod.Namespace).Bind(context.TODO(), binding, metav1.CreateOptions{})
	if err != nil {
		fmt.Println("binding error: ", err)
		return err
	}

	// Emit a Kubernetes event that the Pod was scheduled successfully.
	message := fmt.Sprintf("Successfully assigned %s to %s and GPU UUID is %s", pod.ObjectMeta.Name, bestNode.ObjectMeta.Name, devId)
	event := postevent.MakeBindEvent(pod, message)
	log.Println(message)
	err = postevent.PostEvent(event)
	if nil != err {
		fmt.Println("binding>postEvent error: ", err)
		return err
	}

	return nil
}
