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
	"sort"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SRegion struct {
	multicloud.SRegion
	client *SUcloudClient

	RegionID string

	izones []cloudprovider.ICloudZone
	ivpcs  []cloudprovider.ICloudVpc

	storageCache *SStoragecache

	latitude      float64
	longitude     float64
	fetchLocation bool
}

func (self *SRegion) GetId() string {
	return self.RegionID
}

func (self *SRegion) GetName() string {
	if name, exist := UCLOUD_REGION_NAMES[self.GetId()]; exist {
		return fmt.Sprintf("%s %s", CLOUD_PROVIDER_UCLOUD_CN, name)
	}

	return fmt.Sprintf("%s %s", CLOUD_PROVIDER_UCLOUD_CN, self.GetId())
}

func (self *SRegion) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", CLOUD_PROVIDER_UCLOUD, self.GetId())
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

func (self *SRegion) GetGeographicInfo() cloudprovider.SGeographicInfo {
	if info, ok := LatitudeAndLongitude[self.GetId()]; ok {
		return info
	}
	return cloudprovider.SGeographicInfo{}
}

func (self *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	instance, err := self.GetInstanceByID(id)
	if err != nil {
		return nil, err
	}
	return &instance, nil
}

func (self *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return self.GetDisk(id)
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

// https://docs.ucloud.cn/api/unet-api/describe_eip
func (self *SRegion) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	params := NewUcloudParams()
	eips := make([]SEip, 0)
	err := self.DoListAll("DescribeEIP", params, &eips)
	if err != nil {
		return nil, err
	}

	ieips := []cloudprovider.ICloudEIP{}
	for i := range eips {
		eip := eips[i]
		eip.region = self
		ieips = append(ieips, &eip)
	}

	return ieips, nil
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

func (self *SRegion) GetEipById(eipId string) (SEip, error) {
	params := NewUcloudParams()
	params.Set("EIPIds.0", eipId)
	eips := make([]SEip, 0)
	err := self.DoListAll("DescribeEIP", params, &eips)
	if err != nil {
		return SEip{}, err
	}

	if len(eips) == 1 {
		eip := eips[0]
		eip.region = self
		return eip, nil
	} else if len(eips) == 0 {
		return SEip{}, cloudprovider.ErrNotFound
	} else {
		return SEip{}, fmt.Errorf("GetEipById %d eip found", len(eips))
	}
}

func (self *SRegion) GetIEipById(id string) (cloudprovider.ICloudEIP, error) {
	eip, err := self.GetEipById(id)
	return &eip, err
}

// https://docs.ucloud.cn/api/unet-api/delete_firewall
func (self *SRegion) DeleteSecurityGroup(vpcId, secgroupId string) error {
	params := NewUcloudParams()
	params.Set("FWId", secgroupId)
	return self.DoAction("DeleteFirewall", params, nil)
}

// https://docs.ucloud.cn/api/unet-api/describe_firewall
// 绑定防火墙组的资源类型，默认为全部资源类型。枚举值为："unatgw"，NAT网关； "uhost"，云主机； "upm"，物理云主机； "hadoophost"，hadoop节点； "fortresshost"，堡垒机； "udhost"，私有专区主机；"udockhost"，容器；"dbaudit"，数据库审计.
// todo: 是否需要过滤出仅绑定云主机的安全组？
func (self *SRegion) SyncSecurityGroup(secgroupId string, vpcId string, name string, desc string, rules []secrules.SecurityRule) (string, error) {
	if len(secgroupId) > 0 {
		_, err := self.GetSecurityGroupById(secgroupId)
		if err == cloudprovider.ErrNotFound {
			secgroupId = ""
		} else if err != nil {
			return "", err
		}
	}

	if len(secgroupId) == 0 {
		extID, err := self.CreateDefaultSecurityGroup(name, desc)
		if err != nil {
			return "", err
		}
		secgroupId = extID
	}

	return secgroupId, self.syncSecgroupRules(secgroupId, rules)
}

func (self *SRegion) CreateIVpc(name string, desc string, cidr string) (cloudprovider.ICloudVpc, error) {
	params := NewUcloudParams()
	params.Set("Name", name)
	params.Set("Remark", desc)
	for i, cidr := range strings.Split(cidr, ",") {
		params.Set(fmt.Sprintf("Network.%d", i), cidr)
	}

	vpcId := ""
	err := self.DoAction("CreateVPC", params, &vpcId)
	if err != nil {
		return nil, err
	}

	return self.GetIVpcById(vpcId)
}

// https://docs.ucloud.cn/api/udisk-api/describe_udisk_snapshot
func (self *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	params := NewUcloudParams()
	snapshots := make([]SSnapshot, 0)
	err := self.DoListAll("DescribeUDiskSnapshot", params, &snapshots)
	if err != nil {
		return nil, err
	}

	isnapshots := make([]cloudprovider.ICloudSnapshot, 0)
	for i := range snapshots {
		snapshots[i].region = self
		isnapshots = append(isnapshots, &snapshots[i])
	}

	return isnapshots, nil
}

func (self *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	if len(snapshotId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}

	params := NewUcloudParams()
	snapshots := make([]SSnapshot, 0)
	err := self.DoListAll("DescribeUDiskSnapshot", params, &snapshots)
	if err != nil {
		return nil, err
	}

	for i := range snapshots {
		if snapshots[i].SnapshotID == snapshotId {
			snapshot := snapshots[i]
			snapshot.region = self
			return &snapshot, nil
		}
	}

	return nil, cloudprovider.ErrNotFound
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

func (self *SRegion) getStoragecache() *SStoragecache {
	if self.storageCache == nil {
		self.storageCache = &SStoragecache{region: self}
	}
	return self.storageCache
}

func (self *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	storageCache := self.getStoragecache()
	return []cloudprovider.ICloudStoragecache{storageCache}, nil
}

func (self *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	storageCache := self.getStoragecache()
	if storageCache.GetGlobalId() == id {
		return storageCache, nil
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SRegion) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetILoadBalancerAcls() ([]cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetILoadBalancerById(loadbalancerId string) (cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetILoadBalancerAclById(aclId string) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetILoadBalancerCertificateById(certId string) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) CreateILoadBalancer(loadbalancer *cloudprovider.SLoadbalancer) (cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) CreateILoadBalancerAcl(acl *cloudprovider.SLoadbalancerAccessControlList) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) CreateILoadBalancerCertificate(cert *cloudprovider.SLoadbalancerCertificate) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetSkus(zoneId string) ([]cloudprovider.ICloudSku, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) GetProvider() string {
	return CLOUD_PROVIDER_UCLOUD
}

func (self *SRegion) DoListAll(action string, params SParams, result interface{}) error {
	params.Set("Region", self.GetId())
	return self.client.DoListAll(action, params, result)
}

// return total,lenght,error
func (self *SRegion) DoListPart(action string, limit int, offset int, params SParams, result interface{}) (int, int, error) {
	params.Set("Region", self.GetId())
	return self.client.DoListPart(action, limit, offset, params, result)
}

func (self *SRegion) DoAction(action string, params SParams, result interface{}) error {
	params.Set("Region", self.GetId())
	return self.client.DoAction(action, params, result)
}

func (self *SRegion) fetchInfrastructure() error {
	if err := self.fetchZones(); err != nil {
		return err
	}

	if err := self.fetchIVpcs(); err != nil {
		return err
	}

	for i := 0; i < len(self.ivpcs); i += 1 {
		vpc := self.ivpcs[i].(*SVPC)
		wire := SWire{region: self, vpc: vpc}
		vpc.addWire(&wire)

		for j := 0; j < len(self.izones); j += 1 {
			zone := self.izones[j].(*SZone)
			zone.addWire(&wire)
		}
	}
	return nil
}

func (self *SRegion) fetchZones() error {
	type Region struct {
		RegionID   int64  `json:"RegionId"`
		RegionName string `json:"RegionName"`
		IsDefault  bool   `json:"IsDefault"`
		BitMaps    string `json:"BitMaps"`
		Region     string `json:"Region"`
		Zone       string `json:"Zone"`
	}

	params := NewUcloudParams()
	regions := make([]Region, 0)
	err := self.client.DoListAll("GetRegion", params, &regions)
	if err != nil {
		return err
	}

	for _, r := range regions {
		if r.Region != self.GetId() {
			continue
		}

		szone := SZone{}
		szone.ZoneId = r.Zone
		szone.RegionId = r.Region
		szone.region = self
		self.izones = append(self.izones, &szone)
	}

	return nil
}

func (self *SRegion) fetchIVpcs() error {
	vpcs := make([]SVPC, 0)
	params := NewUcloudParams()
	err := self.DoListAll("DescribeVPC", params, &vpcs)
	if err != nil {
		return err
	}

	for i := range vpcs {
		vpc := vpcs[i]
		vpc.region = self
		self.ivpcs = append(self.ivpcs, &vpc)
	}

	return nil
}

// https://docs.ucloud.cn/api/uhost-api/describe_uhost_instance
func (self *SRegion) GetInstanceByID(instanceId string) (SInstance, error) {
	params := NewUcloudParams()
	params.Set("UHostIds.0", instanceId)
	instances := make([]SInstance, 0)
	err := self.DoAction("DescribeUHostInstance", params, &instances)
	if err != nil {
		return SInstance{}, err
	}

	if len(instances) == 1 {
		return instances[0], nil
	} else if len(instances) == 0 {
		return SInstance{}, cloudprovider.ErrNotFound
	} else {
		return SInstance{}, fmt.Errorf("GetInstanceByID %s %d found.", instanceId, len(instances))
	}
}

func inList(s int, lst []int) bool {
	for _, e := range lst {
		if s == e {
			return true
		}
	}

	return false
}

func toUcloudSecurityRules(rules []secrules.SecurityRule) ([]string, error) {
	ps := make([]int, 0)
	for _, rule := range rules {
		if rule.Direction == secrules.SecurityRuleIngress && !inList(rule.Priority, ps) {
			ps = append(ps, rule.Priority)
		}
	}

	if len(ps) > 3 {
		return nil, fmt.Errorf("unable map local security group rule priority  %v to LOW/MEDIUM/LOW", ps)
	}

	sort.Ints(ps)
	pmap := map[int]string{}
	for i, p := range ps {
		pmap[p] = []string{"LOW", "MEDIUM", "HIGH"}[i]
	}

	ucloudRules := make([]string, 0)
	for _, rule := range rules {
		if rule.Direction == secrules.SecurityRuleIngress {
			// GRE协议被忽略了
			ucloudRules = append(ucloudRules, toUcloudSecRule(rule, pmap)...)
		}
	}

	return ucloudRules, nil
}

// GRE协议被忽略了
func toUcloudSecRule(rule secrules.SecurityRule, pmap map[int]string) []string {
	net := rule.IPNet.String()
	priority := pmap[rule.Priority]
	action := "DROP"
	if rule.Action == secrules.SecurityRuleAllow {
		action = "ACCEPT"
	}

	rules := make([]string, 0)
	if len(rule.Ports) > 0 {
		for _, port := range rule.Ports {
			_rules := generatorRule(rule.Protocol, priority, net, action, port, port)
			rules = append(rules, _rules...)
		}
	} else {
		_rules := generatorRule(rule.Protocol, priority, net, action, rule.PortStart, rule.PortEnd)
		rules = append(rules, _rules...)
	}

	return rules
}

func generatorRule(protocol, priority, net, action string, startPort, endPort int) []string {
	rules := make([]string, 0)

	var ports string
	if startPort <= 0 || endPort <= 0 {
		ports = "1-65535"
	} else if startPort == endPort {
		ports = fmt.Sprintf("%d", startPort)
	} else {
		ports = fmt.Sprintf("%d-%d", startPort, endPort)
	}

	template := fmt.Sprintf("%s|%s|%s|%s|%s|", "%s", "%s", net, action, priority)
	switch protocol {
	case secrules.PROTO_ANY:
		rules = append(rules, fmt.Sprintf(template, "TCP", ports))
		rules = append(rules, fmt.Sprintf(template, "UDP", ports))
		rules = append(rules, fmt.Sprintf(template, "ICMP", ""))
	case secrules.PROTO_TCP:
		rules = append(rules, fmt.Sprintf(template, "TCP", ports))
	case secrules.PROTO_UDP:
		rules = append(rules, fmt.Sprintf(template, "UDP", ports))
	case secrules.PROTO_ICMP:
		rules = append(rules, fmt.Sprintf(template, "ICMP", ""))
	}

	return rules
}

// https://docs.ucloud.cn/api/unet-api/update_firewall
func (self *SRegion) syncSecgroupRules(secgroupId string, rules []secrules.SecurityRule) error {
	_rules, err := toUcloudSecurityRules(rules)
	if err != nil {
		return err
	}

	params := NewUcloudParams()
	params.Set("FWId", secgroupId)
	for i, r := range _rules {
		N := i + 1
		params.Set(fmt.Sprintf("Rule.%d", N), r)
	}

	return self.DoAction("UpdateFirewall", params, nil)
}

func (self *SRegion) GetClient() *SUcloudClient {
	return self.client
}
