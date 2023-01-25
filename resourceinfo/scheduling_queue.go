package resourceinfo

import (
	"fmt"
	"sync"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
)

const (
	UnscehdulableDuration                time.Duration = 30 * time.Second //Unschedulable Pod Duration
	UnschedulableAndUnresolvableDuration time.Duration = 1 * time.Minute  //UnschedulableAndUnresolvable Pod Duration
	ErrorDuration                        time.Duration = 3 * time.Minute  //Error Pod Duration
)

type SchedulingQueue interface {
	Add(pod *v1.Pod) error               //activeQ insert
	AddBackoffQ(qp *QueuedPodInfo) error //backoffQ insert
	// AddUnschedulableIfNotPresent(pod *QueuedPodInfo) error
	Pop() (*QueuedPodInfo, error)        //activeQ Pop
	Update(oldPod, newPod *v1.Pod) error //update pod in activeQ/backoffQ/unschedulable
	Delete(pod *v1.Pod) error            //delete pod in activeQ/backoffQ/unschedulable
	MoveAllToActiveOrBackoffQueue(flag bool) error
	// PendingPods() []*v1.Pod
	Close()
	Run()
}

type PriorityQueue struct { //스케줄러에서 사용하는 우선순위 큐
	activeQ  *Heap
	BackoffQ *Heap
	lock     *sync.Mutex
	cond     *sync.Cond
	stop     chan struct{}
	closed   bool
}

var _ SchedulingQueue = &PriorityQueue{} // priority queue implement scheduling queue

func NewSchedulingQueue() SchedulingQueue {
	return NewPriorityQueue()
}

func NewPriorityQueue() *PriorityQueue {
	sq := &PriorityQueue{
		activeQ:  New(podInfoKeyFunc, podsCompareActivePriority),
		BackoffQ: New(podInfoKeyFunc, podsCompareBackoffCompleted),
		lock:     new(sync.Mutex),
		stop:     make(chan struct{}),
		closed:   false,
	}
	sq.cond = sync.NewCond(sq.lock)
	return sq
}

func (p *PriorityQueue) Run() {
	go wait.Until(p.flushBackoffQCompleted, 3*time.Second, p.stop) //backoffQ >> activeQ flush routine
	// go wait.Until(p.flushUnschedulablePodsLeftover, 30*time.Second, p.stop)
}

func (p *PriorityQueue) Close() {
	p.lock.Lock()
	defer p.lock.Unlock()
	close(p.stop)
	p.closed = true
	p.cond.Broadcast()
}

func (p *PriorityQueue) Add(pod *v1.Pod) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	was_empty := p.Empty()
	pInfo := NewQueuedPodInfo(pod)
	p.activeQ.Add(pInfo)
	p.PrintActiveQ()

	if was_empty {
		p.cond.Broadcast()
	}

	return nil
}

func (p *PriorityQueue) AddBackoffQ(qp *QueuedPodInfo) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	//queue pod info update
	qp.Timestamp = time.Now()
	p.BackoffQ.Add(qp)
	p.PrintBackoffQ()

	return nil
}

func (p *PriorityQueue) Pop() (*QueuedPodInfo, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	for p.Empty() {
		if p.closed {
			return nil, fmt.Errorf("queueClosed")
		}
		p.cond.Wait()
	}

	obj, err := p.activeQ.Pop()
	if err != nil {
		return nil, err
	}
	pInfo := obj.(*QueuedPodInfo)
	pInfo.Attempts++

	return pInfo, nil
}

func (p *PriorityQueue) MoveAllToActiveOrBackoffQueue(flushAllPods bool) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	broadcast := false
	for {
		rawPodInfo := p.BackoffQ.Peek()
		if rawPodInfo == nil {
			break
		}

		qp := rawPodInfo.(*QueuedPodInfo)
		_, err := p.BackoffQ.Pop()
		if err != nil {
			KETI_LOG_L3(fmt.Sprintf("Unable to pop pod from backoff queue despite backoff completion {%s}", qp.Pod.Name))
			break
		}

		if flushAllPods { // flush all pods
			qp.Timestamp = time.Now()
			qp.PriorityScore = qp.UserPriority + qp.Attempts*10
			p.activeQ.Add(rawPodInfo)
			broadcast = true
		} else { // flush only unschedulable pods
			if qp.Status.Code == Unschedulable {
				qp.Timestamp = time.Now()
				qp.PriorityScore = qp.UserPriority + qp.Attempts*10
				p.activeQ.Add(rawPodInfo)
				broadcast = true
			} else {
				continue
			}
		}
	}

	if broadcast {
		p.cond.Broadcast()
	}

	return nil
}

