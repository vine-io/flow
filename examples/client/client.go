package main

import (
	"context"
	"encoding/xml"
	"flag"
	"io"
	"os"
	"reflect"

	"github.com/vine-io/flow"
	"github.com/vine-io/flow/api"
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

type Config struct {
	Id string
}

func (c *ClientEcho) Echo(ctx context.Context, request *pb.EchoRequest, response *pb.EchoResponse) error {
	response.Reply = request.Echo
	return nil
}

func (c *ClientEcho) Ping(ctx context.Context, request *pb.PingRequest, response *pb.PingResponse) error {
	response.Out = "pong"
	return nil
}

var _ flow.Step = (*ClientStep)(nil)

type ClientStep struct {
	*Config `inject:""`

	Echo     *pb.Echo `flow:"entity"`
	EchoArgs *pb.Echo `flow:"ctx:echo"`
	A        string   `flow:"ctx:a"`
	c        string
	List     []*pb.Echo `flow:"ctx:list"`
}

func (c *ClientStep) Owner() reflect.Type {
	return reflect.TypeOf(new(pb.Echo))
}

func (c *ClientStep) Prepare(ctx *flow.PipeSessionCtx) error {
	log.Infof("entity echo = %v, id=%v", c.Echo, c.Id)
	log.Infof("args echo = %v", c.EchoArgs)
	c.c = "test"
	return nil
}

func (c *ClientStep) Commit(ctx *flow.PipeSessionCtx) (map[string]any, error) {
	log.Infof("entity echo = %v, id=%v", c.Echo, c.Id)
	log.Infof("args echo = %v", c.EchoArgs)
	log.Infof("a = %v", c.A)
	log.Infof("list = %v", c.List)

	ctx.Log().Info("test log1")
	ctx.Log().Info("test log2")
	return map[string]any{"a": "bbb"}, nil
}

func (c *ClientStep) Rollback(ctx *flow.PipeSessionCtx) error {
	return nil
}

func (c *ClientStep) Cancel(ctx *flow.PipeSessionCtx) error {
	log.Infof("c = %s", c.c)
	return nil
}

func (c *ClientStep) Desc() string {
	return ""
}

func main() {
	flag.Parse()

	cfgf := &Config{Id: *id}
	// 加载 Entity, Echo, Step
	s := flow.NewClientStore()
	s.Load(&flow.Empty{}, &flow.EmptyEcho{}, &flow.CellStep{}, &ClientStep{}, &pb.Echo{}, &flow.TestStep{})
	if err := s.Provides(&flow.Empty{}, cfgf); err != nil {
		log.Fatal(err)
	}
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

	//out, err := client.Step(ctx, *id, new(ClientStep), map[string]string{"a": "2"})
	//if err != nil {
	//	log.Fatal(err)
	//}
	//log.Info(out)

	items := map[string]any{
		"a":    "a",
		"b":    "1",
		"echo": &pb.Echo{Name: "hello echo"},
		"list": []*pb.Echo{
			&pb.Echo{Name: "hello"},
			&pb.Echo{Name: "hello echo"},
		},
	}
	step := &flow.TestStep{}

	echoEntity1 := &pb.Echo{Name: "hello1"}
	sw := flow.NewSubWorkflowBuilder(echoEntity1, flow.WithName("test subprocess"), flow.WithId("sub")).
		Steps(flow.NewStepBuilder(&ClientStep{}, "1", echoEntity1),
			flow.NewStepBuilder(&flow.CellStep{}, "1", &flow.Empty{}))

	// 创建 workflow
	wid := "demo1"
	echoEntity2 := &pb.Echo{Name: "hello2"}
	d, dataObjects, properties, err := flow.NewWorkFlowBuilder(flow.WithName("w"), flow.WithId(wid)).
		Items(items).
		Steps(
			flow.NewStepBuilder(step, "1", &flow.Empty{}),
			flow.NewStepBuilder(&ClientStep{}, "1", echoEntity2),
			sw,
		).
		ToProcessDefinitions()
	if err != nil {
		log.Fatalf("create a new workflow %v", err)
	}

	data, _ := xml.MarshalIndent(d, "", " ")
	definitionPath := "_output/sample.bpmn"
	os.WriteFile(definitionPath, data, os.ModePerm)

	_ = dataObjects
	_ = properties

	// 发送数据到服务端，执行工作流，并监控 workflow 数据变化
	watcher, err := client.ExecuteWorkflowInstance(ctx, wid, "test", string(data), dataObjects, properties, true)
	if err != nil {
		log.Fatalf("execute workflow: %v", err)
	}

	for {
		result, err := watcher.Next()
		if err == io.EOF {
			log.Infof("worflow done!")
			break
		}
		if err != nil {
			log.Error(err)
			break
		}
		if result == nil {
			break
		}

		//log.Infof("key = %v, type = %v, action = %v", result.Key, result.Type, result.Action)
		switch result.Type {
		case api.EventType_ET_WORKFLOW:
			//log.Infof("workflow: %v", string(result.Value))
		case api.EventType_ET_STATUS:
			//log.Infof("status: %v", string(result.Value))
		case api.EventType_ET_TRACE:
			log.Infof("trace: %v", string(result.Value))
		case api.EventType_ET_BPMN:
			log.Infof("bpmn: %v", string(result.Value))
		case api.EventType_ET_STEP:
			//log.Infof("step: %v", string(result.Value))
		}
	}

	//select {}
}
