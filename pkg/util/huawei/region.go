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

package huawei

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/huawei/client"
	"yunion.io/x/onecloud/pkg/util/huawei/obs"
)

type Locales struct {
	EnUs string `json:"en-us"`
	ZhCN string `json:"zh-cn"`
}

// https://support.huaweicloud.com/api-iam/zh-cn_topic_0067148043.html
type SRegion struct {
	multicloud.SRegion

	client    *SHuaweiClient
	ecsClient *client.Client
	obsClient *obs.ObsClient // 对象存储client.请勿直接引用。

	Description    string  `json:"description"`
	ID             string  `json:"id"`
	Locales        Locales `json:"locales"`
	ParentRegionID string  `json:"parent_region_id"`
	Type           string  `json:"type"`

	izones []cloudprovider.ICloudZone
	ivpcs  []cloudprovider.ICloudVpc

	storageCache *SStoragecache
}

func (self *SRegion) GetClient() *SHuaweiClient {
	return self.client
}

func (self *SRegion) getECSClient() (*client.Client, error) {
	var err error

	if len(self.client.projectId) > 0 {
		project, err := self.client.GetProjectById(self.client.projectId)
		if err != nil {
			return nil, err
		}

		regionId := strings.Split(project.Name, "_")[0]
		if regionId != self.ID {
			log.Debugf("project %s not in region %s", self.client.projectId, self.ID)
			return nil, nil
		}
	}

	if self.ecsClient == nil {
		self.ecsClient, err = client.NewClientWithAccessKey(self.ID, self.client.projectId, self.client.accessKey, self.client.secret, self.client.debug)
		if err != nil {
			return nil, err
		}
	}

	return self.ecsClient, err
}

func (self *SRegion) getOBSClient() (*obs.ObsClient, error) {
	if self.obsClient == nil {
		endpoint := fmt.Sprintf("obs.%s.myhuaweicloud.com", self.GetId())
		obsClient, err := obs.New(self.client.accessKey, self.client.secret, endpoint)
		if err != nil {
			return nil, err
		}

		self.obsClient = obsClient
	}

	return self.obsClient, nil
}

func (self *SRegion) fetchZones() error {
	zones := make([]SZone, 0)
	err := doListAll(self.ecsClient.Zones.List, nil, &zones)
	if err != nil {
		return err
	}

	self.izones = make([]cloudprovider.ICloudZone, 0)
	for i := range zones {
		zone := zones[i]
		zone.region = self
		self.izones = append(self.izones, &zone)
	}
	return nil
}

func (self *SRegion) fetchIVpcs() error {
	// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090625.html
	vpcs := make([]SVpc, 0)
	querys := map[string]string{
		"limit": "2048",
	}
	err := doListAllWithMarker(self.ecsClient.Vpcs.List, querys, &vpcs)
	if err != nil {
		return err
	}

	self.ivpcs = make([]cloudprovider.ICloudVpc, 0)
	for i := range vpcs {
		vpc := vpcs[i]
		vpc.region = self
		self.ivpcs = append(self.ivpcs, &vpc)
	}
	return nil
}

func (self *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	instance, err := self.GetInstanceByID(id)
	if err != nil {
		return nil, err
	}
	return &instance, err
}

func (self *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return self.GetDisk(id)
}

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	if info, ok := LatitudeAndLongitude[self.ID]; ok {
		return info
	}
	return cloudprovider.SGeographicInfo{}
}

func (self *SRegion) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerById(loadbalancerId string) (cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerAclById(aclId string) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetILoadBalancerCertificateById(certId string) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateILoadBalancerCertificate(cert *cloudprovider.SLoadbalancerCertificate) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetILoadBalancerAcls() ([]cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

// https://support.huaweicloud.com/api-iam/zh-cn_topic_0057845622.html
func (self *SRegion) GetId() string {
	return self.ID
}

func (self *SRegion) GetName() string {
	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_HUAWEI_CN, self.Locales.ZhCN)
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_HUAWEI, self.ID)
}

func (self *SRegion) GetStatus() string {
	return api.CLOUD_REGION_STATUS_INSERVER
}

func (self *SRegion) Refresh() error {
	return nil
}

func (self *SRegion) IsEmulated() bool {
	return false
}

func (self *SRegion) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SRegion) GetLatitude() float32 {
	if locationInfo, ok := LatitudeAndLongitude[self.ID]; ok {
		return locationInfo.Latitude
	}
	return 0.0
}

func (self *SRegion) GetLongitude() float32 {
	if locationInfo, ok := LatitudeAndLongitude[self.ID]; ok {
		return locationInfo.Longitude
	}
	return 0.0
}

