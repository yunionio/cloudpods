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
	"strconv"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

// https://help.aliyun.com/document_detail/60933.html?spm=a2c4g.11186623.6.726.38f82ca9U1Gtxw
type SElasticcache struct {
	multicloud.SElasticcacheBase

	region    *SRegion
	attribute *SElasticcacheAttribute
	netinfo   []SNetInfo

	Config              string      `json:"Config"`
	HasRenewChangeOrder bool        `json:"HasRenewChangeOrder"`
	InstanceID          string      `json:"InstanceId"`
	UserName            string      `json:"UserName"`
	ArchitectureType    string      `json:"ArchitectureType"`
	ZoneID              string      `json:"ZoneId"`
	PrivateIP           string      `json:"PrivateIp"`
	VSwitchID           string      `json:"VSwitchId"`
	VpcID               string      `json:"VpcId"`
	NetworkType         string      `json:"NetworkType"`
	Qps                 int64       `json:"QPS"`
	PackageType         string      `json:"PackageType"`
	IsRDS               bool        `json:"IsRds"`
	EngineVersion       string      `json:"EngineVersion"`
	ConnectionDomain    string      `json:"ConnectionDomain"`
	InstanceName        string      `json:"InstanceName"`
	ReplacateID         string      `json:"ReplacateId"`
	Bandwidth           int64       `json:"Bandwidth"`
	ChargeType          TChargeType `json:"ChargeType"`
	InstanceType        string      `json:"InstanceType"`
	Tags                Tags        `json:"Tags"`
	InstanceStatus      string      `json:"InstanceStatus"`
	Port                int         `json:"Port"`
	InstanceClass       string      `json:"InstanceClass"`
	CreateTime          time.Time   `json:"CreateTime"`
	EndTime             time.Time   `json:"EndTime"`
	RegionID            string      `json:"RegionId"`
	NodeType            string      `json:"NodeType"`
	CapacityMB          int         `json:"Capacity"`
	Connections         int64       `json:"Connections"`
}

type SElasticcacheAttribute struct {
	Config              string      `json:"Config"`
	HasRenewChangeOrder string      `json:"HasRenewChangeOrder"`
	InstanceID          string      `json:"InstanceId"`
	ZoneID              string      `json:"ZoneId"`
	ArchitectureType    string      `json:"ArchitectureType"`
	PrivateIP           string      `json:"PrivateIp"`
	VSwitchID           string      `json:"VSwitchId"`
	Engine              string      `json:"Engine"`
	VpcID               string      `json:"VpcId"`
	NetworkType         string      `json:"NetworkType"`
	Qps                 int64       `json:"QPS"`
	PackageType         string      `json:"PackageType"`
	ReplicaID           string      `json:"ReplicaId"`
	IsRDS               bool        `json:"IsRds"`
	MaintainStartTime   string      `json:"MaintainStartTime"`
	VpcAuthMode         string      `json:"VpcAuthMode"`
	ConnectionDomain    string      `json:"ConnectionDomain"`
	EngineVersion       string      `json:"EngineVersion"`
	InstanceName        string      `json:"InstanceName"`
	Bandwidth           int64       `json:"Bandwidth"`
	ChargeType          TChargeType `json:"ChargeType"`
	AuditLogRetention   string      `json:"AuditLogRetention"`
	MaintainEndTime     string      `json:"MaintainEndTime"`
	ReplicationMode     string      `json:"ReplicationMode"`
	InstanceType        string      `json:"InstanceType"`
	InstanceStatus      string      `json:"InstanceStatus"`
	Tags                Tags        `json:"Tags"`
	Port                int64       `json:"Port"`
	InstanceClass       string      `json:"InstanceClass"`
	CreateTime          time.Time   `json:"CreateTime"`
	NodeType            string      `json:"NodeType"`
	RegionID            string      `json:"RegionId"`
	AvailabilityValue   string      `json:"AvailabilityValue"`
	CapacityMB          int         `json:"Capacity"`
	Connections         int64       `json:"Connections"`
	SecurityIPList      string      `json:"SecurityIPList"`
}

