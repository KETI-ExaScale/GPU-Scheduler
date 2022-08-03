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
	"math"
	"os"
	"sort"
	"strings"
	"time"

	// batchv1 "k8s.io/api/batch/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	"gpu-scheduler/framework"
	r "gpu-scheduler/resourceinfo"
)

//metric update, score init
func (sched *GPUScheduler) UpdateCache() error {
	fmt.Println("[STEP 1] Update Scheduler Resource Info")
	fmt.Println("# Sending gRPC request...")

	sched.ScheduleResult.InitResult()
	sched.NodeInfoCache.AvailableNodeCount = 0

	for nodeName, nodeInfo := range sched.NodeInfoCache.NodeInfoList {
		if nodeInfo.GRPCHost == "" {
			ip := r.GetMetricCollectorIP(nodeInfo.Pods)
			if ip == "" {
				fmt.Printf("node {%v} cannot find GPU Metric Collector\n", nodeName)
			} else {
				nodeInfo.GRPCHost = ip
			}
		}

		nodeInfo.PluginResult.InitPluginResult()
		err := nodeInfo.NodeMetric.GetNodeMetric(nodeInfo.GRPCHost)
		if err != nil {
			fmt.Println("<error> failed to get node metric, reason:", err)
			nodeInfo.PluginResult.IsFiltered = true
			continue
		}

		for _, uuid := range nodeInfo.NodeMetric.GPU_UUID {
			err := nodeInfo.GPUMetrics[uuid].GetGPUMetric(uuid, nodeInfo.GRPCHost)
			if err != nil {
				fmt.Println("<error> failed to get gpu metric, reason:", err)
				continue
			}

			nodeInfo.PluginResult.GPUCountUp()
		}

		sched.NodeInfoCache.AvailableNodeCount++
	}

	return nil
}

func (sched *GPUScheduler) checkScheduleType() int {
	test := sched.NewPod.Pod.GetAnnotations()
	targetCluster := test["clusterName"]
	// targetCluster := "kubernetes"
	fmt.Println("\n1. check pod annotation[clustername]: ", targetCluster)
	fmt.Println("sched.ClusterInfoCache.MyClusterName: ", sched.ClusterInfoCache.MyClusterName)
	if len(targetCluster) == 0 {
		//do cluster scheduling
		return 1
	} else if targetCluster != sched.ClusterInfoCache.MyClusterName {
		//do cluster binding
		sched.NewPod.TargetCluster = targetCluster
		return 2
	} else {
		//do node scheduling
		return 3
	}
}

func (sched *GPUScheduler) preScheduling(ctx context.Context) {
	processorLock.Lock()
	defer processorLock.Unlock()

	sched.NewPod = sched.NextPod()
	if sched.NewPod == nil || sched.NewPod.Pod == nil {
		return
	}
	fmt.Println("- schedule pod { ", sched.NewPod.Pod.Name, " }")

	// flag := sched.checkScheduleType()

	// if flag == 1 {
	// 	fmt.Println("- need cluster scheduling")
	// 	sched.clusterScheduleOne(ctx)
	// } else if flag == 2 {
	// 	fmt.Println("- need cluster binding")
	// 	go sched.createPodToAnotherCluster(*sched.NewPod)
	// } else {
	// 	fmt.Println("- need node scheduling")
	sched.nodeScheduleOne(ctx)
	// }
}

func patchPodAnnotationClusterNameMarshal(clusterName string) ([]byte, error) {
	fmt.Println("(test) patchPodAnnotationClusterNameMarshal")
	patchAnnotations := map[string]interface{}{
		"metadata": map[string]map[string]string{"annotations": {
			"clusterName": clusterName,
		}}}

	return json.Marshal(patchAnnotations)
}

//write GPU_ID to annotation
func patchPodAnnotationClusterName(clusterName string) error {
	fmt.Println("\n3. Write clusterName in Pod Annotation")

	patchedAnnotationBytes, err := patchPodAnnotationClusterNameMarshal(clusterName)
	if err != nil {
		return fmt.Errorf("failed to generate patched annotations,reason: %v", err)
	}

	_, err = Scheduler.NodeInfoCache.HostKubeClient.CoreV1().Pods(Scheduler.NewPod.Pod.Namespace).Patch(context.TODO(), Scheduler.NewPod.Pod.Name, types.StrategicMergePatchType, patchedAnnotationBytes, metav1.PatchOptions{})
	if err != nil {
		fmt.Println("patchPodAnnotation error: ", err)
		return err
	}

	// fmt.Println("updated annotation[clustername]: ", Scheduler.NewPod.Pod.Annotations["clusterName"])
	//파드 어노테이션은 ㄴnewpod에 반영이 안됨

	return nil
}

