package zstack

import (
	"net"
	"strings"

	"yunion.io/x/onecloud/pkg/cloudprovider"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/pkg/util/secrules"
)

type SSecurityGroupRule struct {
	ZStackBasic
	SecurityGroupUUID       string `json:"securityGroupUuid"`
	Type                    string `json:"type"`
	IPVersion               int    `json:"ipVersion"`
	StartPort               int    `json:"startPort"`
	EndPort                 int    `json:"endPort"`
	Protocol                string `json:"protocol"`
	State                   string `json:"state"`
	AllowedCIDR             string `json:"allowedCidr"`
	RemoteSecurityGroupUUID string `json:"remoteSecurityGroupUuid"`
	ZStackTime
}

type SSecurityGroup struct {
	region *SRegion

	ZStackBasic
	State     string `json:"state"`
	IPVersion int    `json:"ipVersion"`
	ZStackTime
	InternalID             int                  `json:"internalId"`
	Rules                  []SSecurityGroupRule `json:"rules"`
	AttachedL3NetworkUUIDs []string             `json:"attachedL3NetworkUuids"`
}

func (region *SRegion) GetSecurityGroup(secgroupId string) (*SSecurityGroup, error) {
	secgroups, err := region.GetSecurityGroups(secgroupId, "")
	if err != nil {
		return nil, err
	}
	if len(secgroups) == 1 {
		if secgroups[0].UUID == secgroupId {
			return &secgroups[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}
	if len(secgroups) == 0 || len(secgroupId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (region *SRegion) GetSecurityGroups(secgroupId string, instanceId string) ([]SSecurityGroup, error) {
	secgroups := []SSecurityGroup{}
	params := []string{}
	if len(secgroupId) > 0 {
		params = append(params, "q=uuid="+secgroupId)
	}
	if len(instanceId) > 0 {
		params = append(params, "q=vmNic.vmInstanceUuid="+instanceId)
	}
	err := region.client.listAll("security-groups", params, &secgroups)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(secgroups); i++ {
		secgroups[i].region = region
	}
	return secgroups, nil
}

func (self *SSecurityGroup) GetVpcId() string {
	return api.NORMAL_VPC_ID
}

func (self *SSecurityGroup) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	return data
}

func (self *SSecurityGroup) GetId() string {
	return self.UUID
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.UUID
}

func (self *SSecurityGroup) GetDescription() string {
	return self.Description
}

func (self *SSecurityGroup) GetRules() ([]secrules.SecurityRule, error) {
	rules := []secrules.SecurityRule{}
	priority := 100
	outRuleCount := 0
	for i := 0; i < len(self.Rules); i++ {
		if self.Rules[i].IPVersion == 4 {
			rule := secrules.SecurityRule{
				Direction: secrules.DIR_IN,
				Action:    secrules.SecurityRuleAllow,
				Priority:  priority,
				Protocol:  secrules.PROTO_ANY,
				PortStart: self.Rules[i].StartPort,
				PortEnd:   self.Rules[i].EndPort,
			}
			_, ipNet, err := net.ParseCIDR(self.Rules[i].AllowedCIDR)
			if err != nil {
				return nil, err
			}
			rule.IPNet = ipNet
			if self.Rules[i].Type == "Egress" {
				rule.Direction = secrules.DIR_OUT
				outRuleCount++
			}
			if self.Rules[i].Protocol != "ALL" {
				rule.Protocol = strings.ToLower(self.Rules[i].Protocol)
			}
			rules = append(rules, rule)
			priority--
		}
	}
	if outRuleCount != 0 {
		rule := secrules.MustParseSecurityRule("out:deny any")
		rule.Priority = 1
		rules = append(rules, *rule)
	}
	return rules, nil
}

func (self *SSecurityGroup) GetName() string {
	return self.Name
}

func (self *SSecurityGroup) GetStatus() string {
	return ""
}

func (self *SSecurityGroup) IsEmulated() bool {
	return false
}

func (self *SSecurityGroup) Refresh() error {
	new, err := self.region.GetSecurityGroup(self.UUID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SSecurityGroup) GetProjectId() string {
	return ""
}
