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
	"strconv"
	"strings"
	"time"

	"github.com/aokoli/goutils"
	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/billing"
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
	switch self.InstanceStatus {
	case "Normal":
		return api.ELASTIC_CACHE_STATUS_RUNNING
	case "Creating":
		return api.ELASTIC_CACHE_STATUS_DEPLOYING
	case "Changing":
		return api.ELASTIC_CACHE_STATUS_CHANGING
	case "Inactive":
		return api.ELASTIC_CACHE_STATUS_INACTIVE
	case "Flushing":
		return api.ELASTIC_CACHE_STATUS_FLUSHING
	case "Released":
		return api.ELASTIC_CACHE_STATUS_RELEASED
	case "Transforming":
		return api.ELASTIC_CACHE_STATUS_TRANSFORMING
	case "Unavailable":
		return api.ELASTIC_CACHE_STATUS_UNAVAILABLE
	case "Error":
		return api.ELASTIC_CACHE_STATUS_ERROR
	case "Migrating":
		return api.ELASTIC_CACHE_STATUS_MIGRATING
	case "BackupRecovering":
		return api.ELASTIC_CACHE_STATUS_BACKUPRECOVERING
	case "MinorVersionUpgrading":
		return api.ELASTIC_CACHE_STATUS_MINORVERSIONUPGRADING
	case "NetworkModifying":
		return api.ELASTIC_CACHE_STATUS_NETWORKMODIFYING
	case "SSLModifying":
		return api.ELASTIC_CACHE_STATUS_SSLMODIFYING
	case "MajorVersionUpgrading":
		return api.ELASTIC_CACHE_STATUS_MAJORVERSIONUPGRADING
	default:
		return api.ELASTIC_CACHE_STATUS_MAJORVERSIONUPGRADING
	}
}

func (self *SElasticcache) Refresh() error {
	cache, err := self.region.GetElasticCacheById(self.GetId())
	if err != nil {
		return err
	}

	err = jsonutils.Update(self, cache)
	if err != nil {
		return err
	}

	return nil
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
	switch self.ArchitectureType {
	case "rwsplit":
		return api.ELASTIC_CACHE_ARCH_TYPE_RWSPLIT
	case "cluster":
		return api.ELASTIC_CACHE_ARCH_TYPE_CLUSTER
	case "standard":
		if self.NodeType == "single" {
			return api.ELASTIC_CACHE_ARCH_TYPE_SINGLE
		} else if self.NodeType == "double" {
			return api.ELASTIC_CACHE_ARCH_TYPE_MASTER
		}
	}

	return ""
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

func (self *SElasticcache) GetICloudElasticcacheAccountByName(accountName string) (cloudprovider.ICloudElasticcacheAccount, error) {
	account, err := self.region.GetElasticCacheAccountByName(self.GetId(), accountName)
	if err != nil {
		return nil, err
	}

	account.cacheDB = self
	return account, nil
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
	if count >= 1 {
		self.attribute = &rets[0]
		return self.attribute, nil
	} else {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "elasticcache.GetAttribute %s", self.GetId())
	}
}

