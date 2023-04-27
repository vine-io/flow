package main

import (
	"context"
	"flag"
	"fmt"
	"reflect"

	"github.com/vine-io/flow"
	pb "github.com/vine-io/flow/examples/pb"
	log "github.com/vine-io/vine/lib/logger"
)

var (
	address = flag.String("address", "127.0.0.1:43300", "Set the address of flow service")
	name    = flag.String("name", "flow", "Set the name of flow service")
	id      = flag.String("id", "1", "Set id for flow client")
)

var _ pb.HelloFlowHandler = (*ClientEcho)(nil)

type ClientEcho struct {
}

func (c *ClientEcho) Echo(ctx context.Context, request *pb.EchoRequest, response *pb.EchoResponse) error {
	response.Reply = request.Echo
	return fmt.Errorf("test error")
}

func (c *ClientEcho) Ping(ctx context.Context, request *pb.PingRequest, response *pb.PingResponse) error {
	response.Out = "pong"
	return nil
}

var _ flow.Step = (*ClientStep)(nil)

type ClientStep struct {
	Echo     *pb.Echo `flow:"entity"`
	EchoArgs *pb.Echo `flow:"args:echo"`
}

func (c *ClientStep) Owner() reflect.Type {
	return reflect.TypeOf(new(pb.Echo))
}

func (c *ClientStep) Prepare(ctx *flow.PipeSessionCtx) error {
	log.Infof("entity echo = %v", c.Echo)
	log.Infof("args echo = %v", c.EchoArgs)
	return nil
}

func (c *ClientStep) Commit(ctx *flow.PipeSessionCtx) error {
	return nil
}

func (c *ClientStep) Rollback(ctx *flow.PipeSessionCtx) error {
	return nil
}

func (c *ClientStep) Cancel(ctx *flow.PipeSessionCtx) error {
	return nil
}

func (c *ClientStep) String() string {
	return ""
}

func main() {
	flag.Parse()

	// 加载 Entity, Echo, Step
	s := flow.NewClientStore()
	s.Load(&flow.Empty{}, &flow.EmptyEcho{}, &ClientStep{}, &pb.Echo{}, &flow.TestStep{})
	pb.RegisterHelloFlowHandler(s, &ClientEcho{})

	// 创建 client
	cfg := flow.NewConfig(*name, *id, *address)
	cfg.WithStore(s)
	client, err := flow.NewClient(cfg, map[string]string{})
	if err != nil {
		log.Fatalf("create client: %v", err)
	}

	// 创建一条 pipe grpc 流连接
	// 这个步骤是关键，执行工作流时，每个步骤都会寻找对应的执行者(就是 client)
	// 建立 pipe 连接就表示这个客户端就是一个 step 执行这
	// 同时这个 pipe 具有重连机制
	pipe, err := client.NewSession()
	if err != nil {
		log.Fatalf("create new pipe connect: %v", err)
	}
	defer pipe.Close()

	log.Info("ping request")
	fc := pb.NewHelloFlowClient(*id, client)
	ctx := context.TODO()

	echoReply, err := fc.Echo(ctx, &pb.EchoRequest{"echo"})
	if err != nil {
		log.Fatal(err)
	}
	log.Info(echoReply.Reply)

	pong, err := fc.Ping(ctx, &pb.PingRequest{})
	if err != nil {
		log.Fatal(err)
	}
	log.Info(pong.Out)

	//items := map[string]string{
	//	"a": "a",
	//	"b": "1",
	//}
	//entity := &flow.Empty{Name: "empty"}
	//step := &flow.TestStep{}
	//
	//// 创建 workflow
	//wf := client.NewWorkflow(flow.WithName("w"), flow.WithId("3")).
	//	Items(items).
	//	Entities(entity, &pb.Echo{Name: "hello"}).
	//	Steps(
	//		flow.NewStepBuilder(step, "1").Build(),
	//		flow.NewStepBuilder(&ClientStep{}, "1").Arg("echo", &pb.Echo{Name: "hello"}).Build(),
	//	).
	//	Build()
	//
	//// 发送数据到服务端，执行工作流，并监控 workflow 数据变化
	//watcher, err := client.ExecuteWorkflow(ctx, wf, true)
	//if err != nil {
	//	log.Fatalf("execute workflow: %v", err)
	//}
	//
	//for {
	//	result, err := watcher.Next()
	//	if err == io.EOF {
	//		log.Infof("worflow done!")
	//		break
	//	}
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//	_ = result
	//	//log.Info(result)
	//}
	//
	//select {}
}
