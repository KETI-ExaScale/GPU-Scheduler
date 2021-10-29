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
