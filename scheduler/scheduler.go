package scheduler

import (
	"context"
	"fmt"
	"gpu-scheduler/framework"
	r "gpu-scheduler/resourceinfo"
	"strings"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

var processorLock = &sync.Mutex{}

var Scheduler *GPUScheduler

type GPUScheduler struct {
	NodeInfoCache           *r.NodeCache
	SchedulingPolicy        *SchedulingPolicy
	SchedulingQueue         r.SchedulingQueue
	NewPod                  *r.QueuedPodInfo
	Framework               framework.GPUSchedulerInterface
	ScheduleResult          *r.ScheduleResult
	ClusterInfoCache        *r.ClusterCache
	MetricAnalysisModuleIP  string
	ClusterManagerHost      string
	AvailableClusterManager bool
}

func NewGPUScheduler(hostKubeClient *kubernetes.Clientset) (*GPUScheduler, error) {
	sq := r.NewSchedulingQueue()
	sp := NewSchedulingPolicy()
	nc := r.NewNodeInfoCache(hostKubeClient)
	err := nc.InitNodeInfoCache()
	if err != nil {
		//내 클러스터에는 배포 가능한 노드가 없음(노드를 가져올 수 없음)
		//다른 클러스터엔 가능할수도
		r.KETI_LOG_L3(fmt.Sprintf("[error] cannot find any node in cluster!-%s", err))
	}
	fwk := framework.GPUPodFramework()
	sr := r.NewScheduleResult()
	// cc, err := r.NewClusterCache()
	// cmhost := ""

	// if r.GPU_SCHEDUER_DEBUGG_LEVEL == r.LEVEL1 {
	// 	cc.DumpClusterInfo()
	// }

	// if err != nil {
	// 	r.KETI_LOG_L3("[warning] kubeconfig error / scheduling is only available to my cluster")
	// 	//내 클러스터에만 배포 가능
	// } else {
	// 	cmhost = findClusterManagerHost(hostKubeClient)
	// 	if cmhost == "" {
	// 		r.KETI_LOG_L2("[warning] cannot find cluster-manager in cluster / scheduling is only available to my cluster")
	// 		//내 클러스터에만 배포 가능
	// 	}
	// }

	ma := GetMetricAnalysisModuleIP(hostKubeClient) //예외처리 필요

	return &GPUScheduler{
		NodeInfoCache:           nc,
		SchedulingPolicy:        sp,
		SchedulingQueue:         sq,
		NewPod:                  nil,
		Framework:               fwk,
		ScheduleResult:          sr,
		ClusterInfoCache:        &r.ClusterCache{}, //임시
		MetricAnalysisModuleIP:  ma,
		ClusterManagerHost:      "", //임시
		AvailableClusterManager: false,
	}, nil
}

func (sched *GPUScheduler) Run(ctx context.Context) {
	sched.SchedulingQueue.Run()
	wait.UntilWithContext(ctx, sched.preScheduling, 0) // Schedule Pod Routine
	sched.SchedulingQueue.Close()
}

type SchedulingPolicy struct {
	NodeWeight                float64
	GPUWeight                 float64
	NVLinkWeightPercentage    int64
	GPUAllocatePrefer         string
	NodeReservationPermit     bool
	PodReSchedulePermit       bool
	AvoidNVLinkOneGPU         bool
	MultiNodeAllocationPermit bool
	NonGPUNodePrefer          bool
	MultiGPUNodePrefer        bool
	LeastScoreNodePrefer      bool
	AvoidHighScoreNode        bool
}

func NewSchedulingPolicy() *SchedulingPolicy {
	return &SchedulingPolicy{
		NodeWeight:                0.4,
		GPUWeight:                 0.6,
		NVLinkWeightPercentage:    10,
		GPUAllocatePrefer:         "spread",
		NodeReservationPermit:     false,
		PodReSchedulePermit:       true,
		AvoidNVLinkOneGPU:         false,
		MultiNodeAllocationPermit: false,
		NonGPUNodePrefer:          true,
		MultiGPUNodePrefer:        true,
		LeastScoreNodePrefer:      false,
		AvoidHighScoreNode:        false,
	}
}

func (sched *GPUScheduler) NextPod() *r.QueuedPodInfo {
	newPod, err := sched.SchedulingQueue.Pop()
	if err != nil {
		r.KETI_LOG_L3(fmt.Sprintf("[error] Get NextPod Error : %s", err))
		return nil
	}

	return newPod
}

func findClusterManagerHost(hostKubeClient *kubernetes.Clientset) string {
	pods, _ := hostKubeClient.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})

	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, "keti-cluster-manager") && pod.Status.Phase == "Running" {
			return pod.Status.PodIP
		}
	}
	return ""
}

