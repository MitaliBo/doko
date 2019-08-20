package main

import (
	"errors"
	"fmt"
	dtypes "github.com/docker/docker/api/types"
	consul "github.com/hashicorp/consul/api"
	"strings"
)

const (
	ManagedServiceIDPrefix = "doko-svc-"

	LabelServiceNameKey          = "doko.name"
	LabelServicePortKey          = "doko.port"
	LabelServiceTagsKey          = "doko.tags"
	LabelServiceCheckKey         = "doko.check"
	LabelServiceCheckIntervalKey = "doko.check.interval"
	LabelServiceCheckTimeoutKey  = "doko.check.timeout"
	LabelServiceCheckHTTPPathKey = "doko.check.http.path"
	LabelServiceMetaKeyPrefix    = "doko.meta."

	LabelServiceCheckHTTP = "http"
	LabelServiceCheckGRPC = "grpc"
)

var (
	// ErrMissingNameLabel container has no labels, should ignore
	ErrMissingNameLabel = errors.New("missing label 'doko.name'")
)

func IsServiceIDManaged(id string) bool {
	return strings.HasPrefix(id, ManagedServiceIDPrefix)
}

type Service struct {
	ID            string
	Name          string
	Port          int
	Tags          []string
	Check         string
	CheckInterval string
	CheckTimeout  string
	CheckHTTPPath string
	Meta          map[string]string
}

func (s Service) ToAgentServiceRegistration() (reg *consul.AgentServiceRegistration) {
	reg = &consul.AgentServiceRegistration{
		ID:   s.ID,
		Name: s.Name,
		Port: s.Port,
		Tags: s.Tags,
		Meta: s.Meta,
	}
	switch s.Check {
	case LabelServiceCheckHTTP:
		reg.Checks = []*consul.AgentServiceCheck{
			{
				Name:     "(doko) HTTP Check",
				HTTP:     fmt.Sprintf("http://127.0.0.1:%d%s", s.Port, s.CheckHTTPPath),
				Interval: s.CheckInterval,
				Timeout:  s.CheckTimeout,
			},
		}
	case LabelServiceCheckGRPC:
		reg.Checks = []*consul.AgentServiceCheck{
			{
				Name:     "(doko) gRPC Check",
				GRPC:     fmt.Sprintf("127.0.0.1:%d", s.Port),
				Interval: s.CheckInterval,
				Timeout:  s.CheckTimeout,
			},
		}
	}
	return
}

func ServiceFromContainer(c dtypes.Container) (s Service, err error) {
	if len(c.Labels) == 0 {
		err = ErrMissingNameLabel
		return
	}

	// name
	s.Name = cleanServiceName(c.Labels[LabelServiceNameKey])
	if len(s.Name) == 0 {
		err = ErrMissingNameLabel
		return
	}

	// port
	declaredPort := cleanServicePort(c.Labels[LabelServicePortKey])
	if declaredPort == 0 {
		err = errors.New("missing label 'doko.port'")
		return
	}
	switch c.HostConfig.NetworkMode {
	case "default":
		for _, p := range c.Ports {
			if int(p.PrivatePort) == declaredPort {
				s.Port = int(p.PublicPort)
				break
			}
		}
		if s.Port == 0 {
			err = fmt.Errorf("container port '%d' is not published", declaredPort)
			return
		}
	case "host":
		s.Port = declaredPort
	default:
		err = fmt.Errorf("network mode '%s' not supported", c.HostConfig.NetworkMode)
		return
	}

	// id
	shortID := shortenID(c.ID)
	s.ID = ManagedServiceIDPrefix + shortID

	// tags
	s.Tags = cleanServiceTags(strings.Split(c.Labels[LabelServiceTagsKey], ","))

	// check
	s.Check = cleanServiceCheck(c.Labels[LabelServiceCheckKey])

	switch s.Check {
	case LabelServiceCheckHTTP, LabelServiceCheckGRPC:
		s.CheckTimeout = cleanContainerLabel(c.Labels[LabelServiceCheckTimeoutKey])
		if len(s.CheckTimeout) == 0 {
			s.CheckTimeout = "5s"
		}
		s.CheckInterval = cleanContainerLabel(c.Labels[LabelServiceCheckIntervalKey])
		if len(s.CheckInterval) == 0 {
			s.CheckInterval = "10s"
		}
		if s.Check == LabelServiceCheckHTTP {
			s.CheckHTTPPath = cleanContainerLabel(c.Labels[LabelServiceCheckHTTPPathKey])
			if len(s.CheckHTTPPath) == 0 {
				s.CheckHTTPPath = "/_health"
			}
		}
	case "":
	default:
		err = fmt.Errorf("unknown check type '%s'", s.Check)
		return
	}

	// meta
	s.Meta = map[string]string{}
	for k, v := range c.Labels {
		if strings.HasPrefix(k, LabelServiceMetaKeyPrefix) {
			s.Meta[k[len(LabelServiceMetaKeyPrefix):]] = cleanContainerLabel(v)
		}
	}

	return
}
