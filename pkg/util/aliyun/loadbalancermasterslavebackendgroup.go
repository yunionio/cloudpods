package aliyun

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SLoadbalancerMasterSlaveBackendGroup struct {
	lb *SLoadbalancer

	MasterSlaveServerGroupId   string
	MasterSlaveServerGroupName string
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) GetName() string {
	return backendgroup.MasterSlaveServerGroupName
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) GetId() string {
	return backendgroup.MasterSlaveServerGroupId
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) GetGlobalId() string {
	return backendgroup.GetId()
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) GetStatus() string {
	return models.LB_STATUS_ENABLED
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) IsEmulated() bool {
	return false
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) Refresh() error {
	return nil
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) IsDefault() bool {
	return false
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) GetType() string {
	return models.LB_BACKENDGROUP_TYPE_MASTER_SLAVE
}

func (region *SRegion) GetLoadbalancerMasterSlaveBackendgroups(loadbalancerId string) ([]SLoadbalancerMasterSlaveBackendGroup, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	body, err := region.lbRequest("DescribeMasterSlaveServerGroups", params)
	if err != nil {
		return nil, err
	}
	backendgroups := []SLoadbalancerMasterSlaveBackendGroup{}
	return backendgroups, body.Unmarshal(&backendgroups, "MasterSlaveServerGroups", "MasterSlaveServerGroup")
}

func (region *SRegion) GetLoadbalancerMasterSlaveBackends(backendgroupId string) ([]SLoadbalancerMasterSlaveBackend, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["MasterSlaveServerGroupId"] = backendgroupId
	body, err := region.lbRequest("DescribeMasterSlaveServerGroupAttribute", params)
	if err != nil {
		return nil, err
	}
	backends := []SLoadbalancerMasterSlaveBackend{}
	return backends, body.Unmarshal(&backends, "MasterSlaveBackendServers", "MasterSlaveBackendServer")
}

func (backendgroup *SLoadbalancerMasterSlaveBackendGroup) GetILoadbalancerBackends() ([]cloudprovider.ICloudLoadbalancerBackend, error) {
	backends, err := backendgroup.lb.region.GetLoadbalancerMasterSlaveBackends(backendgroup.MasterSlaveServerGroupId)
	if err != nil {
		return nil, err
	}
	ibackends := []cloudprovider.ICloudLoadbalancerBackend{}
	for i := 0; i < len(backends); i++ {
		backends[i].lbbg = backendgroup
		ibackends = append(ibackends, &backends[i])
	}
	return ibackends, nil
}
