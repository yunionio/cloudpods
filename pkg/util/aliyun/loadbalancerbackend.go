package aliyun

import (
	"fmt"

	"yunion.io/x/jsonutils"
)

type SLoadbalancerBackend struct {
	lbbg *SLoadbalancerBackendgroup

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
	return nil
}

func (backend *SLoadbalancerBackend) GetWeight() int {
	return backend.Weight
}

func (backend *SLoadbalancerBackend) GetPort() int {
	return backend.Port
}

func (backend *SLoadbalancerBackend) GetAddress() string {
	return ""
}

func (backend *SLoadbalancerBackend) GetBackendType() string {
	return ""
}

func (backend *SLoadbalancerBackend) GetBackendId() string {
	return ""
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
