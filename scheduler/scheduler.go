package scheduler

import (
	"context"
	"fmt"
	"gpu-scheduler/framework"
	r "gpu-scheduler/resourceinfo"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

var processorLock = &sync.Mutex{}

var Scheduler *GPUScheduler

type GPUScheduler struct {
	NodeInfoCache    *r.NodeCache
	SchedulingPolicy *SchedulingPolicy
	SchedulingQueue  *r.SchedulingQueue
	NewPod           *r.QueuedPodInfo
	Framework        framework.GPUSchedulerInterface
	ScheduleResult   *r.ScheduleResult
	ClusterInfoCache *r.ClusterCache
}

func NewGPUScheduler(hostKubeClient *kubernetes.Clientset) *GPUScheduler {
	sq := r.NewQueue()

	sp := NewSchedulingPolicy()

	nc := r.NewNodeInfoCache(hostKubeClient)
	nc.InitNodeInfoCache( /*sq*/ )
	nc.DumpCache()

	fwk := framework.GPUPodSpreadFramework()

	sr := r.NewScheduleResult()
	cc := r.NewClusterCache()

	return &GPUScheduler{
		NodeInfoCache:    nc,
		SchedulingPolicy: sp,
		SchedulingQueue:  sq,
		NewPod:           nil,
		Framework:        fwk,
		ScheduleResult:   sr,
		ClusterInfoCache: cc,
	}
}

func (sched *GPUScheduler) Run(ctx context.Context) {
	sched.SchedulingQueue.Run()
	wait.UntilWithContext(ctx, sched.preScheduling, 0)
	sched.SchedulingQueue.Close()
}

type SchedulingPolicy struct {
	NodeWeight             float64
	GPUWeight              float64
	ReSchedulePermit       bool
	NodeReservationPermit  bool
	NVLinkWeightPercentage int64
	GPUAllocatePrefer      bool
	changed                <-chan bool
}

func NewSchedulingPolicy() *SchedulingPolicy {
	var (
		nodeWeight             float64
		gpuWeight              float64
		reschedulePermit       bool
		nodeReservetionPermit  bool
		nvlinkWeightPercentage int64
		gpuAllocatePrefer      bool
	)

	p, err := exec.Command("cat", "/gpu-scheduler-configmap/node-gpu-score-weight").Output()
	if err == nil {
		nodeWeight, _ = strconv.ParseFloat(strings.Split(string(p), " ")[0], 64)
		gpuWeight, _ = strconv.ParseFloat(strings.Split(string(p), " ")[1], 64)
	}

	p, err = exec.Command("cat", "/gpu-scheduler-configmap/pod-re-schedule-permit").Output()
	if err == nil {

		reschedulePermit, _ = strconv.ParseBool(string(p))
	}

	p, err = exec.Command("cat", "/gpu-scheduler-configmap/node-reservation-permit").Output()
	if err == nil {
		nodeReservetionPermit, _ = strconv.ParseBool(string(p))
	}

	p, err = exec.Command("cat", "/gpu-scheduler-configmap/nvlink-weight-percentage").Output()
	if err == nil {
		nvlinkWeightPercentage, _ = strconv.ParseInt(string(p), 0, 64)
	}

	p, err = exec.Command("cat", "/gpu-scheduler-configmap/least-allocated-pod-prefer").Output()
	if err == nil {
		pp := string(p)
		if pp == "spread" {
			gpuAllocatePrefer = true
		} else {
			gpuAllocatePrefer = false
		}
	}

	fmt.Println("<GPU Scheduler Policy List>")
	fmt.Println("1.", r.Policy1)
	fmt.Println("  -node weight : ", nodeWeight)
	fmt.Println("  -gpu weight : ", gpuWeight)
	fmt.Println("2.", r.Policy2)
	fmt.Println("  -value : ", reschedulePermit)
	fmt.Println("3.", r.Policy3)
	fmt.Println("  -value : ", nodeReservetionPermit)
	fmt.Println("4.", r.Policy4)
	fmt.Println("  -value : ", nvlinkWeightPercentage)
	fmt.Println("5.", r.Policy5)
	fmt.Println("  -value : ", gpuAllocatePrefer)

	return &SchedulingPolicy{
		NodeWeight:             nodeWeight,
		GPUWeight:              gpuWeight,
		ReSchedulePermit:       reschedulePermit,
		NodeReservationPermit:  nodeReservetionPermit,
		NVLinkWeightPercentage: nvlinkWeightPercentage,
		GPUAllocatePrefer:      gpuAllocatePrefer,
		changed:                make(chan bool),
	}
}

func (sched *GPUScheduler) NextPod() *r.QueuedPodInfo {
	newPod, err := sched.SchedulingQueue.Pop_AvtiveQ()
	if err != nil {
		fmt.Print("NextPod Error : ", err)
		return nil
	}

	return newPod
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
