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

	// batchv1 "k8s.io/api/batch/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	r "gpu-scheduler/resourceinfo"
)

const (
	scheduleTypeError  = 0
	clusterScheuduling = 1
	nodeScheduling     = 2
	clusterBinding     = 3
)

// metric update, score init
func (sched *GPUScheduler) InitScore() {
	r.KETI_LOG_L3("[scheduling] STEP 1. init result")

	sched.ScheduleResult.InitResult()
	sched.NodeInfoCache.AvailableNodeCount = sched.NodeInfoCache.TotalNodeCount

	for _, nodeInfo := range sched.NodeInfoCache.NodeInfoList {
		nodeInfo.PluginResult.InitPluginResult()
		nodeInfo.PluginResult.AvailableGPUCount = int(nodeInfo.TotalGPUCount)
	}
}

func (sched *GPUScheduler) checkScheduleType() int {
	targetCluster := sched.NewPod.TargetCluster
	r.KETI_LOG_L1(fmt.Sprintf("[debugg] check pod annotation (clusterName): %s", targetCluster))

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

	sched.NewPod = sched.NextPod() // 스케줄링 큐에서 획득
	if sched.NewPod == nil || sched.NewPod.Pod == nil {
		return
	}

	r.KETI_LOG_L3("\n[scheduling] pod scheduling start")
	r.KETI_LOG_L2(fmt.Sprintf("[debugg] pod name: %s", sched.NewPod.Pod.Name))
	r.KETI_LOG_L1(fmt.Sprintf("[debugg] pod request resource: %+v", sched.NewPod.RequestedResource))

	flag := sched.checkScheduleType()

	if flag == clusterScheuduling {
		r.KETI_LOG_L1("[debugg] need cluster scheduling")
		sched.clusterScheduleOne(ctx)
	} else if flag == clusterBinding {
		r.KETI_LOG_L1("[debugg] need cluster binding")
		sched.createPodToAnotherCluster(ctx, *sched.NewPod)
	} else if flag == nodeScheduling {
		r.KETI_LOG_L1("[debugg] need node scheduling")
		sched.nodeScheduleOne(ctx)
	} else {
		r.KETI_LOG_L3("[error] cannot create to target cluster, target cluster is unavailable")
	}
}

// write clusterName to pod annotation
func (sched *GPUScheduler) patchPodAnnotationClusterName(clusterName string) error {
	r.KETI_LOG_L2("[scheduling] write clusterName in pod annotation")

	patchAnnotations := map[string]interface{}{
		"metadata": map[string]map[string]string{"annotations": {
			"clusterName": clusterName,
		}}}

	patchedAnnotationBytes, err := json.Marshal(patchAnnotations)
	if err != nil {
		return fmt.Errorf("[error] failed to generate patched annotations,reason: %v", err)
	}

	_, err = Scheduler.NodeInfoCache.HostKubeClient.CoreV1().Pods(sched.NewPod.Pod.Namespace).Patch(context.TODO(), Scheduler.NewPod.Pod.Name, types.StrategicMergePatchType, patchedAnnotationBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("[error] patchPodAnnotation error: %v", err)
	}

	//파드 어노테이션은 sched.newpod에 반영이 안됨

	return nil
}

func (sched *GPUScheduler) createPodToAnotherCluster(ctx context.Context, qpod r.QueuedPodInfo) {
	r.KETI_LOG_L2("[scheduling] Create Pod To Another Cluster")

	targetCluster := qpod.TargetCluster
	if !sched.ClusterInfoCache.ClusterInfoList[targetCluster].Avaliable {
		fmt.Println(fmt.Errorf("[error] your target cluster {%s} is unavailable", targetCluster))
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
		fmt.Printf("annotation: %v\n", qpod.Pod.GetAnnotations())
		err = sched.ClusterInfoCache.MyClusterInfo.Clientset.BatchV1().Jobs(qpod.Pod.Namespace).Delete(context.TODO(), jobName, deleteOptions)
		if err != nil {
			r.KETI_LOG_L3(fmt.Sprintf("[error] failed to delete job - %s", err))
			return
		}
		//job은 job으로 배포?
		_, err = targetClientset.CoreV1().Pods(qpod.Pod.Namespace).Create(context.TODO(), newPod, metav1.CreateOptions{})
		// _, err = targetClientset.BatchV1().Jobs(qpod.Pod.Namespace).Create(context.TODO(), newPod., metav1.CreateOptions{})
		if err != nil {
			r.KETI_LOG_L3(fmt.Sprintf("[error] failed to create pod {%s} to target cluster {%s} - %s", qpod.Pod.Name, qpod.TargetCluster, err))
			sched.SchedulingQueue.AddBackoffQ(&qpod)
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
			r.KETI_LOG_L3(fmt.Sprintf("[error] failed to delete pod - %s", err))
			return
		}
		//pod는 pod로 배포?
		_, err = targetClientset.CoreV1().Pods(qpod.Pod.Namespace).Create(context.TODO(), newPod, metav1.CreateOptions{})
		if err != nil {
			r.KETI_LOG_L3(fmt.Sprintf("[error] failed to create pod {%s} to target cluster {%s} - %s", qpod.Pod.Name, qpod.TargetCluster, err))
			sched.SchedulingQueue.AddBackoffQ(&qpod)
			return
		}
	}

	r.KETI_LOG_L1(fmt.Sprintf("[scheduling] successfully create pod {%s} to target cluster\n", qpod.Pod.Name))

	// sched.deletePodFromSchedulingQueue(qpod)
}

