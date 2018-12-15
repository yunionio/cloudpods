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

type Rules struct {
	Rule []Rule
}

type Listener struct {
	Protocol string
	Port     int
}

type Listeners struct {
	Listener []Listener
}

type AssociatedObjects struct {
	Rules     Rules
	Listeners Listeners
}

type SLoadbalancerBackendGroup struct {
	lb *SLoadbalancer

	VServerGroupId    string
	VServerGroupName  string
	AssociatedObjects AssociatedObjects
}

func (backendgroup *SLoadbalancerBackendGroup) GetName() string {
	return backendgroup.VServerGroupName
}

func (backendgroup *SLoadbalancerBackendGroup) GetId() string {
	return backendgroup.VServerGroupId
}

func (backendgroup *SLoadbalancerBackendGroup) GetGlobalId() string {
	return backendgroup.VServerGroupId
}

func (backendgroup *SLoadbalancerBackendGroup) GetStatus() string {
	return ""
}

func (backendgroup *SLoadbalancerBackendGroup) IsDefault() bool {
	return true
}

func (backendgroup *SLoadbalancerBackendGroup) GetType() string {
	return "standard"
}

func (backendgroup *SLoadbalancerBackendGroup) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (backendgroup *SLoadbalancerBackendGroup) IsEmulated() bool {
	return false
}

func (backendgroup *SLoadbalancerBackendGroup) Refresh() error {
	loadbalancerBackendgroups, err := backendgroup.lb.region.GetLoadbalancerBackendgroups(backendgroup.lb.LoadBalancerId)
	if err != nil {
		return err
	}
	for _, loadbalancerBackendgroup := range loadbalancerBackendgroups {
		if loadbalancerBackendgroup.VServerGroupId == backendgroup.VServerGroupId {
			return jsonutils.Update(backendgroup, loadbalancerBackendgroup)
		}
	}
	return cloudprovider.ErrNotFound
}

func (region *SRegion) GetLoadbalancerBackendgroups(loadbalancerId string) ([]SLoadbalancerBackendGroup, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	body, err := region.lbRequest("DescribeVServerGroups", params)
	if err != nil {
		return nil, err
	}
	backendgroups := []SLoadbalancerBackendGroup{}
	return backendgroups, body.Unmarshal(&backendgroups, "VServerGroups", "VServerGroup")
}

func (backendgroup *SLoadbalancerBackendGroup) GetILoadbalancerBackends() ([]cloudprovider.ICloudLoadbalancerBackend, error) {
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
