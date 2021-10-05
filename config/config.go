package config

var (
	ip   = "influxdb.gpu.svc.cluster.local"
	port = "8086"
	URL  = "http://" + ip + ":" + port
)

const SchedulerName = "gpu-scheduler"
