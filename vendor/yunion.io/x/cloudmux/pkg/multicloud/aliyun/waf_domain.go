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

package aliyun

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SWafDomain struct {
	multicloud.SResourceBase
	AliyunTags
	region *SRegion

	insId           string
	name            string
	HttpToUserIp    int           `json:"HttpToUserIp"`
	HttpPort        []int         `json:"HttpPort"`
	IsAccessProduct int           `json:"IsAccessProduct"`
	Resourcegroupid string        `json:"ResourceGroupId"`
	Readtime        int           `json:"ReadTime"`
	SourceIps       []string      `json:"SourceIps"`
	Ipfollowstatus  int           `json:"IpFollowStatus"`
	Clustertype     int           `json:"ClusterType"`
	Loadbalancing   int           `json:"LoadBalancing"`
	Cname           string        `json:"Cname"`
	Writetime       int           `json:"WriteTime"`
	HTTP2Port       []interface{} `json:"Http2Port"`
	Version         int           `json:"Version"`
	Httpsredirect   int           `json:"HttpsRedirect"`
	Connectiontime  int           `json:"ConnectionTime"`
	Accesstype      string        `json:"AccessType"`
	HttpsPort       []int         `json:"HttpsPort"`
	AccessHeaders   []string
}

func (self *SRegion) DescribeDomain(id, domain string) (*SWafDomain, error) {
	params := map[string]string{
		"RegionId":   self.RegionId,
		"InstanceId": id,
		"Domain":     domain,
	}
	resp, err := self.wafRequest("DescribeDomain", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeDomain")
	}
	ret := &SWafDomain{region: self, name: domain, insId: id}
	err = resp.Unmarshal(ret, "Domain")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return ret, nil
}

func (self *SRegion) DeleteDomain(id, domain string) error {
	params := map[string]string{
		"RegionId":   self.RegionId,
		"InstanceId": id,
		"Domain":     domain,
	}
	_, err := self.wafRequest("DeleteDomain", params)
	return errors.Wrapf(err, "DeleteDomain")
}

func (self *SRegion) DescribeDomainNames(id string) ([]string, error) {
	params := map[string]string{
		"RegionId":   self.RegionId,
		"InstanceId": id,
	}
	resp, err := self.wafRequest("DescribeDomainNames", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeDomainNames")
	}
	domains := []string{}
	err = resp.Unmarshal(&domains, "DomainNames")
	return domains, errors.Wrapf(err, "resp.Unmarshal")
}

func (self *SRegion) SetDomainRuleGroup(insId, domain, ruleGroupId string) error {
	params := map[string]string{
		"RegionId":    self.RegionId,
		"InstanceId":  insId,
		"Domains":     domain,
		"RuleGroupId": ruleGroupId,
	}
	_, err := self.wafRequest("SetDomainRuleGroup", params)
	return err
}

func (self *SRegion) DescribeDomainRuleGroup(insId, domain string) (string, error) {
	params := map[string]string{
		"RegionId":   self.RegionId,
		"InstanceId": insId,
		"Domain":     domain,
	}
	resp, err := self.wafRequest("DescribeDomainRuleGroup", params)
	if err != nil {
		return "", errors.Wrapf(err, "DescribeDomainRuleGroup")
	}
	return resp.GetString("RuleGroupId")
}

func (self *SRegion) GetICloudWafInstances() ([]cloudprovider.ICloudWafInstance, error) {
	wafs, err := self.GetICloudWafInstancesV1()
	if err != nil {
		return nil, err
	}
	wafv2, err := self.GetICloudWafInstancesV2()
	if err != nil {
		return nil, err
	}
	return append(wafs, wafv2...), nil
}

func (self *SRegion) GetICloudWafInstanceById(id string) (cloudprovider.ICloudWafInstance, error) {
	ins, err := self.DescribeInstanceSpecInfo()
	if err != nil {
		ins, err = self.DescribeWafInstance()
		if err != nil {
			return nil, err
		}
		return self.DescribeDomainV2(ins.InstanceId, id)
	}
	return self.DescribeDomain(ins.InstanceId, id)
}

