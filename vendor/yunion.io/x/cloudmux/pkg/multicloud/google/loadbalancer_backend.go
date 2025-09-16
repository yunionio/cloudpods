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

type SLoadbalancerBackend struct {
	lbbg *SLoadBalancerBackendGroup

	backendService SBackendServices       //
	instanceGroup  SInstanceGroup         // 实例组
	Backend        SInstanceGroupInstance // backend

	Instance string
	IpAddr   string

	Port int `json:"port"`
}

func (self *SLoadbalancerBackend) GetId() string {
	if len(self.Instance) > 0 {
		return fmt.Sprintf("%s::%s::%d", self.Instance, self.IpAddr, self.Port)
	}
	return fmt.Sprintf("%s::%s::%s::%d", self.lbbg.GetGlobalId(), self.instanceGroup.GetGlobalId(), self.GetBackendId(), self.Port)
}

func (self *SLoadbalancerBackend) GetName() string {
	if len(self.Instance) > 0 {
		return self.Instance
	}
	segs := strings.Split(self.Backend.Instance, "/")
	name := ""
	if len(segs) > 0 {
		name = segs[len(segs)-1]
	}
	return fmt.Sprintf("%s::%s::%d", self.instanceGroup.GetName(), name, self.Port)
}

func (self *SLoadbalancerBackend) GetGlobalId() string {
	return self.GetId()
}

func (self *SLoadbalancerBackend) GetDescription() string {
	return ""
}

func (self *SLoadbalancerBackend) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SLoadbalancerBackend) Refresh() error {
	return nil
}

func (self *SLoadbalancerBackend) GetWeight() int {
	return 0
}

func (self *SLoadbalancerBackend) GetPort() int {
	return self.Port
}

func (self *SLoadbalancerBackend) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SLoadbalancerBackend) GetBackendType() string {
	return api.LB_BACKEND_GUEST
}

func (self *SLoadbalancerBackend) GetBackendRole() string {
	return api.LB_BACKEND_ROLE_DEFAULT
}

func (self *SLoadbalancerBackend) GetBackendId() string {
	if len(self.Instance) > 0 {
		return self.Instance
	}
	r := SResourceBase{
		Name:     "",
		SelfLink: self.Backend.Instance,
	}
	return r.GetGlobalId()
}

func (self *SLoadbalancerBackend) GetIpAddress() string {
	return self.IpAddr
}

func (self *SLoadbalancerBackend) Update(ctx context.Context, opts *cloudprovider.SLoadbalancerBackend) error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadBalancerBackendGroup) GetLoadbalancerBackends() ([]SLoadbalancerBackend, error) {
	if self.backends != nil {
		return self.backends, nil
	}

	_igs, err := self.lb.GetInstanceGroupsMap()
	if err != nil {
		return nil, errors.Wrap(err, "GetInstanceGroupsMap")
	}

	igs := make([]SInstanceGroup, 0)
	for i := range self.backendService.Backends {
		backend := self.backendService.Backends[i]
		if v, ok := _igs[backend.Group]; ok {
			igs = append(igs, v)
		}
	}

	ret := make([]SLoadbalancerBackend, 0)
	for _, backend := range self.backendService.Backends {
		if strings.Contains(backend.Group, "networkEndpointGroups") {
			endpoints := struct {
				Items []struct {
					NetworkEndpoint struct {
						IpAddress string
						Port      int
						Instance  string
					}
				}
			}{}
			err = self.lb.region.PostBySelfId(backend.Group+"/listNetworkEndpoints", &endpoints)
			if err != nil {
				return nil, errors.Wrapf(err, "listNetworkEndpoints")
			}
			for _, item := range endpoints.Items {
				backend := SLoadbalancerBackend{
					lbbg:           self,
					backendService: self.backendService,
					Instance:       item.NetworkEndpoint.Instance,
					IpAddr:         item.NetworkEndpoint.IpAddress,
					Port:           item.NetworkEndpoint.Port,
				}
				ret = append(ret, backend)
			}
		}
	}
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
					backend := SLoadbalancerBackend{
						lbbg:           self,
						backendService: self.backendService,
						instanceGroup:  ig,
						Backend:        bs[n],
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
				for p := range fr.Ports {
					port, _ := strconv.Atoi(fr.Ports[p])
					if port <= 0 {
						continue
					}

					for n := range bs {
						backend := SLoadbalancerBackend{
							lbbg:           self,
							backendService: self.backendService,
							instanceGroup:  ig,
							Backend:        bs[n],
							Port:           port,
						}

						ret = append(ret, backend)
					}
				}
			}
		}
	}

	self.backends = ret
	return ret, nil
}
