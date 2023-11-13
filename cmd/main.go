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
	"fmt"
	r "gpu-scheduler/resourceinfo"
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
	//reschedulingTest() // 에러파드 재스케줄링 테스트

	quitChan := make(chan struct{})
	var wg sync.WaitGroup

	hostConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}
	hostKubeClient := kubernetes.NewForConfigOrDie(hostConfig)

	informerFactory := informers.NewSharedInformerFactory(hostKubeClient, 0)

	s.Scheduler, err = s.NewGPUScheduler(hostKubeClient) // GPU Scheduler Create
	if err != nil {
		log.Fatal(err)
	}

	err = s.Scheduler.InitClusterManager() // Init Cluster Manager
	if err != nil {
		r.KETI_LOG_L3(fmt.Sprintf("[error] init cluster manager : %s", err))
	}

	s.AddAllEventHandlers(s.Scheduler, informerFactory) // Scheduler Event Handler (node,pod craete/delete/update | policy update)
	wg.Add(1)
	go informerFactory.Start(quitChan)

	wg.Add(1)
	ctx, cancel := context.WithCancel(context.Background())
	s.Scheduler.Run(ctx) // Schedule Pod Routine & Scheduling Queue Flush Routine

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case <-signalChan:
			r.KETI_LOG_L3(fmt.Sprint("[signal] shitdown signal received, exiting..."))
			close(quitChan)
			cancel()
			wg.Wait()
			os.Exit(0)
		}
	}
}

// func reschedulingTest() {
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
