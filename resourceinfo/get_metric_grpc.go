package resourceinfo

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"gpu-scheduler/config"
	pb "gpu-scheduler/proto"

	"google.golang.org/grpc"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

//Get Node/GPU Metrics by gRPC

func GetNodeMetric(nodeName string, ip string) *NodeMetric {
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

	if config.Debugg {
		fmt.Println(" |NodeMetric|", totalGPUCount, nodeCPU, nodeMemory, uuids)
	}

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

func GetGPUMetrics(uuids []string, ip string) []*GPUMetric {
	var tempGPUMetrics []*GPUMetric

	for _, uuid := range uuids {
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
		tempGPUMetrics = append(tempGPUMetrics, newGPUMetric)

		if config.Debugg {
			fmt.Println(" |GPUMetric|", newGPUMetric)
		}
	}

	return tempGPUMetrics
}
