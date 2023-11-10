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
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
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
	period             time.Duration
	mu                 sync.RWMutex
	PodStates          map[string]*PodState //스케줄링한 파드 전부(생성/대기)
	NodeInfoList       map[string]*NodeInfo
	ImageStates        map[string]*imageState
	TotalNodeCount     int
	AvailableNodeCount int
	HostKubeClient     *kubernetes.Clientset
	// ttl                    time.Duration
	// assumedPods            sets.String          //스케줄링 완료된 파드 이름, 생성되면 지움
}

func NewNodeInfoCache(hostKubeClient *kubernetes.Clientset) *NodeCache {
	return &NodeCache{
		period:             cleanAssumedPeriod,
		PodStates:          make(map[string]*PodState),
		NodeInfoList:       make(map[string]*NodeInfo),
		ImageStates:        make(map[string]*imageState),
		TotalNodeCount:     0,
		AvailableNodeCount: 0,
		HostKubeClient:     hostKubeClient,
		// ttl:                    durationToExpireAssumedPod,
		// assumedPods:            make(sets.String),
	}
}

func (c *NodeCache) InitNodeInfoCache( /*scheduliungQ *SchedulingQueue*/ ) error {
	nodes, err := c.HostKubeClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}

	for _, node := range nodes.Items {
		c.AddNode(&node) // Nodeinfolist > Node, Imagestate
		// if !IsMasterNode(&node) { // Not Schedule To Master Node
		// 	c.AddNode(node)
		// }
	}

	if GPU_SCHEDUER_DEBUGG_LEVEL == LEVEL1 { //LEVEL = 1,2,3
		c.DumpCache()
	}

	return nil
}

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

// metric update, score init
func (c *NodeCache) DumpCache() error {
	KETI_LOG_L2("\n-----:: Dump Node Metric Cache ::-----")

	KETI_LOG_L2(fmt.Sprintf(">Total Node Count(available/total): (%d/%d)<", c.AvailableNodeCount, c.TotalNodeCount))
	for nodeName, nodeInfo := range c.NodeInfoList {
		if !nodeInfo.PluginResult.IsFiltered {
			KETI_LOG_L2(fmt.Sprintf(">> NodeName : %s", nodeName))
			// fmt.Println("(2) pods: ")
			// for _, pod := range nodeInfo.Pods {
			// 	fmt.Println("# ", pod.Pod.Name)
			// }
			KETI_LOG_L2(fmt.Sprintf("- Num Of Pods:%d", len(nodeInfo.Pods)))

			// fmt.Print("(3) image: ")
			// for imageName, _ := range nodeInfo.ImageStates {
			// 	fmt.Print(imageName, ", ")
			// }
			// fmt.Println()

			KETI_LOG_L2(fmt.Sprintf("- Num Of Images:%d", len(nodeInfo.ImageStates)))

			KETI_LOG_L2(fmt.Sprintf("- Total GPU Count:%d", nodeInfo.TotalGPUCount))
			for i, uuid := range nodeInfo.GPU_UUID {
				fmt.Println("# ", i, ":", uuid)
			}

			//KETI_LOG_L1("# Used Ports: [")
			//for port, _ := range nodeInfo.UsedPorts {
			//	KETI_LOG_L1(fmt.Sprintf("%s,", port))
			//	KETI_LOG_L1("]")
			//}

			KETI_LOG_L2("- NVLink List: ")
			for _, nvlink := range nodeInfo.NVLinkList {
				KETI_LOG_L3(fmt.Sprintf("[%s:%s:%d]", nvlink.GPU1, nvlink.GPU2, nvlink.Lane))
			}
		} else {
			KETI_LOG_L2(fmt.Sprintf(">> NodeName : %s -> filtered node", nodeName))
		}
	}
	KETI_LOG_L2("----------------------------------------")

	return nil
}

func (c *NodeCache) NodeCountDown() {
	c.AvailableNodeCount--
}

func (c *NodeCache) NodeCountUP() {
	c.AvailableNodeCount++
}

