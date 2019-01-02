package openstack

import (
	"net"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/pkg/utils"
)

type SSecurityGroupRule struct {
	Direction       string
	Ethertype       string
	ID              string
	PortRangeMax    int
	PortRangeMin    int
	Protocol        string
	RemoteGroupID   string
	RemoteIpPrefix  string
	SecurityGroupID string
	ProjectID       string
	RevisionNumber  int
	Tags            []string
	TenantID        string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Description     string
}

type SSecurityGroup struct {
	vpc *SVpc

	Description        string
	ID                 string
	Name               string
	SecurityGroupRules []SSecurityGroupRule
	ProjectID          string
	RevisionNumber     int
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Tags               []string
	TenantID           string
}

func (region *SRegion) GetSecurityGroup(secgroupId string) (*SSecurityGroup, error) {
	_, resp, err := region.Get("network", "/v2.0/security-groups/"+secgroupId, "", nil)
	if err != nil {
		return nil, err
	}
	secgroup := &SSecurityGroup{}
	return secgroup, resp.Unmarshal(secgroup, "security_group")
}

func (region *SRegion) GetSecurityGroups() ([]SSecurityGroup, error) {
	_, resp, err := region.Get("network", "/v2.0/security-groups", "", nil)
	if err != nil {
		return nil, err
	}
	secgroups := []SSecurityGroup{}
	return secgroups, resp.Unmarshal(&secgroups, "security_groups")
}

func (secgroup *SSecurityGroup) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (secgroup *SSecurityGroup) GetVpcId() string {
	return "normal"
}

func (secgroup *SSecurityGroup) GetId() string {
	return secgroup.ID
}

func (secgroup *SSecurityGroup) GetGlobalId() string {
	return secgroup.ID
}

func (secgroup *SSecurityGroup) GetDescription() string {
	return secgroup.Description
}

func (secgroup *SSecurityGroup) GetName() string {
	if len(secgroup.Name) > 0 {
		return secgroup.Name
	}
	return secgroup.ID
}

func (secgroup *SSecurityGroupRule) String() string {
	rules := secgroup.toRules()
	result := []string{}
	for _, rule := range rules {
		result = append(result, rule.String())
	}
	return strings.Join(result, ";")
}

func (secgrouprule *SSecurityGroupRule) toRules() []secrules.SecurityRule {
	rules := []secrules.SecurityRule{}
	// 暂时忽略IPv6安全组规则,忽略远端也是安全组的规则
	if secgrouprule.Ethertype != "IPv4" || len(secgrouprule.RemoteGroupID) > 0 {
		return rules
	}
	rule := secrules.SecurityRule{
		Direction:   secrules.DIR_IN,
		Action:      secrules.SecurityRuleAllow,
		Description: secgrouprule.Description,
		Priority:    1,
	}
	if utils.IsInStringArray(secgrouprule.Protocol, []string{"", "0", "any"}) {
		rule.Protocol = secrules.PROTO_ANY
	} else if utils.IsInStringArray(secgrouprule.Protocol, []string{"6", "tcp"}) {
		rule.Protocol = secrules.PROTO_TCP
	} else if utils.IsInStringArray(secgrouprule.Protocol, []string{"17", "udp"}) {
		rule.Protocol = secrules.PROTO_UDP
	} else if utils.IsInStringArray(secgrouprule.Protocol, []string{"1", "icmp"}) {
		rule.Protocol = secrules.PROTO_ICMP
	} else {
		return rules
	}
	if secgrouprule.Direction == "egress" {
		rule.Direction = secrules.DIR_OUT
	}
	if len(secgrouprule.RemoteIpPrefix) == 0 {
		secgrouprule.RemoteIpPrefix = "0.0.0.0/0"
	}
	_, ipnet, err := net.ParseCIDR(secgrouprule.RemoteIpPrefix)
	if err != nil {
		return rules
	}
	rule.IPNet = ipnet
	if secgrouprule.PortRangeMax > 0 && secgrouprule.PortRangeMin > 0 {
		if secgrouprule.PortRangeMax == secgrouprule.PortRangeMin {
			rule.Ports = []int{secgrouprule.PortRangeMax}
		} else {
			rule.PortStart = secgrouprule.PortRangeMin
			rule.PortEnd = secgrouprule.PortRangeMax
		}
	}
	if err := rule.ValidateRule(); err != nil {
		return rules
	}
	return []secrules.SecurityRule{rule}
}

func (secgroup *SSecurityGroup) GetRules() ([]secrules.SecurityRule, error) {
	rules := []secrules.SecurityRule{}
	for _, rule := range secgroup.SecurityGroupRules {
		subRules := rule.toRules()
		rules = append(rules, subRules...)
	}
	return rules, nil
}

func (secgroup *SSecurityGroup) GetStatus() string {
	return ""
}

func (secgroup *SSecurityGroup) IsEmulated() bool {
	return false
}

func (secgroup *SSecurityGroup) Refresh() error {
	new, err := secgroup.vpc.region.GetSecurityGroup(secgroup.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(secgroup, new)
}

func (secgroup *SRegion) SyncSecurityGroup(secgroupId string, vpcId string, name string, desc string, rules []secrules.SecurityRule) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (region *SRegion) DeleteSecurityGroup(vpcId, secGroupId string) error {
	return cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateSecurityGroup(name, description string) (*SSecurityGroup, error) {
	return nil, cloudprovider.ErrNotImplemented
}