func (sched *GPUScheduler) clusterScheduleOne(ctx context.Context) {
	r.KETI_LOG_L2("[scheduling] request cluster scheduling to cluster manager")

	if !sched.ClusterInfoCache.Available {
		r.KETI_LOG_L1("[warning] not available get other cluster config / cluster scheduling is only available to my cluster")
		sched.nodeScheduleOne(ctx)
		return
	}

	if sched.ClusterManagerHost == "" {
		host := findClusterManagerHost(sched.NodeInfoCache.HostKubeClient)
		if host == "" {
			r.KETI_LOG_L1("[warning] cannot find cluster-manager in cluster / cluster scheduling is only available to my cluster")
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
	if err != nil || !success {
		r.KETI_LOG_L3(fmt.Sprintf("[error] cluster manager get best cluster failed - %s", err))
		sched.NewPod.TargetCluster = sched.ClusterInfoCache.MyClusterName
	} else {
		sched.NewPod.TargetCluster = targetCluster
	}

	r.KETI_LOG_L3(fmt.Sprintf("[debugg] targetCluster: %s | mycluster: %s", targetCluster, sched.ClusterInfoCache.MyClusterName))

	if targetCluster == sched.ClusterInfoCache.MyClusterName {
		r.KETI_LOG_L1("[debugg] target cluster name is my cluster!")
		sched.nodeScheduleOne(ctx)
	} else {
		r.KETI_LOG_L1("[debugg] target cluster name is not my cluster!")
		sched.createPodToAnotherCluster(ctx, *sched.NewPod)
		// sched.deletePodFromSchedulingQueue(sched.NewPod)
		return
	}
}

func (sched *GPUScheduler) nodeScheduleOne(ctx context.Context) {
	r.KETI_LOG_L2("[scheduling] node scheduling start")

	newPod := sched.NewPod

	//[STEP 1] Update Scheduler Cache
	sched.InitScore() //score init

	// sched.NodeInfoCache.DumpCache() //테스트용

	//[STEP 2,3] Filtering/Scoring Node/GPU
	err := sched.schedulePod()
	if err != nil {
		r.KETI_LOG_L3(fmt.Sprintf("[error] Failed Pod Scheduling >> Reason:%s", err.Error()))
		newPod.Status.Reasons = err.Error()
		//스케줄링 실패 원인 분석
		sched.SchedulingQueue.AddBackoffQ(newPod)
		return
	}

	//[STEP 4]Get Best Node/GPU
	sched.GetBestNodeAndGPU()
	sched.NodeInfoCache.UpdatePodState(newPod.Pod, r.Assumed)

	//[STEP 5] Binding Pod To Node
	go sched.Binding(ctx, *newPod, *sched.ScheduleResult)
}

func (sched *GPUScheduler) schedulePod() error {
	//[STEP 2] Filtering Stage
	r.KETI_LOG_L3("[scheduling] STEP 2. run filtering plugins")
	sched.Framework.RunNodeFilteringPlugins(sched.NodeInfoCache, sched.NewPod)

	// if sched.SchedulingPolicy.NonGPUNodePrefer /*sched.SchedulingPolicy.GPUFilteringTemperature*/ {
	// 	sched.Framework.RunGPUCountFilteringPlugin(sched.NodeInfoCache, sched.NewPod)
	// 	// 	sched.Framework.RunGPUTemperatureFilteringPlugin(sched.NodeInfoCache, sched.NewPod)
	// }
	sched.Framework.RunGPUCountFilteringPlugin(sched.NodeInfoCache, sched.NewPod)
	if sched.NodeInfoCache.AvailableNodeCount == 0 {
		return fmt.Errorf("filtered all nodes")
	}

	//[STEP 3] Scoring Stage
	r.KETI_LOG_L3("[scheduling] STEP 3. get analysis metric score")
	err := sched.NodeInfoCache.GetAnalysisScore(sched.MetricAnalysisModuleIP)
	if err != nil {
		return fmt.Errorf("[error] cannot connect analysis engine: %s", err)
	}

	return nil
}

func (sched *GPUScheduler) GetBestNodeAndGPU() {
	r.KETI_LOG_L3("[scheduling] STEP 4. calculate best node/gpu")
	var scoreRate []string

	for nodeName, nodeInfo := range sched.NodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			sched.getTotalScore(nodeInfo, sched.NewPod.RequestedResource.GPUCount)
			inserted := false
			for i, scoreName := range scoreRate {
				if nodeInfo.PluginResult.TotalScore >= sched.NodeInfoCache.NodeInfoList[scoreName].PluginResult.TotalScore {
					scoreRate = append(scoreRate[:i], append([]string{nodeName}, scoreRate[i:]...)...)
					inserted = true
					break
				}
			}
			if !inserted {
				scoreRate = append(scoreRate, nodeName)
			}
		}
	}

	selectedNodeName := ""
	if sched.SchedulingPolicy.LeastScoreNodePrefer {
		selectedNodeName = scoreRate[len(scoreRate)-1]
	} else if sched.SchedulingPolicy.AvoidHighScoreNode {
		if len(scoreRate) == 1 {
			selectedNodeName = scoreRate[0]
		} else {
			selectedNodeName = scoreRate[1]
		}
	} else {
		selectedNodeName = scoreRate[0]
	}

	sched.ScheduleResult.BestNode = selectedNodeName
	sched.ScheduleResult.BestGPU = sched.NodeInfoCache.NodeInfoList[selectedNodeName].PluginResult.BestGPU
	sched.ScheduleResult.TotalScore = sched.NodeInfoCache.NodeInfoList[selectedNodeName].PluginResult.TotalScore

	// r.KETI_LOG_L3(fmt.Sprintf("(policy 1) %s : nodeWeight=%.1f, gpuWeight=%.1f --> OK", r.Policy1, sched.SchedulingPolicy.NodeWeight, sched.SchedulingPolicy.GPUWeight))
	// r.KETI_LOG_L3(fmt.Sprintf("(policy 2) %s : %v ", r.Policy2, sched.SchedulingPolicy.NVLinkWeightPercentage))
	// fmt.Sprintf("(policy 3) %s : %v ", r.Policy3, sched.SchedulingPolicy.GPUAllocatePrefer))
	// r.KETI_LOG_L3(fmt.Sprintf("(policy 4) %s : %v ", r.Policy4, sched.SchedulingPolicy.NodeReservationPermit))
	// r.KETI_LOG_L3(fmt.Sprintf("(policy 5) %s : %v ", r.Policy5, sched.SchedulingPolicy.PodReSchedulePermit))
	// r.KETI_LOG_L3(fmt.Sprintf("(policy 6) %s : %v ", r.Policy6, sched.SchedulingPolicy.AvoidNVLinkOneGPU))
	// r.KETI_LOG_L3(fmt.Sprintf("(policy 7) %s : %v ", r.Policy7, sched.SchedulingPolicy.MultiNodeAllocationPermit))
	// r.KETI_LOG_L3(fmt.Sprintf("(policy 8) %s : %v ", r.Policy8, sched.SchedulingPolicy.NonGPUNodePrefer))
	// r.KETI_LOG_L3(fmt.Sprintf("(policy 9) %s : %v ", r.Policy9, sched.SchedulingPolicy.MultiGPUNodePrefer))
	// r.KETI_LOG_L3(fmt.Sprintf("(policy 10) %s : %v ", r.Policy10, sched.SchedulingPolicy.LeastScoreNodePrefer))
	// r.KETI_LOG_L3(fmt.Sprintf("(policy 11) %s : %v ", r.Policy11, sched.SchedulingPolicy.AvoidHighScoreNode))

	fmt.Printf("(policy 1) %s : nodeWeight=%.1f, gpuWeight=%.1f --> OK", r.Policy1, sched.SchedulingPolicy.NodeWeight, sched.SchedulingPolicy.GPUWeight)
	fmt.Printf("(policy 2) %s : %v ", r.Policy2, sched.SchedulingPolicy.NVLinkWeightPercentage)
	if sched.SchedulingPolicy.NVLinkWeightPercentage != 0 {
		fmt.Println(" --> OK")
	}
	fmt.Printf("(policy 3) %s : %v --> OK", r.Policy3, sched.SchedulingPolicy.GPUAllocatePrefer)
	fmt.Printf("(policy 4) %s : %v ", r.Policy4, sched.SchedulingPolicy.NodeReservationPermit)
	if sched.SchedulingPolicy.NodeReservationPermit {
		fmt.Println(" --> OK")
	}
	fmt.Printf("(policy 5) %s : %v ", r.Policy5, sched.SchedulingPolicy.PodReSchedulePermit)
	if sched.SchedulingPolicy.PodReSchedulePermit {
		fmt.Println(" --> OK")
	}
	fmt.Printf("(policy 6) %s : %v ", r.Policy6, sched.SchedulingPolicy.AvoidNVLinkOneGPU)
	if sched.SchedulingPolicy.AvoidNVLinkOneGPU {
		fmt.Println(" --> OK")
	}
	fmt.Printf("(policy 7) %s : %v ", r.Policy7, sched.SchedulingPolicy.MultiNodeAllocationPermit)
	if sched.SchedulingPolicy.MultiNodeAllocationPermit {
		fmt.Println(" --> OK")
	}
	fmt.Printf("(policy 8) %s : %v ", r.Policy8, sched.SchedulingPolicy.NonGPUNodePrefer)
	if sched.SchedulingPolicy.NonGPUNodePrefer {
		fmt.Println(" --> OK")
	}
	fmt.Printf("(policy 9) %s : %v ", r.Policy9, sched.SchedulingPolicy.MultiGPUNodePrefer)
	if sched.SchedulingPolicy.MultiGPUNodePrefer {
		fmt.Println(" --> OK")
	}
	fmt.Printf("(policy 10) %s : %v ", r.Policy10, sched.SchedulingPolicy.LeastScoreNodePrefer)
	if sched.SchedulingPolicy.LeastScoreNodePrefer {
		fmt.Println(" --> OK")
	}
	fmt.Printf("(policy 11) %s : %v ", r.Policy11, sched.SchedulingPolicy.AvoidHighScoreNode)
	if sched.SchedulingPolicy.AvoidHighScoreNode {
		fmt.Println(" --> OK")
	}

	r.KETI_LOG_L3(fmt.Sprintf("[scheduling] scheduling result: best node = %s", sched.ScheduleResult.BestNode))

	if sched.NewPod.RequestedResource.GPUCount != 0 {
		r.KETI_LOG_L3(fmt.Sprintf("[scheduling] scheduling result: best gpu = %s", sched.ScheduleResult.BestGPU))
	}

}

