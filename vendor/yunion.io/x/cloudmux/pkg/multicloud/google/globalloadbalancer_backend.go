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

package google

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SGlobalLoadbalancerBackend struct {
	lbbg *SGlobalLoadBalancerBackendGroup

	backendService SBackendServices             //
	instanceGroup  SGlobalInstanceGroup         // 实例组
	Backend        SGlobalInstanceGroupInstance // backend

	Port int `json:"port"`
}

func (self SGlobalLoadbalancerBackend) GetId() string {
	return fmt.Sprintf("%s::%s::%s::%d", self.lbbg.GetGlobalId(), self.instanceGroup.GetGlobalId(), self.GetBackendId(), self.Port)
}

func (self SGlobalLoadbalancerBackend) GetName() string {
	segs := strings.Split(self.Backend.Instance, "/")
	name := ""
	if len(segs) > 0 {
		name = segs[len(segs)-1]
	}
	return fmt.Sprintf("%s::%s::%d", self.instanceGroup.GetName(), name, self.Port)
}

func (self SGlobalLoadbalancerBackend) GetGlobalId() string {
	return self.GetId()
}

func (self SGlobalLoadbalancerBackend) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self SGlobalLoadbalancerBackend) GetDescription() string {
	return ""
}

func (self SGlobalLoadbalancerBackend) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self SGlobalLoadbalancerBackend) Refresh() error {
	return nil
}

func (self SGlobalLoadbalancerBackend) IsEmulated() bool {
	return true
}

func (self SGlobalLoadbalancerBackend) GetSysTags() map[string]string {
	return nil
}

func (self SGlobalLoadbalancerBackend) GetTags() (map[string]string, error) {
	return nil, nil
}

func (self SGlobalLoadbalancerBackend) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotSupported
}

func (self SGlobalLoadbalancerBackend) GetWeight() int {
	return 0
}

func (self SGlobalLoadbalancerBackend) GetPort() int {
	return self.Port
}

func (self SGlobalLoadbalancerBackend) GetBackendType() string {
	return api.LB_BACKEND_GUEST
}

func (self SGlobalLoadbalancerBackend) GetBackendRole() string {
	return api.LB_BACKEND_ROLE_DEFAULT
}

func (self SGlobalLoadbalancerBackend) GetBackendId() string {
	vm := &SInstance{}
	err := self.lbbg.lb.region.client.GetBySelfId(self.Backend.Instance, vm)
	if err != nil {
		return getGlobalId(self.Backend.Instance)
	}
	return vm.GetGlobalId()
}

func (self SGlobalLoadbalancerBackend) GetIpAddress() string {
	return ""
}

func (self SGlobalLoadbalancerBackend) SyncConf(ctx context.Context, port, weight int) error {
	return cloudprovider.ErrNotSupported
}

func (self *SGlobalLoadBalancerBackendGroup) GetILoadbalancerBackends() ([]cloudprovider.ICloudLoadbalancerBackend, error) {
	backends, err := self.GetLoadbalancerBackends()
	if err != nil {
		return nil, errors.Wrap(err, "GetLoadbalancerBackends")
	}

	ibackends := make([]cloudprovider.ICloudLoadbalancerBackend, len(backends))
	for i := range backends {
		ibackends[i] = &backends[i]
	}

	return ibackends, nil
}

func (self *SGlobalLoadBalancerBackendGroup) GetLoadbalancerBackends() ([]SGlobalLoadbalancerBackend, error) {
	if self.backends != nil {
		return self.backends, nil
	}

	_igs, err := self.lb.GetInstanceGroupsMap()
	if err != nil {
		return nil, errors.Wrap(err, "GetInstanceGroupsMap")
	}

	igs := make([]SGlobalInstanceGroup, 0)
	for i := range self.backendService.Backends {
		backend := self.backendService.Backends[i]
		if v, ok := _igs[backend.Group]; ok {
			igs = append(igs, v)
		}
	}

	ret := make([]SGlobalLoadbalancerBackend, 0)
	for i := range igs {
		ig := igs[i]
		// http lb
		if self.lb.isHttpLb {
			for j := range ig.NamedPorts {
				np := ig.NamedPorts[j]
				if np.Name != self.backendService.PortName {
					continue
				}

				bs, err := ig.GetInstances()
				if err != nil {
					return nil, errors.Wrap(err, "GetInstances")
				}

				for n := range bs {
					backend := SGlobalLoadbalancerBackend{
						lbbg:           self,
						instanceGroup:  ig,
						Backend:        bs[n],
						backendService: self.backendService,
						Port:           int(np.Port),
					}

					ret = append(ret, backend)
				}
			}
		} else {
			// tcp & udp lb
			bs, err := ig.GetInstances()
			if err != nil {
				return nil, errors.Wrap(err, "GetInstances")
			}

			frs, err := self.lb.GetForwardingRules()
			if err != nil {
				return nil, errors.Wrap(err, "GetForwardingRules")
			}

			for m := range frs {
				fr := frs[m]
				if fr.Ports == nil || len(fr.Ports) == 0 {
					ports := strings.Split(fr.PortRange, "-")
					if len(ports) == 2 && ports[0] == ports[1] {
						port, _ := strconv.Atoi(ports[0])
						for n := range bs {
							backend := SGlobalLoadbalancerBackend{
								lbbg:           self,
								instanceGroup:  ig,
								Backend:        bs[n],
								backendService: self.backendService,
								Port:           port,
							}

							ret = append(ret, backend)
						}
					}
				} else {
					for p := range fr.Ports {
						port, _ := strconv.Atoi(fr.Ports[p])
						if port <= 0 {
							continue
						}

						for n := range bs {
							backend := SGlobalLoadbalancerBackend{
								lbbg:           self,
								instanceGroup:  ig,
								Backend:        bs[n],
								backendService: self.backendService,
								Port:           port,
							}

							ret = append(ret, backend)
						}
					}
				}
			}
		}
	}

	self.backends = ret
	return ret, nil
}
