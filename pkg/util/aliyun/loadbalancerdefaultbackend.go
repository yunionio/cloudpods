package aliyun

import (
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SLoadbalancerDefaultBackend struct {
	lbbg *SLoadbalancerDefaultBackendGroup

	ServerId string
	Weight   int
}

func (backend *SLoadbalancerDefaultBackend) GetName() string {
	return backend.ServerId
}

func (backend *SLoadbalancerDefaultBackend) GetId() string {
	return fmt.Sprintf("%s/%s", backend.lbbg.lb.LoadBalancerId, backend.ServerId)
}

func (backend *SLoadbalancerDefaultBackend) GetGlobalId() string {
	return backend.GetId()
}

func (backend *SLoadbalancerDefaultBackend) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (backend *SLoadbalancerDefaultBackend) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (backend *SLoadbalancerDefaultBackend) IsEmulated() bool {
	return false
}

func (backend *SLoadbalancerDefaultBackend) Refresh() error {
	return nil
}

func (backend *SLoadbalancerDefaultBackend) GetWeight() int {
	return backend.Weight
}

func (backend *SLoadbalancerDefaultBackend) GetPort() int {
	return 0
}

func (backend *SLoadbalancerDefaultBackend) GetBackendType() string {
	return api.LB_BACKEND_GUEST
}

func (backend *SLoadbalancerDefaultBackend) GetBackendRole() string {
	return api.LB_BACKEND_ROLE_DEFAULT
}

func (backend *SLoadbalancerDefaultBackend) GetBackendId() string {
	return backend.ServerId
}

func (backend *SLoadbalancerDefaultBackend) GetProjectId() string {
	return ""
}

func (backend *SLoadbalancerDefaultBackend) SyncConf(port, weight int) error {
	params := map[string]string{}
	params["RegionId"] = backend.lbbg.lb.region.RegionId
	params["LoadBalancerId"] = backend.lbbg.lb.LoadBalancerId
	loadbalancer, err := backend.lbbg.lb.region.GetLoadbalancerDetail(backend.lbbg.lb.LoadBalancerId)
	if err != nil {
		return err
	}
	servers := jsonutils.NewArray()
	for i := 0; i < len(loadbalancer.BackendServers.BackendServer); i++ {
		_backend := loadbalancer.BackendServers.BackendServer[i]
		_backend.lbbg = backend.lbbg
		if _backend.GetGlobalId() == backend.GetGlobalId() {
			_backend.Weight = weight
		}
		servers.Add(
			jsonutils.Marshal(
				map[string]string{
					"ServerId": _backend.ServerId,
					"Weight":   fmt.Sprintf("%d", _backend.Weight),
				},
			))
	}

	params["BackendServers"] = servers.String()
	_, err = backend.lbbg.lb.region.lbRequest("SetBackendServers", params)
	return err
}