func (self *SWafDomain) GetId() string {
	return self.name
}

func (self *SWafDomain) GetStatus() string {
	return api.WAF_STATUS_AVAILABLE
}

func (self *SWafDomain) GetUpstreamPort() int {
	return 0
}

func (self *SWafDomain) GetUpstreamScheme() string {
	return ""
}

func (self *SWafDomain) GetWafType() cloudprovider.TWafType {
	return cloudprovider.WafTypeDefault
}

func (self *SWafDomain) GetEnabled() bool {
	return true
}

func (self *SWafDomain) GetName() string {
	return self.name
}

func (self *SWafDomain) GetGlobalId() string {
	return self.name
}

func (self *SWafDomain) GetIsAccessProduct() bool {
	return self.IsAccessProduct == 1
}

func (self *SWafDomain) GetAccessHeaders() []string {
	return self.AccessHeaders
}

func (self *SWafDomain) GetHttpPorts() []int {
	return self.HttpPort
}

func (self *SWafDomain) GetHttpsPorts() []int {
	return self.HttpsPort
}

func (self *SWafDomain) GetCname() string {
	return self.Cname
}

func (self *SWafDomain) GetCertId() string {
	return ""
}

func (self *SWafDomain) GetCertName() string {
	return ""
}

func (self *SWafDomain) GetSourceIps() []string {
	return self.SourceIps
}

func (self *SWafDomain) GetCcList() []string {
	return []string{}
}

func (self *SWafDomain) Delete() error {
	return self.region.DeleteDomain(self.insId, self.name)
}

func (self *SWafDomain) GetDefaultAction() *cloudprovider.DefaultAction {
	return &cloudprovider.DefaultAction{
		Action:        cloudprovider.WafActionAllow,
		InsertHeaders: map[string]string{},
	}
}

type ManagedRuleGroup struct {
	waf *SWafDomain

	insId       string
	domain      string
	ruleGroupId string
}

func (self *ManagedRuleGroup) GetName() string {
	return "RuleGroup"
}

func (self *ManagedRuleGroup) GetDesc() string {
	return "规则组"
}

func (self *ManagedRuleGroup) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.insId, self.domain)
}

func (self *ManagedRuleGroup) GetPriority() int {
	return 0
}

func (self *ManagedRuleGroup) GetAction() *cloudprovider.DefaultAction {
	return nil
}

func (self *ManagedRuleGroup) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (self *ManagedRuleGroup) Update(opts *cloudprovider.SWafRule) error {
	for _, statement := range opts.Statements {
		if len(statement.RuleGroupId) == 0 {
			return self.waf.region.SetDomainRuleGroup(self.insId, self.domain, statement.RuleGroupId)
		} else if len(statement.ManagedRuleGroupName) > 0 {
			switch statement.ManagedRuleGroupName {
			case "严格规则":
				return self.waf.region.SetDomainRuleGroup(self.insId, self.domain, "1011")
			case "中等规则":
				return self.waf.region.SetDomainRuleGroup(self.insId, self.domain, "1012")
			case "宽松规则":
				return self.waf.region.SetDomainRuleGroup(self.insId, self.domain, "1013")
			}
		}
	}
	return nil
}

func (self *ManagedRuleGroup) GetStatementCondition() cloudprovider.TWafStatementCondition {
	return cloudprovider.WafStatementConditionNone
}

func (self *ManagedRuleGroup) GetStatements() ([]cloudprovider.SWafStatement, error) {
	groupName := self.ruleGroupId
	switch self.ruleGroupId {
	case "1011":
		groupName = "严格规则"
	case "1012":
		groupName = "中等规则"
	case "1013":
		groupName = "宽松规则"
	}
	return []cloudprovider.SWafStatement{
		cloudprovider.SWafStatement{
			ManagedRuleGroupName: groupName,
			RuleGroupId:          self.ruleGroupId,
		},
	}, nil
}

