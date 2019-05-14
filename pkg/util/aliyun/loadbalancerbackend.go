package aliyun

import (
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SLoadbalancerBackend struct {
	lbbg *SLoadbalancerBackendGroup

	ServerId string
	Port     int
	Weight   int
}

func (backend *SLoadbalancerBackend) GetName() string {
	return backend.ServerId
}

func (backend *SLoadbalancerBackend) GetId() string {
	return fmt.Sprintf("%s/%s", backend.lbbg.VServerGroupId, backend.ServerId)
}

func (backend *SLoadbalancerBackend) GetGlobalId() string {
	return backend.GetId()
}

func (backend *SLoadbalancerBackend) GetStatus() string {
	return api.LB_STATUS_ENABLED
}

func (backend *SLoadbalancerBackend) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (backend *SLoadbalancerBackend) IsEmulated() bool {
	return false
}

func (backend *SLoadbalancerBackend) Refresh() error {
	loadbalancerBackends, err := backend.lbbg.lb.region.GetLoadbalancerBackends(backend.lbbg.VServerGroupId)
	if err != nil {
		return err
	}
	for _, loadbalancerBackend := range loadbalancerBackends {
		if loadbalancerBackend.ServerId == backend.ServerId {
			return jsonutils.Update(backend, loadbalancerBackend)
		}
	}
	return cloudprovider.ErrNotFound
}

func (backend *SLoadbalancerBackend) GetWeight() int {
	return backend.Weight
}

func (backend *SLoadbalancerBackend) GetPort() int {
	return backend.Port
}

func (backend *SLoadbalancerBackend) GetBackendType() string {
	return api.LB_BACKEND_GUEST
}

func (backend *SLoadbalancerBackend) GetBackendRole() string {
	return api.LB_BACKEND_ROLE_DEFAULT
}

func (backend *SLoadbalancerBackend) GetBackendId() string {
	return backend.ServerId
}

func (region *SRegion) GetLoadbalancerBackends(backendgroupId string) ([]SLoadbalancerBackend, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["VServerGroupId"] = backendgroupId
	body, err := region.lbRequest("DescribeVServerGroupAttribute", params)
	if err != nil {
		return nil, err
	}
	backends := []SLoadbalancerBackend{}
	return backends, body.Unmarshal(&backends, "BackendServers", "BackendServer")
}

func (backend *SLoadbalancerBackend) GetProjectId() string {
	return ""
}

func (backend *SLoadbalancerBackend) SyncConf(port, weight int) error {
	err := backend.lbbg.lb.region.RemoveBackendVServer(backend.lbbg.lb.LoadBalancerId, backend.lbbg.VServerGroupId, backend.ServerId, backend.Port)
	if err != nil {
		return err
	}
	return backend.lbbg.lb.region.AddBackendVServer(backend.lbbg.lb.LoadBalancerId, backend.lbbg.VServerGroupId, backend.ServerId, weight, port)
}
