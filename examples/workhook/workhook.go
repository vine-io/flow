package main

import (
	"context"
	"flag"

	"github.com/vine-io/flow"
	"github.com/vine-io/flow/api"
	log "github.com/vine-io/vine/lib/logger"
)

var (
	address = flag.String("address", "127.0.0.1:43300", "Set the address of flow service")
	name    = flag.String("name", "flow", "Set the name of flow service")
	id      = flag.String("id", "2", "Set id for flow client")
)

type Config struct {
	Id string
}

func main() {
	flag.Parse()

	cfgf := &Config{Id: *id}
	// 加载 Entity, Echo, Step
	s := flow.NewClientStore()
	if err := s.Provides(&flow.Empty{}, cfgf); err != nil {
		log.Fatal(err)
	}

	// 创建 client
	cfg := flow.NewConfig(*name, *id, *address)
	cfg.WithStore(s)
	client, err := flow.NewClient(cfg, map[string]string{})
	if err != nil {
		log.Fatalf("create client: %v", err)
	}

	w, err := client.WorkHook(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	for {
		next, err := w.Next()
		if err != nil {
			log.Info(err)
			return
		}

		switch next.Action {
		case api.HookAction_HA_UP, api.HookAction_HA_DOWN:
			log.Info(next.Worker)
		}
	}
}
