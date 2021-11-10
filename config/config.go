package config

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// //influx 사용 X
// var (
// 	ip   = "influxdb.gpu.svc.cluster.local"
// 	port = "8086"
// 	URL  = "http://" + ip + ":" + port
// )

var Host_config, _ = rest.InClusterConfig()
var Host_kubeClient = kubernetes.NewForConfigOrDie(Host_config)

//Debugging Print
var (
	N         = float64(4)
	Debugg    = true
	Score     = false
	Metric    = false
	Re        = false
	Filtering = true
	Scoring   = true
)

const SchedulerName = "gpu-scheduler"
