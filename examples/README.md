# 简介
flow 实例代码，分成 server 和 client 部分

# 服务端
flow server 依赖于 etcd，所以需要先启动 etcd

```bash
etcd
```

启动服务端
```bash
go run server/server.go --endpoints=127.0.0.1:2379
```

# 客户端
每个建立 pipe 连接的客户端都为 worker。启动客户端
```bash
go run client/client.go 
```

查看输出结果
```bash
2023-03-23 15:28:57 file=client/client.go:71 level=info name:"w" wid:"2" action:EA_CREATE type:ET_STEP key:"/vine.io/flow/wf/2/step/e85d973a-7945-4d6e-881e-220be6348ffb" value:"{\"name\":\"github.com/vine-io/flow.EmptyStep\",\"uid\":\"e85d973a-7945-4d6e-881e-220be6348ffb\",\"worker\":\"2\",\"entity\":\"github.com/vine-io/flow.Empty\",\"injects\":[\"a\",\"b\",\"c\"]}" 
2023-03-23 15:28:57 file=client/client.go:71 level=info name:"w" wid:"2" action:EA_UPDATE type:ET_WORKFLOW key:"/vine.io/flow/wf/2/status" value:"{\"state\":\"failed\",\"msg\":\"pipe 2 not found\",\"action\":\"prepare\",\"step\":\"github.com/vine-io/flow.EmptyStep_e85d973a-7945-4d6e-881e-220be6348ffb\"}" 
2023-03-23 15:28:57 file=client/client.go:65 level=info worflow done!
➜  flow (main) ✗ go run examples/client/client.go
2023-03-23 15:30:18 file=client/client.go:71 level=info name:"w" wid:"2" action:EA_CREATE type:ET_STEP key:"/vine.io/flow/wf/2/step/604aae6d-849f-41a5-8a8d-a9b6e47a4f2c" value:"{\"name\":\"github.com/vine-io/flow.EmptyStep\",\"uid\":\"604aae6d-849f-41a5-8a8d-a9b6e47a4f2c\",\"worker\":\"2\",\"entity\":\"github.com/vine-io/flow.Empty\",\"injects\":[\"a\",\"b\",\"c\"]}" 
2023-03-23 15:30:18 file=flow/client.go:440 level=error failed to get receive data: EOF
2023-03-23 15:30:18 file=client/client.go:65 level=info worflow done!
➜  flow (main) ✗ go run examples/client/client.go
2023-03-23 15:36:01 file=client/client.go:69 level=fatal rpc error: code = InvalidArgument desc = {"id":"7e5945a0-6fdf-4c9a-83cc-aa74d07464cd","code":400,"detail":"worker 2 not found","status":"Bad Request"}
exit status 1
➜  flow (main) ✗ go run examples/client/client.go
2023-03-23 15:36:13 file=client/client.go:71 level=info name:"w" wid:"2" action:EA_CREATE type:ET_STEP key:"/vine.io/flow/wf/2/step/c5570d4c-78ba-46dd-ba79-a0f11969c6df" value:"{\"name\":\"github.com/vine-io/flow.EmptyStep\",\"uid\":\"c5570d4c-78ba-46dd-ba79-a0f11969c6df\",\"worker\":\"1\",\"entity\":\"github.com/vine-io/flow.Empty\",\"injects\":[\"a\",\"b\",\"c\"]}" 
2023-03-23 15:36:13 file=flow/step.go:102 level=info a = a, b = 1
2023-03-23 15:36:13 file=client/client.go:71 level=info name:"w" wid:"2" action:EA_UPDATE type:ET_ITEM key:"/vine.io/flow/wf/2/item/c" value:"\"ok\"" 
2023-03-23 15:36:13 file=client/client.go:71 level=info name:"w" wid:"2" action:EA_UPDATE type:ET_ENTITY key:"/vine.io/flow/wf/2/entity/github.com/vine-io/flow.Empty" value:"{\"kind\":\"github.com/vine-io/flow.Empty\",\"id\":\"1\",\"raw\":\"eyJuYW1lIjoiZW1wdHkifQ==\"}" 
2023-03-23 15:36:13 file=client/client.go:71 level=info name:"w" wid:"2" action:EA_UPDATE type:ET_WORKFLOW key:"/vine.io/flow/wf/2/status" value:"{\"state\":\"running\",\"action\":\"commit\",\"progress\":\"0.00\",\"step\":\"github.com/vine-io/flow.EmptyStep_c5570d4c-78ba-46dd-ba79-a0f11969c6df\"}" 
2023-03-23 15:36:13 file=client/client.go:71 level=info name:"w" wid:"2" action:EA_UPDATE type:ET_STEP key:"/vine.io/flow/wf/2/step/c5570d4c-78ba-46dd-ba79-a0f11969c6df" value:"{\"name\":\"github.com/vine-io/flow.EmptyStep\",\"uid\":\"c5570d4c-78ba-46dd-ba79-a0f11969c6df\",\"worker\":\"1\",\"entity\":\"github.com/vine-io/flow.Empty\",\"injects\":[\"a\",\"b\",\"c\"]}" 
2023-03-23 15:36:13 file=flow/step.go:112 level=info commit
2023-03-23 15:36:13 file=flow/step.go:113 level=info c = "ok"
2023-03-23 15:36:13 file=client/client.go:71 level=info name:"w" wid:"2" action:EA_UPDATE type:ET_ENTITY key:"/vine.io/flow/wf/2/entity/github.com/vine-io/flow.Empty" value:"{\"kind\":\"github.com/vine-io/flow.Empty\",\"id\":\"1\",\"raw\":\"eyJuYW1lIjoiY29tbWl0dGVkIn0=\"}" 
2023-03-23 15:36:13 file=client/client.go:71 level=info name:"w" wid:"2" action:EA_UPDATE type:ET_WORKFLOW key:"/vine.io/flow/wf/2/status" value:"{\"state\":\"running\",\"action\":\"cancel\",\"step\":\"github.com/vine-io/flow.EmptyStep_c5570d4c-78ba-46dd-ba79-a0f11969c6df\"}" 
2023-03-23 15:36:13 file=client/client.go:71 level=info name:"w" wid:"2" action:EA_UPDATE type:ET_STEP key:"/vine.io/flow/wf/2/step/c5570d4c-78ba-46dd-ba79-a0f11969c6df" value:"{\"name\":\"github.com/vine-io/flow.EmptyStep\",\"uid\":\"c5570d4c-78ba-46dd-ba79-a0f11969c6df\",\"worker\":\"1\",\"entity\":\"github.com/vine-io/flow.Empty\",\"injects\":[\"a\",\"b\",\"c\"]}" 
2023-03-23 15:36:13 file=flow/step.go:125 level=info cancel
2023-03-23 15:36:13 file=client/client.go:71 level=info name:"w" wid:"2" action:EA_UPDATE type:ET_ENTITY key:"/vine.io/flow/wf/2/entity/github.com/vine-io/flow.Empty" value:"{\"kind\":\"github.com/vine-io/flow.Empty\",\"id\":\"1\",\"raw\":\"eyJuYW1lIjoiY2FuY2VsIn0=\"}" 
2023-03-23 15:36:13 file=client/client.go:65 level=info worflow done!
```