type SNetInfo struct {
	ConnectionString  string  `json:"ConnectionString"`
	Port              string  `json:"Port"`
	DBInstanceNetType string  `json:"DBInstanceNetType"`
	VPCID             string  `json:"VPCId"`
	VPCInstanceID     string  `json:"VPCInstanceId"`
	IPAddress         string  `json:"IPAddress"`
	IPType            string  `json:"IPType"`
	Upgradeable       string  `json:"Upgradeable"`
	ExpiredTime       *string `json:"ExpiredTime,omitempty"`
}

func (self *SElasticcache) GetId() string {
	return self.InstanceID
}

func (self *SElasticcache) GetName() string {
	return self.InstanceName
}

func (self *SElasticcache) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticcache) GetStatus() string {
	// todo: fix
	return self.InstanceStatus
}

func (self *SElasticcache) GetBillingType() string {
	return convertChargeType(self.ChargeType)
}

func (self *SElasticcache) GetCreatedAt() time.Time {
	return self.CreateTime
}

func (self *SElasticcache) GetExpiredAt() time.Time {
	return convertExpiredAt(self.EndTime)
}

func (self *SElasticcache) GetInstanceType() string {
	return self.InstanceClass
}

func (self *SElasticcache) GetCapacityMB() int {
	return self.CapacityMB
}

func (self *SElasticcache) GetArchType() string {
	return self.ArchitectureType
}

func (self *SElasticcache) GetNodeType() string {
	return self.NodeType
}

func (self *SElasticcache) GetEngine() string {
	return self.InstanceType
}

func (self *SElasticcache) GetEngineVersion() string {
	return self.EngineVersion
}

func (self *SElasticcache) GetVpcId() string {
	return self.VpcID
}

func (self *SElasticcache) GetZoneId() string {
	zone, err := self.region.getZoneById(self.ZoneID)
	if err != nil {
		log.Errorf("failed to find zone for elasticcache %s error: %v", self.GetId(), err)
		return ""
	}
	return zone.GetGlobalId()
}

func (self *SElasticcache) GetNetworkType() string {
	switch self.NetworkType {
	case "VPC":
		return api.LB_NETWORK_TYPE_VPC
	case "CLASSIC":
		return api.LB_NETWORK_TYPE_CLASSIC
	default:
		return api.LB_NETWORK_TYPE_VPC
	}
}

func (self *SElasticcache) GetNetworkId() string {
	return self.VSwitchID
}

func (self *SElasticcache) GetPrivateDNS() string {
	return self.ConnectionDomain
}

func (self *SElasticcache) GetPrivateIpAddr() string {
	return self.PrivateIP
}

func (self *SElasticcache) GetPrivateConnectPort() int {
	return self.Port
}

func (self *SElasticcache) GetPublicDNS() string {
	pub, err := self.GetPublicNetInfo()
	if err != nil {
		log.Errorf("SElasticcache.GetPublicDNS %s", err)
		return ""
	}

	if pub != nil {
		return pub.ConnectionString
	}

	return ""
}

func (self *SElasticcache) GetPublicIpAddr() string {
	pub, err := self.GetPublicNetInfo()
	if err != nil {
		log.Errorf("SElasticcache.GetPublicIpAddr %s", err)
	}

	if pub != nil {
		return pub.IPAddress
	}

	return ""
}

func (self *SElasticcache) GetPublicConnectPort() int {
	pub, err := self.GetPublicNetInfo()
	if err != nil {
		log.Errorf("SElasticcache.GetPublicConnectPort %s", err)
	}

	if pub != nil {
		port, _ := strconv.Atoi(pub.Port)
		return port
	}

	return 0
}

func (self *SElasticcache) GetMaintainStartTime() string {
	attr, err := self.GetAttribute()
	if err != nil {
		log.Errorf("SElasticcache.GetMaintainStartTime %s", err)
	}

	if attr != nil {
		return attr.MaintainStartTime
	}

	return ""
}