type PodState struct {
	Pod   *corev1.Pod
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

func (cache *NodeCache) RemovePodStates(pod *corev1.Pod) {
	ok, _ := cache.CheckPodStateExist(pod)
	if ok {
		key, err := GetPodKey(pod)
		if err != nil {
			return
		}
		delete(cache.PodStates, key)
	} else {
		return
	}
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

	if ok, _ := cache.CheckPodStateExist(&pod); ok {
		cache.UpdatePodState(&pod, s)
	} else {
		ps := &PodState{
			Pod:   &pod,
			State: s,
		}
		cache.PodStates[key] = ps
	}

	return nil
}

func (cache *NodeCache) UpdatePodState(pod *corev1.Pod, s string) error {
	key, err := GetPodKey(pod)
	if err != nil {
		return err
	}

	cache.PodStates[key].State = s
	cache.PodStates[key].Pod = pod
	return nil
}

func (cache *NodeCache) CheckPodStateExist(pod *corev1.Pod) (bool, string) {
	key, err := GetPodKey(pod)
	if err != nil {
		return false, ""
	}

	if podState, ok := cache.PodStates[key]; ok {
		return true, podState.State
	} else {
		return false, ""
	}
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

// func (cache *NodeCache) AssumePod(pod *corev1.Pod) error {
// 	cache.mu.Lock()
// 	defer cache.mu.Unlock()

// 	return cache.UpdatePodState(pod, Assumed)
// }

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
func (cache *NodeCache) addPod(pod *corev1.Pod) {
	n, ok := cache.NodeInfoList[pod.Spec.NodeName]
	if ok {
		n.AddPod(*pod)
	}
}

// Assumes that lock is already acquired.
func (cache *NodeCache) updatePod(oldPod, newPod *corev1.Pod) error {

	if err := cache.removePod(oldPod); err != nil {
		return err
	}

	cache.addPod(newPod)
	return nil
}

// Assumes that lock is already acquired.
// Removes a pod from the cached node info. If the node information was already
// removed and there are no more pods left in the node, cleans up the node from
// the cache.
func (cache *NodeCache) removePod(pod *corev1.Pod) error {
	n, ok := cache.NodeInfoList[pod.Spec.NodeName]
	if !ok {
		return fmt.Errorf("Node not found when trying to remove pod{%s}", pod.Name)
	} else {
		if err := n.RemovePod(pod); err != nil {
			return err
		}
		if len(n.Pods) == 0 && n.node == nil {
			cache.removeNodeInfoFromList(pod.Spec.NodeName)
		}
	}
	return nil
}

func (cache *NodeCache) AddPod(pod *corev1.Pod, state string) error {
	key, err := GetPodKey(pod)
	if err != nil {
		return err
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	_, ok := cache.PodStates[key]
	switch {
	case ok:
		if err := cache.UpdatePodState(pod, state); err != nil {
			return err
		} else {
			cache.addPod(pod)
		}
	case !ok:
		if err := cache.AddPodState(*pod, state); err != nil {
			return err
		} else {
			cache.addPod(pod)
		}
	}
	return nil
}

func (cache *NodeCache) UpdatePod(oldPod, newPod *corev1.Pod) error {
	key, err := GetPodKey(oldPod)
	if err != nil {
		return err
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	_, ok := cache.PodStates[key]
	switch {
	case ok:
		cache.UpdatePodState(newPod, BindingFinished)
		return cache.updatePod(oldPod, newPod)
	case !ok:
		if err := cache.AddPodState(*oldPod, BindingFinished); err != nil {
			return err
		} else {
			cache.updatePod(oldPod, newPod)
		}
	}
	return nil
}

func (cache *NodeCache) RemovePod(pod *corev1.Pod) error {
	key, err := GetPodKey(pod)
	if err != nil {
		return err
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()

	currState, ok := cache.PodStates[key]
	switch {
	case ok:
		cache.RemovePodStates(pod)
		return cache.removePod(currState.Pod)
	case !ok:
		return nil
	}

	// if currState.Pod.Spec.NodeName != pod.Spec.NodeName {
	// 	klog.ErrorS(nil, "Pod was added to a different node than it was assumed", "pod", klog.KObj(pod), "assumedNode", klog.KRef("", pod.Spec.NodeName), "currentNode", klog.KRef("", currState.Pod.Spec.NodeName))
	// 	if pod.Spec.NodeName != "" {
	// 		// An empty NodeName is possible when the scheduler misses a Delete
	// 		// event and it gets the last known state from the informer cache.
	// 		klog.ErrorS(nil, "scheduler cache is corrupted and can badly affect scheduling decisions")
	// 		os.Exit(1)
	// 	}
	// }

	return nil
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

func (cache *NodeCache) AddNode(node *corev1.Node) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	n, ok := cache.NodeInfoList[node.Name]
	if ok {
		cache.RemoveNode(n.Node())
	}
	n = NewNodeInfo()
	cache.NodeInfoList[node.Name] = n
	cache.TotalNodeCount++

	podsInNode, _ := cache.HostKubeClient.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + node.Name,
	})

	// pods, podswithaffinity, podswithrequirequiredantiaffinity
	// usedport, pvcrefcounts, requested
	for _, pod := range podsInNode.Items { //add pod in nodeinfocache
		cache.addPod(&pod)
		cache.AddPodState(pod, BindingFinished)
	}

	n.InitNodeInfo(node, cache.HostKubeClient)
	// n.Avaliable = false //gpu metric collector가 없다면 스케줄링 대상 노드 X

	cache.addNodeImageStates(node, n)
	n.SetNode(node)

	return nil
}

func (cache *NodeCache) UpdateNode(oldNode, newNode *corev1.Node) error {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	n, ok := cache.NodeInfoList[newNode.Name]
	if !ok {
		n = NewNodeInfo()
		cache.NodeInfoList[newNode.Name] = n
	} else {
		cache.removeNodeImageStates(n.Node())
	}

	cache.addNodeImageStates(newNode, n)
	n.SetNode(newNode)

	return nil
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

	for _, image := range node.Status.Images {
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
	nodeInfo.ImageStates = newSum
	// fmt.Println("newsum:", len(newSum)) //100개
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
