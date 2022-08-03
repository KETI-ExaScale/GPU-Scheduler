package resourceinfo

import (
	"context"
	"strings"
	"time"

	pb "gpu-scheduler/proto"

	"google.golang.org/grpc"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

//Get Node/GPU Metrics by gRPC

func (nm *NodeMetric) GetNodeMetric(ip string) error {
	host := ip + ":9000"
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	defer conn.Close()
	if err != nil {
		return err
	}
	grpcClient := pb.NewUserClient(conn)
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	r, err := grpcClient.GetNode(ctx, &pb.GetNodeRequest{})
	if err != nil {
		cancel()
		return err
	}
	result := r.GetNodeMessage()
	cancel()

	nm.TotalGPUCount = result.GpuCount
	nm.MilliCPUTotal = result.NodeTotalcpu
	nm.MilliCPUUsed = result.NodeCpu
	nm.MemoryTotal = result.NodeTotalmemory
	nm.MemoryUsed = result.NodeMemory
	nm.StorageTotal = result.NodeTotalstorage
	nm.StorageUsed = result.NodeStorage
	nm.GPU_UUID = stringToArray(result.GpuUuid)
	nm.MaxGPUMemory = result.MaxGpuMemory

	// 	fmt.Println(" |NodeMetric|", nm)

	return nil
}

//'[abc abc]' : string -> ['abc' 'abc'] : []string
func stringToArray(str string) []string {
	str = strings.Trim(str, "[]")
	return strings.Split(str, " ")
}

func (gm *GPUMetric) GetGPUMetric(uuid string, ip string) error {
	host := ip + ":9000"
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()
	grpcClient := pb.NewUserClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	p, err := grpcClient.GetGPU(ctx, &pb.GetGPURequest{GpuUuid: uuid})
	if err != nil {
		cancel()
		return err
	}
	result := p.GetGpuMessage()
	cancel()

	gm.GPUName = result.GpuName
	gm.GPUIndex = result.GpuIndex
	gm.GPUPowerUsed = result.GpuPower
	gm.GPUPowerTotal = result.GpuTpower
	gm.GPUMemoryTotal = int64(result.GpuTotal)
	gm.GPUMemoryFree = int64(result.GpuFree)
	gm.GPUMemoryUsed = int64(result.GpuUsed)
	gm.GPUTemperature = result.GpuTemp
	gm.PodCount = result.MpsCount
	gm.GPUFlops = result.GpuFlops
	gm.GPUArch = result.GpuArch
	gm.GPUUtil = result.GpuUtil

	return nil
}
