package huawei

/*
https://support.huaweicloud.com/usermanual-vpc/zh-cn_topic_0073379079.html
安全组的限制
默认情况下，一个用户可以创建100个安全组。
默认情况下，一个安全组最多只允许拥有50条安全组规则。
默认情况下，一个弹性云服务器或辅助网卡最多只能被添加到5个安全组中。
在创建私网弹性负载均衡时，需要选择弹性负载均衡所在的安全组。请勿删除默认规则或者确保满足以下规则：
出方向：允许发往同一个安全组的报文可以通过，或者允许对端负载均衡器报文通过。
入方向：允许来自同一个安全组的报文可以通过，或者允许对端负载均衡器报文通过。
*/

import (
	"net"
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/secrules"
)

type SecurityGroupRule struct {
	Direction       string `json:"direction"`
	Ethertype       string `json:"ethertype"`
	ID              string `json:"id"`
	Description     string `json:"description"`
	SecurityGroupID string `json:"security_group_id"`
	RemoteGroupID   string `json:"remote_group_id"`
}

type SecurityGroupRuleDetail struct {
	Direction       string `json:"direction"`
	Ethertype       string `json:"ethertype"`
	ID              string `json:"id"`
	Description     string `json:"description"`
	PortRangeMax    int64  `json:"port_range_max"`
	PortRangeMin    int64  `json:"port_range_min"`
	Protocol        string `json:"protocol"`
	RemoteGroupID   string `json:"remote_group_id"`
	RemoteIPPrefix  string `json:"remote_ip_prefix"`
	SecurityGroupID string `json:"security_group_id"`
	TenantID        string `json:"tenant_id"`
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090615.html
type SSecurityGroup struct {
	vpc *SVpc

	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	Description         string              `json:"description"`
	VpcID               string              `json:"vpc_id"`
	EnterpriseProjectID string              `json:"enterprise_project_id "`
	SecurityGroupRules  []SecurityGroupRule `json:"security_group_rules"`
}

func (self *SSecurityGroup) GetId() string {
	return self.ID
}

func (self *SSecurityGroup) GetVpcId() string {
	// 无vpc关联的安全组统一返回normal
	if len(self.VpcID) == 0 {
		return "normal"
	}

	return self.VpcID
}

func (self *SSecurityGroup) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.ID
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.ID
}

func (self *SSecurityGroup) GetStatus() string {
	return ""
}

func (self *SSecurityGroup) Refresh() error {
	if new, err := self.vpc.region.GetSecurityGroupDetails(self.GetId()); err != nil {
		return err
	} else {
		return jsonutils.Update(self, new)
	}
}

func (self *SSecurityGroup) IsEmulated() bool {
	return false
}

func (self *SSecurityGroup) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	return data
}

func (self *SSecurityGroup) GetDescription() string {
	return self.Description
}

func (self *SSecurityGroup) GetRules() ([]secrules.SecurityRule, error) {
	rules := make([]secrules.SecurityRule, 0)
	for _, r := range self.SecurityGroupRules {
		// 忽略了源地址是安全组的规则
		if len(r.RemoteGroupID) > 0 {
			continue
		}

		rule, err := self.GetSecurityRule(r.ID)
		if err != nil {
			return rules, err
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

func (self *SSecurityGroup) GetSecurityRule(ruleId string) (secrules.SecurityRule, error) {
	remoteRule := SecurityGroupRuleDetail{}
	err := DoGet(self.vpc.region.ecsClient.SecurityGroupRules.Get, ruleId, nil, &remoteRule)
	if err != nil {
		return secrules.SecurityRule{}, err
	}

	var direction secrules.TSecurityRuleDirection
	if remoteRule.Direction == "ingress" {
		direction = secrules.SecurityRuleIngress
	} else {
		direction = secrules.SecurityRuleEgress
	}

	protocol := "any"
	if remoteRule.Protocol != "" {
		protocol = remoteRule.Protocol
	}

	// todo: 没考虑ipv6。可能报错
	ipNet := &net.IPNet{}
	if len(remoteRule.RemoteIPPrefix) > 0 {
		_, ipNet, err = net.ParseCIDR(remoteRule.RemoteIPPrefix)
	}

	rule := secrules.SecurityRule{
		Priority:    0,
		Action:      secrules.SecurityRuleAllow,
		IPNet:       ipNet,
		Protocol:    protocol,
		Direction:   direction,
		PortStart:   int(remoteRule.PortRangeMin),
		PortEnd:     int(remoteRule.PortRangeMax),
		Ports:       nil,
		Description: remoteRule.Description,
	}
	return rule, err
}

func (self *SRegion) GetSecurityGroupDetails(secGroupId string) (*SSecurityGroup, error) {
	securitygroup := SSecurityGroup{}
	err := DoGet(self.ecsClient.SecurityGroups.Get, secGroupId, nil, &securitygroup)
	return &securitygroup, err
}

func (self *SRegion) GetSecurityGroups(vpcId string, limit int, marker string) ([]SSecurityGroup, int, error) {
	querys := map[string]string{}
	if len(vpcId) > 0 {
		querys["vpc_id"] = vpcId
	}

	if len(marker) > 0 {
		querys["marker"] = marker
	}

	querys["limit"] = strconv.Itoa(limit)
	securitygroups := make([]SSecurityGroup, 0)
	err := DoList(self.ecsClient.SecurityGroups.List, querys, &securitygroups)
	return securitygroups, len(securitygroups), err
}
