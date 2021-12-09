package resourceinfo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gpu-scheduler/config"
	pb "gpu-scheduler/proto"

	"google.golang.org/grpc"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

//Get Node/GPU Metrics by gRPC

func GetNodeMetric(ip string) (*NodeMetric, error) {
	host := ip + ":9000"
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	defer conn.Close()
	if err != nil {
		return nil, err
	}
	grpcClient := pb.NewUserClient(conn)
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	r, err := grpcClient.GetNode(ctx, &pb.GetNodeRequest{})
	if err != nil {
		cancel()
		return nil, err
	}
	result := r.GetNodeMessage()
	cancel()

	totalGPUCount := result.GpuCount
	milliCPUTotal := result.NodeTotalcpu
	milliCPUUsed := result.NodeCpu
	memoryTotal := result.NodeTotalmemory
	memoryUsed := result.NodeMemory
	storageTotal := result.NodeTotalstorage
	storageUsed := result.NodeStorage
	uuids := stringToArray(result.GpuUuid)
	maxGPUMemory := result.MaxGpuMemory

	if config.GPUMemoryTotalMost < maxGPUMemory {
		config.GPUMemoryTotalMost = maxGPUMemory
	}

	newNodeMetric := &NodeMetric{
		MilliCPUTotal: milliCPUTotal,
		MilliCPUUsed:  milliCPUUsed,
		MemoryTotal:   memoryTotal,
		MemoryUsed:    memoryUsed,
		StorageTotal:  storageTotal,
		StorageUsed:   storageUsed,
		TotalGPUCount: totalGPUCount,
		GPU_UUID:      uuids,
		MaxGPUMemory:  maxGPUMemory,
	}

	if config.Metric {
		fmt.Println(" |NodeMetric|", newNodeMetric)
	}

	return newNodeMetric, nil
}

//'[abc abc]' : string -> ['abc' 'abc'] : []string
func stringToArray(str string) []string {
	str = strings.Trim(str, "[]")
	return strings.Split(str, " ")
}

func GetGPUMetrics(uuids []string, ip string) ([]*GPUMetric, error) {
	var gpuMetrics []*GPUMetric

	for _, uuid := range uuids {
		host := ip + ":9000"
		conn, err := grpc.Dial(host, grpc.WithInsecure())
		if err != nil {
			return nil, err
		}
		defer conn.Close()
		grpcClient := pb.NewUserClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		p, err := grpcClient.GetGPU(ctx, &pb.GetGPURequest{GpuUuid: uuid})
		if err != nil {
			cancel()
			return nil, err
		}
		result := p.GetGpuMessage()
		cancel()

		gpuName := result.GpuName
		gpuIndex := result.GpuIndex
		gpuPower := result.GpuPower
		gpuMemoryTotal := int64(result.GpuTotal)
		gpuMemoryFree := int64(result.GpuFree)
		gpuMemoryUsed := int64(result.GpuUsed)
		gpuTemperature := result.GpuTemp
		podCount := result.MpsCount

		newGPUMetric := &GPUMetric{
			GPUName:        gpuName,
			UUID:           uuid,
			GPUIndex:       gpuIndex,
			GPUPower:       gpuPower,
			GPUMemoryTotal: gpuMemoryTotal,
			GPUMemoryFree:  gpuMemoryFree,
			GPUMemoryUsed:  gpuMemoryUsed,
			GPUTemperature: gpuTemperature,
			IsFiltered:     false,
			GPUScore:       0,
			PodCount:       podCount,
		}
		gpuMetrics = append(gpuMetrics, newGPUMetric)

		if config.Metric {
			fmt.Println(" |GPUMetric |", newGPUMetric)
		}
	}

	return gpuMetrics, nil
}
