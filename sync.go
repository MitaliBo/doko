package main

import (
	"context"
	dtypes "github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	consul "github.com/hashicorp/consul/api"
	"log"
	"reflect"
	"strconv"
	"strings"
)

const (
	labelServiceNameKey       = "docons/name"
	labelServicePortKey       = "docons/port"
	labelServiceTagsKey       = "docons/tags"
	labelServiceMetaKeyPrefix = "docons/meta-"

	metaDoconsKey   = "docons"
	metaDoconsValue = "true"
)

// Service interchange struct for docker container and consul service
type Service struct {
	ID   string
	Name string
	Port int
	Tags []string
	Meta map[string]string
}

func listDockerServices(client *docker.Client) (svcs map[string]Service, err error) {
	svcs = map[string]Service{}
	var cs []dtypes.Container
	if cs, err = client.ContainerList(context.Background(), dtypes.ContainerListOptions{}); err != nil {
		return
	}
	for _, c := range cs {
		if c.Labels == nil {
			continue
		}
		name := c.Labels[labelServiceNameKey]
		if len(name) == 0 {
			continue
		}
		port, _ := strconv.Atoi(c.Labels[labelServicePortKey])
		if port == 0 {
			log.Println("label", labelServicePortKey, "is missing:", c.Names)
			continue
		}
		tags := cleanStrSlice(strings.Split(c.Labels[labelServiceTagsKey], ","))
		meta := map[string]string{}
		for k, v := range c.Labels {
			if strings.HasPrefix(k, labelServiceMetaKeyPrefix) {
				meta[k[len(labelServiceMetaKeyPrefix):]] = v
			}
		}
		meta[metaDoconsKey] = metaDoconsValue
		svcs[c.ID] = Service{
			ID:   c.ID,
			Name: name,
			Port: port,
			Tags: tags,
			Meta: meta,
		}
	}
	return
}

func listConsulServices(client *consul.Client) (svcs map[string]Service, err error) {
	svcs = map[string]Service{}
	var cs map[string]*consul.AgentService
	if cs, err = client.Agent().Services(); err != nil {
		return
	}
	for _, s := range cs {
		if s.Meta == nil || s.Meta[metaDoconsKey] != metaDoconsValue {
			continue
		}
		svcs[s.ID] = Service{
			ID:   s.ID,
			Name: s.Service,
			Port: s.Port,
			Tags: s.Tags,
			Meta: s.Meta,
		}
	}
	return
}

func synchronize(dc *docker.Client, cc *consul.Client) (err error) {
	var dsvcs map[string]Service
	var csvcs map[string]Service

	if dsvcs, err = listDockerServices(dc); err != nil {
		return
	}
	if csvcs, err = listConsulServices(cc); err != nil {
		return
	}

	for _, s := range csvcs {
		if _, ok := dsvcs[s.ID]; !ok {
			if err = cc.Agent().ServiceDeregister(s.ID); err != nil {
				return
			}
		}
	}

	for _, s := range dsvcs {
		if reflect.DeepEqual(s, csvcs[s.ID]) {
			continue
		}
		if err = cc.Agent().ServiceRegister(&consul.AgentServiceRegistration{
			ID:   s.ID,
			Name: s.Name,
			Port: s.Port,
			Tags: s.Tags,
			Meta: s.Meta,
		}); err != nil {
			return
		}
	}

	return
}
