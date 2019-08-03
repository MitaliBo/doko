package main

import (
	"context"
	dtypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	docker "github.com/docker/docker/client"
	consul "github.com/hashicorp/consul/api"
	"log"
	"os"
	"time"
)

var (
	dclient *docker.Client
	cclient *consul.Client

	syncs = make(chan interface{}, 10)
)

func exit(err *error) {
	if *err == nil {
		log.Println("exited with error:", (*err).Error())
		os.Exit(1)
	}
}

func main() {
	var err error
	defer exit(&err)

	// docker client
	if dclient, err = docker.NewEnvClient(); err != nil {
		return
	}

	// consul client
	if cclient, err = consul.NewClient(consul.DefaultConfig()); err != nil {
		return
	}

	// run debounced sync routine
	go debounce(time.Second*3, syncs, synchronizeWithRetry)

	// trigger initial sync
	syncs <- nil

	// watch
	watch(context.Background())
}

func synchronizeWithRetry() {
	if err := synchronize(dclient, cclient); err != nil {
		log.Printf("failed to synchronize: %s", err.Error())
		// re-trigger sync
		syncs <- nil
	}
}

func watch(ctx context.Context) {
	// full sync every 30 seconds, resolving any kind of race condition
	tk := time.NewTicker(time.Second * 30)
	defer tk.Stop()

	// build events filter
	fargs := filters.NewArgs()
	fargs.Add("scope", "local")
	fargs.Add("type", "container")
	fargs.Add("event", "start")
	fargs.Add("event", "die")

	// streaming events with retry
outerLoop:
	for {
		evch, erch := dclient.Events(ctx, dtypes.EventsOptions{Filters: fargs})

	innerLoop:
		for {
			select {
			case <-tk.C:
				syncs <- nil
			case <-evch:
				syncs <- nil
			case err := <-erch:
				if err != nil {
					log.Printf("docker events streaming failed: %s", err.Error())
				}
				break innerLoop
			case <-ctx.Done():
				break outerLoop
			}
		}

		time.Sleep(time.Second)
	}
}
