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

package scheduler

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	r "gpu-scheduler/resourceinfo"
)

func patchPodAnnotationUUID(bestGPU string) ([]byte, error) {
	patchAnnotations := map[string]interface{}{
		"metadata": map[string]map[string]string{"annotations": {
			"UUID": bestGPU,
		}}}

	return json.Marshal(patchAnnotations)
}

//write GPU_ID to annotation
func patchPodAnnotation(bestGPU string) error {
	fmt.Println("- write gpu uuid in pod annotation")

	patchedAnnotationBytes, err := patchPodAnnotationUUID(bestGPU)
	if err != nil {
		return fmt.Errorf("<error> failed to generate annotations - %s", err)
	}

	_, err = Scheduler.NodeInfoCache.HostKubeClient.CoreV1().Pods(Scheduler.NewPod.Pod.Namespace).Patch(context.TODO(), Scheduler.NewPod.Pod.Name, types.StrategicMergePatchType, patchedAnnotationBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("<error> failed to patch annotations - %s", err)
	}

	return nil
}

func (sched *GPUScheduler) Binding(ctx context.Context, newpod r.QueuedPodInfo, result r.ScheduleResult) {
	fmt.Println("[STEP 5] Binding Pod To Target Node {", newpod.Pod.Name, "}")

	_, cancel := context.WithCancel(ctx)
	defer cancel()

	// if sched.NewPod.IsDeleted() {
	// 	return
	// }

	ip := sched.ClusterManagerHost
	if ip != "" {
		reScore := int(float64(result.TotalScore) * 0.9) //노드 스코어를 줄여야함 -> 구체적으로
		fmt.Println("<test> score: ", result.TotalScore, ",", reScore)
		success, err := UpdateNodeScore(ip, result.BestNode, reScore)
		if err != nil {
			fmt.Println("<error> update node score grpc error - ", err)
		}
		if !success {
			fmt.Println("<error> failed update node score")
		}
	}

	//파드 스펙에 GPU 업데이트
	if newpod.IsGPUPod {
		err := patchPodAnnotation(result.BestGPU)
		if err != nil {
			sched.SchedulingQueue.Add_BackoffQ(&newpod)
			fmt.Println("<error> failed to generate patched annotations - ", err)
			return
		}
	}

	binding := &corev1.Binding{
		ObjectMeta: metav1.ObjectMeta{
			Name: newpod.Pod.Name,
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Binding",
		},
		Target: corev1.ObjectReference{
			APIVersion: "v1",
			Kind:       "Node",
			Name:       result.BestNode,
		},
	}

	err := sched.NodeInfoCache.HostKubeClient.CoreV1().Pods(newpod.Pod.Namespace).Bind(context.TODO(), binding, metav1.CreateOptions{})
	if err != nil {
		fmt.Println("<error> binding error - ", err)
		sched.SchedulingQueue.Add_BackoffQ(&newpod)
		return
	}

	// Emit a Kubernetes event that the Pod was scheduled successfully.
	// message := fmt.Sprintf("<Binding Success> Successfully assigned %s", sched.NewPod.Pod.Name)
	// event := newpod.MakeBindEvent(message)
	// fmt.Println("<Binding Success> Successfully assigned", newpod.Pod.Name)
	fmt.Printf("-----:: Successfully Assigned Pod {%s} ::-----\n", newpod.Pod.Name)
	// PostEvent(event)
}
