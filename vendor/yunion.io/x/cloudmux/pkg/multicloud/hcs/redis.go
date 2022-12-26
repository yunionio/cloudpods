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

package hcs

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423020.html
type SElasticcache struct {
	multicloud.SElasticcacheBase
	HcsTags

	region *SRegion

	Name                  string   `json:"name"`
	Engine                string   `json:"engine"`
	CapacityGB            int      `json:"capacity"`
	IP                    string   `json:"ip"`
	DomainName            string   `json:"domainName"`
	Port                  int      `json:"port"`
	Status                string   `json:"status"`
	Libos                 bool     `json:"libos"`
	Description           string   `json:"description"`
	Task                  string   `json:"task"`
	MaxMemoryMB           int      `json:"max_memory"`
	UsedMemoryMB          int      `json:"used_memory"`
	InstanceId            string   `json:"instance_id"`
	ResourceSpecCode      string   `json:"resource_spec_code"`
	EngineVersion         string   `json:"engine_version"`
	InternalVersion       string   `json:"internal_version"`
	ChargingMode          int      `json:"charging_mode"`
	CapacityMinor         string   `json:"capacity_minor"`
	VpcId                 string   `json:"vpc_id"`
	VpcName               string   `json:"vpc_name"`
	TaskStatus            string   `json:"task_status"`
	CreatedAt             string   `json:"created_at"`
	ErrorCode             string   `json:"error_code"`
	UserId                string   `json:"user_id"`
	UserName              string   `json:"user_name"`
	MaintainBegin         string   `json:"maintain_begin"`
	MaintainEnd           string   `json:"maintain_end"`
	NoPasswordAccess      string   `json:"no_password_access"`
	AccessUser            string   `json:"access_user"`
	EnablePublicip        bool     `json:"enable_publicip"`
	PublicipId            string   `json:"publicip_id"`
	PublicipAddress       string   `json:"publicip_address"`
	EnableSSL             bool     `json:"enable_ssl"`
	ServiceUpgrade        bool     `json:"service_upgrade"`
	ServiceTaskId         string   `json:"service_task_id"`
	IsFree                string   `json:"is_free"`
	EnterpriseProjectId   string   `json:"enterprise_project_id"`
	AvailableZones        []string `json:"available_zones"`
	SubnetId              string   `json:"subnet_id"`
	SecurityGroupId       string   `json:"security_group_id"`
	BackendAddrs          []string `json:"backend_addrs"`
	ProductId             string   `json:"product_id"`
	SecurityGroupName     string   `json:"security_group_name"`
	SubnetName            string   `json:"subnet_name"`
	OrderId               string   `json:"order_id"`
	SubnetCIdR            string   `json:"subnet_cidr"`
	InstanceBackupPolicy  string   `json:"instance_backup_policy"`
	EnterpriseProjectName string   `json:"enterprise_project_name"`
}

func (self *SElasticcache) GetId() string {
	return self.InstanceId
}

func (self *SElasticcache) GetName() string {
	return self.Name
}

func (self *SElasticcache) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticcache) GetProjectId() string {
	return self.EnterpriseProjectId
}

func (self *SElasticcache) Refresh() error {
	cache, err := self.region.GetElasticCache(self.GetId())
	if err != nil {
		return errors.Wrap(err, "Elasticcache.Refresh.GetElasticCache")
	}

	err = jsonutils.Update(self, cache)
	if err != nil {
		return errors.Wrap(err, "Elasticcache.Refresh.Update")
	}

	return nil
}

func (self *SElasticcache) GetStatus() string {
	switch self.Status {
	case "RUNNING":
		return api.ELASTIC_CACHE_STATUS_RUNNING
	case "CREATING":
		return api.ELASTIC_CACHE_STATUS_DEPLOYING
	case "CREATEFAILED":
		return api.ELASTIC_CACHE_STATUS_CREATE_FAILED
	case "ERROR":
		return api.ELASTIC_CACHE_STATUS_ERROR
	case "RESTARTING":
		return api.ELASTIC_CACHE_STATUS_RESTARTING
	case "FROZEN":
		return api.ELASTIC_CACHE_STATUS_UNAVAILABLE
	case "EXTENDING":
		return api.ELASTIC_CACHE_STATUS_CHANGING
	case "RESTORING":
		return api.ELASTIC_CACHE_STATUS_TRANSFORMING // ?
	case "FLUSHING":
		return api.ELASTIC_CACHE_STATUS_FLUSHING
	}

	return ""
}

