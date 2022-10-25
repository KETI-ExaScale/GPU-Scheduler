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
	"sort"
	"strings"
	"time"

	// batchv1 "k8s.io/api/batch/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"gpu-scheduler/framework"
	r "gpu-scheduler/resourceinfo"
)

const (
	scheduleTypeError  = 0
	clusterScheuduling = 1
	nodeScheduling     = 2
	clusterBinding     = 3
)

//metric update, score init
func (sched *GPUScheduler) UpdateCache() {
	r.KETI_LOG_L3("[STEP 1] Update Scheduler Resource Info")
	r.KETI_LOG_L2("- Sending gRPC Request To Metric Collector ...")

	sched.ScheduleResult.InitResult()
	sched.NodeInfoCache.AvailableNodeCount = 0

	for nodeName, nodeInfo := range sched.NodeInfoCache.NodeInfoList {
		nodeInfo.PluginResult.InitPluginResult()
		nodeInfo.NodeMetric.InitNVLinkList()

		if nodeInfo.MetricCollectorIP == "" {
			ip := r.GetMetricCollectorIP(nodeInfo.Pods)
			if ip == "" {
				r.KETI_LOG_L2(fmt.Sprintf("- node {%s} cannot find GPU Metric Collector", nodeName))
				nodeInfo.PluginResult.IsFiltered = true
				continue
			} else {
				nodeInfo.MetricCollectorIP = ip
			}
		}

		err := nodeInfo.NodeMetric.GetNodeMetric(nodeInfo.MetricCollectorIP)
		if err != nil {
			r.KETI_LOG_L3(fmt.Sprintf("<error> failed to get node metric - %s", err))
			nodeInfo.PluginResult.IsFiltered = true
			continue
		}

		for _, uuid := range nodeInfo.NodeMetric.GPU_UUID {
			err := nodeInfo.GPUMetrics[uuid].GetGPUMetric(uuid, nodeInfo.MetricCollectorIP)
			if err != nil {
				r.KETI_LOG_L3(fmt.Sprintf("<error> failed to get gpu metric - %s", err))
				nodeInfo.PluginResult.GPUScores[uuid].IsFiltered = true
				continue
			}

			nodeInfo.PluginResult.GPUCountUp()
		}

		sched.NodeInfoCache.NodeCountUP()
	}
}

func (sched *GPUScheduler) checkScheduleType() int {
	targetCluster := sched.NewPod.Pod.Annotations["clusterName"]
	r.KETI_LOG_L1(fmt.Sprintf("\n1. check pod annotation[clusterName]: %s", targetCluster))
	if targetCluster == "" {
		if sched.ClusterInfoCache.Available && sched.AvailableClusterManager {
			return clusterScheuduling //do cluster scheduling
		}
		return nodeScheduling //do node scheduling
	} else if targetCluster != sched.ClusterInfoCache.MyClusterName {
		sched.NewPod.TargetCluster = targetCluster
		if !sched.ClusterInfoCache.Available {
			return scheduleTypeError //cannot create to target cluster
		}
		return clusterBinding //do cluster binding
	} else { //targetCluster == myCluster
		return nodeScheduling //do node scheduling
	}
}

func (sched *GPUScheduler) preScheduling(ctx context.Context) {
	processorLock.Lock()
	defer processorLock.Unlock()

	sched.NewPod = sched.NextPod()
	if sched.NewPod == nil || sched.NewPod.Pod == nil {
		return
	}

	r.KETI_LOG_L3("\n-----:: Pod Scheduling Start ::-----")
	r.KETI_LOG_L2(fmt.Sprintf("- pod name: %s", sched.NewPod.Pod.Name))

	r.KETI_LOG_L1(fmt.Sprintf("<test> pod request resource: %+v", sched.NewPod.RequestedResource))

	flag := sched.checkScheduleType()

	if flag == clusterScheuduling {
		r.KETI_LOG_L1("- need cluster scheduling")
		sched.clusterScheduleOne(ctx)
	} else if flag == clusterBinding {
		r.KETI_LOG_L1("- need cluster binding")
		sched.createPodToAnotherCluster(ctx, *sched.NewPod)
	} else if flag == nodeScheduling {
		r.KETI_LOG_L1("- need node scheduling")
		sched.nodeScheduleOne(ctx)
	} else {
		r.KETI_LOG_L3("<error> cannot create to target cluster, target cluster is unavailable")
	}
}