func (p *PriorityQueue) Len() int {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.activeQ.Len()
}

func (p *PriorityQueue) Empty() bool {
	return p.activeQ.Len() == 0
}

func (p *PriorityQueue) flushBackoffQCompleted() {
	p.lock.Lock()
	defer p.lock.Unlock()
	broadcast := false
	for {
		rawPodInfo := p.BackoffQ.Peek()
		if rawPodInfo == nil {
			break
		}
		qp := rawPodInfo.(*QueuedPodInfo)
		var boTime time.Time

		if qp.Status.Code == Unschedulable {
			boTime = qp.Timestamp.Add(UnscehdulableDuration)
		} else if qp.Status.Code == UnschedulableAndUnresolvable {
			boTime = qp.Timestamp.Add(UnschedulableAndUnresolvableDuration)
		} else if qp.Status.Code == Error {
			boTime = qp.Timestamp.Add(ErrorDuration)
		} else {
			KETI_LOG_L3(fmt.Sprintf("QueuePodInfo unknown status :%d {%s}", qp.Status.Code, qp.Pod.Name))
		}

		if !boTime.Before(time.Now()) {
			break
		}

		_, err := p.BackoffQ.Pop()
		if err != nil {
			KETI_LOG_L3(fmt.Sprintf("Unable to pop from backoff queue despite backoff completion {%s}", qp.Pod.Name))
			break
		}

		qp.Timestamp = time.Now()
		qp.PriorityScore = qp.UserPriority + qp.Attempts*10
		qp.Status.Code = Wait
		p.activeQ.Add(rawPodInfo)
		broadcast = true
	}

	if broadcast {
		p.cond.Broadcast()
	}
}

func updatePod(oldPodInfo interface{}, newPod *v1.Pod) *QueuedPodInfo {
	pInfo := oldPodInfo.(*QueuedPodInfo)
	pInfo.Update(newPod)
	return pInfo
}

func (p *PriorityQueue) Update(oldPod, newPod *v1.Pod) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if oldPod != nil {
		oldPodInfo := newQueuedPodInfoForLookup(oldPod)
		if oldPodInfo, exists, _ := p.activeQ.Get(oldPodInfo); exists {
			pInfo := updatePod(oldPodInfo, newPod)
			return p.activeQ.Update(pInfo)
		}

		if oldPodInfo, exists, _ := p.BackoffQ.Get(oldPodInfo); exists {
			pInfo := updatePod(oldPodInfo, newPod)
			return p.BackoffQ.Update(pInfo)
		}
	}

	// // If the pod is in the unschedulable queue, updating it may make it schedulable.
	// if usPodInfo := p.unschedulablePods.get(newPod); usPodInfo != nil {
	// 	pInfo := updatePod(usPodInfo, newPod)
	// 	p.PodNominator.UpdateNominatedPod(oldPod, pInfo.PodInfo)
	// 	if isPodUpdated(oldPod, newPod) {
	// 		if p.isPodBackingoff(usPodInfo) {
	// 			if err := p.BackoffQ.Add(pInfo); err != nil {
	// 				return err
	// 			}
	// 			p.unschedulablePods.delete(usPodInfo.Pod)
	// 		} else {
	// 			if err := p.activeQ.Add(pInfo); err != nil {
	// 				return err
	// 			}
	// 			p.unschedulablePods.delete(usPodInfo.Pod)
	// 			p.cond.Broadcast()
	// 		}
	// 	} else {
	// 		// Pod update didn't make it schedulable, keep it in the unschedulable queue.
	// 		p.unschedulablePods.addOrUpdate(pInfo)
	// 	}

	// 	return nil
	// }

	// If pod is not in any of the queues, we put it in the active queue.
	pInfo := NewQueuedPodInfo(newPod)
	if err := p.activeQ.Add(pInfo); err != nil {
		return err
	}
	p.cond.Broadcast()
	return nil
}