func (self *SElasticcache) GetMaintainEndTime() string {
	attr, err := self.GetAttribute()
	if err != nil {
		log.Errorf("SElasticcache.GetMaintainEndTime %s", err)
	}

	if attr != nil {
		return attr.MaintainEndTime
	}

	return ""
}

func (self *SElasticcache) GetICloudElasticcacheAccounts() ([]cloudprovider.ICloudElasticcacheAccount, error) {
	accounts, err := self.region.GetElasticCacheAccounts(self.GetId())
	if err != nil {
		return nil, err
	}

	iaccounts := make([]cloudprovider.ICloudElasticcacheAccount, len(accounts))
	for i := range accounts {
		accounts[i].cacheDB = self
		iaccounts[i] = &accounts[i]
	}

	return iaccounts, nil
}

func (self *SElasticcache) GetICloudElasticcacheAcls() ([]cloudprovider.ICloudElasticcacheAcl, error) {
	acls, err := self.region.GetElasticCacheAcls(self.GetId())
	if err != nil {
		return nil, err
	}

	iacls := make([]cloudprovider.ICloudElasticcacheAcl, len(acls))
	for i := range acls {
		acls[i].cacheDB = self
		iacls[i] = &acls[i]
	}

	return iacls, nil
}

func (self *SElasticcache) GetICloudElasticcacheBackups() ([]cloudprovider.ICloudElasticcacheBackup, error) {
	start := self.CreateTime.Format("2006-01-02T15:04Z")
	end := time.Now().Format("2006-01-02T15:04Z")
	backups, err := self.region.GetElasticCacheBackups(self.GetId(), start, end)
	if err != nil {
		return nil, err
	}

	ibackups := make([]cloudprovider.ICloudElasticcacheBackup, len(backups))
	for i := range backups {
		backups[i].cacheDB = self
		ibackups[i] = &backups[i]
	}

	return ibackups, nil
}

func (self *SElasticcache) GetICloudElasticcacheParameters() ([]cloudprovider.ICloudElasticcacheParameter, error) {
	parameters, err := self.region.GetElasticCacheParameters(self.GetId())
	if err != nil {
		return nil, err
	}

	iparameters := make([]cloudprovider.ICloudElasticcacheParameter, len(parameters))
	for i := range parameters {
		parameters[i].cacheDB = self
		iparameters[i] = &parameters[i]
	}

	return iparameters, nil
}

func (self *SElasticcache) GetAttribute() (*SElasticcacheAttribute, error) {
	if self.attribute != nil {
		return self.attribute, nil
	}

	params := make(map[string]string)
	params["RegionId"] = self.region.RegionId
	params["InstanceId"] = self.GetId()

	rets := []SElasticcacheAttribute{}
	err := DoListAll(self.region.kvsRequest, "DescribeInstanceAttribute", params, []string{"Instances", "DBInstanceAttribute"}, &rets)
	if err != nil {
		return nil, errors.Wrap(err, "elasticcache.GetAttribute")
	}

	count := len(rets)
	if count == 1 {
		self.attribute = &rets[0]
		return self.attribute, nil
	} else if count == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "elasticcache.GetAttribute %s", self.GetId())
	} else {
		return nil, errors.Wrapf(cloudprovider.ErrDuplicateId, "elasticcache.GetAttribute %s.expect 1 found %d ", self.GetId(), count)
	}
}

func (self *SElasticcache) GetNetInfo() ([]SNetInfo, error) {
	if self.netinfo != nil && len(self.netinfo) > 0 {
		return self.netinfo, nil
	}

	params := make(map[string]string)
	params["RegionId"] = self.region.RegionId
	params["InstanceId"] = self.GetId()

	rets := []SNetInfo{}
	err := DoListAll(self.region.kvsRequest, "DescribeDBInstanceNetInfo", params, []string{"NetInfoItems", "InstanceNetInfo"}, &rets)
	if err != nil {
		return nil, errors.Wrap(err, "elasticcache.GetNetInfo")
	}

	self.netinfo = rets
	return self.netinfo, nil
}

