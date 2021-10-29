package resourceinfo

import (
	"context"
	"fmt"
	"log"
	"time"

	_ "github.com/influxdata/influxdb1-client" // this is important because of the bug in go mod
	client "github.com/influxdata/influxdb1-client/v2"

	pb "gpu-scheduler/proto"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"

	"google.golang.org/grpc"
)

func GetGPUMetrics(c client.Client, uuids []string, ip string) []*GPUMetric {
	var tempGPUMetrics []*GPUMetric

	for _, uuid := range uuids {
		// q := client.Query{
		// 	Command:  fmt.Sprintf("SELECT last(*) FROM gpumetric where UUID='%s'", uuid),
		// 	Database: "metric",
		// }
		// response, err := c.Query(q)
		// if err != nil || response.Error() != nil {
		// 	fmt.Println("InfluxDB error: ", err)
		// 	return nil
		// }
		// myGPUMetric := response.Results[0].Series[0].Values[0]

		// gpuName := fmt.Sprintf("%s", myGPUMetric[1])
		// mpsIndex, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[2]))
		// gpuPower, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[3]))
		// gpuMemoryTotal, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[5]))
		// gpuMemoryFree, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[4]))
		// gpuMemoryUsed, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[6]))
		// gpuTemperature, _ := strconv.Atoi(fmt.Sprintf("%s", myGPUMetric[7]))

		host := ip + ":9000"
		conn, err := grpc.Dial(host, grpc.WithInsecure())
		if err != nil {
			fmt.Println("gRPC Error!!!: ", err)
		}
		defer conn.Close()
		grpcClient := pb.NewUserClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		p, err := grpcClient.GetGPU(ctx, &pb.GetGPURequest{GpuUuid: uuid})
		if err != nil {
			log.Fatalf("not gpu greet: %v", err)
		}
		result := p.GetGpuMessage()
		cancel()

		gpuName := result.GpuName
		mpsIndex := result.GpuIndex
		gpuPower := result.GpuPower
		gpuMemoryTotal := int64(result.GpuTotal)
		gpuMemoryFree := int64(result.GpuFree)
		gpuMemoryUsed := int64(result.GpuUsed)
		gpuTemperature := result.GpuTemp

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