type SDefenseTypeRule struct {
	insId       string
	domain      string
	defenseType string
	action      cloudprovider.TWafAction
}

func (self *SDefenseTypeRule) GetName() string {
	switch self.defenseType {
	case "waf":
		return "正则防护引擎"
	case "dld":
		return "大数据深度学习引擎"
	case "ac_cc":
		return "CC安全防护"
	case "antifraud":
		return "数据风控"
	case "normalized":
		return "主动防御"
	}
	return self.defenseType
}

func (self *SDefenseTypeRule) GetDesc() string {
	return ""
}

func (self *SDefenseTypeRule) GetGlobalId() string {
	return fmt.Sprintf("%s-%s-%s", self.insId, self.domain, self.defenseType)
}

func (self *SDefenseTypeRule) GetPriority() int {
	return 0
}

func (self *SDefenseTypeRule) GetAction() *cloudprovider.DefaultAction {
	return &cloudprovider.DefaultAction{
		Action: self.action,
	}
}

func (self *SDefenseTypeRule) GetStatementCondition() cloudprovider.TWafStatementCondition {
	return cloudprovider.WafStatementConditionNone
}

func (self *SDefenseTypeRule) GetStatements() ([]cloudprovider.SWafStatement, error) {
	return []cloudprovider.SWafStatement{}, nil
}

func (self *SDefenseTypeRule) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (self *SDefenseTypeRule) Update(opts *cloudprovider.SWafRule) error {
	return cloudprovider.ErrNotSupported
}

func (self *SWafDomain) GetRules() ([]cloudprovider.ICloudWafRule, error) {
	ruleGroupId, err := self.region.DescribeDomainRuleGroup(self.insId, self.name)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeDomainRuleGroup")
	}
	ret := []cloudprovider.ICloudWafRule{}
	ret = append(ret, &ManagedRuleGroup{
		waf:         self,
		insId:       self.insId,
		domain:      self.name,
		ruleGroupId: ruleGroupId,
	})
	for _, defenseType := range []string{
		"waf",
		"dld",
		"ac_cc",
		"antifraud",
		"normalized",
	} {
		act, _ := self.region.DescribeProtectionModuleMode(self.insId, self.name, defenseType)
		ret = append(ret, &SDefenseTypeRule{
			insId:       self.insId,
			domain:      self.name,
			defenseType: defenseType,
			action:      act,
		})
	}
	return ret, nil
}

type SIpSegement struct {
	IpV6s string
	Ips   string
}

func (self *SRegion) DescribeWafSourceIpSegment(insId string) (*SIpSegement, error) {
	params := map[string]string{
		"RegionId":   self.RegionId,
		"InstanceId": insId,
	}
	resp, err := self.wafRequest("DescribeWafSourceIpSegment", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeWafSourceIpSegment")
	}
	ret := &SIpSegement{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, errors.Wrapf(err, "")
	}
	return ret, nil
}

func (self *SRegion) CreateICloudWafInstance(opts *cloudprovider.WafCreateOptions) (cloudprovider.ICloudWafInstance, error) {
	ins, err := self.DescribeInstanceSpecInfo()
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeInstanceSpecInfo")
	}
	waf, err := self.CreateDomain(ins.InstanceId, opts.Name, opts.SourceIps, opts.CloudResources)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateDomain")
	}
	return waf, nil
}