func (sched *GPUScheduler) getNewClusterClient(clusterName string) (*kubernetes.Clientset, error) {
	fmt.Println("get new cluster client: ", clusterName)
	config, err := clientcmd.BuildConfigFromFlags("https://10.0.5.66:6443", os.Getenv("KUBECONFIG"))
	config.TLSClientConfig = rest.TLSClientConfig{Insecure: true}
	// config.BearerToken =
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	// config.list_kube_config_contexts()
	return clientset, nil
}

func (sched *GPUScheduler) createPodToAnotherCluster(qpod r.QueuedPodInfo) {
	fmt.Println("**Create Pod To Another Cluster**")

	new_clientset, err := sched.getNewClusterClient(qpod.TargetCluster)
	if err != nil {
		fmt.Println("failed to get new clinent set/ pod=", qpod.Pod.Name)
		fmt.Println("err: ", err)
		sched.SchedulingQueue.Add_BackoffQ(&qpod)
		return
	}

	newPod := new_clientset.CoreV1().Pods(qpod.Pod.Namespace)
	_, err = newPod.Create(context.TODO(), qpod.Pod, metav1.CreateOptions{})
	if err != nil {
		fmt.Println("failed to create pod to another cluster/ pod=", qpod.Pod.Name, " /clustername=", qpod.TargetCluster)
		fmt.Println("err: ", err)
		sched.SchedulingQueue.Add_BackoffQ(&qpod)
		return
	}

	fmt.Println("cluster create success - ", qpod.Pod.Name)
	sched.deletePodFromSchedulingQueue(qpod)
}

func (sched *GPUScheduler) clusterScheduleOne(ctx context.Context) {
	fmt.Println("\n2. cluster scheduling start")
	// fmt.Println("2-1. cluster Filtering")
	// fmt.Println("-cluster{kubernetes, gpu1}")
	// fmt.Println("2-2. cluster Scoring")
	// fmt.Println("-cluster{kubernetes} average node score : 80")
	// fmt.Println("-cluster{gpu1} average node score : 60")
	findCluster := false
	clusterToCreat := ""
	for _, _ = range sched.ClusterInfoCache.ClusterInfoList {
		clusterToCreat = "kubernetes"
		fmt.Println("<cluster to create>: ", clusterToCreat)
		sched.NewPod.TargetCluster = clusterToCreat
		findCluster = true
		err := patchPodAnnotationClusterName(clusterToCreat)
		if err != nil {
			sched.SchedulingQueue.Add_BackoffQ(sched.NewPod)
			fmt.Println("failed to generate patched annotations,reason: ", err)
			return
		}
		// clusterToCreat = clusterInfo.ClusterName
		// if clusterToCreat != sched.ClusterInfoCache.MyClusterName {
		// 	fmt.Println("<cluster to create>: ", "kubernetes")
		// 	sched.NewPod.TargetCluster = clusterToCreat
		// 	err := patchPodAnnotationClusterName(clusterToCreat)
		// 	if err != nil {
		// 		sched.SchedulingQueue.Add_BackoffQ(sched.NewPod)
		// 		fmt.Println("failed to generate patched annotations,reason: ", err)
		// 		return
		// 	}
		// 	findCluster = true
		// 	break
		// }
	}

	if !findCluster {
		sched.SchedulingQueue.Add_BackoffQ(sched.NewPod)
		fmt.Println("cannot find cluster")
		return
	}

	fmt.Println("clusterToCreate: ", clusterToCreat, "| myClusterName: ", sched.ClusterInfoCache.MyClusterName)

	if clusterToCreat == sched.ClusterInfoCache.MyClusterName {
		fmt.Println("\ntarget cluster name is my cluster!")
		sched.nodeScheduleOne(ctx)
	} else {
		fmt.Println("\nntarget cluster name is not my cluster!")
		go sched.createPodToAnotherCluster(*sched.NewPod)
		sched.deletePodFromSchedulingQueue(sched.NewPod)
		return
	}
}

