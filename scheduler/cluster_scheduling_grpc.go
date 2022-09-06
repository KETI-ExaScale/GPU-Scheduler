package scheduler

import (
	"context"
	"fmt"
	"time"

	pb "gpu-scheduler/proto"

	"google.golang.org/grpc"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

const portNumber = "8686"

type InitStruct struct {
	NodeName string
	Score    int64
	GPUCount int64
}

func InitMyClusterManager(ip string, infoList []InitStruct) (bool, error) {
	fmt.Println("#Init my cluster manager called")
	host := ip + ":" + portNumber
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		fmt.Println("<error> update node score1 - ", err)
		return false, err
	}
	defer conn.Close()
	grpcClient := pb.NewClusterClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)

	var requestMessageList []*pb.RequestMessage
	for _, info := range infoList {
		fmt.Println("#Init Info (", info.NodeName, ",", info.Score, ",", info.GPUCount, ")")
		var requestMessage = &pb.RequestMessage{
			NodeName:  info.NodeName,
			NodeScore: info.Score,
			GpuCount:  info.GPUCount,
		}
		requestMessageList = append(requestMessageList, requestMessage)
	}

	var initRequestMessage = &pb.InitMyClusterRequest{
		RequestMessage: requestMessageList,
	}
	p, err := grpcClient.InitMyCluster(ctx, initRequestMessage)
	if err != nil {
		cancel()
		fmt.Println("<error> update node score2 - ", err)
		return false, err
	}

	success := p.Success

	cancel()

	return success, nil
}

func GetBestCluster(ip string, gpu int, filtercluster []string) (string, bool, error) {
	fmt.Println("#Get Best Cluster Called")
	host := ip + ":" + portNumber
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		fmt.Println("<error> get best cluster1 - ", err)
		return "", false, err
	}
	defer conn.Close()
	grpcClient := pb.NewClusterClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)

	var requestMessage = &pb.ClusterSchedulingRequest{
		GpuCount:        int64(gpu),
		FilteredCluster: filtercluster,
	}
	p, err := grpcClient.RequestClusterScheduling(ctx, requestMessage)
	if err != nil {
		cancel()
		fmt.Println("<error> get best cluster2 - ", err)
		return "", false, err
	}

	targetCluster := p.ClusterName
	success := p.Success

	cancel()

	return targetCluster, success, nil
}

func UpdateNodeScore(ip string, node string, score int) (bool, error) {
	fmt.Println("update node score: ", node, "-", score)
	host := ip + ":" + portNumber
	conn, err := grpc.Dial(host, grpc.WithInsecure())
	if err != nil {
		fmt.Println("<error> update node score1 - ", err)
		return false, err
	}
	defer conn.Close()
	grpcClient := pb.NewClusterClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)

	var requestMessage = &pb.RequestMessage{
		NodeName:  node,
		NodeScore: int64(score),
	}
	var updateClusterMessage = &pb.UpdateMyClusterRequest{
		RequestMessage: requestMessage,
	}
	p, err := grpcClient.UpdateMyCluster(ctx, updateClusterMessage)
	if err != nil {
		cancel()
		fmt.Println("<error> update node score2 - ", err)
		return false, err
	}

	success := p.Success

	cancel()

	return success, nil
}
