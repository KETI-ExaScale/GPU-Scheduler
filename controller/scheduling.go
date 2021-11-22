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
	"errors"
	"fmt"
	"gpu-scheduler/algorithm/predicates"
	"gpu-scheduler/algorithm/priorities"
	"gpu-scheduler/config"
	resource "gpu-scheduler/resourceinfo"
	"log"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var processorLock = &sync.Mutex{}

//새로 생성된 파드 감시
func MonitorUnscheduledPods(done chan struct{}, wg *sync.WaitGroup) {
	if config.Debugg {
		fmt.Println("[A] Watching New Pod")
	}

	pods, errc := WatchNewPods() //새롭게 들어온 파드 얻음
	for {
		select {
		case err := <-errc:
			fmt.Println(err)
		case pod := <-pods:
			processorLock.Lock()
			time.Sleep(2 * time.Second)
			if config.Debugg {
				fmt.Println("----------------------------------------------------------------------------------------------------------------------------")
				fmt.Print("<New Pod ADDED> ")
			}
			err := SchedulePod(pod) //새롭게 들어온 파드 스케줄링
			if err != nil {
				fmt.Println("Failed Pod Scheduling: ", err)
				if config.Debugg {
					fmt.Println("----------------------------------------------------------------------------------------------------------------------------")
				}
			}
			processorLock.Unlock()
		case <-done:
			wg.Done()
			log.Println("Stopped scheduler.")
			return
		}
	}
}

//스케줄링 실패한 파드 일정 주기로 재스케줄링
func ReconcileRescheduledPods(interval int, done chan struct{}, wg *sync.WaitGroup) {
	if config.Debugg {
		fmt.Println("[B] ReScheduling Loop")
	}

	for {
		select {
		case <-time.After(time.Duration(interval) * time.Second):
			t := time.Now()
			if config.Re {
				fmt.Printf("<Re> Reschedule After time duration [%d:%d:%d]\n", t.Hour(), t.Minute(), t.Second())
			}
			err := SchedulePods()
			if err != nil {
				log.Println("Failed Pod ReScheduling: ", err)
				if config.Debugg {
					fmt.Println("----------------------------------------------------------------------------------------------------------------------------")
				}
			}
		case <-done:
			wg.Done()
			log.Println("Stopped reconciliation loop.")
			return
		}
	}
}

func WatchNewPods() (<-chan *corev1.Pod, <-chan error) {
	pods := make(chan *corev1.Pod)
	errc := make(chan error, 1)

	go func() {
		for {
			watch, err := config.Host_kubeClient.CoreV1().Pods("").Watch(context.TODO(), metav1.ListOptions{
				FieldSelector: fmt.Sprintf("spec.schedulerName=%s,spec.nodeName=", config.SchedulerName),
			})
			if err != nil {
				fmt.Println("WatchNewPods error: ", err) //debug
				errc <- err
			}
			for event := range watch.ResultChan() {
				if event.Type == "ADDED" {
					pods <- event.Object.(*corev1.Pod)
				}
			}
		}
	}()
	return pods, errc
}

func GetUnscheduledPods() ([]*corev1.Pod, error) {
	rescheduledPods := make([]*corev1.Pod, 0)

	podList, err := config.Host_kubeClient.CoreV1().Pods(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.schedulerName=%s,status.phase=Pending,spec.nodeName=", config.SchedulerName),
	})

	if err != nil {
		fmt.Println("getUnscheduledPods error: ", err)
		return rescheduledPods, err
	}

	for _, pod := range podList.Items {
		if pod.ObjectMeta.Annotations["failedCount"] != "" {
			//fmt.Println("pod.Spec.NodeName: ", pod.Spec.NodeName, "pod.Spec.SchedulerName: ", pod.Spec.SchedulerName, "pod.Status.Phase: ", pod.Status.Phase, "pod.ObjectMeta.Annotations[failedcount]:", pod.ObjectMeta.Annotations["failedCount"])
			rescheduledPods = append(rescheduledPods, &pod)
		}
	}

	return rescheduledPods, nil
}

const (
	policy1 = "node-gpu-score-weight"
	policy2 = "pod-re-schedule-permit"
)

func SchedulePod(pod *corev1.Pod) error {
	fmt.Println("PodName:", pod.ObjectMeta.Name)

	if config.Policy {
		weightPolicy := fmt.Sprintf("{node weight : %v} {gpu weight : %v}", config.NodeWeight, config.GPUWeight)
		reSchedulePolicy := "reSchedule :" + config.ReSchedule

		fmt.Println("<GPU Scheduler Policy List>")
		fmt.Println("              NAME             |  STATUS |              POLICIES                  ")
		fmt.Printf(" %-30v| Enabled | %-40v\n", policy1, weightPolicy)
		fmt.Printf(" %-30v| Enabled | %-40v\n", policy2, reSchedulePolicy)
	}

	//get a new pod
	newPod := resource.GetNewPodInfo(pod)

	//[step0] update nodeInfoList
	var nodeInfoList []*resource.NodeInfo
	*resource.AvailableNodeCount = 0

	nodeInfoList, err := resource.NodeUpdate(nodeInfoList)
	if err != nil {
		return errors.New("error get multimetric")
	}

	if *resource.AvailableNodeCount == 0 {
		return errors.New("there is no node to schedule")
	}

	//[step1] Filtering Stage
	nodes, err := predicates.Filtering(newPod, nodeInfoList)
	if err != nil {
		return errors.New("every node is filtered")
	}

	//[step2] Scoring Stage
	nodes, err = priorities.Scoring(nodes, newPod)
	if err != nil {
		return errors.New("scoring error")
	}

	//Get Best Node/GPU
	result := resource.GetBestNodeAneGPU(nodes, newPod.RequestedResource.GPUMPS)

	//[step3] Binding Stage
	err = Binding(newPod, result)
	if err != nil {
		return errors.New("binding error")
	}

	return nil
}

func SchedulePods() error { //called by reconcileUnscheduledPods
	processorLock.Lock()
	defer processorLock.Unlock()
	pods, err := GetUnscheduledPods()
	if err != nil {
		log.Println("SchedulePods>getUnscheduledPods error: ", err)
		return err
	}
	for _, pod := range pods { //스케줄링 대기중인 파드들 하나씩 스케줄링
		if config.Debugg {
			fmt.Println("----------------------------------------------------------------------------------------------------------------------------")
			fmt.Print("<Reschedule Pod ADDED> ")
		}
		err := SchedulePod(pod)
		if err != nil {
			return err
		}
	}
	return nil
}
