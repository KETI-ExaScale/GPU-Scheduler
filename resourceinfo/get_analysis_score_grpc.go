package resourceinfo

import (
	"context"
	"time"

	pb "gpu-scheduler/proto/score"

	"google.golang.org/grpc"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func (nc *NodeCache) GetAnalysisScore(ip string) error {
	host := ip + ":9322"
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()
	grpcClient := pb.NewMetricGRPCClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	res, err := grpcClient.GetScore(ctx, &pb.Request{})
	if err != nil {
		cancel()
		return err
	}

	cancel()

	for nodeName, nodeInfo := range nc.NodeInfoList {
		if resNodeScore, nodeExist := res.Scores[nodeName]; nodeExist {
			nodeInfo.PluginResult.NodeScore = int(resNodeScore.NodeScore)
			for gpuName, gpuScore := range nodeInfo.PluginResult.GPUScores {
				if resGPUScore, gpuExist := resNodeScore.GpuScores[gpuName]; gpuExist {
					gpuScore.GPUScore = int(resGPUScore.GpuScore)
				} else {
					gpuScore.IsFiltered = true
				}
			}
		} else {
			nodeInfo.PluginResult.IsFiltered = true
		}

	}

	return nil
}