func (self *SElasticcache) GetNetInfo() ([]SNetInfo, error) {
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

// https://help.aliyun.com/document_detail/66742.html?spm=a2c4g.11186623.6.731.54c123d2P02qhk
func (self *SElasticcache) GetPublicNetInfo() (*SNetInfo, error) {
	nets, err := self.GetNetInfo()
	if err != nil {
		return nil, err
	}

	for i := range nets {
		if nets[i].IPType == "Public" {
			return &nets[i], nil
		}
	}

	return nil, nil
}

func (self *SRegion) GetElasticCaches(instanceIds []string) ([]SElasticcache, error) {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	if instanceIds != nil && len(instanceIds) > 0 {
		params["InstanceIds"] = strings.Join(instanceIds, ",")
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
		return nil, cloudprovider.ErrNotFound
	} else {
		return nil, errors.Wrapf(cloudprovider.ErrDuplicateId, "region.GetElasticCacheById %s.expect 1 found %d ", instanceId, len(caches))
	}
}

func (self *SRegion) GetIElasticcacheById(id string) (cloudprovider.ICloudElasticcache, error) {
	ec, err := self.GetElasticCacheById(id)
	if err != nil {
		return nil, err
	}

	return ec, nil
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

func (self *SRegion) GetElasticCacheAccountByName(instanceId string, accountName string) (*SElasticcacheAccount, error) {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["InstanceId"] = instanceId
	params["AccountName"] = accountName

	ret := []SElasticcacheAccount{}
	err := DoListAll(self.kvsRequest, "DescribeAccounts", params, []string{"Accounts", "Account"}, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "region.GetElasticCacheAccounts")
	}

	if len(ret) == 1 {
		return &ret[0], nil
	} else if len(ret) == 0 {
		return nil, cloudprovider.ErrNotFound
	} else {
		return nil, errors.Wrap(fmt.Errorf("%d account with name %s found", len(ret), accountName), "region.GetElasticCacheAccountByName")
	}
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

// https://help.aliyun.com/document_detail/60873.html?spm=a2c4g.11174283.6.715.7412dce0qSYemb
func (self *SRegion) CreateIElasticcaches(ec *cloudprovider.SCloudElasticCacheInput) (cloudprovider.ICloudElasticcache, error) {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["InstanceClass"] = ec.InstanceType
	params["InstanceName"] = ec.InstanceName
	params["InstanceType"] = ec.Engine
	params["EngineVersion"] = ec.EngineVersion

	if len(ec.Password) > 0 {
		params["UserName"] = ec.UserName
		params["Password"] = ec.Password
	}

	if len(ec.ZoneIds) > 0 {
		params["ZoneId"] = ec.ZoneIds[0]
	}

	if len(ec.PrivateIpAddress) > 0 {
		params["PrivateIpAddress"] = ec.PrivateIpAddress
	}

	if len(ec.NodeType) > 0 {
		params["NodeType"] = ec.NodeType
	}

	params["NetworkType"] = ec.NetworkType
	params["VpcId"] = ec.VpcId
	params["VSwitchId"] = ec.NetworkId
	params["ChargeType"] = ec.ChargeType
	if ec.ChargeType == billing.BILLING_TYPE_PREPAID && ec.BC != nil {
		if ec.BC.GetMonths() >= 1 && ec.BC.GetMonths() <= 9 {
			params["Period"] = strconv.Itoa(ec.BC.GetMonths())
		} else if ec.BC.GetMonths() == 12 || ec.BC.GetMonths() == 24 || ec.BC.GetMonths() == 36 {
			params["Period"] = strconv.Itoa(ec.BC.GetMonths())
		} else {
			return nil, fmt.Errorf("region.CreateIElasticcaches invalid billing cycle.reqired month (1~9) or  year(1~3)")
		}
	}

	ret := &SElasticcache{}
	err := DoAction(self.kvsRequest, "CreateInstance", params, []string{}, ret)
	if err != nil {
		return nil, errors.Wrap(err, "region.CreateIElasticcaches")
	}

	ret.region = self
	return ret, nil
}

// https://help.aliyun.com/document_detail/116215.html?spm=a2c4g.11174283.6.736.6b9ddce0c5nsw6
func (self *SElasticcache) Restart() error {
	params := make(map[string]string)
	params["InstanceId"] = self.GetId()
	params["EffectiveTime"] = "0" // 立即重启

	err := DoAction(self.region.kvsRequest, "RestartInstance", params, nil, nil)
	if err != nil {
		return errors.Wrap(err, "elasticcache.Restart")
	}

	return nil
}

// https://help.aliyun.com/document_detail/60898.html?spm=a2c4g.11186623.6.713.56ec1603KS0xA0
func (self *SElasticcache) Delete() error {
	params := make(map[string]string)
	params["InstanceId"] = self.GetId()

	err := DoAction(self.region.kvsRequest, "DeleteInstance", params, nil, nil)
	if err != nil {
		return errors.Wrap(err, "elasticcache.Delete")
	}

	return nil
}

// https://help.aliyun.com/document_detail/60903.html?spm=a2c4g.11186623.6.711.3f062c92aRJNfw
func (self *SElasticcache) ChangeInstanceSpec(spec string) error {
	params := make(map[string]string)
	params["InstanceId"] = self.GetId()
	params["InstanceClass"] = strings.Split(spec, ":")[0]
	params["AutoPay"] = "true" // 自动付款

	err := DoAction(self.region.kvsRequest, "ModifyInstanceSpec", params, nil, nil)
	if err != nil {
		return errors.Wrap(err, "elasticcache.ChangeInstanceSpec")
	}

	return nil
}

// https://help.aliyun.com/document_detail/61000.html?spm=a2c4g.11186623.6.730.57d66cb0QlQS86
func (self *SElasticcache) SetMaintainTime(maintainStartTime, maintainEndTime string) error {
	params := make(map[string]string)
	params["InstanceId"] = self.GetId()
	params["MaintainStartTime"] = maintainStartTime
	params["MaintainEndTime"] = maintainEndTime

	err := DoAction(self.region.kvsRequest, "ModifyInstanceMaintainTime", params, nil, nil)
	if err != nil {
		return errors.Wrap(err, "elasticcache.SetMaintainTime")
	}

	return nil
}

// https://help.aliyun.com/document_detail/125795.html?spm=a2c4g.11186623.6.719.51542b3593cDKO
func (self *SElasticcache) AllocatePublicConnection(port int) (string, error) {
	if port < 1024 {
		port = 6379
	}

	suffix, _ := goutils.RandomAlphabetic(4)
	conn := self.GetId() + strings.ToLower(suffix)

	params := make(map[string]string)
	params["InstanceId"] = self.GetId()
	params["Port"] = strconv.Itoa(port)
	params["ConnectionStringPrefix"] = conn

	err := DoAction(self.region.kvsRequest, "AllocateInstancePublicConnection", params, nil, nil)
	if err != nil {
		return "", errors.Wrap(err, "elasticcache.AllocatePublicConnection")
	}

	return conn, nil
}

// https://help.aliyun.com/document_detail/125796.html?spm=a2c4g.11186623.6.720.702a3b23Qayopy
func (self *SElasticcache) ReleasePublicConnection() error {
	publicConn := self.GetPublicDNS()
	if len(publicConn) == 0 {
		log.Debugf("elasticcache.ReleasePublicConnection public connect is empty")
		return nil
	}

	params := make(map[string]string)
	params["InstanceId"] = self.GetId()
	params["CurrentConnectionString"] = publicConn

	err := DoAction(self.region.kvsRequest, "ReleaseInstancePublicConnection", params, nil, nil)
	if err != nil {
		return errors.Wrap(err, "elasticcache.ReleasePublicConnection")
	}

	return nil
}

// https://help.aliyun.com/document_detail/93603.html?spm=a2c4g.11186623.6.732.10666cf9UVgNPb 修改链接地址

// https://help.aliyun.com/document_detail/95973.html?spm=a2c4g.11186623.6.742.4698126aH0s4Q5
func (self *SElasticcache) CreateAccount(input cloudprovider.SCloudElasticCacheAccountInput) (cloudprovider.ICloudElasticcacheAccount, error) {
	params := make(map[string]string)
	params["InstanceId"] = self.GetId()
	params["AccountName"] = input.AccountName
	params["AccountPassword"] = input.AccountPassword
	if len(input.AccountPrivilege) > 0 {
		params["AccountPrivilege"] = input.AccountPrivilege
	}

	if len(input.Description) > 0 {
		params["AccountDescription"] = input.Description
	}

	err := DoAction(self.region.kvsRequest, "CreateAccount", params, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "elasticcache.CreateAccount")
	}

	return self.GetICloudElasticcacheAccountByName(input.AccountName)
}

func (self *SElasticcache) CreateAcl(aclName, securityIps string) (cloudprovider.ICloudElasticcacheAcl, error) {
	acl := &SElasticcacheAcl{}
	acl.cacheDB = self
	acl.SecurityIPGroupName = aclName
	acl.SecurityIPList = securityIps
	return acl, self.region.createAcl(self.GetId(), aclName, securityIps)
}

// https://help.aliyun.com/document_detail/61113.html?spm=a2c4g.11186623.6.770.16a61e7admr0xy
func (self *SElasticcache) UpdateInstanceParameters(config jsonutils.JSONObject) error {
	params := make(map[string]string)
	params["InstanceId"] = self.GetId()
	params["Config"] = config.String()

	err := DoAction(self.region.kvsRequest, "ModifyInstanceConfig", params, nil, nil)
	if err != nil {
		return errors.Wrap(err, "elasticcache.UpdateInstanceParameters")
	}

	return nil
}

// https://help.aliyun.com/document_detail/61075.html?spm=a2c4g.11186623.6.749.4cba126a2U9xNa
func (self *SElasticcache) CreateBackup() (cloudprovider.ICloudElasticcacheBackup, error) {
	params := make(map[string]string)
	params["InstanceId"] = self.GetId()

	// 目前没有查询备份ID的接口，因此，备份ID没什么用
	err := DoAction(self.region.kvsRequest, "CreateBackup", params, []string{"BackupJobID"}, nil)
	if err != nil {
		return nil, errors.Wrap(err, "elasticcache.CreateBackup")
	}

	return nil, nil
}

// https://help.aliyun.com/document_detail/61077.html?spm=a2c4g.11186623.6.750.3d7630be7ziUfg
func (self *SElasticcache) UpdateBackupPolicy(config cloudprovider.SCloudElasticCacheBackupPolicyUpdateInput) error {
	params := make(map[string]string)
	params["InstanceId"] = self.GetId()
	params["PreferredBackupPeriod"] = config.PreferredBackupPeriod
	params["PreferredBackupTime"] = config.PreferredBackupTime

	err := DoAction(self.region.kvsRequest, "ModifyBackupPolicy", params, nil, nil)
	if err != nil {
		return errors.Wrap(err, "elasticcache.UpdateBackupPolicy")
	}

	return nil
}

// https://help.aliyun.com/document_detail/60931.html?spm=a2c4g.11186623.6.728.5c57292920UKx3
func (self *SElasticcache) FlushInstance() error {
	params := make(map[string]string)
	params["InstanceId"] = self.GetId()

	err := DoAction(self.region.kvsRequest, "FlushInstance", params, nil, nil)
	if err != nil {
		return errors.Wrap(err, "elasticcache.FlushInstance")
	}

	return nil
}

// https://help.aliyun.com/document_detail/98531.html?spm=5176.11065259.1996646101.searchclickresult.4df474c38Sc2SO
func (self *SElasticcache) UpdateAuthMode(noPwdAccess bool) error {
	params := make(map[string]string)
	params["InstanceId"] = self.GetId()
	if noPwdAccess {
		params["VpcAuthMode"] = "Close"
	} else {
		params["VpcAuthMode"] = "Open"
	}

	err := DoAction(self.region.kvsRequest, "ModifyInstanceVpcAuthMode", params, nil, nil)
	if err != nil {
		return errors.Wrap(err, "elasticcacheAccount.UpdateAuthMode")
	}

	return nil
}

func (self *SElasticcache) GetAuthMode() string {
	attribute, err := self.GetAttribute()
	if err != nil {
		log.Errorf("elasticcache.GetAuthMode %s", err)
	}
	switch attribute.VpcAuthMode {
	case "Open":
		return "on"
	default:
		return "off"
	}
}

func (self *SElasticcache) GetICloudElasticcacheAccount(accountId string) (cloudprovider.ICloudElasticcacheAccount, error) {
	segs := strings.Split(accountId, "/")
	if len(segs) < 2 {
		return nil, errors.Wrap(fmt.Errorf("%s", accountId), "elasticcache.GetICloudElasticcacheAccount invalid account id ")
	}

	return self.GetICloudElasticcacheAccountByName(segs[1])
}

func (self *SElasticcache) GetICloudElasticcacheAcl(aclId string) (cloudprovider.ICloudElasticcacheAcl, error) {
	acls, err := self.GetICloudElasticcacheAcls()
	if err != nil {
		return nil, err
	}

	for _, acl := range acls {
		if acl.GetId() == aclId {
			return acl, nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func (self *SElasticcache) GetICloudElasticcacheBackup(backupId string) (cloudprovider.ICloudElasticcacheBackup, error) {
	backups, err := self.GetICloudElasticcacheBackups()
	if err != nil {
		return nil, err
	}

	for _, backup := range backups {
		if backup.GetId() == backupId {
			return backup, nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}
