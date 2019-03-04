package aliyun

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/consts"
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
	return ""
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
	return consts.LB_BACKEND_GUEST
}

func (backend *SLoadbalancerBackend) GetBackendRole() string {
	return consts.LB_BACKEND_ROLE_DEFAULT
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
