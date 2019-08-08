package main

import (
	"context"
	"flag"
	dtypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	docker "github.com/docker/docker/client"
	consul "github.com/hashicorp/consul/api"
	"log"
	"os"
	"time"
)

var (
	optDeregister bool

	dclient *docker.Client
	cclient *consul.Client

	syncs = make(chan interface{}, 10)
)

func exit(err *error) {
	if *err != nil {
		log.Println("exited with error:", (*err).Error())
		os.Exit(1)
	}
}

func main() {
	var err error
	defer exit(&err)

	// ensure instance id
	if err = ensureInstanceID(); err != nil {
		return
	}

	// parse flag
	flag.BoolVar(&optDeregister, "deregister", false, "one shot run to deregister self from consul")
	flag.Parse()

	// consul client
	if cclient, err = consul.NewClient(consul.DefaultConfig()); err != nil {
		return
	}

	// deregister if demanded
	if optDeregister {
		err = deregisterInstance()
		return
	}

	// docker client
	if dclient, err = docker.NewEnvClient(); err != nil {
		return
	}

	// register self to consul
	if err = registerInstance(); err != nil {
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
	stk := time.NewTicker(time.Second * 30)
	defer stk.Stop()

	// notify consul TTL check every 5 seconds
	ctk := time.NewTicker(time.Second * 5)
	defer ctk.Stop()

	// build events filter
	fargs := filters.NewArgs()
	fargs.Add("scope", "local")
	fargs.Add("type", "container")
	fargs.Add("event", "start")
	fargs.Add("event", "die")

	// streaming events with retry
outerLoop:
	for {
		ech, errch := dclient.Events(ctx, dtypes.EventsOptions{Filters: fargs})

	innerLoop:
		for {
			select {
			case <-ctk.C:
				if err := notifyInstanceRunning(); err != nil {
					log.Printf("failed to notify consul TTL check: %s", err.Error())
				}
			case <-stk.C:
				syncs <- nil
			case <-ech:
				syncs <- nil
			case err := <-errch:
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
