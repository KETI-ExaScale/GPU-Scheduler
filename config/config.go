package config

import (
	"os/exec"
	"strconv"
	"strings"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

//Kube-client
var (
	Host_config, _  = rest.InClusterConfig()
	Host_kubeClient = kubernetes.NewForConfigOrDie(Host_config)
)

//Policy
var (
	weightPolicy, _     = exec.Command("cat", "/tmp/node-gpu-score-weight").Output()
	NodeWeight, _       = strconv.ParseFloat(strings.Split(string(weightPolicy), " ")[0], 64)
	GPUWeight, _        = strconv.ParseFloat(strings.Split(string(weightPolicy), " ")[1], 64)
	reSchedulePolicy, _ = exec.Command("cat", "/tmp/pod-re-schedule-permit").Output()
	ReSchedule          = "reSchedule :" + string(reSchedulePolicy)
	LeastPod            = true
)

//const variable
const (
	N             = float64(4)
	G             = float64(3)
	SchedulerName = "gpu-scheduler"
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
	Weight    = true
)

// //influx 사용 X
// var (
// 	ip   = "influxdb.gpu.svc.cluster.local"
// 	port = "8086"
// 	URL  = "http://" + ip + ":" + port
// )