func (sched *GPUScheduler) InitClusterManager() error {
	if sched.ClusterManagerHost == "" {
		return nil
	}

	// initPod := r.NewQueuedPodInfo(nil) //Run Test Init Pod

	sched.InitScore() //score init
	if sched.NodeInfoCache.AvailableNodeCount == 0 {
		r.KETI_LOG_L3("[error] there isn't node to schedule")
		return nil
	}

	// sched.Framework.RunScoringPlugins(sched.NodeInfoCache, initPod)

	var initList []InitStruct
	for nodeName, nodeInfo := range sched.NodeInfoCache.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			nodeScore := calcInitNodeScore(nodeInfo, sched.SchedulingPolicy.NodeWeight, sched.SchedulingPolicy.GPUWeight)
			initStruct := InitStruct{nodeName, nodeScore, nodeInfo.TotalGPUCount}
			initList = append(initList, initStruct)
		}
	}

	success, err := InitMyClusterManager(sched.ClusterManagerHost, initList)
	if err != nil || !success {
		sched.AvailableClusterManager = false
		return err
	}
	sched.AvailableClusterManager = true
	return nil
}

func calcInitNodeScore(nodeInfo *r.NodeInfo, nodeWeight float64, gpuWeight float64) int64 {
	nodeScore := float64(nodeInfo.PluginResult.NodeScore)
	gpuScore := float64(0)
	cnt := float64(0)
	for _, score := range nodeInfo.PluginResult.GPUScores {
		gpuScore += float64(score.GPUScore)
		cnt++
	}
	gpuScore = gpuScore / cnt
	totalScore := nodeScore*nodeWeight + gpuScore*gpuWeight
	r.KETI_LOG_L1(fmt.Sprintf("[debugg] nodescore:%.3f, gpuscore:%.3f, totalscore:%.3f", nodeScore, gpuScore, totalScore))
	return int64(totalScore)
}

func GetMetricAnalysisModuleIP(hostKubeClient *kubernetes.Clientset) string {
	pods, _ := hostKubeClient.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})

	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, "keti-analysis-engine") && pod.Status.Phase == "Running" {
			return pod.Status.PodIP
		}
	}
	return ""
}

// func (sched *GPUScheduler) InitPluginResult() {
// 	sched.NodeInfoCache.AvailableNodeCount = 0
// 	sched.PluginResult.Scores = nil
// }

// func (sched *GPUScheduler) RunScheduler(quit chan struct{}, wg *sync.WaitGroup) {
// 	fmt.Println("[A] Run Scheduling Routine")

// 	for {
// 		select {
// 		case <-quit:
// 			wg.Done()
// 			log.Println("Stopped scheduler.")
// 			return
// 		default:
// 			newPod, err := sched.SchedulingQueue.Wait_and_pop()
// 			if err != nil {
// 				continue
// 			}
// 			sched.printSchedulingQ()
// 			fmt.Println("--------------------------------------------------------------------------------------------------------------------------")
// 			processorLock.Lock()
// 			if !newPod.IsDeleted() {
// 				fmt.Println("<<<newpod>>> ", newPod.Pod.Name)
// 				err := sched.schduleOne(newPod) //새롭게 들어온 파드 스케줄링
// 				if err != nil {
// 					fmt.Println("Failed Pod Scheduling: ", err)
// 					sched.printReSchedulingQ()
// 				}
// 			}

// 			processorLock.Unlock()

// 			//time.Sleep(2 * time.Second)
// 		}
// 	}
// }