func (sched *GPUScheduler) getTotalScore(nodeinfo *r.NodeInfo, requestedGPU int) {
	score := nodeinfo.PluginResult
	if !sched.NewPod.IsGPUPod || nodeinfo.PluginResult.AvailableGPUCount == 0 {
		score.TotalScore = int(float64(score.NodeScore) * sched.SchedulingPolicy.NodeWeight)
		return
	}
	if sched.SchedulingPolicy.MultiGPUNodePrefer {
		score.TotalScore = score.NodeScore * (10 + nodeinfo.PluginResult.AvailableGPUCount) / 10
		r.KETI_LOG_L2(fmt.Sprintf("[debugg] policy#9 :: %d * %d / 10 = %d", score.NodeScore, (10 + nodeinfo.PluginResult.AvailableGPUCount), score.TotalScore))
	}
	sched.getTotalGPUScore(nodeinfo, requestedGPU)

	if sched.NewPod.IsGPUPod && nodeinfo.PluginResult.AvailableGPUCount == 0 {
		score.TotalScore = int(math.Round(float64(score.NodeScore) * 0.1))
		r.KETI_LOG_L2(fmt.Sprintf("[debugg] total score(node score * 0.1) = %d * 0.1 = %d", score.NodeScore, score.TotalScore))
	} else {
		score.TotalScore = int(math.Round(float64(score.NodeScore)*sched.SchedulingPolicy.NodeWeight +
			float64(score.TotalGPUScore)*sched.SchedulingPolicy.GPUWeight))

		r.KETI_LOG_L2(fmt.Sprintf("[debugg] policy#1 :: total score(nodeScore * nodeWeight + gpuScore * gpuWeight) = %d * %.1f + %d * %.1f = %d", score.NodeScore,
			sched.SchedulingPolicy.NodeWeight, score.TotalGPUScore, sched.SchedulingPolicy.GPUWeight, score.TotalScore))
	}
}

