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
	dockerClient *docker.Client
	consulClient *consul.Client

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
	if dockerClient, err = docker.NewEnvClient(); err != nil {
		return
	}

	// consul client
	if consulClient, err = consul.NewClient(consul.DefaultConfig()); err != nil {
		return
	}

	// run debounced sync routine
	go debounce(time.Second*3, syncs, notify)

	// trigger initial sync
	syncs <- nil

	// watch
	if err = watch(); err != nil {
		return
	}
}

func notify() {
	if err := synchronize(dockerClient, consulClient); err != nil {
		log.Println("SYNC: failed to sync:", err.Error())
		// re-trigger sync
		syncs <- nil
	}
}

func watch() (err error) {
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	// ticker to ensure full synchronization
	tk := time.NewTicker(time.Second * 20)
	defer tk.Stop()

	// limit events to local container
	fargs := filters.NewArgs()
	fargs.Add("scope", "local")
	fargs.Add("type", "container")
	fargs.Add("event", "start")
	fargs.Add("event", "die")

	// start streaming
	vchan, echan := dockerClient.Events(ctx, dtypes.EventsOptions{Filters: fargs})

	for {
		select {
		case <-tk.C:
			syncs <- nil
		case <-vchan:
			syncs <- nil
		case err = <-echan:
			return
		}
	}
}
