package main

import (
	"crypto/rand"
	"encoding/hex"
	consul "github.com/hashicorp/consul/api"
	"io/ioutil"
	"os"
	"strings"
)

const (
	instanceIDFile      = "doko-id"
	instanceServiceName = "doko"
)

var (
	instanceID        string
	instanceServiceID string
	instanceCheckID   string
)

func ensureInstanceID() (err error) {
	var buf []byte
	if buf, err = ioutil.ReadFile(instanceIDFile); err != nil {
		if !os.IsNotExist(err) {
			return
		}
	}
	if len(buf) == 0 {
		id := make([]byte, 12, 12)
		if _, err = rand.Read(id); err != nil {
			return
		}
		instanceID = hex.EncodeToString(id)
		if err = ioutil.WriteFile(instanceIDFile, []byte(instanceID), 0644); err != nil {
			return
		}
	} else {
		instanceID = strings.TrimSpace(string(buf))
	}

	instanceServiceID = "doko-ins-" + instanceID
	instanceCheckID = "doko-ins-chk-" + instanceID
	return
}

func registerInstance() (err error) {
	if err = cclient.Agent().ServiceRegister(&consul.AgentServiceRegistration{
		Name: instanceServiceName,
		ID:   instanceServiceID,
		Port: 0,
		Check: &consul.AgentServiceCheck{
			CheckID: instanceCheckID,
			Name:    "(doko) Internal Alive Check",
			TTL:     "10s",
		},
	}); err != nil {
		return
	}
	return
}

func deregisterInstance() error {
	return cclient.Agent().ServiceDeregister(instanceServiceID)
}

func notifyInstanceRunning() error {
	return cclient.Agent().PassTTL(instanceCheckID, "RUNNING")
}