//write clusterName to pod annotation
func (sched *GPUScheduler) patchPodAnnotationClusterName(clusterName string) error {
	r.KETI_LOG_L2("- write clusterName in pod annotation")

	patchAnnotations := map[string]interface{}{
		"metadata": map[string]map[string]string{"annotations": {
			"clusterName": clusterName,
		}}}

	patchedAnnotationBytes, err := json.Marshal(patchAnnotations)
	if err != nil {
		return fmt.Errorf("<error> failed to generate patched annotations,reason: %v", err)
	}

	_, err = Scheduler.NodeInfoCache.HostKubeClient.CoreV1().Pods(sched.NewPod.Pod.Namespace).Patch(context.TODO(), Scheduler.NewPod.Pod.Name, types.StrategicMergePatchType, patchedAnnotationBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("<error> patchPodAnnotation error: %v", err)
	}

	//파드 어노테이션은 sched.newpod에 반영이 안됨

	return nil
}

func (sched *GPUScheduler) createPodToAnotherCluster(ctx context.Context, qpod r.QueuedPodInfo) {
	r.KETI_LOG_L2("- Create Pod To Another Cluster")

	targetCluster := qpod.TargetCluster
	if !sched.ClusterInfoCache.ClusterInfoList[targetCluster].Avaliable {
		fmt.Println(fmt.Errorf("<error> your target cluster {%s} is unavailable", targetCluster))
		qpod.FilteredCluster = append(qpod.FilteredCluster, targetCluster) // add filtered cluster
		sched.clusterScheduleOne(ctx)                                      //re cluster scheduling
		return
	}

	err := sched.patchPodAnnotationClusterName(qpod.TargetCluster)
	if err != nil {
		r.KETI_LOG_L3(fmt.Sprintf("%s", err))
		sched.nodeScheduleOne(ctx) //create my cluster
		return
	}

	targetClientset := sched.ClusterInfoCache.ClusterInfoList[qpod.TargetCluster].Clientset

	period := int64(0)
	deletePolicy := metav1.DeletePropagationForeground
	deleteOptions := metav1.DeleteOptions{
		PropagationPolicy:  &deletePolicy,
		GracePeriodSeconds: &period,
	}

	//delete origin pod in my cluster
	//job과 Pod 말고 deployment인 경우 존재??
	jobName := qpod.Pod.ObjectMeta.Labels["job-name"] //job이 있다면 job-name 존재
	if jobName != "" {
		//누락되는 정보가 없을까???
		newPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        qpod.Pod.ObjectMeta.Name,
				Namespace:   qpod.Pod.Namespace,
				Annotations: qpod.Pod.GetAnnotations(),
			},
			Spec:     qpod.Pod.Spec,
			Status:   corev1.PodStatus{},
			TypeMeta: qpod.Pod.TypeMeta,
		}

		r.KETI_LOG_L1(fmt.Sprintf("<test> jobname::%s", jobName))
		err = sched.ClusterInfoCache.MyClusterInfo.Clientset.BatchV1().Jobs(qpod.Pod.Namespace).Delete(context.TODO(), jobName, deleteOptions)
		if err != nil {
			r.KETI_LOG_L3(fmt.Sprintf("<error> failed to delete job - %s", err))
			return
		}
		//job은 job으로 배포?
		_, err = targetClientset.CoreV1().Pods(qpod.Pod.Namespace).Create(context.TODO(), newPod, metav1.CreateOptions{})
		// _, err = targetClientset.BatchV1().Jobs(qpod.Pod.Namespace).Create(context.TODO(), newPod., metav1.CreateOptions{})
		if err != nil {
			r.KETI_LOG_L3(fmt.Sprintf("<error> failed to create pod {%s} to target cluster {%s} - %s", qpod.Pod.Name, qpod.TargetCluster, err))
			sched.SchedulingQueue.Add_BackoffQ(&qpod)
			return
		}
	} else {
		//누락되는 정보가 없을까???
		newPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        qpod.Pod.ObjectMeta.Name,
				Namespace:   qpod.Pod.Namespace,
				Annotations: qpod.Pod.GetAnnotations(),
			},
			Spec:     qpod.Pod.Spec,
			Status:   corev1.PodStatus{},
			TypeMeta: qpod.Pod.TypeMeta,
		}

		err = sched.ClusterInfoCache.MyClusterInfo.Clientset.CoreV1().Pods(qpod.Pod.Namespace).Delete(context.TODO(), qpod.Pod.Name, deleteOptions)
		if err != nil {
			r.KETI_LOG_L3(fmt.Sprintf("<error> failed to delete pod - %s", err))
			return
		}
		//pod는 pod로 배포?
		_, err = targetClientset.CoreV1().Pods(qpod.Pod.Namespace).Create(context.TODO(), newPod, metav1.CreateOptions{})
		if err != nil {
			r.KETI_LOG_L3(fmt.Sprintf("<error> failed to create pod {%s} to target cluster {%s} - %s", qpod.Pod.Name, qpod.TargetCluster, err))
			sched.SchedulingQueue.Add_BackoffQ(&qpod)
			return
		}
	}

	r.KETI_LOG_L1(fmt.Sprintf("-----:: Successfully Create Pod {%s} To Target Cluster ::-----\n", qpod.Pod.Name))

	// sched.deletePodFromSchedulingQueue(qpod)
}

