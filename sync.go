package main

import (
	"context"
	"fmt"
	dtypes "github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	consul "github.com/hashicorp/consul/api"
	"log"
	"reflect"
	"strconv"
	"strings"
)

const (
	ServiceIDPrefix = "doko-srv-"
	CheckIDPrefix   = "doko-chk-"

	LabelServiceNameKey       = "doko.name"
	LabelServicePortKey       = "doko.port"
	LabelServiceTagsKey       = "doko.tags"
	LabelServiceMetaKeyPrefix = "doko.meta."
	LabelServiceCheckKey      = "doko.check"
	LabelServiceCheckHTTP     = "http"
)

type Service struct {
	ID   string
	Name string
	Port int
	Tags []string
	Meta map[string]string
}

type Check struct {
	ID        string
	ServiceID string
	URL       string
}

func queryDocker(client *docker.Client) (svcs map[string]Service, chks map[string]Check, err error) {
	svcs = map[string]Service{}
	chks = map[string]Check{}
	var cs []dtypes.Container
	if cs, err = client.ContainerList(context.Background(), dtypes.ContainerListOptions{}); err != nil {
		return
	}
	for _, c := range cs {
		if c.Labels == nil {
			continue
		}

		shortID := shortenID(c.ID)

		svc := Service{
			Name: c.Labels[LabelServiceNameKey],
			ID:   ServiceIDPrefix + shortID,
			Tags: cleanStrSlice(strings.Split(c.Labels[LabelServiceTagsKey], ",")),
			Meta: map[string]string{},
		}

		if len(svc.Name) == 0 {
			continue
		}

		port, _ := strconv.Atoi(c.Labels[LabelServicePortKey])
		if port == 0 {
			log.Printf("container %s: label %s is missing or invalid", shortID, LabelServicePortKey)
			continue
		}
		if c.HostConfig.NetworkMode == "host" {
			svc.Port = port
		} else if c.HostConfig.NetworkMode == "default" {
			for _, p := range c.Ports {
				if int(p.PrivatePort) == port {
					svc.Port = int(p.PublicPort)
					break
				}
			}
			if svc.Port == 0 {
				log.Printf("container %s: port %d is not mapped", shortID, port)
				continue
			}
		} else {
			log.Printf("container %s: network mode %s is not supported", shortID, c.HostConfig.NetworkMode)
			continue
		}

		for k, v := range c.Labels {
			if strings.HasPrefix(k, LabelServiceMetaKeyPrefix) {
				svc.Meta[k[len(LabelServiceMetaKeyPrefix):]] = v
			}
		}

		svcs[svc.ID] = svc

		check := strings.ToLower(c.Labels[LabelServiceCheckKey])

		if check == LabelServiceCheckHTTP {
			chk := Check{
				ID:        CheckIDPrefix + shortID,
				ServiceID: svc.ID,
				URL:       fmt.Sprintf("http://127.0.0.1:%d/_health", svc.Port),
			}
			chks[chk.ID] = chk
		} else if len(check) > 0 {
			log.Printf("container %s: check mode %s is not supported", shortID, check)
		}
	}
	return
}

func queryConsul(client *consul.Client) (svcs map[string]Service, chks map[string]Check, err error) {
	svcs = map[string]Service{}
	chks = map[string]Check{}
	var ss map[string]*consul.AgentService
	if ss, err = client.Agent().Services(); err != nil {
		return
	}
	for _, s := range ss {
		if !strings.HasPrefix(s.ID, ServiceIDPrefix) {
			continue
		}
		svc := Service{
			ID:   s.ID,
			Name: s.Service,
			Port: s.Port,
			Tags: s.Tags,
			Meta: s.Meta,
		}
		svcs[svc.ID] = svc
	}
	var cs map[string]*consul.AgentCheck
	if cs, err = client.Agent().Checks(); err != nil {
		return
	}
	for _, c := range cs {
		if !strings.HasPrefix(c.CheckID, CheckIDPrefix) {
			continue
		}
		chk := Check{
			ID:        c.CheckID,
			ServiceID: c.ServiceID,
			URL:       c.Definition.HTTP,
		}
		chks[chk.ID] = chk
	}
	return
}

func synchronize(dc *docker.Client, cc *consul.Client) (err error) {
	var dsvcs map[string]Service
	var dchks map[string]Check
	var csvcs map[string]Service
	var cchks map[string]Check

	if dsvcs, dchks, err = queryDocker(dc); err != nil {
		return
	}
	if csvcs, cchks, err = queryConsul(cc); err != nil {
		return
	}

	for _, s := range csvcs {
		if _, ok := dsvcs[s.ID]; !ok {
			log.Printf("service %s(%s) no longer exists, deregistering", s.Name, s.ID)
			if err = cc.Agent().ServiceDeregister(s.ID); err != nil {
				return
			}
		}
	}

	for _, c := range cchks {
		if _, ok := dchks[c.ID]; !ok {
			log.Printf("check %s no longer exists, deregistering", c.ID)
			if err = cc.Agent().CheckDeregister(c.ID); err != nil {
				return
			}
		}
	}

	for _, s := range dsvcs {
		if reflect.DeepEqual(s, csvcs[s.ID]) {
			continue
		}
		log.Printf("service create/update %s(%s), %+v", s.Name, s.ID, s)
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

	for _, c := range dchks {
		// no DeepEqual because consul checks returns no URL
		if c.ServiceID == cchks[c.ID].ServiceID {
			continue
		}
		log.Printf("check create/update for %s, %+v", c.ServiceID, c)
		if err = cc.Agent().CheckRegister(&consul.AgentCheckRegistration{
			ID:        c.ID,
			ServiceID: c.ServiceID,
			Name:      "doko check for " + c.ServiceID,
			AgentServiceCheck: consul.AgentServiceCheck{
				HTTP:     c.URL,
				Interval: "5s",
			},
		}); err != nil {
			return
		}
	}

	return
}
