/*
Copyright 2017 The Kubernetes Authors.

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
	"fmt"
)

const (
	Ns            = int(10) //노드 스코어링 단계 수
	Gs            = int(10) //GPU 스코어링 단계 수
	SchedulerName = "gpu-scheduler"
	Policy1       = "node-gpu-score-weight"
	Policy2       = "pod-re-schedule-permit"
	Policy3       = "node-reservation-permit"
	Policy4       = "nvlink-weight-percentage"
	Policy5       = "gpu-allocate-prefer"
)

const (
	MaxScore int = 100
	MinScore int = 40
)

type ActionType int64

const (
	Add    ActionType = 1 << iota // 1
	Delete                        // 10
	// UpdateNodeXYZ is only applicable for Node events.
	UpdateNodeAllocatable // 100
	UpdateNodeLabel       // 1000
	UpdateNodeTaint       // 10000
	UpdateNodeCondition   // 100000

	All ActionType = 1<<iota - 1 // 111111

	// Use the general Update type if you don't either know or care the specific sub-Update type to use.
	Update = UpdateNodeAllocatable | UpdateNodeLabel | UpdateNodeTaint | UpdateNodeCondition
)

const (
	//--[UserPriority]--
	LowPriority    = 10
	MiddlePriority = 50
	HighPriority   = 100
	Immediatly     = 1500
)

// GVK is short for group/version/kind, which can uniquely represent a particular API resource.
type GVK string

// Constants for GVKs.
const (
	Pod                   GVK = "Pod"
	Node                  GVK = "Node"
	PersistentVolume      GVK = "PersistentVolume"
	PersistentVolumeClaim GVK = "PersistentVolumeClaim"
	Service               GVK = "Service"
	StorageClass          GVK = "storage.k8s.io/StorageClass"
	CSINode               GVK = "storage.k8s.io/CSINode"
	CSIDriver             GVK = "storage.k8s.io/CSIDriver"
	CSIStorageCapacity    GVK = "storage.k8s.io/CSIStorageCapacity"
	WildCard              GVK = "*"
)

const GPU_SCHEDUER_DEBUGG_LEVEL = 1

func KETI_LOG_L1(log string) { //자세한 출력, DumpClusterInfo DumpNodeInfo
	if GPU_SCHEDUER_DEBUGG_LEVEL < 2 { //LEVEL = 1,2,3
		fmt.Println(log)
	}
}

func KETI_LOG_L2(log string) { // 기본출력
	if GPU_SCHEDUER_DEBUGG_LEVEL < 3 { //LEVEL = 2,3
		fmt.Println(log)
	}
}

func KETI_LOG_L3(log string) { //필수출력, 정량용, 에러
	if GPU_SCHEDUER_DEBUGG_LEVEL < 4 { //LEVEL = 3
		fmt.Println(log)
	}
}