func (sched *GPUScheduler) nodeScheduleOne(ctx context.Context) {
	fmt.Println("node scheduling { ", sched.NewPod.Pod.Name, " }")

	pod := sched.NewPod.Pod

	startTime := time.Now()

	sched.frameworkForPod()

	klog.V(3).InfoS("Attempting to schedule pod", "pod", klog.KObj(pod))

	//[STEP 1] Update Scheduler
	err := sched.UpdateCache() //metric update, score init
	if err != nil {
		fmt.Println("nodeinfolist update error")
		sched.SchedulingQueue.Add_BackoffQ(sched.NewPod)
		return
	}

	sched.NodeInfoCache.DumpCache() //확인용

	//[STEP 2,3] Schedule a pod
	err = sched.schedulePod()
	if err != nil {
		fmt.Println("schedulePod error")
		sched.SchedulingQueue.Add_BackoffQ(sched.NewPod)
		return
	}

	//[STEP 4]get Best Node/GPU
	sched.GetBestNodeAndGPU()

	sched.NodeInfoCache.UpdatePodState(pod, r.Assumed)

	elapsedTime := time.Since(startTime)

	fmt.Println("#Scheduling Time : ", elapsedTime.Seconds())

	//[STEP 5] Binding Stage
	go sched.Binding(ctx, *sched.NewPod, *sched.ScheduleResult)
}

func (sched *GPUScheduler) schedulePod() error {
	if sched.NodeInfoCache.TotalNodeCount == 0 {
		return fmt.Errorf("there is no node to schedule")
	}

	//[STEP 2] Filtering Stage
	err := sched.Framework.RunFilteringPlugins(sched.NodeInfoCache, sched.NewPod)
	if err != nil {
		fmt.Println("Run filtering plugins error")
		return err
	}

	//[STEP 3] Scoring Stage
	err = sched.Framework.RunScoringPlugins(sched.NodeInfoCache, sched.NewPod)
	if err != nil {
		fmt.Println("Run scoring plugins error")
		return err
	}

	return nil
}

// func (sched *GPUScheduler) assume(assumed *corev1.Pod, host string) error {
// 	// Optimistically assume that the binding will succeed and send it to apiserver
// 	// in the background.
// 	// If the binding fails, scheduler will release resources allocated to assumed pod
// 	// immediately.
// 	assumed.Spec.NodeName = host

// 	if err := sched.NodeInfoCache.AssumePod(assumed); err != nil {
// 		klog.ErrorS(err, "Scheduler cache AssumePod failed")
// 		return err
// 	}
// 	// // if "assumed" is a nominated pod, we should remove it from internal cache
// 	// if sched.SchedulingQueue != nil {
// 	// 	sched.SchedulingQueue.DeleteNominatedPodIfExists(assumed)
// 	// }

// 	return nil
// }

func (sched *GPUScheduler) frameworkForPod() {
	if sched.NewPod.IsGPUPod {
		sched.Framework = framework.GPUPodSpreadFramework()
	} else {
		sched.Framework = framework.NonGPUPodFramework()
	}
}

func (sched *GPUScheduler) GetBestNodeAndGPU() {
	fmt.Println("[STEP 4] Get Best Node/GPU")
	for nodeName, nodeInfo := range sched.NodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			sched.getTotalScore(nodeInfo, sched.NewPod.RequestedResource.GPUCount)
			if sched.ScheduleResult.TotalScore < nodeInfo.PluginResult.TotalScore {
				sched.ScheduleResult.BestNode = nodeName
				sched.ScheduleResult.BestGPU = nodeInfo.PluginResult.BestGPU
				sched.ScheduleResult.TotalScore = nodeInfo.PluginResult.TotalScore
			}
		}

	}
	fmt.Println("#Result: BestNode {", sched.ScheduleResult.BestNode, "}")
	fmt.Println("#Result: BestGPU {", sched.ScheduleResult.BestGPU, "}")
}

func (sched *GPUScheduler) getTotalScore(nodeinfo *r.NodeInfo, requestedGPU int) {
	score := nodeinfo.PluginResult
	if !sched.NewPod.IsGPUPod {
		score.TotalScore = score.NodeScore
		return
	}
	sched.getTotalGPUScore(nodeinfo, requestedGPU)
	score.TotalScore = int(math.Round(float64(score.NodeScore)*sched.SchedulingPolicy.NodeWeight +
		float64(score.TotalGPUScore)*sched.SchedulingPolicy.GPUWeight))
}

