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
	N             = float64(4)
	G             = float64(3)
	SchedulerName = "gpu-scheduler"
	Policy1       = "node-gpu-score-weight"
	Policy2       = "pod-re-schedule-permit"
)

var (
	NodeWeight float64
	GPUWeight  float64
	ReSchedule string
	LeastPod   = true
)

var (
	GPUMemoryTotalMost = int64(0)
)

//Debugging Print
var (
	Debugg    = true
	Score     = true
	Metric    = true
	Re        = false
	Filtering = false
	Scoring   = false
	Policy    = true
)

// //influx 사용 X
// var (
// 	ip   = "influxdb.gpu.svc.cluster.local"
// 	port = "8086"
// 	URL  = "http://" + ip + ":" + port
// )
