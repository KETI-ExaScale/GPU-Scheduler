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

package main

import (
	"context"
	s "gpu-scheduler/scheduler"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	log.Println("-----Start GPU Scheduler-----")

	//test() 에러파드 재스케줄링 테스트

	quitChan := make(chan struct{}) //struct타입을 전송할 수 있는 통신용 채널 생성
	var wg sync.WaitGroup           //모든 고루틴이 종료될 때 까지 대기할 때 사용

	hostConfig, _ := rest.InClusterConfig()
	hostKubeClient := kubernetes.NewForConfigOrDie(hostConfig)

	informerFactory := informers.NewSharedInformerFactory(hostKubeClient, 0)

	s.Scheduler = s.NewGPUScheduler(hostKubeClient) //스케줄러 생성

	s.AddAllEventHandlers(s.Scheduler, informerFactory)
	wg.Add(1)
	go informerFactory.Core().V1().Pods().Informer().Run(quitChan)

	//폴리시 이벤트 핸들러 추가

	wg.Add(1)
	ctx, cancel := context.WithCancel(context.Background())
	s.Scheduler.Run(ctx) //schedule_one context

	// wg.Add(1)
	// go s.Scheduler.RunScheduler(quitChan, &wg) //[A]Run Scheduling Routine

	// wg.Add(1)
	// go s.Scheduler.MonitorUnscheduledPods(quitChan, &wg) //[B]MonitorUnscheduledPods (start api server watching)

	// wg.Add(1)
	// go s.Scheduler.WatchLowPerformancePod(quitChan, &wg) //[D]Cache Thread

	// wg.Add(1)
	// go s.Scheduler.WatchSchedulerPolicy(quitChan, &wg) //[E] Watch Scheduler Policy

	// // wg.Add(1)
	// // go s.Scheduler.ReconcileRescheduledPods(30, quitChan, &wg) //[C]Queue Flush(BackOff,Reschedule)

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM) //SIGINT를 지정하여 기다리는 루틴
	for {
		select {
		case <-signalChan:
			log.Printf("Shutdown signal received, exiting...")
			close(quitChan)
			cancel()
			wg.Wait() //모든 고루틴이 종료될 때까지 대기
			os.Exit(0)
		}
	}
}

// func test() {
// 	host_config, _ := rest.InClusterConfig()
// 	host_kubeClient := kubernetes.NewForConfigOrDie(host_config)
// 	selector := fields.SelectorFromSet(fields.Set{"status.phase": "Failed"})
// 	podlist, err := host_kubeClient.CoreV1().Pods(v1.NamespaceAll).List(context.TODO(), metav1.ListOptions{
// 		FieldSelector: selector.String(),
// 		LabelSelector: labels.Everything().String(),
// 	})
// 	if err != nil {
// 		fmt.Errorf("failed to get Pods assigned to node")
// 	}
// 	fmt.Println(podlist)
// 	errorpod := podlist.Items[0]
// 	newpod := errorpod.DeepCopy()

// 	binding := &v1.Binding{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name: errorpod.Name,
// 		},
// 		TypeMeta: metav1.TypeMeta{
// 			APIVersion: "v1",
// 			Kind:       "Binding",
// 		},
// 		Target: v1.ObjectReference{
// 			APIVersion: "v1",
// 			Kind:       "Node",
// 			Name:       "gpuserver2",
// 		},
// 	}
// 	config.Host_kubeClient.CoreV1().Pods(errorpod.Namespace).Delete(context.TODO(), errorpod.Name, metav1.DeleteOptions{})

// 	err = config.Host_kubeClient.CoreV1().Pods(newpod.Namespace).Bind(context.TODO(), binding, metav1.CreateOptions{})
// 	if err != nil {
// 		fmt.Println("binding error: ", err)
// 	}
// }