func (sched *GPUScheduler) getTotalGPUScore(nodeinfo *r.NodeInfo, requestedGPU int) {
	score := nodeinfo.PluginResult
	totalGPUScore, bestGPU := float64(0), ""

	//최종 GPUScore 순으로 정렬
	type gs []*r.GPUScore
	sortedGPUScore := make(gs, 0, len(score.GPUScores))
	for _, d := range score.GPUScores {
		sortedGPUScore = append(sortedGPUScore, d)
	}
	sort.SliceStable(sortedGPUScore, func(i, j int) bool {
		return sortedGPUScore[i].GPUScore > sortedGPUScore[j].GPUScore
	})

	for _, a := range sortedGPUScore {
		fmt.Println("|", a.UUID, " | ", a.GPUScore, " | ", a.IsFiltered, "|")
	}

	//NVLink GPU 존재X
	if nodeinfo.NodeMetric.NVLinkList == nil {
		bestGPUScore := sortedGPUScore[:requestedGPU] //스코어 점수 상위 N개의 GPU
		for _, gpu := range bestGPUScore {
			totalGPUScore += float64(gpu.GPUScore) * float64(1/float64(requestedGPU))
			bestGPU += gpu.UUID + ","
		}
		score.TotalGPUScore = int(math.Round(totalGPUScore))
		score.BestGPU = strings.Trim(bestGPU, ",")

		return
	}

	//NVLink GPU 존재O
	sched.checkNVLinkGPU(nodeinfo)
	gpucnt := requestedGPU
	for gpucnt > 0 {
		if gpucnt == 1 { //하나 선택
			for _, gpu := range sortedGPUScore {
				if !gpu.IsFiltered && !gpu.IsSelected {
					totalGPUScore += float64(gpu.GPUScore) * float64(1/float64(requestedGPU))
					bestGPU += gpu.UUID + ","
					gpu.IsSelected = true
					gpucnt--
					break
				}
			}
		} else { //nvlink 쌍 선택 가능
			var a1 []string
			s1, g1, i1 := float64(0), "", -1
			var i2 []int
			s2, g2 := float64(0), ""

			for i, nvl := range nodeinfo.NodeMetric.NVLinkList {
				if !nvl.IsFiltered && !nvl.IsSelected {
					s1 = float64(nvl.Score)
					g1 += nvl.GPU1 + "," + nvl.GPU2 + ","
					i1 = i
					a1 = append(a1, nvl.GPU1, nvl.GPU2)
					break
				}
			}

			for j, gpu := range sortedGPUScore {
				if !gpu.IsFiltered && !gpu.IsSelected {
					s2 += float64(gpu.GPUScore)
					g2 += gpu.UUID + ","
					i2 = append(i2, j)
				}
				if len(i2) == 2 {
					s2 /= 2
					break
				}
			}

			if s1 > s2 {
				totalGPUScore += float64(s1) * float64(2/float64(requestedGPU))
				bestGPU += g1 + ","
				nodeinfo.NodeMetric.NVLinkList[i1].IsSelected = true
				for _, gpu := range sortedGPUScore {
					if gpu.UUID == a1[0] || gpu.UUID == a1[1] {
						gpu.IsSelected = true
					}
				}
			} else {
				totalGPUScore += float64(s2) * float64(2/float64(requestedGPU))
				bestGPU += g2 + ","
				sortedGPUScore[i2[0]].IsSelected = true
				sortedGPUScore[i2[1]].IsSelected = true
			}

			gpucnt -= 2
		}
	}

}

func (sched *GPUScheduler) checkNVLinkGPU(nodeinfo *r.NodeInfo) {
	fmt.Println("#20. Check NVLink GPU")

	for _, nvl := range nodeinfo.NodeMetric.NVLinkList {
		if nodeinfo.PluginResult.GPUScores[nvl.GPU1].IsFiltered ||
			nodeinfo.PluginResult.GPUScores[nvl.GPU2].IsFiltered {
			nvl.IsFiltered = true
			continue
		}
		score := float64(nodeinfo.PluginResult.GPUScores[nvl.GPU1].GPUScore+
			nodeinfo.PluginResult.GPUScores[nvl.GPU2].GPUScore) / 2
		nvl.Score = int(math.Round(score * float64(1+float64(sched.SchedulingPolicy.NVLinkWeightPercentage)/100)))
	}
}

// func (sched *GPUScheduler) PostEvent(event *corev1.Event) {
// 	_, err := sched.NodeInfoCache.HostKubeClient.CoreV1().Events(event.InvolvedObject.Namespace).Update(context.TODO(), event, metav1.UpdateOptions{})
// 	if err != nil {
// 		fmt.Println("post event failed")
// 	}
// }

// func (sched *GPUScheduler) IsThereAnyNode() bool {
// 	if sched.NodeInfoCache.TotalNodeCount == 0 {
// 		sched.NewPod.FailedScheduling()
// 		message := fmt.Sprintf("pod (%s) failed to fit in any node", sched.NewPod.Pod.ObjectMeta.Name)
// 		log.Println(message)
// 		event := sched.NewPod.MakeNoNodeEvent(message)
// 		PostEvent(event)
// 		return false
// 	}
// 	return true
// }