func (sched *GPUScheduler) clusterScheduleOne(ctx context.Context) {
	r.KETI_LOG_L2("- Request Cluster Scheduling To Cluster Manager")

	if !sched.ClusterInfoCache.Available {
		r.KETI_LOG_L1("- *not available get other cluster config / cluster scheduling is only available to my cluster")
		sched.nodeScheduleOne(ctx)
		return
	}

	if sched.ClusterManagerHost == "" {
		host := findClusterManagerHost(sched.NodeInfoCache.HostKubeClient)
		if host == "" {
			r.KETI_LOG_L1("- *cannot find cluster-manager in cluster /  cluster scheduling is only available to my cluster")
			sched.nodeScheduleOne(ctx)
			return
		} else {
			sched.ClusterManagerHost = host
		}
	}

	ip := sched.ClusterManagerHost
	gpucount := sched.NewPod.PodInfo.RequestedResource.GPUCount
	fc := sched.NewPod.FilteredCluster
	targetCluster, success, err := GetBestCluster(ip, gpucount, fc)
	if err != nil {
		r.KETI_LOG_L3(fmt.Sprintf("<error> cluster manager get best cluster failed - %s", err))
		sched.nodeScheduleOne(ctx)
		return
	}

	sched.NewPod.TargetCluster = targetCluster

	if !success {
		// sched.SchedulingQueue.Add_BackoffQ(sched.NewPod)
		r.KETI_LOG_L3("<error> cluster manager cannot find best cluster")
		sched.nodeScheduleOne(ctx)
		return
	}

	r.KETI_LOG_L3(fmt.Sprintf("# targetCluster: %s | mycluster: %s", targetCluster, sched.ClusterInfoCache.MyClusterName))

	if targetCluster == sched.ClusterInfoCache.MyClusterName {
		r.KETI_LOG_L1("- target cluster name is my cluster!")
		sched.nodeScheduleOne(ctx)
	} else {
		r.KETI_LOG_L1("- target cluster name is not my cluster!")
		sched.createPodToAnotherCluster(ctx, *sched.NewPod)
		// sched.deletePodFromSchedulingQueue(sched.NewPod)
		return
	}
}

