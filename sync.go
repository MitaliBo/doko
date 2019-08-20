package main

import (
	"context"
	dtypes "github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	consul "github.com/hashicorp/consul/api"
	"log"
)

func queryDocker(client *docker.Client) (svcs map[string]Service, err error) {
	svcs = map[string]Service{}
	var cs []dtypes.Container
	if cs, err = client.ContainerList(context.Background(), dtypes.ContainerListOptions{}); err != nil {
		return
	}
	for _, c := range cs {
		var s Service
		if s, err = ServiceFromContainer(c); err != nil {
			if err != ErrMissingNameLabel {
				log.Printf("failed to register container %s: %s", c.ID, err.Error())
			}
			err = nil
			continue
		}
		svcs[s.ID] = s
	}
	return
}

func queryConsul(client *consul.Client) (svcs map[string]bool, err error) {
	svcs = map[string]bool{}
	var ss map[string]*consul.AgentService
	if ss, err = client.Agent().Services(); err != nil {
		return
	}
	for _, s := range ss {
		if !IsServiceIDManaged(s.ID) {
			continue
		}
		svcs[s.ID] = true
	}
	return
}

func synchronize(dc *docker.Client, cc *consul.Client) (err error) {
	var dsvcs map[string]Service
	var csvcs map[string]bool

	if dsvcs, err = queryDocker(dc); err != nil {
		return
	}
	if csvcs, err = queryConsul(cc); err != nil {
		return
	}

	for id := range csvcs {
		if _, ok := dsvcs[id]; ok {
			continue
		}
		log.Printf("service %s no longer exists, deregistering", id)
		if err = cc.Agent().ServiceDeregister(id); err != nil {
			log.Printf("failed to deregister %s: %s", id, err.Error())
			err = nil
			continue
		}
	}

	for _, s := range dsvcs {
		if csvcs[s.ID] {
			continue
		}
		log.Printf("new service %s(%s), registering", s.Name, s.ID)
		if err = cc.Agent().ServiceRegister(s.ToAgentServiceRegistration()); err != nil {
			log.Printf("failed to register %s: %s", s.ID, err.Error())
			return
		}
	}

	return
}
