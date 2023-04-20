# flow

# 实例代码
查看 [examples](https://github.com/vine-io/flow/tree/main/examples)

## 生成 echo 代码
```bash
cd $GOPATH/src
protoc -I. --gogo_out=:. --flow_out=:. github.com/vine-io/flow/examples/pb/hello.proto
```

## 启动 etcd
```bash
etcd
```

## 启动 server 和 client
```bash
go build -o _output/server examples/server/server.go
go build -o _output/client examples/client/client.go

_output/server
_output/client
```

# 下载 protoc-gen-flow
```bash
bash -c tool/install.sh
```