// func (sched *GPUScheduler) MonitorUnscheduledPods(done chan struct{}, wg *sync.WaitGroup) {
// 	fmt.Println("[B] MonitorUnscheduledPods (start api server watching)")

// 	for {
// 		select {
// 		case <-done:
// 			wg.Done()
// 			log.Println("Stopped scheduler.")
// 			return
// 		default:
// 			watch, err := r.ClusterInfo.HostKubeClient.CoreV1().Pods("").Watch(context.TODO(), metav1.ListOptions{
// 				FieldSelector: fmt.Sprintf("spec.schedulerName=%s,spec.nodeName=", r.SchedulerName),
// 			})
// 			if err != nil {
// 				fmt.Println("WatchNewPods error: ", err) //debug
// 			}
// 			for event := range watch.ResultChan() {
// 				if event.Type == "ADDED" {
// 					pod := event.Object.(*corev1.Pod)
// 					sched.inqueSchedulingQ(pod)
// 				} /*else if event.Type == "Deleted" {
// 					pod := event.Object.(*corev1.Pod)
// 					sched.SchedulingQueue.Remove(pod)
// 				}*/
// 			}
// 		}
// 	}
// }

// func (sched *GPUScheduler) inqueSchedulingQ(newpod *corev1.Pod) {
// 	p := getNewPodInfo(newpod)
// 	sched.SchedulingQueue.PushBack(p)
// }

// // func (sched *GPUScheduler) removeSchedulingQ(newpod *corev1.Pod) {
// // 	p := getNewPodInfo(newpod)
// // 	sched.SchedulingQueue.Remove(p)
// // }

// //주기적으로 재 스케줄링 큐의 파드들을 재스케줄링 -> 주기 얼마로?
// func (sched *GPUScheduler) ReconcileRescheduledPods(interval int, done chan struct{}, wg *sync.WaitGroup) {
// 	fmt.Println("[C] Run ReScheduling Routine")

// 	for {
// 		select {
// 		case <-time.After(time.Duration(interval) * time.Second):
// 			fmt.Println("time passed")
// 			tempQ := sched.ReSchedulingQueue
// 			sched.ReSchedulingQueue = list.New()
// 			sched.printReSchedulingQ()
// 			printtempQ(tempQ)
// 			err := sched.schedulePods(tempQ)
// 			if err != nil {
// 				log.Println("Failed Pod ReScheduling: ", err)
// 			}
// 		case <-done:
// 			wg.Done()
// 			log.Println("Stopped scheduler.")
// 			return
// 		}
// 	}
// }

// func (sched *GPUScheduler) schedulePods(q *list.List) error {
// 	processorLock.Lock()
// 	defer processorLock.Unlock()
// 	for p := q.Front(); p != nil; p = p.Next() { //스케줄링 대기중인 파드들 하나씩 스케줄링
// 		pod := p.Value.(*r.Pod)
// 		if !pod.IsDeleted() {
// 			fmt.Println("<<<repod>>> ", pod.Pod.Name)
// 			err := sched.schduleOne(pod) //새롭게 들어온 파드 스케줄링
// 			if err != nil {
// 				fmt.Println("Failed Pod Scheduling: ", err)
// 				sched.printReSchedulingQ()
// 			}
// 		}
// 	}
// 	return nil
// }

// func (sched *GPUScheduler) inqueReScedulingQ(p *r.Pod) {
// 	p.FailedScheduling()
// 	sched.ReSchedulingQueue.PushBack(p)
// }

// func (sched *GPUScheduler) printReSchedulingQ() {
// 	fmt.Println("@print re scheduling queue@")
// 	for rsp := sched.ReSchedulingQueue.Front(); rsp != nil; rsp = rsp.Next() {
// 		a := rsp.Value.(*r.Pod)
// 		fmt.Println("@@", a.Pod.Name, " & ", a.Pod.Annotations["failCount"])
// 	}
// }

// func (sched *GPUScheduler) printSchedulingQ() {
// 	fmt.Println("@print scheduling queue@")
// 	for rsp := sched.SchedulingQueue.activeQ.Front(); rsp != nil; rsp = rsp.Next() {
// 		a := rsp.Value.(*r.Pod)
// 		fmt.Println("@@", a.Pod.Name)
// 	}
// }