func (sched *GPUScheduler) nodeScheduleOne(ctx context.Context) {
	r.KETI_LOG_L2("- Node Scheduling Start")

	pod := sched.NewPod.Pod
	startTime := time.Now()
	sched.frameworkForPod()

	//[STEP 1] Update Scheduler Cache
	sched.UpdateCache() //metric update, score init
	if sched.NodeInfoCache.AvailableNodeCount == 0 {
		r.KETI_LOG_L3("<error> there isn't node to schedule")
		sched.SchedulingQueue.Add_BackoffQ(sched.NewPod)
		return
	}

	sched.NodeInfoCache.DumpCache() //테스트용

	//[STEP 2,3] Filtering/Scoring Node/GPU
	err := sched.schedulePod()
	if err != nil {
		r.KETI_LOG_L3(fmt.Sprintf("<error> pod scheduling failed - %s", err))
		sched.SchedulingQueue.Add_BackoffQ(sched.NewPod)
		return
	}

	//[STEP 4]Get Best Node/GPU
	sched.GetBestNodeAndGPU()
	sched.NodeInfoCache.UpdatePodState(pod, r.Assumed)
	sched.NodeInfoCache.GPUPodCountUp(sched.ScheduleResult.BestNode, sched.ScheduleResult.BestGPU)

	elapsedTime := time.Since(startTime)

	r.KETI_LOG_L2(fmt.Sprintf("# Scheduling Time : %f", elapsedTime.Seconds()))

	//[STEP 5] Binding Pod To Node
	go sched.Binding(ctx, *sched.NewPod, *sched.ScheduleResult)
}

func (sched *GPUScheduler) schedulePod() error {
	//[STEP 2] Filtering Stage
	err := sched.Framework.RunFilteringPlugins(sched.NodeInfoCache, sched.NewPod)
	if err != nil {
		return fmt.Errorf("filtering failed - %s (unschedulable plugins: %s)", err, sched.NewPod.UnschedulablePlugins)
	}

	//[STEP 3] Scoring Stage
	err = sched.Framework.RunScoringPlugins(sched.NodeInfoCache, sched.NewPod)
	if err != nil {
		return fmt.Errorf("scoring failed (reason: %s)", err)
	}

	return nil
}

func (sched *GPUScheduler) frameworkForPod() {
	if sched.NewPod.IsGPUPod {
		if sched.SchedulingPolicy.GPUAllocatePrefer == "spread" {
			sched.Framework = framework.GPUPodSpreadFramework()
		} else {
			sched.Framework = framework.GPUPodBinpackFramework()
		}
	} else {
		sched.Framework = framework.NonGPUPodFramework()
	}
}