func (self *SElasticcache) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SElasticcache) GetCreatedAt() time.Time {
	var createtime time.Time
	if len(self.CreatedAt) > 0 {
		createtime, _ = time.Parse("2006-01-02T15:04:05.000Z", self.CreatedAt)
	}

	return createtime
}

func (self *SElasticcache) GetInstanceType() string {
	return self.ResourceSpecCode
}

func (self *SElasticcache) GetCapacityMB() int {
	return self.CapacityGB * 1024
}

func (self *SElasticcache) GetArchType() string {
	if strings.Contains(self.ResourceSpecCode, "single") {
		return api.ELASTIC_CACHE_ARCH_TYPE_SINGLE
	}
	if strings.Contains(self.ResourceSpecCode, "ha") || strings.Contains(self.ResourceSpecCode, "master") {
		return api.ELASTIC_CACHE_ARCH_TYPE_MASTER
	}
	if strings.Contains(self.ResourceSpecCode, "cluster") {
		return api.ELASTIC_CACHE_ARCH_TYPE_CLUSTER
	}
	if strings.Contains(self.ResourceSpecCode, "proxy") {
		return api.ELASTIC_CACHE_ARCH_TYPE_CLUSTER
	}
	return self.ResourceSpecCode
}

func (self *SElasticcache) GetNodeType() string {
	// single（单副本） | double（双副本)
	if strings.Contains(self.ResourceSpecCode, "single") {
		return "single"
	}
	return "double"
}

func (self *SElasticcache) GetEngine() string {
	return self.Engine
}

func (self *SElasticcache) GetEngineVersion() string {
	return self.EngineVersion
}

func (self *SElasticcache) GetVpcId() string {
	return self.VpcId
}

func (self *SElasticcache) GetZoneId() string {
	if len(self.AvailableZones) > 0 {
		zone, err := self.region.GetZoneById(self.AvailableZones[0])
		if err != nil {
			log.Errorf("elasticcache.GetZoneId %s", err)
			return ""
		}
		return zone.GetGlobalId()
	}
	return ""
}

func (self *SElasticcache) GetNetworkType() string {
	return api.LB_NETWORK_TYPE_VPC
}

func (self *SElasticcache) GetNetworkId() string {
	return self.SubnetId
}

func (self *SElasticcache) GetPrivateDNS() string {
	return self.DomainName
}

func (self *SElasticcache) GetPrivateIpAddr() string {
	return self.IP
}

func (self *SElasticcache) GetPrivateConnectPort() int {
	return self.Port
}

func (self *SElasticcache) GetPublicDNS() string {
	return self.PublicipAddress
}

func (self *SElasticcache) GetPublicIpAddr() string {
	return self.PublicipAddress
}

func (self *SElasticcache) GetPublicConnectPort() int {
	return self.Port
}

func (self *SElasticcache) GetMaintainStartTime() string {
	return self.MaintainBegin
}

func (self *SElasticcache) GetMaintainEndTime() string {
	return self.MaintainEnd
}

func (self *SElasticcache) GetICloudElasticcacheAccounts() ([]cloudprovider.ICloudElasticcacheAccount, error) {
	iaccounts := []cloudprovider.ICloudElasticcacheAccount{}
	iaccount := &SElasticcacheAccount{cacheDB: self}
	iaccounts = append(iaccounts, iaccount)
	return iaccounts, nil
}

func (self *SElasticcache) GetICloudElasticcacheAcls() ([]cloudprovider.ICloudElasticcacheAcl, error) {
	// 华为云使用安全组做访问控制。目前未支持
	return []cloudprovider.ICloudElasticcacheAcl{}, nil
}

