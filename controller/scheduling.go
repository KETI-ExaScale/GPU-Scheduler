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
	"fmt"
	"gpu-scheduler/algorithm/predicates"
	"gpu-scheduler/algorithm/priorities"
	"log"
	"sync"
	"time"

	"gpu-scheduler/config"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var processorLock = &sync.Mutex{}

//새로 생성된 파드 감시
func MonitorUnscheduledPods(done chan struct{}, wg *sync.WaitGroup) {
	fmt.Println("called MonitorUnscheduledPods")
	pods, errc := WatchUnscheduledPods() //새롭게 들어온 파드 얻음
	for {
		select {
		case err := <-errc:
			fmt.Println("MonitorUnschedeuledPods>case err: ", err)
		case pod := <-pods:
			processorLock.Lock()
			time.Sleep(2 * time.Second)
			fmt.Println("Watch->pod->scheduling")
			err := SchedulePod(pod) //새롭게 들어온 파드 스케줄링
			if err != nil {
				fmt.Println("MonitorUnschedeuledPods>case pod: ", err)
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
func ReconcileUnscheduledPods(interval int, done chan struct{}, wg *sync.WaitGroup) {
	fmt.Println("called ReconcileUnscheduledPods")
	for {
		select {
		case <-time.After(time.Duration(interval) * time.Second):
			fmt.Println("called ReconcileUnscheduledPods>time duration")
			err := SchedulePods()
			if err != nil {
				log.Println("ReconcileUnscheduledPods error: ", err)
			}
		case <-done:
			wg.Done()
			log.Println("Stopped reconciliation loop.")
			return
		}
	}
}

func WatchUnscheduledPods() (<-chan *corev1.Pod, <-chan error) {
	fmt.Println("called WatchUnscheduledPods")
	pods := make(chan *corev1.Pod)
	errc := make(chan error, 1)

	host_config, _ := rest.InClusterConfig()
	host_kubeClient := kubernetes.NewForConfigOrDie(host_config)

	go func() {
		for {
			watch, err := host_kubeClient.CoreV1().Pods("").Watch(context.TODO(), metav1.ListOptions{
				FieldSelector: fmt.Sprintf("spec.schedulerName=%s,spec.nodeName=", config.SchedulerName),
			})
			if err != nil {
				fmt.Println("watchUnscheduledPods error: ", err)
				errc <- err
			}
			for event := range watch.ResultChan() {
				if event.Type == "ADDED" {
					fmt.Println("ADDED")
					pods <- event.Object.(*corev1.Pod)
				}
			}
		}
	}()
	return pods, errc
}

func GetUnscheduledPods() ([]*corev1.Pod, error) {
	fmt.Println("called GetUnscheduledPods")
	rescheduledPods := make([]*corev1.Pod, 0)

	host_config, _ := rest.InClusterConfig()
	host_kubeClient := kubernetes.NewForConfigOrDie(host_config)

	podList, err := host_kubeClient.CoreV1().Pods(corev1.NamespaceAll).List(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.schedulerName=%s,status.phase=Pending", config.SchedulerName),
	})

	if err != nil {
		fmt.Println("getUnscheduledPods error: ", err)
		return rescheduledPods, err
	}

	for _, pod := range podList.Items {
		if pod.Spec.NodeName == "" {
			fmt.Println("pod.Spec.NodeName: ", pod.Spec.NodeName, "/pod.Spec.SchedulerName: ", pod.Spec.SchedulerName, "/pod.Status.Phase: ", pod.Status.Phase, "/pod.name: ", pod.Name)
			rescheduledPods = append(rescheduledPods, &pod)
		}
	}

	return rescheduledPods, nil
}

func SchedulePod(pod *corev1.Pod) error {
	fmt.Println("--------------------------------------------------------------")
	fmt.Println("newPodName:", pod.ObjectMeta.Name)

	nodes, err := predicates.Filtering(pod)
	if err != nil {
		fmt.Println("schedulePod>Scoring Filtering: ", err)
		return err
	}

	if len(nodes) == 0 {
		return fmt.Errorf("Unable to schedule pod (%s) failed to fit in any node", pod.ObjectMeta.Name)
	}

	bestNode, err := priorities.Scoring(nodes, pod)
	if err != nil {
		fmt.Println("schedulePod>Scoring error: ", err)
		return err
	}

	err = Binding(pod, bestNode.Node)
	if err != nil {
		fmt.Println("schedulePod>Binding error: ", err)
		return err
	}

	return nil
}

func SchedulePods() error { //called by reconcileUnscheduledPods
	fmt.Println("called SchedulePods")
	processorLock.Lock()
	defer processorLock.Unlock()
	pods, err := GetUnscheduledPods()
	if err != nil {
		log.Println("SchedulePods>getUnscheduledPods error: ", err)
		return err
	}
	for _, pod := range pods { //스케줄링 대기중인 파드들 하나씩 스케줄링
		fmt.Println("reconcile called schedulepods: ", pod.Name)
		err := SchedulePod(pod)
		if err != nil {
			log.Println("SchedulePods>schedulepod error: ", err)
		}
	}
	return nil
}
