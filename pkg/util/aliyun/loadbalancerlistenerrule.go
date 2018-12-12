package aliyun

import (
	"fmt"

	"yunion.io/x/jsonutils"
)

type SLoadbalancerListenerRule struct {
	httpListener  *SLoadbalancerHTTPListener
	httpsListener *SLoadbalancerHTTPSListener

	Domain         string
	ListenerSync   string
	RuleId         string
	RuleName       string
	Url            string
	VServerGroupId string
}

func (lbr *SLoadbalancerListenerRule) GetName() string {
	return lbr.RuleName
}

func (lbr *SLoadbalancerListenerRule) GetId() string {
	return lbr.RuleId
}

func (lbr *SLoadbalancerListenerRule) GetGlobalId() string {
	return lbr.RuleId
}

func (lbr *SLoadbalancerListenerRule) GetStatus() string {
	return ""
}

func (lbr *SLoadbalancerListenerRule) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (lbr *SLoadbalancerListenerRule) IsEmulated() bool {
	return false
}

func (lbr *SLoadbalancerListenerRule) Refresh() error {
	return nil
}

func (lbr *SLoadbalancerListenerRule) GetDomain() string {
	return lbr.Domain
}

func (lbr *SLoadbalancerListenerRule) GetPath() string {
	return lbr.Url
}

func (lbr *SLoadbalancerListenerRule) GetBackendGroupId() string {
	return lbr.VServerGroupId
}

func (region *SRegion) GetLoadbalancerListenerRules(loadbalancerId string, listenerPort int) ([]SLoadbalancerListenerRule, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["LoadBalancerId"] = loadbalancerId
	params["ListenerPort"] = fmt.Sprintf("%d", listenerPort)
	body, err := region.lbRequest("DescribeRules", params)
	if err != nil {
		return nil, err
	}
	rules := []SLoadbalancerListenerRule{}
	return rules, body.Unmarshal(&rules, "Rules", "Rule")
}