func (self *SElasticcache) GetICloudElasticcacheBackups() ([]cloudprovider.ICloudElasticcacheBackup, error) {
	start := self.GetCreatedAt().Format("20060102150405")
	end := time.Now().Format("20060102150405")
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

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423035.html
func (self *SRegion) GetElasticCacheBackups(instanceId, startTime, endTime string) ([]SElasticcacheBackup, error) {
	params := url.Values{}
	params.Set("beginTime", startTime)
	params.Set("endTime", endTime)
	res := fmt.Sprintf("instances/%s/backups", instanceId)
	ret := []SElasticcacheBackup{}
	return ret, self.redisList(res, params, &ret)
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423027.html
func (self *SRegion) GetElasticCacheParameters(instanceId string) ([]SElasticcacheParameter, error) {
	params := url.Values{}
	res := fmt.Sprintf("instances/%s/configs", instanceId)
	ret := []SElasticcacheParameter{}
	return ret, self.redisList(res, params, &ret)
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423044.html
func (self *SRegion) GetElasticCaches() ([]SElasticcache, error) {
	params := url.Values{}
	ret := []SElasticcache{}
	return ret, self.redisList("instances", params, &ret)
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423020.html
func (self *SRegion) GetElasticCache(instanceId string) (*SElasticcache, error) {
	cache := &SElasticcache{region: self}
	res := fmt.Sprintf("instances/%s", instanceId)
	return cache, self.redisGet(res, cache)
}

func (self *SRegion) GetIElasticcacheById(id string) (cloudprovider.ICloudElasticcache, error) {
	ec, err := self.GetElasticCache(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetElasticCache(%s)", id)
	}
	return ec, nil
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423047.html
func (self *SRegion) zoneNameToDcsZoneIds(zoneIds []string) ([]string, error) {
	type Z struct {
		Id                   string `json:"id"`
		Code                 string `json:"code"`
		Name                 string `json:"name"`
		Port                 string `json:"port"`
		ResourceAvailability string `json:"resource_availability"`
	}

	rs := []Z{}
	err := self.redisList("availableZones", url.Values{}, &rs)
	if err != nil {
		return nil, errors.Wrapf(err, "availableZones")
	}
	zoneMap := map[string]string{}
	for i := range rs {
		if rs[i].ResourceAvailability == "true" {
			zoneMap[rs[i].Code] = rs[i].Id
		}
	}

	ret := []string{}
	for _, zone := range zoneIds {
		if id, ok := zoneMap[zone]; ok {
			ret = append(ret, id)
		} else {
			return nil, errors.Wrap(fmt.Errorf("zone %s not found or not available", zone), "region.zoneNameToDcsZoneIds")
		}
	}

	return ret, nil
}

func (self *SRegion) CreateElasticcache(opts *cloudprovider.SCloudElasticCacheInput) (*SElasticcache, error) {
	params := map[string]interface{}{
		"name":               opts.InstanceName,
		"engine":             opts.Engine,
		"engine_version":     opts.EngineVersion,
		"capacity":           opts.CapacityGB,
		"vpc_id":             opts.VpcId,
		"subnet_id":          opts.NetworkId,
		"product_id":         opts.InstanceType,
		"spec_code":          opts.InstanceType,
		"available_zones":    opts.ZoneIds,
		"no_password_access": true,
		"enable_publicip":    false,
	}
	if len(opts.SecurityGroupIds) > 0 {
		params["security_group_id"] = opts.SecurityGroupIds[0]
	} else {
		secgroups, err := self.GetSecurityGroups(opts.VpcId)
		if err != nil {
			return nil, errors.Wrapf(err, "GetSecurityGroups")
		}
		for _, secgroup := range secgroups {
			params["security_group_id"] = secgroup.Id
			break
		}
	}

	if len(opts.ProjectId) > 0 {
		params["enterprise_project_id"] = opts.ProjectId
	}

	if len(opts.Password) > 0 {
		params["no_password_access"] = false
		params["password"] = opts.Password
		if opts.Engine == "Memcache" {
			params["access_user"] = opts.UserName
		}
	}

	if len(opts.EipId) > 0 {
		params["enable_publicip"] = true
		params["publicip_id"] = opts.EipId
	}

	if len(opts.PrivateIpAddress) > 0 {
		params["private_ip"] = opts.PrivateIpAddress
	}

	if len(opts.MaintainBegin) > 0 {
		params["maintain_begin"] = opts.MaintainBegin
		params["maintain_end"] = opts.MaintainEnd
	}

	ret := struct {
		InstanceId string
	}{}
	err := self.redisCreate("instances", params, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "redisCreate")
	}
	return self.GetElasticCache(ret.InstanceId)
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423019.html
func (self *SRegion) CreateIElasticcaches(opts *cloudprovider.SCloudElasticCacheInput) (cloudprovider.ICloudElasticcache, error) {
	cache, err := self.CreateElasticcache(opts)
	if err != nil {
		return nil, err
	}
	return cache, nil
}

func (self *SElasticcache) Restart() error {
	return nil
}

func (self *SElasticcache) Delete() error {
	return self.region.DeleteElasticcache(self.InstanceId)
}

func (self *SRegion) DeleteElasticcache(id string) error {
	res := fmt.Sprintf("instances/%s", id)
	return self.redisDelete(res)
}

func (self *SElasticcache) ChangeInstanceSpec(spec string) error {
	return cloudprovider.ErrNotSupported
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423021.html
func (self *SElasticcache) SetMaintainTime(maintainStartTime, maintainEndTime string) error {
	return cloudprovider.ErrNotImplemented
}

// https://support.huaweicloud.com/usermanual-dcs/dcs-zh-ug-180314001.html
// 目前只有Redis3.0版本密码模式的实例支持通过公网访问Redis实例，其他版本暂不支持公网访问。
// todo: 目前没找到api
func (self *SElasticcache) AllocatePublicConnection(port int) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

// todo: 目前没找到api
func (self *SElasticcache) ReleasePublicConnection() error {
	return cloudprovider.ErrNotSupported
}

func (self *SElasticcache) CreateAccount(account cloudprovider.SCloudElasticCacheAccountInput) (cloudprovider.ICloudElasticcacheAccount, error) {
	return nil, cloudprovider.ErrNotSupported
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423031.html

func (self *SElasticcache) CreateAcl(aclName, securityIps string) (cloudprovider.ICloudElasticcacheAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423029.html
func (self *SElasticcache) UpdateInstanceParameters(config jsonutils.JSONObject) error {
	return cloudprovider.ErrNotImplemented
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423033.html
func (self *SElasticcache) CreateBackup(desc string) (cloudprovider.ICloudElasticcacheBackup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func backupPeriodTrans(config cloudprovider.SCloudElasticCacheBackupPolicyUpdateInput) *jsonutils.JSONArray {
	segs := strings.Split(config.PreferredBackupPeriod, ",")
	ret := jsonutils.NewArray()
	for _, seg := range segs {
		switch seg {
		case "Monday":
			ret.Add(jsonutils.NewString("1"))
		case "Tuesday":
			ret.Add(jsonutils.NewString("2"))
		case "Wednesday":
			ret.Add(jsonutils.NewString("3"))
		case "Thursday":
			ret.Add(jsonutils.NewString("4"))
		case "Friday":
			ret.Add(jsonutils.NewString("5"))
		case "Saturday":
			ret.Add(jsonutils.NewString("6"))
		case "Sunday":
			ret.Add(jsonutils.NewString("7"))
		}
	}

	return ret
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423021.html
func (self *SElasticcache) UpdateBackupPolicy(config cloudprovider.SCloudElasticCacheBackupPolicyUpdateInput) error {
	return nil
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423030.html
// 当前版本，只有DCS2.0实例支持清空数据功能，即flush操作。
func (self *SElasticcache) FlushInstance(input cloudprovider.SCloudElasticCacheFlushInstanceInput) error {
	return nil
}

// SElasticcacheAccount => ResetPassword
func (self *SElasticcache) UpdateAuthMode(noPwdAccess bool, password string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SElasticcache) GetAuthMode() string {
	switch self.NoPasswordAccess {
	case "true":
		return "off"
	default:
		return "on"
	}
}

func (self *SElasticcache) GetSecurityGroupIds() ([]string, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SElasticcache) GetICloudElasticcacheAccount(accountId string) (cloudprovider.ICloudElasticcacheAccount, error) {
	accounts, err := self.GetICloudElasticcacheAccounts()
	if err != nil {
		return nil, errors.Wrap(err, "Elasticcache.GetICloudElasticcacheAccount.Accounts")
	}

	for i := range accounts {
		account := accounts[i]
		if account.GetGlobalId() == accountId {
			return account, nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func (self *SElasticcache) GetICloudElasticcacheAcl(aclId string) (cloudprovider.ICloudElasticcacheAcl, error) {
	return nil, cloudprovider.ErrNotSupported
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

func (instance *SElasticcache) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SElasticcache) UpdateSecurityGroups(secgroupIds []string) error {
	return errors.Wrap(cloudprovider.ErrNotSupported, "UpdateSecurityGroups")
}

func (self *SElasticcache) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (self *SElasticcache) SetAutoRenew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) GetIElasticcaches() ([]cloudprovider.ICloudElasticcache, error) {
	caches, err := self.GetElasticCaches()
	if err != nil {
		return nil, err
	}

	ret := []cloudprovider.ICloudElasticcache{}
	for i := range caches {
		caches[i].region = self
		ret = append(ret, &caches[i])
	}

	return ret, nil
}
