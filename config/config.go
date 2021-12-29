package config

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

//Kube-client
var (
	Host_config, _  = rest.InClusterConfig()
	Host_kubeClient = kubernetes.NewForConfigOrDie(Host_config)
)

//const variable
const (
	N             = float64(4) //노드 스코어링 단계 수
	G             = float64(3) //GPU 스코어링 단계 수
	SchedulerName = "gpu-scheduler"
	Policy1       = "node-gpu-score-weight"
	Policy2       = "pod-re-schedule-permit"
)

// gpu-scheduler policy
var (
	NodeWeight float64
	GPUWeight  float64
	ReSchedule bool
	Temp       bool
	LeastPod   = true
)

// max gpu memory in cluster for scoring
var (
	GPUMemoryTotalMost = int64(0)
)

//debug print
var (
	Debugg    = false
	Score     = false
	Metric    = false
	Re        = false
	Filtering = false
	Scoring   = false
	Policy    = true
)
