package main

import (
	"fmt"
	"net"
	"net/url"

	sharedconsul "github.com/example/ats-platform/internal/shared/consul"
	"github.com/hashicorp/consul/api"
)

type serviceDiscovery struct {
	client *api.Client
}

type serviceInstance struct {
	Name    string
	Address string
	Port    int
}

func newServiceDiscovery(consulAddr string) (*serviceDiscovery, error) {
	cfg := api.DefaultConfig()
	cfg.Address = consulAddr

	client, err := api.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("create consul client: %w", err)
	}

	return &serviceDiscovery{client: client}, nil
}

func (d *serviceDiscovery) resolve(baseName string, protocol sharedconsul.Protocol) (*serviceInstance, error) {
	serviceName := sharedconsul.ServiceName(baseName, protocol)
	services, _, err := d.client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return nil, fmt.Errorf("discover service %s: %w", serviceName, err)
	}
	if len(services) == 0 {
		return nil, fmt.Errorf("no healthy instances for service %s", serviceName)
	}

	entry := services[0]
	address := entry.Service.Address
	if address == "" {
		address = entry.Node.Address
	}
	if address == "" || entry.Service.Port == 0 {
		return nil, fmt.Errorf("service %s returned incomplete address info", serviceName)
	}

	return &serviceInstance{
		Name:    serviceName,
		Address: normalizeServiceAddress(address),
		Port:    entry.Service.Port,
	}, nil
}

func normalizeServiceAddress(address string) string {
	if address != "host.docker.internal" {
		return address
	}

	if _, err := net.LookupHost(address); err == nil {
		return address
	}

	return "127.0.0.1"
}

func (i *serviceInstance) baseURL() string {
	return fmt.Sprintf("http://%s:%d", i.Address, i.Port)
}

func (i *serviceInstance) externalURL() string {
	base := i.baseURL()
	if parsed, err := url.Parse(base); err == nil {
		return parsed.String()
	}
	return base
}
