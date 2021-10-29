package resourceinfo

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/influxdata/influxdb1-client" // this is important because of the bug in go mod
	client "github.com/influxdata/influxdb1-client/v2"

	pb "gpu-scheduler/proto"

	"google.golang.org/grpc"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func GetNodeMetric(c client.Client, nodeName string, ip string) *NodeMetric {
	// q := client.Query{
	// 	Command:  fmt.Sprintf("SELECT last(*) FROM multimetric where NodeName='%s'", nodeName),
	// 	Database: "metric",
	// }
	// response, err := c.Query(q)
	// if err != nil || response.Error() != nil {
	// 	fmt.Println("InfluxDB error: ", err)
	// 	return nil
	// }
	// myNodeMetric := response.Results[0].Series[0].Values[0]

	// totalGPUCount, _ := strconv.Atoi(fmt.Sprintf("%s", myNodeMetric[1]))
	// nodeCPU := fmt.Sprintf("%s", myNodeMetric[2])
	// nodeMemory := fmt.Sprintf("%s", myNodeMetric[3])
	// uuids := stringToArray(myNodeMetric[5].(string))

	// fmt.Println(" |NodeMetric|", totalGPUCount, nodeCPU, nodeMemory, uuids)

	//gRPC test
	host := ip + ":9000"
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		fmt.Println("gRPC Error!!!: ", err)
	}
	defer conn.Close()
	grpcClient := pb.NewUserClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	r, err := grpcClient.GetNode(ctx, &pb.GetNodeRequest{})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	result := r.GetNodeMessage()
	cancel()

	totalGPUCount := result.GpuCount
	nodeCPU := result.NodeCpu
	nodeMemory := result.NodeMemory
	uuids := stringToArray(result.GpuUuid)

	fmt.Println(" |NodeMetric|", totalGPUCount, nodeCPU, nodeMemory, uuids)

	return &NodeMetric{
		NodeCPU:       nodeCPU,
		NodeMemory:    nodeMemory,
		TotalGPUCount: totalGPUCount,
		GPU_UUID:      uuids,
	}
}

//'[abc abc]' : string -> ['abc' 'abc'] : []string
func stringToArray(str string) []string {
	str = strings.Trim(str, "[]")
	return strings.Split(str, " ")
}

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
