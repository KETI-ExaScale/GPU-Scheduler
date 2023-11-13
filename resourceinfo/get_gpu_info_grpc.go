package resourceinfo

import (
	"context"
	"time"

	pb "gpu-scheduler/proto/api"

	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func (ni *NodeInfo) GetGPUInitInfo(node *corev1.Node) error {
	internalIP := ""
	for _, address := range node.Status.Addresses {
		if address.Type == "InternalIP" {
			internalIP = address.Address
			break
		}
	}

	host := internalIP + ":9321"
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()
	grpcClient := pb.NewTravelerClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	res, err := grpcClient.NodeGPUInfo(ctx, &pb.Request{})
	if err != nil {
		cancel()
		return err
	}

	cancel()

	for _, nvlink := range res.NvlinkInfo {
		nvl := NewNVLink(nvlink.Gpu1Uuid, nvlink.Gpu2Uuid, nvlink.Lanecount)
		ni.NVLinkList = append(ni.NVLinkList, &nvl)
	}

	for uuid := range res.IndexUuidMap {
		ni.GPU_UUID = append(ni.GPU_UUID, uuid)
	}

	ni.TotalGPUCount = int64(res.TotalGpuCount)

	for _, uuid := range ni.GPU_UUID {
		ni.PluginResult.GPUScores[uuid] = NewGPUScore(uuid)
		ni.PluginResult.GPUCountUp()
	}

	return nil
}
