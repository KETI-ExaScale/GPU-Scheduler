/*
Copyright 2015 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resourceinfo

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

var (
	cleanAssumedPeriod         = 1 * time.Second
	durationToExpireAssumedPod = 15 * time.Minute
)

const (
	Pending         = "Pending"
	Assumed         = "Assumed"
	BindingFinished = "BindingFinished"
)

type NodeCache struct {
	period time.Duration
	// ttl                    time.Duration
	mu sync.RWMutex
	// assumedPods            sets.String          //스케줄링 완료된 파드 이름, 생성되면 지움
	PodStates              map[string]*PodState //스케줄링한 파드 전부(생성/대기)
	NodeInfoList           map[string]*NodeInfo
	ImageStates            map[string]*imageState
	GPUMemoryMostInCluster int64
	TotalNodeCount         int
	AvailableNodeCount     int
	HostKubeClient         *kubernetes.Clientset
}

func NewNodeInfoCache(hostKubeClient *kubernetes.Clientset) *NodeCache {
	return &NodeCache{
		period: cleanAssumedPeriod,
		// ttl:                    durationToExpireAssumedPod,
		// assumedPods:            make(sets.String),
		PodStates:              make(map[string]*PodState),
		NodeInfoList:           make(map[string]*NodeInfo),
		ImageStates:            make(map[string]*imageState),
		GPUMemoryMostInCluster: 0,
		TotalNodeCount:         0,
		AvailableNodeCount:     0,
		HostKubeClient:         hostKubeClient,
	}
}

func (c *NodeCache) InitNodeInfoCache( /*scheduliungQ *SchedulingQueue*/ ) {
	nodes, _ := c.HostKubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	// pods, _ := c.HostKubeClient.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})

	for _, node := range nodes.Items {
<<<<<<< HEAD
		if !IsMasterNode(&node) {
=======
		if !isNonGPUNode(&node) { //nongpunode여도 추가하는걸로 수정하기
>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85
			fmt.Println("[node name]: ", node.Name)
			c.AddNode(node) //nodeinfolist에 node 추가, imagestate에 이미지 추가
		}
	}

	// for _, pod := range pods.Items {
	// 	podPtr := (*corev1.Pod)(unsafe.Pointer(&pod))
	// 	if pod.Spec.NodeName == "" {
	// 		fmt.Println("[pod name]: ", pod.Name)
	// 		scheduliungQ.Add_AvtiveQ(podPtr)
	// 	}
	// }
}

<<<<<<< HEAD
const (
	control_plane = "node-role.kubernetes.io/control-plane"
	master        = "node-role.kubernetes.io/master"
)

func IsMasterNode(node *corev1.Node) bool {
	if _, ok := node.Labels[control_plane]; ok {
		return true
	} else if _, ok := node.Labels[master]; ok {
		return true
	} else {
		return false
	}
}

=======
>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85
//metric update, score init
func (c *NodeCache) DumpCache() error {
	fmt.Println("[Dump Node Info Cache]")

<<<<<<< HEAD
	fmt.Println("(0) available node count: ", c.AvailableNodeCount)
	for nodeName, nodeInfo := range c.NodeInfoList {
		fmt.Println("===Node List===")
=======
	for nodeName, nodeInfo := range c.NodeInfoList {
>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85
		fmt.Println("(1) node name {", nodeName, "}")

		fmt.Println("(2) pods: ")
		a := 1
		for _, pod := range nodeInfo.Pods {
			fmt.Println("- ", a, ":", pod.Pod.Name)
			a++
		}

		// fmt.Print("(3) image: ")
		// for imageName, _ := range nodeInfo.ImageStates {
		// 	fmt.Print(imageName, ", ")
		// }
		// fmt.Println()

		fmt.Print("(3) num of image: ", len(nodeInfo.ImageStates), "\n")

		fmt.Println("(4) gpu name: ")
		for i, uuid := range nodeInfo.NodeMetric.GPU_UUID {
			fmt.Println("- ", i, ":", uuid)
		}

<<<<<<< HEAD
		fmt.Println("(5) total gpu count: ", nodeInfo.NodeMetric.TotalGPUCount)
=======
		fmt.Println("(5) available node count: ", c.AvailableNodeCount)
>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85

		fmt.Print("(6) used ports: ")
		for port, _ := range nodeInfo.UsedPorts {
			fmt.Print(port, ", ")
		}
		fmt.Println()

		fmt.Println("(7) nvlink list: ")
<<<<<<< HEAD
		for _, nvlink := range nodeInfo.NodeMetric.NVLinkList {
			fmt.Println("-", nvlink.GPU1, ":", nvlink.GPU2, ":", nvlink.Link)
		}
		fmt.Println()
=======
>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85
	}

	return nil
}

func (c *NodeCache) NodeCountDown() {
	c.AvailableNodeCount--
}

type PodState struct {
	Pod *corev1.Pod
	// Used by assumedPod to determinate expiration.
	deadline *time.Time
	// Used to block cache from expiring assumedPod if binding still runs
	State string
}

type imageState struct {
	// Size of the image
	size int64
	// A set of node names for nodes having this image present
	nodes sets.String
}

// createImageStateSummary returns a summarizing snapshot of the given image's state.
func (cache *NodeCache) createImageStateSummary(state *imageState) *ImageStateSummary {
	return &ImageStateSummary{
		Size:     state.size,
		NumNodes: len(state.nodes),
	}
}

// removeNodeInfoFromList removes a NodeInfo from the "cache.nodes" doubly
// linked list.
// We assume cache lock is already acquired.
func (cache *NodeCache) removeNodeInfoFromList(name string) {
	delete(cache.NodeInfoList, name)
}

func (cache *NodeCache) removePodStates(node *corev1.Node) {
	n, ok := cache.NodeInfoList[node.Name]
	if !ok {
		return
	}

	for _, pod := range n.Pods {
		key, err := GetPodKey(pod.Pod)
		if err != nil {
			return
		}
		delete(cache.PodStates, key)
	}
}

func (cache *NodeCache) AddPodState(pod corev1.Pod, s string) error {
	key, err := GetPodKey(&pod)
	if err != nil {
		return err
	}
	ps := &PodState{
		Pod:      &pod,
		deadline: nil,
		State:    s,
	}
	cache.PodStates[key] = ps

	return nil
}

<<<<<<< HEAD
func (cache *NodeCache) UpdatePodState(pod *corev1.Pod, s string) error {
	key, err := GetPodKey(pod)
	if err != nil {
		return err
	}

	cache.PodStates[key].State = s
	return nil
=======
func (cache *NodeCache) ChangePodState(pod *corev1.Pod, s string) {
	key, err := GetPodKey(pod)
	if err != nil {
		return
	}

	cache.PodStates[key].State = s
>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85
}

func (cache *NodeCache) CheckPodStateExist(pod *corev1.Pod) (bool, string) {
	key, err := GetPodKey(pod)
	if err != nil {
		return false, ""
	}
	// for a, b := range cache.PodStates {
	// 	fmt.Println("|", a, "|", b.Pod.Name, "|", b.State, "|")
	// }
	if podState, ok := cache.PodStates[key]; ok {
		return true, podState.State
	}
	return false, ""
}

// // NodeCount returns the number of nodes in the cache.
// // DO NOT use outside of tests.
// func (cache *Cache) NodeCount() int {
// 	cache.mu.RLock()
// 	defer cache.mu.RUnlock()
// 	return len(cache.nodeInfoList)
// }

// // PodCount returns the number of pods in the cache (including those from deleted nodes).
// // DO NOT use outside of tests.
// func (cache *Cache) PodCount() (int, error) {
// 	cache.mu.RLock()
// 	defer cache.mu.RUnlock()
// 	// podFilter is expected to return true for most or all of the pods. We
// 	// can avoid expensive array growth without wasting too much memory by
// 	// pre-allocating capacity.
// 	count := 0
// 	for _, n := range cache.nodeInfoList {
// 		count += len(n.Pods)
// 	}
// 	return count, nil
// }

<<<<<<< HEAD
// func (cache *NodeCache) AssumePod(pod *corev1.Pod) error {
// 	cache.mu.Lock()
// 	defer cache.mu.Unlock()

// 	return cache.UpdatePodState(pod, Assumed)
// }
=======
func (cache *NodeCache) AssumePod(pod *corev1.Pod) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	return cache.AddPodState(*pod, Assumed)
}
>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85

// func (cache *Cache) FinishBinding(pod *corev1.Pod) error {
// 	return cache.finishBinding(pod, time.Now())
// }

// // finishBinding exists to make tests determinitistic by injecting now as an argument
// func (cache *Cache) finishBinding(pod *corev1.Pod, now time.Time) error {
// 	key, err := GetPodKey(pod)
// 	if err != nil {
// 		return err
// 	}

// 	cache.mu.RLock()
// 	defer cache.mu.RUnlock()

// 	klog.V(5).InfoS("Finished binding for pod, can be expired", "pod", klog.KObj(pod))
// 	currState, ok := cache.podStates[key]
// 	if ok && cache.assumedPods.Has(key) {
// 		dl := now.Add(cache.ttl)
// 		currState.bindingFinished = true
// 		currState.deadline = &dl
// 	}
// 	return nil
// }

// func (cache *Cache) ForgetPod(pod *corev1.Pod) error {
// 	key, err := GetPodKey(pod)
// 	if err != nil {
// 		return err
// 	}

// 	cache.mu.Lock()
// 	defer cache.mu.Unlock()

// 	currState, ok := cache.podStates[key]
// 	if ok && currState.pod.Spec.NodeName != pod.Spec.NodeName {
// 		return fmt.Errorf("pod %v was assumed on %v but assigned to %v", key, pod.Spec.NodeName, currState.pod.Spec.NodeName)
// 	}

// 	// Only assumed pod can be forgotten.
// 	if ok && cache.assumedPods.Has(key) {
// 		return cache.removePod(pod)
// 	}
// 	return fmt.Errorf("pod %v wasn't assumed so cannot be forgotten", key)
// }

// Assumes that lock is already acquired.
func (cache *NodeCache) addPod(pod *corev1.Pod, status string) error {
	// key, err := GetPodKey(pod)
	// if err != nil {
	// 	return err
	// }

	n, ok := cache.NodeInfoList[pod.Spec.NodeName]
	if ok {
		n.AddPod(*pod)
		cache.AddPodState(*pod, BindingFinished)
		// cache.moveNodeInfoToHead(pod.Spec.NodeName)
		// ps := &podState{
		// 	pod: pod,
		// }
		// cache.PodStates[key] = ps
		// if assumePod {
		// 	cache.assumedPods.Insert(key)
		// }
	}

	return nil
}

// Assumes that lock is already acquired.
func (cache *NodeCache) updatePod(oldPod, newPod *corev1.Pod) error {

	ok, oldState := cache.CheckPodStateExist(oldPod)
	if !ok {
		cache.AddPodState(*oldPod, Pending)
	}

	if err := cache.removePod(oldPod); err != nil {
		return err
	}

	return cache.addPod(newPod, oldState)
}

// Assumes that lock is already acquired.
// Removes a pod from the cached node info. If the node information was already
// removed and there are no more pods left in the node, cleans up the node from
// the cache.
func (cache *NodeCache) removePod(pod *corev1.Pod) error {
	key, err := GetPodKey(pod)
	if err != nil {
		return err
	}

	n, ok := cache.NodeInfoList[pod.Spec.NodeName]
	if !ok {
		klog.ErrorS(nil, "Node not found when trying to remove pod", "node", klog.KRef("", pod.Spec.NodeName), "pod", klog.KObj(pod))
	} else {
		if err := n.RemovePod(pod); err != nil {
			return err
		}
		if len(n.Pods) == 0 && n.node == nil {
			cache.removeNodeInfoFromList(pod.Spec.NodeName)
		}
	}

	delete(cache.PodStates, key)
	// delete(cache.assumedPods, key)
	return nil
}

func (cache *NodeCache) AddPod(pod *corev1.Pod) error {
	fmt.Println("cache.AddPod: ", pod.Name)
	key, err := GetPodKey(pod)
	if err != nil {
		return err
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	_, ok := cache.PodStates[key]
	switch {
	case ok:
		// if cache.assumedPods.Has(key) {
		// 	if currState.pod.Spec.NodeName != pod.Spec.NodeName {
		// 		// The pod was added to a different node than it was assumed to.
		// 		klog.InfoS("Pod was added to a different node than it was assumed", "pod", klog.KObj(pod), "assumedNode", klog.KRef("", pod.Spec.NodeName), "currentNode", klog.KRef("", currState.pod.Spec.NodeName))
		// 		if err = cache.updatePod(currState.pod, pod); err != nil {
		// 			klog.ErrorS(err, "Error occurred while updating pod")
		// 		}
		// 	} else {
		// 		delete(cache.assumedPods, key)
		// 		cache.PodStates[key].deadline = nil
		// 		cache.PodStates[key].pod = pod
		// 	}
		// } else {
		// 	cache.PodStates[key].deadline = nil
		// 	cache.PodStates[key].pod = pod
		// }
		fmt.Println("already exist status")
	case !ok:
		if err = cache.addPod(pod, BindingFinished); err != nil {
			klog.ErrorS(err, "Error occurred while adding pod")
		}
	default:
		return fmt.Errorf("pod %v was already in added state", key)
	}

	return nil
}

func (cache *NodeCache) UpdatePod(oldPod, newPod *corev1.Pod) error {
	fmt.Println("cache.UpdatePod: ", oldPod.Name)
	// key, err := GetPodKey(oldPod)
	// if err != nil {
	// 	return err
	// }

	cache.mu.Lock()
	defer cache.mu.Unlock()

	// currState, ok := cache.PodStates[key]
	// // An assumed pod won't have Update/Remove event. It needs to have Add event
	// // before Update event, in which case the state would change from Assumed to Added.
	// if ok && !cache.assumedPods.Has(key) {
	// 	if currState.pod.Spec.NodeName != newPod.Spec.NodeName {
	// 		klog.ErrorS(nil, "Pod updated on a different node than previously added to", "pod", klog.KObj(oldPod))
	// 		klog.ErrorS(nil, "scheduler cache is corrupted and can badly affect scheduling decisions")
	// 		os.Exit(1)
	// 	}
	return cache.updatePod(oldPod, newPod)
	// }
	// return fmt.Errorf("pod %v is not added to scheduler cache, so cannot be updated", key)
}

func (cache *NodeCache) RemovePod(pod *corev1.Pod) error {
	fmt.Println("cache.RemovePod: ", pod.Name)
	key, err := GetPodKey(pod)
	if err != nil {
		return err
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	currState, ok := cache.PodStates[key]
	if !ok {
		return fmt.Errorf("pod %v is not found in scheduler cache, so cannot be removed from it", key)
	}
	if currState.Pod.Spec.NodeName != pod.Spec.NodeName {
		klog.ErrorS(nil, "Pod was added to a different node than it was assumed", "pod", klog.KObj(pod), "assumedNode", klog.KRef("", pod.Spec.NodeName), "currentNode", klog.KRef("", currState.Pod.Spec.NodeName))
		if pod.Spec.NodeName != "" {
			// An empty NodeName is possible when the scheduler misses a Delete
			// event and it gets the last known state from the informer cache.
			klog.ErrorS(nil, "scheduler cache is corrupted and can badly affect scheduling decisions")
			os.Exit(1)
		}
	}
	return cache.removePod(currState.Pod)
}

// func (cache *NodeCache) IsAssumedPod(pod *corev1.Pod) (bool, error) {
// 	key, err := GetPodKey(pod)
// 	if err != nil {
// 		return false, err
// 	}

// 	cache.mu.RLock()
// 	defer cache.mu.RUnlock()

// 	return cache.assumedPods.Has(key), nil
// }

// func (cache *NodeCache) IsAssumed(pod *corev1.Pod) bool {
// 	key, _ := GetPodKey(pod)
// 	cache.mu.RLock()
// 	defer cache.mu.RUnlock()

// 	return cache.assumedPods.Has(key)
// }

// func (cache *NodeCache) RemoveAssumePod(pod *corev1.Pod) error {
// 	key, err := GetPodKey(pod)
// 	if err != nil {
// 		return err
// 	}

// 	cache.mu.RLock()
// 	defer cache.mu.RUnlock()

// 	delete(cache.assumedPods, key)

// 	return nil

// }

// // GetPod might return a pod for which its node has already been deleted from
// // the main cache. This is useful to properly process pod update events.
// func (cache *Cache) GetPod(pod *corev1.Pod) (*corev1.Pod, error) {
// 	key, err := GetPodKey(pod)
// 	if err != nil {
// 		return nil, err
// 	}

// 	cache.mu.RLock()
// 	defer cache.mu.RUnlock()

// 	podState, ok := cache.podStates[key]
// 	if !ok {
// 		return nil, fmt.Errorf("pod %v does not exist in scheduler cache", key)
// 	}

// 	return podState.pod, nil
// }

func (cache *NodeCache) AddNode(node corev1.Node) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	n, ok := cache.NodeInfoList[node.Name]
	if ok {
		cache.RemoveNode(n.Node())
	}
	n = NewNodeInfo()
	cache.NodeInfoList[node.Name] = n
<<<<<<< HEAD
	cache.TotalNodeCount++
=======
>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85

	podsInNode, _ := cache.HostKubeClient.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + node.Name,
	})

	// pods, podswithaffinity, podswithrequirequiredantiaffinity, usedport,
	// pvcrefcounts, requested
	for _, pod := range podsInNode.Items {
		fmt.Println("# pod: ", pod.Name)
		if err := cache.addPod(&pod, BindingFinished); err != nil {
			klog.ErrorS(err, "Error occurred while adding pod")
		}
	}

<<<<<<< HEAD
	err := n.InitNodeInfo(&node, cache.HostKubeClient)
	if err != nil {
		fmt.Println("<error> cannot init node info - ", err)
		return err
	}

	cache.GPUMemoryMostInCluster = Max(cache.GPUMemoryMostInCluster, n.NodeMetric.MaxGPUMemory)
	cache.DumpCache() //확인용
	cache.addNodeImageStates(&node, n)
	n.SetNode(&node)
=======
	n.InitNodeInfo(&node, cache.HostKubeClient)
	cache.GPUMemoryMostInCluster = Max(cache.GPUMemoryMostInCluster, n.NodeMetric.MaxGPUMemory)

	cache.addNodeImageStates(&node, n)
	n.SetNode(&node)
	cache.TotalNodeCount++
>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85

	return nil
}

<<<<<<< HEAD
func (cache *NodeCache) UpdateNode(oldNode, newNode *corev1.Node) error {
=======
//이미지만 업데이트? 다른 리소스 업데이트면 어떻게?
func (cache *NodeCache) UpdateNode(oldNode, newNode *corev1.Node) error /* *NodeInfo */ {
>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85
	cache.mu.Lock()
	defer cache.mu.Unlock()

	n, ok := cache.NodeInfoList[newNode.Name]
	if !ok {
		n = NewNodeInfo()
		cache.NodeInfoList[newNode.Name] = n
<<<<<<< HEAD
=======
		// cache.nodeTree.addNode(newNode)
>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85
	} else {
		cache.removeNodeImageStates(n.Node())
	}

<<<<<<< HEAD
=======
	n.InitNodeInfo(newNode, cache.HostKubeClient)
	cache.GPUMemoryMostInCluster = Max(cache.GPUMemoryMostInCluster, n.NodeMetric.MaxGPUMemory)

	// cache.nodeTree.updateNode(oldNode, newNode)
>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85
	cache.addNodeImageStates(newNode, n)
	n.SetNode(newNode)

	return nil
<<<<<<< HEAD
=======
	/*return n.Clone()*/
>>>>>>> c78b3aab458596cbc06a1a80d03f7cb202c02a85
}

// RemoveNode removes a node from the cache's tree.
// The node might still have pods because their deletion events didn't arrive
// yet. Those pods are considered removed from the cache, being the node tree
// the source of truth.
// However, we keep a ghost node with the list of pods until all pod deletion
// events have arrived. A ghost node is skipped from snapshots.
func (cache *NodeCache) RemoveNode(node *corev1.Node) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	n, ok := cache.NodeInfoList[node.Name]
	if !ok {
		return fmt.Errorf("node %v is not found", node.Name)
	}

	cache.removeNodeImageStates(node)
	cache.removePodStates(node)
	n.RemoveNode()
	cache.removeNodeInfoFromList(node.Name)
	cache.TotalNodeCount--

	return nil
}

// addNodeImageStates adds states of the images on given node to the given nodeInfo and update the imageStates in
// scheduler cache. This function assumes the lock to scheduler cache has been acquired.
func (cache *NodeCache) addNodeImageStates(node *corev1.Node, nodeInfo *NodeInfo) {
	newSum := make(map[string]*ImageStateSummary)

	// fmt.Println("image num [")
	for _, image := range node.Status.Images {
		// fmt.Print(i, ",")
		for _, name := range image.Names {
			// update the entry in imageStates
			state, ok := cache.ImageStates[name]
			if !ok {
				state = &imageState{
					size:  image.SizeBytes,
					nodes: sets.NewString(node.Name),
				}
				cache.ImageStates[name] = state
			} else {
				state.nodes.Insert(node.Name)
			}
			// create the imageStateSummary for this image
			if _, ok := newSum[name]; !ok {
				newSum[name] = cache.createImageStateSummary(state)
			}
		}
	}
	// fmt.Println("]")
	nodeInfo.ImageStates = newSum
}

// removeNodeImageStates removes the given node record from image entries having the node
// in imageStates cache. After the removal, if any image becomes free, i.e., the image
// is no longer available on any node, the image entry will be removed from imageStates.
func (cache *NodeCache) removeNodeImageStates(node *corev1.Node) {
	if node == nil {
		return
	}

	for _, image := range node.Status.Images {
		for _, name := range image.Names {
			state, ok := cache.ImageStates[name]
			if ok {
				state.nodes.Delete(node.Name)
				if len(state.nodes) == 0 {
					// Remove the unused image to make sure the length of
					// imageStates represents the total number of different
					// images on all nodes
					delete(cache.ImageStates, name)
				}
			}
		}
	}
}

// func (cache *Cache) run() {
// 	go wait.Until(cache.cleanupExpiredAssumedPods, cache.period, cache.stop)
// }

// func (cache *Cache) cleanupExpiredAssumedPods() {
// 	cache.cleanupAssumedPods(time.Now())
// }

// // cleanupAssumedPods exists for making test deterministic by taking time as input argument.
// // It also reports metrics on the cache size for nodes, pods, and assumed pods.
// func (cache *Cache) cleanupAssumedPods(now time.Time) {
// 	cache.mu.Lock()
// 	defer cache.mu.Unlock()
// 	// defer cache.updateMetrics()

// 	// The size of assumedPods should be small
// 	for key := range cache.assumedPods {
// 		ps, ok := cache.podStates[key]
// 		if !ok {
// 			klog.ErrorS(nil, "Key found in assumed set but not in podStates, potentially a logical error")
// 			os.Exit(1)
// 		}
// 		if !ps.bindingFinished {
// 			klog.V(5).InfoS("Could not expire cache for pod as binding is still in progress",
// 				"pod", klog.KObj(ps.pod))
// 			continue
// 		}
// 		if now.After(*ps.deadline) {
// 			klog.InfoS("Pod expired", "pod", klog.KObj(ps.pod))
// 			if err := cache.removePod(ps.pod); err != nil {
// 				klog.ErrorS(err, "ExpirePod failed", "pod", klog.KObj(ps.pod))
// 			}
// 		}
// 	}
// }