func (sched *GPUScheduler) GetBestNodeAndGPU() {
	r.KETI_LOG_L3("[STEP 4] Get Best Node/GPU")
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
	r.KETI_LOG_L3(fmt.Sprintf("# Scheduling Result: BestNode {%s}", sched.ScheduleResult.BestNode))
	r.KETI_LOG_L3(fmt.Sprintf("# Scheduling Result: BestGPU {%s}", sched.ScheduleResult.BestGPU))
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

	r.KETI_LOG_L3(fmt.Sprintf("(policy 1) node-gpu-score-weight : nodeWeight=%.1f, gpuWeight=%.1f", sched.SchedulingPolicy.NodeWeight, sched.SchedulingPolicy.GPUWeight))
	r.KETI_LOG_L3(fmt.Sprintf("# Total Score(nodeScore * nodeWeight + gpuScore * gpuWeight) = %d * %.1f + %d * %.1f = %d", score.NodeScore,
		sched.SchedulingPolicy.NodeWeight, score.TotalGPUScore, sched.SchedulingPolicy.GPUWeight, score.TotalScore))
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
		r.KETI_LOG_L1(fmt.Sprintf("<test> %s | %d | %v", a.UUID, a.GPUScore, a.IsFiltered))
	}

	//NVLink GPU 고려 X
	if requestedGPU == 1 || nodeinfo.NodeMetric.NVLinkList == nil {
		bestGPUScore := sortedGPUScore[:requestedGPU] //스코어 점수 상위 N개의 GPU
		for _, gpu := range bestGPUScore {
			totalGPUScore += float64(gpu.GPUScore) * float64(1/float64(requestedGPU))
			bestGPU += gpu.UUID + ","
		}
		score.TotalGPUScore = int(math.Round(totalGPUScore))
		score.BestGPU = strings.Trim(bestGPU, ",")

		return
	}

	//NVLink GPU 고려 O
	sched.checkNVLinkGPU(nodeinfo)

	//NVLink Score순으로 정렬
	type ns []r.NVLink
	sortedNVLinkScore := make(ns, 0, len(nodeinfo.NodeMetric.NVLinkList))
	for _, n := range nodeinfo.NodeMetric.NVLinkList {
		sortedNVLinkScore = append(sortedNVLinkScore, *n)
	}
	sort.SliceStable(sortedNVLinkScore, func(i, j int) bool {
		return sortedNVLinkScore[i].Score > sortedNVLinkScore[j].Score
	})

	// for _, nvl := range sortedNVLinkScore {
	// 	fmt.Println("<test> NVLink GPU Pair: ", nvl.GPU1, "|", nvl.GPU2, "|", nvl.Score)
	// }

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
			//NVLink쌍 점수 계산
			var nvlink_pair []string
			nvlink_score, nvlink_selected, nvlink_index := float64(0), "", -1
			for i, nvl := range sortedNVLinkScore {
				if !nvl.IsFiltered && !nvl.IsSelected {
					// fmt.Println("**", nvl.Score, " ", nvl.GPU1, " ", nvl.GPU2)
					nvlink_score = float64(nvl.Score)
					nvlink_selected += nvl.GPU1 + "," + nvl.GPU2 + ","
					nvlink_index = i
					nvlink_pair = append(nvlink_pair, nvl.GPU1, nvl.GPU2)
					break
				}
			}

			//상위 GPU쌍 점수 계산
			var high_gpu []int
			high_score, high_selected := float64(0), ""
			for j, gpu := range sortedGPUScore {
				if !gpu.IsFiltered && !gpu.IsSelected {
					high_score += float64(gpu.GPUScore)
					high_selected += gpu.UUID + ","
					high_gpu = append(high_gpu, j)
				}
				if len(high_gpu) == 2 {
					high_score /= 2
					// fmt.Println("***", high_score)
					break
				}
			}

			if nvlink_score >= high_score { //NVLink쌍 선택
				totalGPUScore += float64(nvlink_score) * float64(2/float64(requestedGPU))
				bestGPU += nvlink_selected + ","
				sortedNVLinkScore[nvlink_index].IsSelected = true
				for _, gpu := range sortedGPUScore {
					if gpu.UUID == nvlink_pair[0] || gpu.UUID == nvlink_pair[1] {
						gpu.IsSelected = true
					}
				}
			} else { //상위 GPU쌍 선택
				totalGPUScore += float64(high_score) * float64(2/float64(requestedGPU))
				bestGPU += high_selected + ","
				sortedGPUScore[high_gpu[0]].IsSelected = true
				sortedGPUScore[high_gpu[1]].IsSelected = true
			}

			gpucnt -= 2
		}
	}

	score.TotalGPUScore = int(math.Round(totalGPUScore))
	score.BestGPU = strings.Trim(bestGPU, ",")

}

func (sched *GPUScheduler) checkNVLinkGPU(nodeinfo *r.NodeInfo) {
	r.KETI_LOG_L2("S#20. Check NVLink GPU")
	r.KETI_LOG_L3(fmt.Sprintf("(policy 2) nvlink-weight-percentage : %d %%\n", sched.SchedulingPolicy.NVLinkWeightPercentage))
	for _, nvl := range nodeinfo.NodeMetric.NVLinkList {
		if nodeinfo.PluginResult.GPUScores[nvl.GPU1].IsFiltered ||
			nodeinfo.PluginResult.GPUScores[nvl.GPU2].IsFiltered {
			nvl.IsFiltered = true
			continue
		}
		score := float64(nodeinfo.PluginResult.GPUScores[nvl.GPU1].GPUScore+
			nodeinfo.PluginResult.GPUScores[nvl.GPU2].GPUScore) / 2
		nvl.Score = int(math.Round(score * float64(1+float64(sched.SchedulingPolicy.NVLinkWeightPercentage)/100)))
		r.KETI_LOG_L3(fmt.Sprintf("# linked gpu{%s}:gpu{%s} score = (%d+%d) / 2 * %.2f = %d\n",
			nvl.GPU1, nvl.GPU2, nodeinfo.PluginResult.GPUScores[nvl.GPU1].GPUScore,
			nodeinfo.PluginResult.GPUScores[nvl.GPU2].GPUScore, float64(1+float64(sched.SchedulingPolicy.NVLinkWeightPercentage)/100), nvl.Score))
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
