package resourceinfo

import (
	"fmt"
	"strconv"

	_ "github.com/influxdata/influxdb1-client" // this is important because of the bug in go mod
	client "github.com/influxdata/influxdb1-client/v2"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func GetGPUMetrics(c client.Client, uuids []string) []*GPUMetric {
	var tempGPUMetrics []*GPUMetric

	for _, uuid := range uuids {
		q := client.Query{
			Command:  fmt.Sprintf("SELECT last(*) FROM gpumetric where UUID='%s'", uuid),
			Database: "metric",
		}
		response, err := c.Query(q)
		if err != nil || response.Error() != nil {
			fmt.Println("InfluxDB error: ", err)
			return nil
		}
		myGPUMetric := response.Results[0].Series[0].Values[0]

		gpuName := fmt.Sprintf("%s", myGPUMetric[1])
		mpsIndex, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[2]))
		gpuPower, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[3]))
		gpuMemoryTotal, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[5]))
		gpuMemoryFree, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[4]))
		gpuMemoryUsed, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[6]))
		gpuTemperature, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[7]))

		newGPUMetric := &GPUMetric{
			GPUName:        gpuName,
			UUID:           uuid,
			MPSIndex:       mpsIndex,
			GPUPower:       gpuPower,
			GPUMemoryTotal: gpuMemoryTotal,
			GPUMemoryFree:  gpuMemoryFree,
			GPUMemoryUsed:  gpuMemoryUsed,
			GPUTemperature: gpuTemperature,
			IsFiltered:     false,
			GPUScore:       0,
		}

		fmt.Println(" |GPUMetric|", newGPUMetric)
		tempGPUMetrics = append(tempGPUMetrics, newGPUMetric)
	}

	return tempGPUMetrics
}