func (self *SElasticcache) GetPublicNetInfo() (*SNetInfo, error) {
	nets, err := self.GetNetInfo()
	if err != nil {
		return nil, err
	}

	for i := range nets {
		if nets[i].DBInstanceNetType == "2" || nets[i].IPType == "Public" {
			return &nets[i], nil
		}
	}

	return nil, nil
}

func (self *SRegion) GetElasticCaches(instanceIds []string) ([]SElasticcache, error) {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	if instanceIds != nil && len(instanceIds) > 0 {
		params["InstanceIds"] = jsonutils.Marshal(instanceIds).String()
	}

	ret := []SElasticcache{}
	err := DoListAll(self.kvsRequest, "DescribeInstances", params, []string{"Instances", "KVStoreInstance"}, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "region.GetElasticCaches")
	}

	for i := range ret {
		ret[i].region = self
	}

	return ret, nil
}

func (self *SRegion) GetElasticCacheById(instanceId string) (*SElasticcache, error) {
	caches, err := self.GetElasticCaches([]string{instanceId})
	if err != nil {
		return nil, errors.Wrapf(err, "region.GetElasticCacheById %s", instanceId)
	}

	if len(caches) == 1 {
		return &caches[0], nil
	} else if len(caches) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "region.GetElasticCacheById %s", instanceId)
	} else {
		return nil, errors.Wrapf(cloudprovider.ErrDuplicateId, "region.GetElasticCacheById %s.expect 1 found %d ", instanceId, len(caches))
	}
}

// https://help.aliyun.com/document_detail/95802.html?spm=a2c4g.11186623.6.746.143e782f3Pfkfg
func (self *SRegion) GetElasticCacheAccounts(instanceId string) ([]SElasticcacheAccount, error) {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["InstanceId"] = instanceId

	ret := []SElasticcacheAccount{}
	err := DoListAll(self.kvsRequest, "DescribeAccounts", params, []string{"Accounts", "Account"}, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "region.GetElasticCacheAccounts")
	}

	return ret, nil
}

// https://help.aliyun.com/document_detail/63889.html?spm=a2c4g.11186623.6.764.3cb43852R7lnoS
func (self *SRegion) GetElasticCacheAcls(instanceId string) ([]SElasticcacheAcl, error) {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["InstanceId"] = instanceId

	ret := []SElasticcacheAcl{}
	err := DoListAll(self.kvsRequest, "DescribeSecurityIps", params, []string{"SecurityIpGroups", "SecurityIpGroup"}, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "region.GetElasticCacheAcls")
	}

	return ret, nil
}

// https://help.aliyun.com/document_detail/61081.html?spm=a2c4g.11186623.6.754.10613852qAbEQV
func (self *SRegion) GetElasticCacheBackups(instanceId, startTime, endTime string) ([]SElasticcacheBackup, error) {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["InstanceId"] = instanceId
	params["StartTime"] = startTime
	params["EndTime"] = endTime

	ret := []SElasticcacheBackup{}
	err := DoListAll(self.kvsRequest, "DescribeBackups", params, []string{"Backups", "Backup"}, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "region.GetElasticCacheBackups")
	}

	return ret, nil
}

// https://help.aliyun.com/document_detail/93078.html?spm=a2c4g.11186623.6.769.58011975YYL5Gl
func (self *SRegion) GetElasticCacheParameters(instanceId string) ([]SElasticcacheParameter, error) {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["DBInstanceId"] = instanceId

	ret := []SElasticcacheParameter{}
	err := DoListAll(self.kvsRequest, "DescribeParameters", params, []string{"RunningParameters", "Parameter"}, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "region.GetElasticCacheParameters")
	}

	return ret, nil
}