func (self *SRegion) fetchInfrastructure() error {
	_, err := self.getECSClient()
	if err != nil {
		return err
	}

	if err := self.fetchZones(); err != nil {
		return err
	}

	if err := self.fetchIVpcs(); err != nil {
		return err
	}

	for i := 0; i < len(self.ivpcs); i += 1 {
		vpc := self.ivpcs[i].(*SVpc)
		wire := SWire{region: self, vpc: vpc}
		vpc.addWire(&wire)

		for j := 0; j < len(self.izones); j += 1 {
			zone := self.izones[j].(*SZone)
			zone.addWire(&wire)
		}
	}
	return nil
}

func (self *SRegion) GetIZones() ([]cloudprovider.ICloudZone, error) {
	if self.izones == nil {
		var err error
		err = self.fetchInfrastructure()
		if err != nil {
			return nil, err
		}
	}
	return self.izones, nil
}

func (self *SRegion) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	if self.ivpcs == nil {
		err := self.fetchInfrastructure()
		if err != nil {
			return nil, err
		}
	}
	return self.ivpcs, nil
}

func (self *SRegion) GetEipById(eipId string) (SEipAddress, error) {
	var eip SEipAddress
	err := DoGet(self.ecsClient.Eips.Get, eipId, nil, &eip)
	eip.region = self
	return eip, err
}

// 返回参数分别为eip 列表、列表长度、error。
// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090598.html
func (self *SRegion) GetEips() ([]SEipAddress, error) {
	querys := make(map[string]string)

	eips := make([]SEipAddress, 0)
	err := doListAllWithMarker(self.ecsClient.Eips.List, querys, &eips)
	for i := range eips {
		eips[i].region = self
	}
	return eips, err
}

func (self *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	_, err := self.getECSClient()
	if err != nil {
		return nil, err
	}

	eips, err := self.GetEips()
	if err != nil {
		return nil, err
	}

	ret := make([]cloudprovider.ICloudEIP, len(eips))
	for i := 0; i < len(eips); i += 1 {
		eips[i].region = self
		ret[i] = &eips[i]
	}
	return ret, nil
}

