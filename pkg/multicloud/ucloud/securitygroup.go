// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ucloud

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
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
// 只有入方向规则
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
	return api.NORMAL_VPC_ID
}

func (self *SRegion) GetSecurityGroupById(secGroupId string) (*SSecurityGroup, error) {
	secgroups, err := self.GetSecurityGroups(secGroupId, "", "")
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

func randomString(prefix string, length int) string {
	bytes := []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	result := []byte{}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < length; i++ {
		result = append(result, bytes[r.Intn(len(bytes))])
	}
	return prefix + string(result)
}

func (self *SRegion) CreateDefaultSecurityGroup(name, description string) (string, error) {
	// 减少安全组名称冲突
	name = randomString(name, 4)
	return self.CreateSecurityGroup(name, description, []string{"TCP|1-65535|0.0.0.0/0|ACCEPT|LOW", "UDP|1-65535|0.0.0.0/0|ACCEPT|LOW", "ICMP||0.0.0.0/0|ACCEPT|LOW"})
}

// https://docs.ucloud.cn/api/unet-api/create_firewall
func (self *SRegion) CreateSecurityGroup(name, description string, rules []string) (string, error) {
	params := NewUcloudParams()
	params.Set("Name", name)
	params.Set("Remark", description)
	if len(rules) == 0 {
		return "", fmt.Errorf("CreateSecurityGroup required at least one rule")
	}

	for i, rule := range rules {
		params.Set(fmt.Sprintf("Rule.%d", i), rule)
	}

	type Firewall struct {
		FWId string
	}

	firewall := Firewall{}
	err := self.DoAction("CreateFirewall", params, &firewall)
	if err != nil {
		return "", err
	}

	return firewall.FWId, nil
}

// https://docs.ucloud.cn/api/unet-api/describe_firewall
func (self *SRegion) GetSecurityGroups(secGroupId string, resourceId string, name string) ([]SSecurityGroup, error) {
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
		if strings.Contains(err.Error(), "not exist") {
			return nil, cloudprovider.ErrNotFound
		}
		return nil, err
	}

	result := []SSecurityGroup{}

	for i := range secgroups {
		if len(name) == 0 || secgroups[i].Name == name {
			secgroups[i].region = self
			result = append(result, secgroups[i])
		}
	}

	return result, nil
}

func (self *SSecurityGroup) SyncRules(rules []secrules.SecurityRule) error {
	// 如果是空规则，onecloud。默认拒绝所有流量
	if len(rules) == 0 {
		_, IpNet, _ := net.ParseCIDR("0.0.0.0/0")
		rules = []secrules.SecurityRule{{
			Priority:    0,
			Action:      secrules.SecurityRuleDeny,
			IPNet:       IpNet,
			Protocol:    secrules.PROTO_ANY,
			Direction:   secrules.SecurityRuleIngress,
			PortStart:   0,
			PortEnd:     0,
			Ports:       nil,
			Description: "",
		}}
	}
	return self.region.syncSecgroupRules(self.FWID, rules)
}

func (self *SSecurityGroup) Delete() error {
	return self.region.DeleteSecurityGroup(self.FWID)
}
