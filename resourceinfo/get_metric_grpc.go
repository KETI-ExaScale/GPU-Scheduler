package resourceinfo

import (
	"context"
	"fmt"
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
		fmt.Println("<error> get node metric1 - ", err)
		return err
	}
	grpcClient := pb.NewUserClient(conn)
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	r, err := grpcClient.GetNode(ctx, &pb.GetNodeRequest{})
	if err != nil {
		cancel()
		fmt.Println("<error> get node metric2 - ", err)
		return err
	}
	result := r.GetNodeMessage()
	cancel()

	// nm.TotalGPUCount = result.GpuCount
	// nm.MilliCPUTotal = result.NodeTotalcpu
	nm.MilliCPUUsed = result.NodeCpu
	// nm.MemoryTotal = result.NodeTotalmemory
	nm.MemoryUsed = result.NodeMemory
	// nm.StorageTotal = result.NodeTotalstorage
	nm.StorageUsed = result.NodeStorage
	// nm.GPU_UUID = stringToArray(result.GpuUuid)
	// nm.MaxGPUMemory = result.MaxGpuMemory

	// Gpu1UuidArr1 := result.Gpu1Uuid
	// Gpu2UuidArr2 := result.Gpu2Uuid
	// LinkcountArr3 := result.Linkcount

	// for i, value := range Gpu1UuidArr1 {
	// 	nvl := NewNVLink(value, Gpu2UuidArr2[i], LinkcountArr3[i])
	// 	nm.NVLinkList = append(nm.NVLinkList, nvl)
	// }

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
		fmt.Println("<error> get gpu metric1 - ", err)
		return err
	}
	defer conn.Close()
	grpcClient := pb.NewUserClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	p, err := grpcClient.GetGPU(ctx, &pb.GetGPURequest{GpuUuid: uuid})
	if err != nil {
		cancel()
		fmt.Println("<error> get gpu metric2 - ", err)
		return err
	}
	result := p.GetGpuMessage()
	cancel()

	// gm.GPUName = result.GpuName
	// gm.GPUIndex = result.GpuIndex
	gm.GPUPowerUsed = result.GpuPower
	// gm.GPUPowerTotal = result.GpuTpower
	// gm.GPUMemoryTotal = int64(result.GpuTotal)
	gm.GPUMemoryFree = int64(result.GpuFree)
	gm.GPUMemoryUsed = int64(result.GpuUsed)
	gm.GPUTemperature = result.GpuTemp
	gm.PodCount = result.MpsCount
	// gm.GPUFlops = result.GpuFlops
	// gm.GPUArch = result.GpuArch
	// gm.GPUUtil = result.GpuUtil

	return nil
}

func (ni *NodeInfo) GetInitMetric(ip string) error {
	fmt.Println("get init metric")
	host := ip + ":9000"
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()
	grpcClient := pb.NewUserClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	i, err := grpcClient.GetInitData(ctx, &pb.InitRequest{})
	if err != nil {
		cancel()
		fmt.Println("<error> get init metric - ", err)
		return err
	}

	inode := i.GetInitNode()
	igpu := i.GetInitGPU()
	cancel()

	ni.NodeMetric.TotalGPUCount = inode.GpuCount
	ni.NodeMetric.MilliCPUTotal = inode.NodeTotalcpu
	ni.NodeMetric.MilliCPUUsed = inode.NodeCpu
	ni.NodeMetric.MemoryTotal = inode.NodeTotalmemory
	ni.NodeMetric.MemoryUsed = inode.NodeMemory
	ni.NodeMetric.StorageTotal = inode.NodeTotalstorage
	ni.NodeMetric.StorageUsed = inode.NodeStorage
	ni.NodeMetric.GPU_UUID = stringToArray(inode.GpuUuid)
	ni.NodeMetric.MaxGPUMemory = inode.MaxGpuMemory

	Gpu1UuidArr1 := inode.Gpu1Index
	Gpu2UuidArr2 := inode.Gpu2Index
	LinkcountArr3 := inode.Lanecount

	for i, value := range Gpu1UuidArr1 {
		nvl := NewNVLink(value, Gpu2UuidArr2[i], LinkcountArr3[i])
		ni.NodeMetric.NVLinkList = append(ni.NodeMetric.NVLinkList, nvl)
	}

	fmt.Println("+++", ni.NodeMetric.GPU_UUID)

	for i, uuid := range ni.NodeMetric.GPU_UUID {
		fmt.Println("uuid: ", uuid)
		gm := NewGPUMetric()

		gm.GPUName = igpu[i].GpuName
		gm.GPUIndex = igpu[i].GpuIndex
		gm.GPUPowerUsed = igpu[i].GpuPower
		gm.GPUPowerTotal = igpu[i].GpuTpower
		gm.GPUMemoryTotal = int64(igpu[i].GpuTotal)
		gm.GPUMemoryFree = int64(igpu[i].GpuFree)
		gm.GPUMemoryUsed = int64(igpu[i].GpuUsed)
		gm.GPUTemperature = igpu[i].GpuTemp
		gm.PodCount = igpu[i].MpsCount
		gm.GPUFlops = igpu[i].GpuFlops
		gm.GPUArch = igpu[i].GpuArch
		gm.GPUUtil = igpu[i].GpuUtil

		fmt.Print("\\\\", gm.GPUName)

		ni.GPUMetrics[uuid] = gm
		ni.PluginResult.GPUScores[uuid] = NewGPUScore(uuid)
		ni.PluginResult.GPUCountUp()
	}

	return nil
}
