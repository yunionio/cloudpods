package aliyun

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type Rule struct {
	RuleId   string
	RuleName string
	Domain   string
	Url      string
}

type Listener struct {
	Protocol string
	Port     int
}

type AssociatedObject struct {
	Rules     []Rule
	Listeners []Listener
}

type SLoadbalancerBackendgroup struct {
	lb *SLoadbalancer

	VServerGroupId    string
	VServerGroupName  string
	AssociatedObjects []AssociatedObject
}

func (backendgroup *SLoadbalancerBackendgroup) GetName() string {
	return backendgroup.VServerGroupName
}

func (backendgroup *SLoadbalancerBackendgroup) GetId() string {
	return backendgroup.VServerGroupId
}

func (backendgroup *SLoadbalancerBackendgroup) GetGlobalId() string {
	return backendgroup.VServerGroupId
}

func (backendgroup *SLoadbalancerBackendgroup) GetStatus() string {
	return ""
}

func (backendgroup *SLoadbalancerBackendgroup) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (backendgroup *SLoadbalancerBackendgroup) IsEmulated() bool {
	return false
}

func (backendgroup *SLoadbalancerBackendgroup) Refresh() error {
	return nil
}

func (region *SRegion) GetLoadbalancerBackendgroups(lbId string) ([]SLoadbalancerBackendgroup, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	if len(lbId) > 0 {
		params["LoadBalancerId"] = lbId
	}
	body, err := region.lbRequest("DescribeVServerGroups", params)
	if err != nil {
		return nil, err
	}
	backendgroups := []SLoadbalancerBackendgroup{}
	return backendgroups, body.Unmarshal(&backendgroups, "VServerGroups", "VServerGroup")
}

func (backendgroup *SLoadbalancerBackendgroup) GetILoadbalancerBackends() ([]cloudprovider.ICloudLoadbalancerBackend, error) {
	backends, err := backendgroup.lb.region.GetLoadbalancerBackends(backendgroup.VServerGroupId)
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
