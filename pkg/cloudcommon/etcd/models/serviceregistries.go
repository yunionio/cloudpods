package models

import (
	"context"
	"net"
	"strconv"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/etcd/models/base"
)

type SServiceRegistryManager struct {
	base.SEtcdBaseModelManager
}

type SServiceRegistry struct {
	base.SEtcdBaseModel

	Provider    string
	Environment string
	Region      string
	Zone        string

	ServiceType string

	Address string
	Port    int
}

var ServiceRegistryManager *SServiceRegistryManager

func init() {
	ServiceRegistryManager = &SServiceRegistryManager{
		SEtcdBaseModelManager: base.NewEtcdBaseModelManager(
			&SServiceRegistry{},
			"service-registry",
			"service-registries",
		),
	}
}

func (manager *SServiceRegistryManager) Register(ctx context.Context, addr string, port int,
	provider string, env string, region string, zone string, serviceType string) error {

	log.Infof("register service %s %s:%d", serviceType, addr, port)
	sr := SServiceRegistry{}
	sr.ID = net.JoinHostPort(addr, strconv.Itoa(port))
	sr.Provider = provider
	sr.Environment = env
	sr.Region = region
	sr.Zone = zone
	sr.ServiceType = serviceType
	sr.Address = addr
	sr.Port = port

	return manager.Session(ctx, &sr)
}
