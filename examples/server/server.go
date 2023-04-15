package main

import (
	"flag"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/vine-io/flow"
	"github.com/vine-io/vine/core/broker/http"
	"github.com/vine-io/vine/core/registry/mdns"
	vserver "github.com/vine-io/vine/core/server"
	"github.com/vine-io/vine/core/server/grpc"
	log "github.com/vine-io/vine/lib/logger"
	usignal "github.com/vine-io/vine/util/signal"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	addr      = flag.String("address", "127.0.0.1:43300", "Set the address of flow service")
	name      = flag.String("name", "flow", "Set the name of flow service")
	endpoints = flag.String("endpoints", "127.0.0.1:2379", "Set the endpoints of etcd for service")
)

func main() {
	flag.Parse()

	conn, err := clientv3.New(clientv3.Config{
		Endpoints:            strings.Split(*endpoints, ","),
		DialTimeout:          time.Second * 3,
		DialKeepAliveTime:    time.Second * 30,
		DialKeepAliveTimeout: time.Second * 15,
	})

	if err != nil {
		log.Fatalf("connecting to etcd: %v", err)
	}

	scheduler, err := flow.NewScheduler(conn, 10)
	if err != nil {
		log.Fatalf("start scheduler failed: %v", err)
	}

	vbroker := http.NewBroker()
	vbroker.Init()

	reg := mdns.NewRegistry()
	reg.Init()

	s := grpc.NewServer(
		vserver.Name(*name),
		vserver.Address(*addr),
		vserver.Broker(vbroker),
		vserver.Registry(reg),
		grpc.Options(
		//gGrpc.KeepaliveParams(keepalive.ServerParameters{
		//	MaxConnectionIdle:     time.Second * 15,
		//	MaxConnectionAge:      time.Second * 30,
		//	MaxConnectionAgeGrace: time.Second * 5,
		//	Time:                  time.Second * 5,
		//	Timeout:               time.Second * 1,
		//}),
		//gGrpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
		//	MinTime:             5 * time.Second,
		//	PermitWithoutStream: true,
		//}),
		),
	)

	if err = s.Init(); err != nil {
		log.Fatalf("init flow service: %v", err)
	}

	server, err := flow.NewRPCServer(s, scheduler)
	if err != nil {
		log.Fatalf("start flow server: %v", err)
	}

	err = s.Start()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, usignal.Shutdown()...)

	log.Infof("Received signal %s", <-ch)

	select {
	case <-ch:
		s.Stop()
		server.Stop()
	}
}