func (self *SRegion) CreateDomain(insId, domain string, sourceIps []string, cloudResources []cloudprovider.SCloudResource) (*SWafDomain, error) {
	params := map[string]string{
		"RegionId":        self.RegionId,
		"InstanceId":      insId,
		"Domain":          domain,
		"IsAccessProduct": "0",
		"HttpPort":        `["80"]`,
		"HttpsPort":       `["443"]`,
		"Http2Port":       `["80", "443"]`,
	}
	if len(sourceIps) > 0 {
		params["SourceIps"] = jsonutils.Marshal(sourceIps).String()
		params["AccessType"] = "waf-cloud-dns"
	} else if len(cloudResources) > 0 {
		ins := jsonutils.NewArray()
		for _, res := range cloudResources {
			ins.Add(jsonutils.Marshal(map[string]interface{}{"InstanceId": res.Id, "Port": res.Port}))
		}
		params["CloudNativeInstances"] = ins.String()
		params["AccessType"] = "waf-cloud-native"
	} else {
		return nil, errors.Error("missing source ips")
	}
	_, err := self.wafRequest("CreateDomain", params)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateDomain")
	}
	return self.DescribeDomain(insId, domain)
}

func (self *SWafDomain) AddRule(opts *cloudprovider.SWafRule) (cloudprovider.ICloudWafRule, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "AddRule")
}

func (self *SWafDomain) Refresh() error {
	domain, err := self.region.DescribeDomain(self.insId, self.name)
	if err != nil {
		return errors.Wrapf(err, "DescribeDomain")
	}
	return jsonutils.Update(self, domain)
}

func (self *SWafDomain) GetCloudResources() ([]cloudprovider.SCloudResource, error) {
	ret := []cloudprovider.SCloudResource{}
	if len(self.Cname) > 0 {
		ret = append(ret, cloudprovider.SCloudResource{
			Type:          "cname",
			Name:          "CNAME",
			Id:            self.Cname,
			CanDissociate: false,
		})
	}
	ipseg, err := self.region.DescribeWafSourceIpSegment(self.insId)
	if err == nil {
		ret = append(ret, cloudprovider.SCloudResource{
			Type:          "segment_ipv4",
			Name:          "Segment IPv4",
			Id:            ipseg.Ips,
			CanDissociate: false,
		})
		ret = append(ret, cloudprovider.SCloudResource{
			Type:          "segment_ipv6",
			Name:          "Segment IPv6",
			Id:            ipseg.IpV6s,
			CanDissociate: false,
		})
	}
	return ret, nil
}

func (self *SRegion) DescribeProtectionModuleMode(insId, domain, defenseType string) (cloudprovider.TWafAction, error) {
	params := map[string]string{
		"RegionId":    self.RegionId,
		"Domain":      domain,
		"InstanceId":  insId,
		"DefenseType": defenseType,
	}
	resp, err := self.wafRequest("DescribeProtectionModuleMode", params)
	if err != nil {
		return cloudprovider.WafActionNone, errors.Wrapf(err, "DescribeProtectionModuleMode %s", defenseType)
	}
	if !resp.Contains("Mode") {
		return cloudprovider.WafActionNone, nil
	}
	mode, _ := resp.Int("Mode")
	switch defenseType {
	case "waf":
		if mode == 0 {
			return cloudprovider.WafActionBlock, nil
		}
		if mode == 1 {
			return cloudprovider.WafActionAlert, nil
		}
	case "dld":
		if mode == 0 {
			return cloudprovider.WafActionAlert, nil
		}
		if mode == 1 {
			return cloudprovider.WafActionBlock, nil
		}
	case "ac_cc":
		if mode == 0 {
			return cloudprovider.WafActionAllow, nil
		}
		if mode == 1 {
			return cloudprovider.WafActionBlock, nil
		}
	case "antifraud":
		if mode == 0 {
			return cloudprovider.WafActionAlert, nil
		}
		if mode == 1 || mode == 2 {
			return cloudprovider.WafActionBlock, nil
		}
	case "normalized":
		if mode == 0 {
			return cloudprovider.WafActionAlert, nil
		}
		if mode == 1 {
			return cloudprovider.WafActionBlock, nil
		}
	}
	return cloudprovider.WafActionNone, nil
}
