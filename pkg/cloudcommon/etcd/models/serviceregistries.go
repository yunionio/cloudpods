// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	ServiceRegistryManager.SetVirtualObject(ServiceRegistryManager)
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
