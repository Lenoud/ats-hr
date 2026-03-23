package consul

import (
	"fmt"
	"net"
	"strings"

	"github.com/hashicorp/consul/api"
)

// consul 定义一个consul结构体，其内部有一个`*api.Client`字段。
type consul struct {
	client *api.Client
}

// NewConsul 连接至consul服务返回一个consul对象
func NewConsul(addr string) (*consul, error) {
	cfg := api.DefaultConfig()
	cfg.Address = addr
	c, err := api.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	return &consul{c}, nil
}

// GetOutboundIP 获取本机的出口IP
func GetOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}

// ResolveServiceAddress returns the explicit address when provided, otherwise falls back to the detected outbound IP.
func ResolveServiceAddress(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit), nil
	}

	ip, err := GetOutboundIP()
	if err != nil {
		return "", err
	}
	return ip.String(), nil
}

// ServiceID returns the stable Consul service ID used by this helper.
func ServiceID(serviceName string, ip string, port int, uuid string) string {
	return fmt.Sprintf("%s-%s-%d-%s", serviceName, ip, port, uuid)
}

// EndpointServiceID returns the stable Consul service ID for an endpoint.
func EndpointServiceID(endpoint Endpoint, instanceID string) string {
	return ServiceID(ServiceName(endpoint.BaseName, endpoint.Protocol), endpoint.IP, endpoint.Port, instanceID)
}

// RegisterService registers a service endpoint in Consul.
func (c *consul) RegisterService(serviceName string, ip string, port int, uuid string) error {
	srv := &api.AgentServiceRegistration{
		ID:      ServiceID(serviceName, ip, port, uuid),
		Name:    serviceName,
		Tags:    []string{"tcp"},
		Address: ip,
		Port:    port,

		Check: &api.AgentServiceCheck{
			TCP:                            fmt.Sprintf("%s:%d", ip, port), // 关键！
			Interval:                       "5s",
			Timeout:                        "3s",
			DeregisterCriticalServiceAfter: "30s",
		},
	}

	return c.client.Agent().ServiceRegister(srv)
}

// RegisterEndpoint registers an endpoint in Consul using shared naming rules.
func (c *consul) RegisterEndpoint(endpoint Endpoint, instanceID string) error {
	return c.RegisterService(ServiceName(endpoint.BaseName, endpoint.Protocol), endpoint.IP, endpoint.Port, instanceID)
}

// Deregister 注销服务
func (c *consul) Deregister(serviceID string) error {
	return c.client.Agent().ServiceDeregister(serviceID)
}
