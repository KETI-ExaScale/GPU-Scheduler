package config

//influx 사용 X
var (
	ip   = "influxdb.gpu.svc.cluster.local"
	port = "8086"
	URL  = "http://" + ip + ":" + port
)

const SchedulerName = "gpu-scheduler"
