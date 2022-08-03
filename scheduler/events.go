/*
Copyright 2019 The Kubernetes Authors.

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

package scheduler

import r "gpu-scheduler/resourceinfo"

const (
	// PodAdd is the event when a new pod is added to API server.
	PodAdd = "PodAdd"
	// ScheduleAttemptFailure is the event when a schedule attempt fails.
	ScheduleAttemptFailure = "ScheduleAttemptFailure"
	// BackoffComplete is the event when a pod finishes backoff.
	BackoffComplete = "BackoffComplete"
	// ForceActivate is the event when a pod is moved from unschedulablePods/backoffQ
	// to activeQ. Usually it's triggered by plugin implementations.
	ForceActivate = "ForceActivate"
)

var (
	// AssignedPodAdd is the event when a pod is added that causes pods with matching affinity terms
	// to be more schedulable.
	AssignedPodAdd = r.ClusterEvent{Resource: r.Pod, ActionType: r.Add, Label: "AssignedPodAdd"}
	// NodeAdd is the event when a new node is added to the cluster.
	NodeAdd = r.ClusterEvent{Resource: r.Node, ActionType: r.Add, Label: "NodeAdd"}
	// AssignedPodUpdate is the event when a pod is updated that causes pods with matching affinity
	// terms to be more schedulable.
	AssignedPodUpdate = r.ClusterEvent{Resource: r.Pod, ActionType: r.Update, Label: "AssignedPodUpdate"}
	// AssignedPodDelete is the event when a pod is deleted that causes pods with matching affinity
	// terms to be more schedulable.
	AssignedPodDelete = r.ClusterEvent{Resource: r.Pod, ActionType: r.Delete, Label: "AssignedPodDelete"}
	// NodeSpecUnschedulableChange is the event when unschedulable node spec is changed.
	NodeSpecUnschedulableChange = r.ClusterEvent{Resource: r.Node, ActionType: r.UpdateNodeTaint, Label: "NodeSpecUnschedulableChange"}
	// NodeAllocatableChange is the event when node allocatable is changed.
	NodeAllocatableChange = r.ClusterEvent{Resource: r.Node, ActionType: r.UpdateNodeAllocatable, Label: "NodeAllocatableChange"}
	// NodeLabelChange is the event when node label is changed.
	NodeLabelChange = r.ClusterEvent{Resource: r.Node, ActionType: r.UpdateNodeLabel, Label: "NodeLabelChange"}
	// NodeTaintChange is the event when node taint is changed.
	NodeTaintChange = r.ClusterEvent{Resource: r.Node, ActionType: r.UpdateNodeTaint, Label: "NodeTaintChange"}
	// NodeConditionChange is the event when node condition is changed.
	NodeConditionChange = r.ClusterEvent{Resource: r.Node, ActionType: r.UpdateNodeCondition, Label: "NodeConditionChange"}
	// PvAdd is the event when a persistent volume is added in the cluster.
	PvAdd = r.ClusterEvent{Resource: r.PersistentVolume, ActionType: r.Add, Label: "PvAdd"}
	// PvUpdate is the event when a persistent volume is updated in the cluster.
	PvUpdate = r.ClusterEvent{Resource: r.PersistentVolume, ActionType: r.Update, Label: "PvUpdate"}
	// PvcAdd is the event when a persistent volume claim is added in the cluster.
	PvcAdd = r.ClusterEvent{Resource: r.PersistentVolumeClaim, ActionType: r.Add, Label: "PvcAdd"}
	// PvcUpdate is the event when a persistent volume claim is updated in the cluster.
	PvcUpdate = r.ClusterEvent{Resource: r.PersistentVolumeClaim, ActionType: r.Update, Label: "PvcUpdate"}
	// StorageClassAdd is the event when a StorageClass is added in the cluster.
	StorageClassAdd = r.ClusterEvent{Resource: r.StorageClass, ActionType: r.Add, Label: "StorageClassAdd"}
	// StorageClassUpdate is the event when a StorageClass is updated in the cluster.
	StorageClassUpdate = r.ClusterEvent{Resource: r.StorageClass, ActionType: r.Update, Label: "StorageClassUpdate"}
	// CSINodeAdd is the event when a CSI node is added in the cluster.
	CSINodeAdd = r.ClusterEvent{Resource: r.CSINode, ActionType: r.Add, Label: "CSINodeAdd"}
	// CSINodeUpdate is the event when a CSI node is updated in the cluster.
	CSINodeUpdate = r.ClusterEvent{Resource: r.CSINode, ActionType: r.Update, Label: "CSINodeUpdate"}
	// CSIDriverAdd is the event when a CSI driver is added in the cluster.
	CSIDriverAdd = r.ClusterEvent{Resource: r.CSIDriver, ActionType: r.Add, Label: "CSIDriverAdd"}
	// CSIDriverUpdate is the event when a CSI driver is updated in the cluster.
	CSIDriverUpdate = r.ClusterEvent{Resource: r.CSIDriver, ActionType: r.Update, Label: "CSIDriverUpdate"}
	// CSIStorageCapacityAdd is the event when a CSI storage capacity is added in the cluster.
	CSIStorageCapacityAdd = r.ClusterEvent{Resource: r.CSIStorageCapacity, ActionType: r.Add, Label: "CSIStorageCapacityAdd"}
	// CSIStorageCapacityUpdate is the event when a CSI storage capacity is updated in the cluster.
	CSIStorageCapacityUpdate = r.ClusterEvent{Resource: r.CSIStorageCapacity, ActionType: r.Update, Label: "CSIStorageCapacityUpdate"}
	// ServiceAdd is the event when a service is added in the cluster.
	ServiceAdd = r.ClusterEvent{Resource: r.Service, ActionType: r.Add, Label: "ServiceAdd"}
	// ServiceUpdate is the event when a service is updated in the cluster.
	ServiceUpdate = r.ClusterEvent{Resource: r.Service, ActionType: r.Update, Label: "ServiceUpdate"}
	// ServiceDelete is the event when a service is deleted in the cluster.
	ServiceDelete = r.ClusterEvent{Resource: r.Service, ActionType: r.Delete, Label: "ServiceDelete"}
	// WildCardEvent semantically matches all resources on all actions.
	WildCardEvent = r.ClusterEvent{Resource: r.WildCard, ActionType: r.All, Label: "WildCardEvent"}
	// UnschedulableTimeout is the event when a pod stays in unschedulable for longer than timeout.
	UnschedulableTimeout = r.ClusterEvent{Resource: r.WildCard, ActionType: r.All, Label: "UnschedulableTimeout"}
)
