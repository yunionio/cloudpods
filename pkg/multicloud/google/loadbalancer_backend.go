package google

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SLoadbalancerBackend struct {
	lbbg *SLoadBalancerBackendGroup

	backendService SBackendServices       //
	instanceGroup  SInstanceGroup         // 实例组
	Backend        SInstanceGroupInstance // backend

	Port int `json:"port"`
}

func (self *SLoadbalancerBackend) GetId() string {
	return fmt.Sprintf("%s::%s::%s::%d", self.lbbg.GetGlobalId(), self.instanceGroup.GetGlobalId(), self.GetBackendId(), self.Port)
}

func (self *SLoadbalancerBackend) GetName() string {
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

func (self *SLoadbalancerBackend) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (self *SLoadbalancerBackend) Refresh() error {
	return nil
}

func (self *SLoadbalancerBackend) IsEmulated() bool {
	return true
}

func (self *SLoadbalancerBackend) GetSysTags() map[string]string {
	return nil
}

func (self *SLoadbalancerBackend) GetTags() (map[string]string, error) {
	return nil, nil
}

func (self *SLoadbalancerBackend) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotSupported
}

func (self *SLoadbalancerBackend) GetProjectId() string {
	return self.lbbg.GetProjectId()
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
	r := SResourceBase{
		Name:     "",
		SelfLink: self.Backend.Instance,
	}
	return r.GetGlobalId()
}

func (self *SLoadbalancerBackend) GetIpAddress() string {
	return ""
}

func (self *SLoadbalancerBackend) SyncConf(ctx context.Context, port, weight int) error {
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
