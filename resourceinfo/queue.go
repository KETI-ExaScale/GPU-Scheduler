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

// Add adds a pod to the active queue. It should be called only when a new pod
// is added so there is no chance the pod is already in active/unschedulable/backoff queues
func (q *SchedulingQueue) Add_AvtiveQ(pod *v1.Pod) error {
	q.lock.Lock()
	defer q.lock.Unlock()

	was_empty := q.Empty()
	pInfo := newQueuedPodInfo(pod)
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
	fmt.Println("print active q")
	for e := q.activeQ.Front(); e != nil; e = e.Next() {
		fmt.Println("# ", e.Value.(*QueuedPodInfo).Pod.Name)
	}
}

func (q *SchedulingQueue) PrintbackoffQ() {
	fmt.Println("print backoff q")
	for e := q.podBackoffQ.Front(); e != nil; e = e.Next() {
		fmt.Println("# ", e.Value.(*QueuedPodInfo).Pod.Name)
	}
}

func (q *SchedulingQueue) Pop_AvtiveQ() (*QueuedPodInfo, error) {
	// fmt.Println("Pop_ActiveQ")
	q.lock.Lock()
	defer q.lock.Unlock()

	for q.Empty() {
		// fmt.Println("check1")
		if q.closed {
			return nil, fmt.Errorf("queueClosed")
		}
		q.cond.Wait()
		// fmt.Println("check2")
	}

	obj := q.activeQ.Front()
	if obj == nil {
		// fmt.Println("nil!!!")
	}
	pInfo := obj.Value.(*QueuedPodInfo)
	q.activeQ.Remove(obj)

	// fmt.Println("pop active q")
	// q.PrintQ()

	// if pInfo == nil {
	// 	return q.Wait_and_pop()
	// }

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

// func (q *SchedulingQueue) Pop_BackoffQ() (*QueuedPodInfo, error) {
// 	q.lock.Lock()
// 	defer q.lock.Unlock()

// 	for q.Empty() {
// 		if q.closed {
// 			return nil, fmt.Errorf("queueClosed")
// 		}
// 		q.cond.Wait()
// 	}

// 	obj := q.podBackoffQ.Front()
// 	pInfo := obj.Value.(*QueuedPodInfo)
// 	q.podBackoffQ.Remove(obj)

// 	// if pInfo == nil {
// 	// 	return q.Wait_and_pop()
// 	// }

// 	return pInfo, nil
// }

// func (q *SchedulingQueue) Activate() error {
// 	q.lock.Lock()
// 	defer q.lock.Unlock()

// 	for pInfo := q.podBackoffQ.Front(); pInfo != nil; pInfo = pInfo.Next() {
// 		queuedPodInfo := pInfo.Value.(*QueuedPodInfo)
// 		queuedPodInfo.Timestamp = time.Now()
// 		queuedPodInfo.activate = true
// 		queuedPodInfo.Attempts++

// 		q.activeQ.PushBack(pInfo)
// 		q.podBackoffQ.Remove(pInfo)
// 	}

// 	q.cond.Broadcast()

// 	return nil
// }

func (q *SchedulingQueue) Len() int {
	q.lock.Lock()
	defer q.lock.Unlock()

	return q.activeQ.Len()
}

func (q *SchedulingQueue) Empty() bool {
	return q.activeQ.Len() == 0
}

func (q *SchedulingQueue) FlushBackoffQCompleted() {
	// fmt.Println("flushbackoffq")
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
		// fmt.Println("broadcast2")
		q.cond.Broadcast()
	}
}

// func isPodUpdated(oldPod, newPod *v1.Pod) bool {
// 	strip := func(pod *v1.Pod) *v1.Pod {
// 		p := pod.DeepCopy()
// 		p.ResourceVersion = ""
// 		p.Generation = 0
// 		p.Status = v1.PodStatus{}
// 		p.ManagedFields = nil
// 		p.Finalizers = nil
// 		return p
// 	}
// 	return !reflect.DeepEqual(strip(oldPod), strip(newPod))
// }

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

func (q *SchedulingQueue) Update(oldPod, newPod *v1.Pod) error {
	fmt.Println("updatepod in scheduling queue: ", oldPod.Name)
	q.lock.Lock()
	defer q.lock.Unlock()

	was_empty := q.Empty()

	if oldPod != nil {
		if exists, oldPodInfo := q.backoffQLookup(oldPod.UID); exists {
			pInfo := updatePod(oldPodInfo, newPod)
			pInfo.Activate()
			q.activeQ.PushBack(pInfo)
		}
		if exists, oldPodInfo := q.activeQLookup(oldPod.UID); exists {
			pInfo := updatePod(oldPodInfo, newPod)
			q.activeQ.PushBack(pInfo)
		}
		return nil
	}

	// If pod is not in any of the queues, we put it in the active queue.
	pInfo := newQueuedPodInfo(newPod)
	q.activeQ.PushBack(pInfo)

	if was_empty {
		// fmt.Println("broadcast3")
		q.cond.Broadcast()
	}

	return nil
}

// Delete deletes the item from either of the two queues. It assumes the pod is
// only in one queue.
func (q *SchedulingQueue) Delete(pod *v1.Pod) error {
	fmt.Println("delete pod from scheduing queue")
	q.lock.Lock()
	defer q.lock.Unlock()
	// q.PodNominator.DeleteNominatedPodIfExists(pod)

	q.PrintactiveQ()
	q.PrintbackoffQ()

	if exists, podInfo := q.backoffQLookup(pod.UID); exists {
		fmt.Println("delete Backoff Queue pod [", podInfo.PodInfo.Pod.Name, "]")
	}
	if exists, podInfo := q.activeQLookup(pod.UID); exists {
		fmt.Println("delete Active Queue pod [", podInfo.PodInfo.Pod.Name, "]")
	}

	return nil
}
