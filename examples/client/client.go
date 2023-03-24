package main

import (
	"context"
	"flag"
	"io"

	"github.com/vine-io/flow"
	log "github.com/vine-io/vine/lib/logger"
)

var (
	address = flag.String("address", "127.0.0.1:43300", "Set the address of flow service")
	name    = flag.String("name", "flow", "Set the name of flow service")
	id      = flag.String("id", "1", "Set id for flow client")
)

func main() {
	flag.Parse()

	// 加载 Entity, Echo, Step
	flow.Load(&flow.Empty{}, &flow.EmptyEcho{}, &flow.EmptyStep{})

	// 创建 client
	cfg := flow.NewConfig(*name, *id, *address)
	client, err := flow.NewClient(cfg)
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

	items := map[string][]byte{
		"a": []byte("a"),
		"b": []byte("1"),
	}
	entity := &flow.Empty{Name: "empty"}
	step := &flow.EmptyStep{Client: "1"}

	// 创建 workflow
	wf := pipe.NewWorkflow(flow.WithName("w"), flow.WithId("3")).
		Items(items).
		Entities(entity).
		Steps(flow.StepToWorkStep(step)).
		Build()

	ctx := context.TODO()
	// 发送数据到服务端，执行工作流，并监控 workflow 数据变化
	watcher, err := pipe.ExecuteWorkflow(ctx, wf, true)
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
			log.Fatal(err)
		}
		log.Info(result)
	}

	select {}
}