func (sched *GPUScheduler) getTotalGPUScore(nodeinfo *r.NodeInfo, requestedGPU int) {
	score := nodeinfo.PluginResult
	totalGPUScore, bestGPU := float64(0), ""

	if sched.NewPod.IsGPUPod && nodeinfo.PluginResult.AvailableGPUCount == 0 {
		return
	}

	if requestedGPU == 1 && sched.SchedulingPolicy.AvoidNVLinkOneGPU {
		for _, nvlink := range nodeinfo.NVLinkList {
			score.GPUScores[nvlink.GPU1].GPUScore = int(float32(score.GPUScores[nvlink.GPU1].GPUScore) * 0.3)
			score.GPUScores[nvlink.GPU2].GPUScore = int(float32(score.GPUScores[nvlink.GPU2].GPUScore) * 0.3)
			r.KETI_LOG_L2(fmt.Sprintf("[debugg] policy#6 :: gpu {%s} score * 0.3 = %d", nvlink.GPU1, score.GPUScores[nvlink.GPU1].GPUScore))
			r.KETI_LOG_L2(fmt.Sprintf("[debugg] policy#6 :: gpu {%s} score * 0.3 = %d", nvlink.GPU2, score.GPUScores[nvlink.GPU2].GPUScore))
		}
	}

	if sched.SchedulingPolicy.GPUAllocatePrefer == "binpack" {
		for _, gpuScore := range score.GPUScores {
			score := gpuScore.GPUScore + gpuScore.PodCount*100
			r.KETI_LOG_L2(fmt.Sprintf("[debugg] policy#3 = binpack :: %d + %d * 100 = %d", gpuScore.GPUScore, gpuScore.PodCount, score))
			gpuScore.GPUScore = score
		}
	}

	//최종 GPUScore 순으로 정렬
	type gs []*r.GPUScore
	sortedGPUScore := make(gs, 0, len(score.GPUScores))
	for _, d := range score.GPUScores {
		sortedGPUScore = append(sortedGPUScore, d)
	}

	//NVLink GPU 고려 X
	if requestedGPU == 1 || nodeinfo.NVLinkList == nil {
		bestGPUScore := sortedGPUScore[:requestedGPU] //스코어 점수 상위 N개의 GPU

		sort.SliceStable(sortedGPUScore, func(i, j int) bool {
			return sortedGPUScore[i].GPUScore > sortedGPUScore[j].GPUScore
		})

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
	sortedNVLinkScore := make(ns, 0, len(nodeinfo.NVLinkList))
	for _, n := range nodeinfo.NVLinkList {
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
	for _, nvl := range nodeinfo.NVLinkList {
		if nodeinfo.PluginResult.GPUScores[nvl.GPU1].IsFiltered ||
			nodeinfo.PluginResult.GPUScores[nvl.GPU2].IsFiltered {
			nvl.IsFiltered = true
			continue
		}
		score := float64(nodeinfo.PluginResult.GPUScores[nvl.GPU1].GPUScore+nodeinfo.PluginResult.GPUScores[nvl.GPU2].GPUScore) / 2
		nvl.Score = int(math.Round(score * float64(1+float64(sched.SchedulingPolicy.NVLinkWeightPercentage)/100)))
		r.KETI_LOG_L2(fmt.Sprintf("[debugg] policy#2 :: linked gpu{%s}:gpu{%s} score = (%d+%d) / 2 * %.2f = %d",
			nvl.GPU1, nvl.GPU2, nodeinfo.PluginResult.GPUScores[nvl.GPU1].GPUScore,
			nodeinfo.PluginResult.GPUScores[nvl.GPU2].GPUScore, float64(1+float64(sched.SchedulingPolicy.NVLinkWeightPercentage)/100), nvl.Score))
	}
}
