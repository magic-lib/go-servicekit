package consul

import (
	"fmt"
	"github.com/hashicorp/consul/api"
)

type AgentServiceRegistration struct {
	ServiceId   string
	ServiceName string
	Address     string
	Port        string
}

func (cu *consulClient) RegisterService(registration *api.AgentServiceRegistration) error {
	if registration == nil {
		return fmt.Errorf("registration is nil")
	}
	if registration.ID == "" || registration.Name == "" || registration.Address == "" || registration.Port == 0 {
		return fmt.Errorf("registration is invalid")
	}
	if registration.Check == nil {
		// 健康检查配置
		registration.Check = &api.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%d/health", registration.Address, registration.Port), // 健康检查接口
			Interval:                       "10s",                                                                       // 检查间隔
			Timeout:                        "5s",                                                                        // 超时时间
			DeregisterCriticalServiceAfter: "30s",                                                                       // 不健康30s后自动注销
		}
	}
	// 注册服务
	return cu.consulClient.Agent().ServiceRegister(registration)
}
func (cu *consulClient) HttpRegisterService(registration *api.CatalogRegistration) error {
	if registration == nil {
		return fmt.Errorf("registration is nil")
	}
	if registration.ID == "" || registration.Name == "" || registration.Address == "" || registration.Port == 0 {
		return fmt.Errorf("registration is invalid")
	}
	if registration.Check == nil {
		// 健康检查配置
		registration.Check = &api.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%d/health", registration.Address, registration.Port), // 健康检查接口
			Interval:                       "10s",                                                                       // 检查间隔
			Timeout:                        "5s",                                                                        // 超时时间
			DeregisterCriticalServiceAfter: "30s",                                                                       // 不健康30s后自动注销
		}
	}

	catalog := cu.consulClient.Catalog()
	catalog.Register(registration, nil)

	// 构建Consul注册API地址
	registerURL := fmt.Sprintf("http://%s/v1/catalog/register", cu.consulClient.Catalog())

	// 注册服务
	return cu.consulClient.Agent().ServiceRegister(registration)
}
