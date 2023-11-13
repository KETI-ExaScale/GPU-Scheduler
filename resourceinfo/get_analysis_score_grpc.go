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
		nodeInfo.PluginResult.NodeScore = int(res.Scores[nodeName].NodeScore)
		for gpuName, gpuScore := range nodeInfo.PluginResult.GPUScores {
			gpuScore.GPUScore = int(res.Scores[nodeName].GpuScores[gpuName].GpuScore)
		}
	}

	return nil
}