func (self *SRegion) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	ivpcs, err := self.GetIVpcs()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(ivpcs); i += 1 {
		if ivpcs[i].GetGlobalId() == id {
			return ivpcs[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		if izones[i].GetGlobalId() == id {
			return izones[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIEipById(eipId string) (cloudprovider.ICloudEIP, error) {
	eip, err := self.GetEipById(eipId)
	return &eip, err
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0060595555.html
func (self *SRegion) DeleteSecurityGroup(vpcId, secgroupId string) error {
	return DoDelete(self.ecsClient.SecurityGroups.Delete, secgroupId, nil, nil)
}

func (self *SRegion) SyncSecurityGroup(secgroupId string, vpcId string, name string, desc string, rules []secrules.SecurityRule) (string, error) {
	if len(secgroupId) > 0 {
		_, err := self.GetSecurityGroupDetails(secgroupId)
		if err == cloudprovider.ErrNotFound {
			secgroupId = ""
		} else if err != nil {
			return "", errors.Wrapf(err, "self.GetSecurityGroupDetails(%s)", secgroupId)
		}
	}

	if len(secgroupId) == 0 {
		extID, err := self.CreateSecurityGroup(vpcId, name, desc)
		if err != nil {
			return "", errors.Wrap(err, "self.CreateSecurityGroup")
		}
		secgroupId = extID
	}

	// 华为云默认deny。不需要显式指定
	rules = SecurityRuleSetToAllowSet(rules)
	return secgroupId, self.syncSecgroupRules(secgroupId, rules)
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090608.html
func (self *SRegion) CreateIVpc(name string, desc string, cidr string) (cloudprovider.ICloudVpc, error) {
	params := jsonutils.NewDict()
	vpcObj := jsonutils.NewDict()
	vpcObj.Add(jsonutils.NewString(name), "name")
	vpcObj.Add(jsonutils.NewString(cidr), "cidr")
	params.Add(vpcObj, "vpc")

	vpc := SVpc{}
	err := DoCreate(self.ecsClient.Vpcs.Create, params, &vpc)
	vpc.region = self
	return &vpc, err
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090596.html
// size: 1Mbit/s~2000Mbit/s
// bgpType: 5_telcom，5_union，5_bgp，5_sbgp.
// 东北-大连：5_telcom、5_union
// 华南-广州：5_sbgp
// 华东-上海二：5_sbgp
// 华北-北京一：5_bgp、5_sbgp
// 亚太-香港：5_bgp
func (self *SRegion) CreateEIP(eip *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	var ctype TInternetChargeType
	switch eip.ChargeType {
	case api.EIP_CHARGE_TYPE_BY_TRAFFIC:
		ctype = InternetChargeByTraffic
	case api.EIP_CHARGE_TYPE_BY_BANDWIDTH:
		ctype = InternetChargeByBandwidth
	}

	// todo: 如何避免hardcode。集成到cloudmeta服务中？
	if len(eip.BGPType) == 0 {
		switch self.GetId() {
		case "cn-north-1", "cn-east-2", "cn-south-1":
			eip.BGPType = "5_sbgp"
		case "cn-northeast-1":
			eip.BGPType = "5_telcom"
		case "cn-north-4", "ap-southeast-1", "ap-southeast-2", "eu-west-0":
			eip.BGPType = "5_bgp"
		default:
			eip.BGPType = "5_bgp"
		}
	}

	// 华为云EIP名字最大长度64
	if len(eip.Name) > 64 {
		eip.Name = eip.Name[:64]
	}

	ieip, err := self.AllocateEIP(eip.Name, eip.BandwidthMbps, ctype, eip.BGPType)
	ieip.region = self
	if err != nil {
		return nil, err
	}

	err = cloudprovider.WaitStatus(ieip, api.EIP_STATUS_READY, 5*time.Second, 60*time.Second)
	return ieip, err
}

func (self *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.GetSnapshots("", "")
	if err != nil {
		log.Errorf("self.GetSnapshots fail %s", err)
		return nil, err
	}

	ret := make([]cloudprovider.ICloudSnapshot, len(snapshots))
	for i := 0; i < len(snapshots); i += 1 {
		snapshots[i].region = self
		ret[i] = &snapshots[i]
	}
	return ret, nil
}

func (self *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := self.GetSnapshotById(snapshotId)
	return &snapshot, err
}

func (self *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	iHosts := make([]cloudprovider.ICloudHost, 0)

	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		iZoneHost, err := izones[i].GetIHosts()
		if err != nil {
			return nil, err
		}
		iHosts = append(iHosts, iZoneHost...)
	}
	return iHosts, nil
}

func (self *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		ihost, err := izones[i].GetIHostById(id)
		if err == nil {
			return ihost, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	iStores := make([]cloudprovider.ICloudStorage, 0)

	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		iZoneStores, err := izones[i].GetIStorages()
		if err != nil {
			return nil, err
		}
		iStores = append(iStores, iZoneStores...)
	}
	return iStores, nil
}

func (self *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	izones, err := self.GetIZones()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(izones); i += 1 {
		istore, err := izones[i].GetIStorageById(id)
		if err == nil {
			return istore, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_HUAWEI
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0020090615.html
// 目前desc字段并没有用到
func (self *SRegion) CreateSecurityGroup(vpcId string, name string, desc string) (string, error) {
	// 华为不允许创建名称为default的安全组
	if strings.ToLower(name) == "default" {
		name = fmt.Sprintf("%s-%s", vpcId, name)
	}

	params := jsonutils.NewDict()
	secgroupObj := jsonutils.NewDict()
	secgroupObj.Add(jsonutils.NewString(name), "name")
	if len(vpcId) > 0 {
		secgroupObj.Add(jsonutils.NewString(vpcId), "vpc_id")
	}
	params.Add(secgroupObj, "security_group")

	secgroup := SSecurityGroup{}
	err := DoCreate(self.ecsClient.SecurityGroups.Create, params, &secgroup)
	return secgroup.GetId(), err
}

func (self *SRegion) syncSecgroupRules(secgroupId string, rules []secrules.SecurityRule) error {
	var DeleteRules []secrules.SecurityRule
	var AddRules []secrules.SecurityRule

	if secgroup, err := self.GetSecurityGroupDetails(secgroupId); err != nil {
		return errors.Wrapf(err, "syncSecgroupRules.GetSecurityGroupDetails(%s)", secgroupId)
	} else {
		remoteRules, err := secgroup.GetRulesWithExtId()
		if err != nil {
			return errors.Wrap(err, "secgroup.GetRulesWithExtId")
		}

		sort.Sort(secrules.SecurityRuleSet(rules))
		sort.Sort(secrules.SecurityRuleSet(remoteRules))

		i, j := 0, 0
		for i < len(rules) || j < len(remoteRules) {
			if i < len(rules) && j < len(remoteRules) {
				permissionStr := remoteRules[j].String()
				ruleStr := rules[i].String()
				cmp := strings.Compare(permissionStr, ruleStr)
				if cmp == 0 {
					// DeleteRules = append(DeleteRules, remoteRules[j])
					// AddRules = append(AddRules, rules[i])
					i += 1
					j += 1
				} else if cmp > 0 {
					DeleteRules = append(DeleteRules, remoteRules[j])
					j += 1
				} else {
					AddRules = append(AddRules, rules[i])
					i += 1
				}
			} else if i >= len(rules) {
				DeleteRules = append(DeleteRules, remoteRules[j])
				j += 1
			} else if j >= len(remoteRules) {
				AddRules = append(AddRules, rules[i])
				i += 1
			}
		}
	}

	for _, r := range DeleteRules {
		// r.Description 实际存储的是ruleId
		if err := self.delSecurityGroupRule(r.Description); err != nil {
			log.Errorf("delSecurityGroupRule %v error: %s", r, err.Error())
			return err
		}
	}

	for _, r := range AddRules {
		if err := self.addSecurityGroupRules(secgroupId, &r); err != nil {
			log.Errorf("addSecurityGroupRule %v error: %s", r, err.Error())
			return err
		}
	}

	return nil
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0087467071.html
func (self *SRegion) delSecurityGroupRule(secGrpRuleId string) error {
	return DoDelete(self.ecsClient.SecurityGroupRules.Delete, secGrpRuleId, nil, nil)
}

// https://support.huaweicloud.com/api-vpc/zh-cn_topic_0087451723.html
// icmp port对应关系：https://support.huaweicloud.com/api-vpc/zh-cn_topic_0024109590.html
func (self *SRegion) addSecurityGroupRules(secGrpId string, rule *secrules.SecurityRule) error {
	direction := ""
	if rule.Direction == secrules.SecurityRuleIngress {
		direction = "ingress"
	} else {
		direction = "egress"
	}

	protocal := rule.Protocol
	if rule.Protocol == secrules.PROTO_ANY {
		protocal = ""
	}

	// imcp协议默认为any
	if rule.Protocol == secrules.PROTO_ICMP {
		return self.addSecurityGroupRule(secGrpId, direction, "-1", "-1", protocal, rule.IPNet.String())
	}

	if len(rule.Ports) > 0 {
		for _, port := range rule.Ports {
			portStr := fmt.Sprintf("%d", port)
			err := self.addSecurityGroupRule(secGrpId, direction, portStr, portStr, protocal, rule.IPNet.String())
			if err != nil {
				return err
			}
		}
	} else {
		portStart := fmt.Sprintf("%d", rule.PortStart)
		portEnd := fmt.Sprintf("%d", rule.PortEnd)
		err := self.addSecurityGroupRule(secGrpId, direction, portStart, portEnd, protocal, rule.IPNet.String())
		if err != nil {
			return err
		}
	}

	return nil
}

func (self *SRegion) addSecurityGroupRule(secGrpId, direction, portStart, portEnd, protocol, ipNet string) error {
	params := jsonutils.NewDict()
	secgroupObj := jsonutils.NewDict()
	secgroupObj.Add(jsonutils.NewString(secGrpId), "security_group_id")
	secgroupObj.Add(jsonutils.NewString(direction), "direction")
	secgroupObj.Add(jsonutils.NewString(ipNet), "remote_ip_prefix")
	secgroupObj.Add(jsonutils.NewString("IPV4"), "ethertype")
	// 端口为空或者1-65535
	if len(portStart) > 0 && portStart != "0" && portStart != "-1" {
		secgroupObj.Add(jsonutils.NewString(portStart), "port_range_min")
	}
	if len(portEnd) > 0 && portEnd != "0" && portEnd != "-1" {
		secgroupObj.Add(jsonutils.NewString(portEnd), "port_range_max")
	}
	if len(protocol) > 0 {
		secgroupObj.Add(jsonutils.NewString(protocol), "protocol")
	}
	params.Add(secgroupObj, "security_group_rule")

	rule := SecurityGroupRule{}
	return DoCreate(self.ecsClient.SecurityGroupRules.Create, params, &rule)
}

func (region *SRegion) CreateILoadBalancer(loadbalancer *cloudprovider.SLoadbalancer) (cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) CreateILoadBalancerAcl(acl *cloudprovider.SLoadbalancerAccessControlList) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (region *SRegion) GetSkus(zoneId string) ([]cloudprovider.ICloudSku, error) {
	return nil, cloudprovider.ErrNotImplemented
}