func newQueuedPodInfoForLookup(pod *v1.Pod, plugins ...string) *QueuedPodInfo {
	return &QueuedPodInfo{
		PodInfo: &PodInfo{Pod: pod},
	}
}

func (p *PriorityQueue) Delete(pod *v1.Pod) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if err := p.activeQ.Delete(newQueuedPodInfoForLookup(pod)); err != nil {
		// The item was probably not found in the activeQ.
		p.BackoffQ.Delete(newQueuedPodInfoForLookup(pod))
		// p.unschedulablePods.delete(pod)
	}
	return nil
}

func podInfoKeyFunc(obj interface{}) (string, error) {
	return cache.MetaNamespaceKeyFunc(obj.(*QueuedPodInfo).Pod)
}

type ExplicitKey string

func MetaNamespaceKeyFunc(obj interface{}) (string, error) {
	if key, ok := obj.(ExplicitKey); ok {
		return string(key), nil
	}
	meta, err := meta.Accessor(obj)
	if err != nil {
		return "", fmt.Errorf("object has no meta: %v", err)
	}
	if len(meta.GetNamespace()) > 0 {
		return meta.GetNamespace() + "/" + meta.GetName(), nil
	}
	return meta.GetName(), nil
}

// lessFunc_activeQ_우선순위 비교
func podsCompareActivePriority(podInfo1, podInfo2 interface{}) bool {
	pInfo1 := podInfo1.(*QueuedPodInfo)
	pInfo2 := podInfo2.(*QueuedPodInfo)
	bo1 := pInfo1.PriorityScore
	bo2 := pInfo2.PriorityScore
	return bo1 > bo2
}

// lessFunc_backoffQ_큐인서트시간 비교
func podsCompareBackoffCompleted(podInfo1, podInfo2 interface{}) bool {
	pInfo1 := podInfo1.(*QueuedPodInfo)
	pInfo2 := podInfo2.(*QueuedPodInfo)
	bo1 := pInfo1.Timestamp
	bo2 := pInfo2.Timestamp
	return bo1.Before(bo2) //원래는 bo1.Before(bo2)
}

// func (p *PriorityQueue) AddUnschedulableIfNotPresent(pod *QueuedPodInfo) error {
// 	return nil
// }

// func (p *PriorityQueue) flushUnschedulablePodsLeftover() {
// 	p.lock.Lock()
// 	defer p.lock.Unlock()

// 	var podsToMove []*framework.QueuedPodInfo
// 	currentTime := p.clock.Now()
// 	for _, pInfo := range p.unschedulablePods.podInfoMap {
// 		lastScheduleTime := pInfo.Timestamp
// 		if currentTime.Sub(lastScheduleTime) > p.podMaxInUnschedulablePodsDuration {
// 			podsToMove = append(podsToMove, pInfo)
// 		}
// 	}

// 	if len(podsToMove) > 0 {
// 		p.movePodsToActiveOrBackoffQueue(podsToMove, UnschedulableTimeout)
// 	}
// }

func (p *PriorityQueue) PrintActiveQ() {
	items := p.activeQ.List()
	KETI_LOG_L1("<test> active queue\n")
	for i := 0; i < len(items); i++ {
		KETI_LOG_L1(fmt.Sprintf("%d) name: %s, priority: %d, attempt: %d, score: %d", i, items[i].(*QueuedPodInfo).Pod.Name, items[i].(*QueuedPodInfo).UserPriority, items[i].(*QueuedPodInfo).Attempts, items[i].(*QueuedPodInfo).PriorityScore))
	}
}

func (p *PriorityQueue) PrintBackoffQ() {
	items := p.BackoffQ.List()
	KETI_LOG_L1("<test> backoff queue\n")
	for i := 0; i < len(items); i++ {
		KETI_LOG_L1(fmt.Sprintf("%d) name: %s, priority: %d, attempt: %d, score: %d", i, items[i].(*QueuedPodInfo).Pod.Name, items[i].(*QueuedPodInfo).UserPriority, items[i].(*QueuedPodInfo).Attempts, items[i].(*QueuedPodInfo).PriorityScore))
	}
}

// func (p *PriorityQueue) PendingPods() []*v1.Pod {
// 	return nil
// }