// func printtempQ(t *list.List) {
// 	fmt.Println("@print temp queue@")
// 	for rsp := t.Front(); rsp != nil; rsp = rsp.Next() {
// 		a := rsp.Value.(*r.Pod)
// 		fmt.Println("@@", a.Pod.Name, " & ", a.Pod.Annotations["failCount"])
// 	}
// }

// //스케줄링 정책 확인 루틴, 성능 저하 파드 재스케줄링 정책이 바뀌면 알림
// func (sched *GPUScheduler) WatchSchedulerPolicy(quit chan struct{}, wg *sync.WaitGroup) <-chan bool {
// 	fmt.Println("[E] Watch Scheduler Policy")

// 	for {
// 		weightPolicy, err := exec.Command("cat", "/tmp/node-gpu-score-weight").Output()
// 		if err != nil {
// 			fmt.Println("cannot read scheduler policy (node-gpu-score-weight)")
// 		} else {
// 			nodeWeight, _ := strconv.ParseFloat(strings.Split(string(weightPolicy), " ")[0], 64)
// 			gpuWeight, _ := strconv.ParseFloat(strings.Split(string(weightPolicy), " ")[1], 64)
// 			if nodeWeight != sched.SchedulingPolicy.NodeWeight || gpuWeight != sched.SchedulingPolicy.GPUWeight {
// 				fmt.Println("<<<Policy Updated>>> NodeWeight/GPUWeight | ", sched.SchedulingPolicy.NodeWeight, "/", sched.SchedulingPolicy.GPUWeight, "-->", nodeWeight, "/", gpuWeight)
// 				sched.SchedulingPolicy.NodeWeight, sched.SchedulingPolicy.GPUWeight = nodeWeight, gpuWeight
// 			}
// 		}

// 		reSchedulePolicy, err := exec.Command("cat", "/tmp/pod-re-schedule-permit").Output()
// 		if err != nil {
// 			fmt.Println("cannot read scheduler policy (pod-re-schedule-permit)")
// 		} else {
// 			reSchedule, _ := strconv.ParseBool(string(reSchedulePolicy))
// 			if reSchedule != sched.SchedulingPolicy.ReSchedulePermit {
// 				fmt.Println("<<<Policy Updated>>> reSchedule | ", sched.SchedulingPolicy.ReSchedulePermit, "-->", reSchedule)
// 				sched.SchedulingPolicy.ReSchedulePermit = reSchedule
// 				// sched.SchedulingPolicy.changed <- reSchedule
// 			}
// 		}
// 		// time.Sleep(30 * time.Second)
// 	}

// }

// //grpc로 메트릭콜렉터에서 성능저하파드를 받아 재스케줄링을 수행
// func (sched *GPUScheduler) ReconcileLowPerformancePod(done chan struct{}) {
// 	fmt.Println("(low performance pod listening...)")
// 	for {
// 		select {
// 		case <-done:
// 			return
// 		}
// 	}
// }

// //재스케줄링 정책에 따라 재스케줄링 루틴을 수행
// func (sched *GPUScheduler) WatchLowPerformancePod(quit chan struct{}, wg *sync.WaitGroup) {
// 	quitChan := make(chan struct{}) //struct타입을 전송할 수 있는 통신용 채널 생성

// 	if sched.SchedulingPolicy.ReSchedulePermit {
// 		fmt.Println("[D] *Run* Low Performance Pod Reschedule Routine")
// 		go sched.ReconcileLowPerformancePod(quitChan)
// 	}

// 	for {
// 		select {
// 		case reschedule := <-sched.SchedulingPolicy.changed:
// 			if reschedule {
// 				fmt.Println("[D] *Run* Low Performance Pod Reschedule Routine")
// 				go sched.ReconcileLowPerformancePod(quitChan)
// 			} else {
// 				fmt.Println("[D] *Stop* Low Performance Pod Reschedule Routine")
// 				close(quitChan)
// 			}
// 		case <-quit:
// 			wg.Done()
// 			log.Println("Stopped routine.")
// 			return
// 		}
// 	}
// }
