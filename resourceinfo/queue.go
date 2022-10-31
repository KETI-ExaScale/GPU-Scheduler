package resourceinfo

import (
	"container/list"
	"fmt"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

type SchedulingQueue struct {
	activeQ               *list.List
	podBackoffQ           *list.List
	podMaxBackoffDuration time.Duration
	lock                  *sync.Mutex
	cond                  *sync.Cond
	stop                  chan struct{}
	closed                bool
}

func NewQueue() *SchedulingQueue {
	sq := &SchedulingQueue{
		activeQ:               list.New(),
		podBackoffQ:           list.New(),
		podMaxBackoffDuration: time.Second * 30,
		lock:                  new(sync.Mutex),
		stop:                  make(chan struct{}),
		closed:                false,
	}
	sq.cond = sync.NewCond(sq.lock)
	return sq
}

func (q *SchedulingQueue) Run() {
	go wait.Until(q.FlushBackoffQCompleted, 30*time.Second, q.stop)
}

func (q *SchedulingQueue) Close() {
	q.lock.Lock()
	defer q.lock.Unlock()
	close(q.stop)
	q.closed = true
	q.cond.Broadcast()
}

//새 파드 스케줄링 큐로 넣음
func (q *SchedulingQueue) Add_AvtiveQ(pod *v1.Pod) error {
	q.lock.Lock()
	defer q.lock.Unlock()

	was_empty := q.Empty()
	pInfo := NewQueuedPodInfo(pod)
	q.activeQ.PushBack(pInfo)

	// fmt.Println("add active q")
	// q.PrintQ()

	if was_empty {
		// fmt.Println("broadcast1")
		q.cond.Broadcast()
	}

	return nil
}

func (q *SchedulingQueue) PrintactiveQ() {
	KETI_LOG_L1("-print active q")
	for e := q.activeQ.Front(); e != nil; e = e.Next() {
		KETI_LOG_L1(fmt.Sprintf("# %s", e.Value.(*QueuedPodInfo).Pod.Name))
	}
}

func (q *SchedulingQueue) PrintbackoffQ() {
	KETI_LOG_L1("-print backoff q")
	for e := q.podBackoffQ.Front(); e != nil; e = e.Next() {
		KETI_LOG_L1(fmt.Sprintf("# %s", e.Value.(*QueuedPodInfo).Pod.Name))
	}
}

func (q *SchedulingQueue) Pop_AvtiveQ() (*QueuedPodInfo, error) {
	q.lock.Lock()
	defer q.lock.Unlock()

	for q.Empty() {
		if q.closed {
			return nil, fmt.Errorf("queueClosed")
		}
		q.cond.Wait()
	}

	//스케줄링 큐에 파드가 여러개일때 우선순위 스케줄링을 위한 정렬 수행
	if q.activeQ.Len() != 1 {
		//calculate pod priority score
		for obj := q.activeQ.Front(); obj != nil; obj = obj.Next() {
			qPod := obj.Value.(*QueuedPodInfo)
			score := 0
			if qPod.UserPriority == "L" {
				score = 10
			} else if qPod.UserPriority == "M" {
				score = 50
			} else if qPod.UserPriority == "H" {
				score = 100
			} else {
				score = 1500
			}
			qPod.PriorityScore = qPod.Attempts*10 + score
		}

	}

	obj := q.activeQ.Front()
	if obj == nil {
		KETI_LOG_L3("<error> scheduling queue is empty")
	}
	pInfo := obj.Value.(*QueuedPodInfo)
	q.activeQ.Remove(obj)

	pInfo.Attempts++
	return pInfo, nil
}

func (q *SchedulingQueue) Add_BackoffQ(queuePodInfo *QueuedPodInfo) error {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.podBackoffQ.PushBack(queuePodInfo)

	q.PrintbackoffQ()

	return nil
}

func (q *SchedulingQueue) Len() int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.activeQ.Len()
}

func (q *SchedulingQueue) Empty() bool {
	return q.activeQ.Len() == 0
}

func (q *SchedulingQueue) FlushBackoffQCompleted() {
	q.lock.Lock()
	defer q.lock.Unlock()

	was_empty := q.Empty()
	was_not_empty := (q.podBackoffQ.Len() != 0)

	for pInfo := q.podBackoffQ.Front(); pInfo != nil; pInfo = pInfo.Next() {
		queuedPodInfo := pInfo.Value.(*QueuedPodInfo)
		queuedPodInfo.Timestamp = time.Now()
		queuedPodInfo.activate = true
		queuedPodInfo.Attempts++

		q.activeQ.PushBack(queuedPodInfo)
		q.podBackoffQ.Remove(pInfo)
	}

	if was_empty && was_not_empty {
		q.cond.Broadcast()
	}
}

func (q *SchedulingQueue) backoffQLookup(uid types.UID) (bool, *QueuedPodInfo) {
	for pInfo := q.podBackoffQ.Front(); pInfo != nil; pInfo = pInfo.Next() {
		if pInfo.Value.(*QueuedPodInfo).PodUID == uid {
			pod := pInfo.Value.(*QueuedPodInfo)
			q.podBackoffQ.Remove(pInfo)
			return true, pod
		}
	}
	return false, nil
}

func (q *SchedulingQueue) activeQLookup(uid types.UID) (bool, *QueuedPodInfo) {
	for pInfo := q.activeQ.Front(); pInfo != nil; pInfo = pInfo.Next() {
		if pInfo.Value.(*QueuedPodInfo).PodUID == uid {
			pod := pInfo.Value.(*QueuedPodInfo)
			q.activeQ.Remove(pInfo)
			return true, pod
		}

	}
	return false, nil
}

func updatePod(oldPodInfo *QueuedPodInfo, newPod *v1.Pod) *QueuedPodInfo {
	oldPodInfo.PodInfo.Update(newPod)
	return oldPodInfo
}

//스케줄링 큐 안의 파드 정보 업데이트
func (q *SchedulingQueue) Update(oldPod, newPod *v1.Pod) error {
	q.lock.Lock()
	defer q.lock.Unlock()

	was_empty := q.Empty()

	if oldPod != nil {
		if exists, oldPodInfo := q.backoffQLookup(oldPod.UID); exists {
			pInfo := updatePod(oldPodInfo, newPod)
			pInfo.Activate()
			q.activeQ.PushBack(pInfo)
		} else if exists, oldPodInfo := q.activeQLookup(oldPod.UID); exists {
			pInfo := updatePod(oldPodInfo, newPod)
			q.activeQ.PushBack(pInfo)
		}
		return nil
	}

	pInfo := NewQueuedPodInfo(newPod)
	q.activeQ.PushBack(pInfo)

	if was_empty {
		q.cond.Broadcast()
	}

	return nil
}

//스케줄링 큐 안의 파드 삭제
func (q *SchedulingQueue) Delete(pod *v1.Pod) error {
	q.lock.Lock()
	defer q.lock.Unlock()

	q.PrintactiveQ()
	q.PrintbackoffQ()

	if exists, podInfo := q.backoffQLookup(pod.UID); exists {
		KETI_LOG_L1(fmt.Sprintf("- delete Backoff Queue pod [%s]", podInfo.PodInfo.Pod.Name))
	}
	if exists, podInfo := q.activeQLookup(pod.UID); exists {
		KETI_LOG_L1(fmt.Sprintf("- delete Active Queue pod [%s]", podInfo.PodInfo.Pod.Name))
	}

	return nil
}
