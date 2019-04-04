package ucloud

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/util/secrules"
)

// https://docs.ucloud.cn/api/unet-api/describe_firewall
type SSecurityGroup struct {
	region *SRegion
	vpc    *SVPC // 安全组在UCLOUD实际上与VPC是没有直接关联的。这里的vpc字段只是为了统一，仅仅是标记是哪个VPC在操作该安全组。

	CreateTime    int64  `json:"CreateTime"`
	FWID          string `json:"FWId"`
	GroupID       string `json:"GroupId"`
	Name          string `json:"Name"`
	Remark        string `json:"Remark"`
	ResourceCount int    `json:"ResourceCount"`
	Rule          []Rule `json:"Rule"`
	Tag           string `json:"Tag"`
	Type          string `json:"Type"`
}

func (self *SSecurityGroup) GetProjectId() string {
	return self.region.client.projectId
}

type Rule struct {
	DstPort      string `json:"DstPort"`
	Priority     string `json:"Priority"`
	ProtocolType string `json:"ProtocolType"`
	RuleAction   string `json:"RuleAction"`
	SrcIP        string `json:"SrcIP"`
}

func (self *SSecurityGroup) GetId() string {
	return self.FWID
}

func (self *SSecurityGroup) GetName() string {
	if len(self.Name) == 0 {
		return self.GetId()
	}

	return self.Name
}

func (self *SSecurityGroup) GetGlobalId() string {
	return self.GetId()
}

func (self *SSecurityGroup) GetStatus() string {
	return ""
}

func (self *SSecurityGroup) Refresh() error {
	if new, err := self.region.GetSecurityGroupById(self.GetId()); err != nil {
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
	return self.Remark
}

func (self *SSecurityGroup) UcloudSecRuleToOnecloud(rule Rule) secrules.SecurityRule {
	secrule := secrules.SecurityRule{}
	switch rule.Priority {
	case "HIGH":
		secrule.Priority = 90
	case "MEDIUM":
		secrule.Priority = 60
	case "LOW":
		secrule.Priority = 30
	default:
		secrule.Priority = 1
	}

	switch rule.RuleAction {
	case "ACCEPT":
		secrule.Action = secrules.SecurityRuleAllow
	case "DROP":
		secrule.Action = secrules.SecurityRuleDeny
	default:
		secrule.Action = secrules.SecurityRuleDeny
	}

	_, ipNet, err := net.ParseCIDR(rule.SrcIP)
	if err != nil {
		log.Errorln(err)
	}

	secrule.IPNet = ipNet
	secrule.Protocol = strings.ToLower(rule.ProtocolType)
	secrule.Direction = secrules.SecurityRuleIngress
	if rule.DstPort == "" {
		secrule.PortStart = -1
		secrule.PortEnd = -1
	} else if strings.Contains(rule.DstPort, "-") {
		segs := strings.Split(rule.DstPort, "-")
		s, err := strconv.Atoi(segs[0])
		if err != nil {
			log.Errorln(err)
		}
		e, err := strconv.Atoi(segs[1])
		if err != nil {
			log.Errorln(err)
		}
		secrule.PortStart = s
		secrule.PortEnd = e
	} else {
		port, err := strconv.Atoi(rule.DstPort)
		if err != nil {
			log.Errorln(err)
		}

		secrule.PortStart = port
		secrule.PortEnd = port
	}

	return secrule
}

// https://docs.ucloud.cn/network/firewall/firewall
// 貌似没有出方向规则
// todo: fix me
func (self *SSecurityGroup) GetRules() ([]secrules.SecurityRule, error) {
	rules := make([]secrules.SecurityRule, 0)
	for _, r := range self.Rule {
		rule := self.UcloudSecRuleToOnecloud(r)
		rules = append(rules, rule)
	}

	return rules, nil
}

func (self *SSecurityGroup) GetVpcId() string {
	// 无vpc关联的安全组统一返回normal
	return models.NORMAL_VPC_ID
}

func (self *SRegion) GetSecurityGroupById(secGroupId string) (*SSecurityGroup, error) {
	secgroups, err := self.GetSecurityGroups(secGroupId, "")
	if err != nil {
		return nil, err
	}

	if len(secgroups) == 1 {
		return &secgroups[0], nil
	} else if len(secgroups) == 0 {
		return nil, cloudprovider.ErrNotFound
	} else {
		return nil, fmt.Errorf("GetSecurityGroupById %s %d found", secGroupId, len(secgroups))
	}
}

func (self *SRegion) CreateSecurityGroup(name, description string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

// https://docs.ucloud.cn/api/unet-api/describe_firewall
func (self *SRegion) GetSecurityGroups(secGroupId string, resourceId string) ([]SSecurityGroup, error) {
	secgroups := make([]SSecurityGroup, 0)

	params := NewUcloudParams()
	if len(secGroupId) > 0 {
		params.Set("FWId", secGroupId)
	}

	if len(resourceId) > 0 {
		params.Set("ResourceId", resourceId)
		params.Set("ResourceType", "uhost") //  默认只支持"uhost"，云主机
	}
	err := self.DoListAll("DescribeFirewall", params, &secgroups)
	if err != nil {
		return nil, err
	}

	for i := range secgroups {
		secgroups[i].region = self
	}

	return secgroups, nil
}
