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
	"strings"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423020.html
type SElasticcache struct {
	multicloud.SElasticcacheBase

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
	InstanceID            string   `json:"instance_id"`
	ResourceSpecCode      string   `json:"resource_spec_code"`
	EngineVersion         string   `json:"engine_version"`
	InternalVersion       string   `json:"internal_version"`
	ChargingMode          int      `json:"charging_mode"`
	CapacityMinor         string   `json:"capacity_minor"`
	VpcID                 string   `json:"vpc_id"`
	VpcName               string   `json:"vpc_name"`
	TaskStatus            string   `json:"task_status"`
	CreatedAt             string   `json:"created_at"`
	ErrorCode             string   `json:"error_code"`
	UserID                string   `json:"user_id"`
	UserName              string   `json:"user_name"`
	MaintainBegin         string   `json:"maintain_begin"`
	MaintainEnd           string   `json:"maintain_end"`
	NoPasswordAccess      string   `json:"no_password_access"`
	AccessUser            string   `json:"access_user"`
	EnablePublicip        bool     `json:"enable_publicip"`
	PublicipID            string   `json:"publicip_id"`
	PublicipAddress       string   `json:"publicip_address"`
	EnableSSL             bool     `json:"enable_ssl"`
	ServiceUpgrade        bool     `json:"service_upgrade"`
	ServiceTaskID         string   `json:"service_task_id"`
	IsFree                string   `json:"is_free"`
	EnterpriseProjectID   string   `json:"enterprise_project_id"`
	AvailableZones        []string `json:"available_zones"`
	SubnetID              string   `json:"subnet_id"`
	SecurityGroupID       string   `json:"security_group_id"`
	BackendAddrs          []string `json:"backend_addrs"`
	ProductID             string   `json:"product_id"`
	SecurityGroupName     string   `json:"security_group_name"`
	SubnetName            string   `json:"subnet_name"`
	OrderID               string   `json:"order_id"`
	SubnetCIDR            string   `json:"subnet_cidr"`
	InstanceBackupPolicy  string   `json:"instance_backup_policy"`
	EnterpriseProjectName string   `json:"enterprise_project_name"`
}

func (self *SElasticcache) GetId() string {
	return self.InstanceID
}

func (self *SElasticcache) GetName() string {
	return self.Name
}

func (self *SElasticcache) GetGlobalId() string {
	return self.GetId()
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
	// charging_mode “0”：按需计费    “1”：按包年包月计费
	if self.ChargingMode == 1 {
		return billing_api.BILLING_TYPE_PREPAID
	} else {
		return billing_api.BILLING_TYPE_POSTPAID
	}
}

func (self *SElasticcache) GetCreatedAt() time.Time {
	var createtime time.Time
	if len(self.CreatedAt) > 0 {
		createtime, _ = time.Parse("2006-01-02T15:04:05.000Z", self.CreatedAt)
	}

	return createtime
}

func (self *SElasticcache) GetExpiredAt() time.Time {
	var expiredTime time.Time
	if self.ChargingMode == 1 {
		res, err := self.region.GetOrderResourceDetail(self.GetId())
		if err != nil {
			log.Debugln(err)
		}

		expiredTime = res.ExpireTime
	}

	return expiredTime
}

func (self *SElasticcache) GetInstanceType() string {
	// todo: ??
	return self.ResourceSpecCode
}

func (self *SElasticcache) GetCapacityMB() int {
	return self.CapacityGB * 1024
}

func (self *SElasticcache) GetArchType() string {
	/*
	   资源规格标识。

	   dcs.single_node：表示实例类型为单机
	   dcs.master_standby：表示实例类型为主备
	   dcs.cluster：表示实例类型为集群
	*/
	if strings.Contains(self.ResourceSpecCode, "single") {
		return api.ELASTIC_CACHE_ARCH_TYPE_SINGLE
	} else if strings.Contains(self.ResourceSpecCode, "ha") {
		return api.ELASTIC_CACHE_ARCH_TYPE_MASTER
	} else if strings.Contains(self.ResourceSpecCode, "cluster") {
		return api.ELASTIC_CACHE_ARCH_TYPE_CLUSTER
	} else if strings.Contains(self.ResourceSpecCode, "proxy") {
		return api.ELASTIC_CACHE_ARCH_TYPE_CLUSTER
	}

	return ""
}

func (self *SElasticcache) GetNodeType() string {
	return ""
}

func (self *SElasticcache) GetEngine() string {
	return self.Engine
}

func (self *SElasticcache) GetEngineVersion() string {
	return self.EngineVersion
}

func (self *SElasticcache) GetVpcId() string {
	return self.VpcID
}

func (self *SElasticcache) GetZoneId() string {
	if len(self.AvailableZones) > 0 {
		zone, err := self.region.getZoneById(self.AvailableZones[0])
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
	return self.SubnetID
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
	params := make(map[string]string)
	params["instance_id"] = instanceId
	params["beginTime"] = startTime
	params["endTime"] = endTime

	backups := make([]SElasticcacheBackup, 0)
	err := doListAll(self.ecsClient.Elasticcache.ListBackups, params, &backups)
	if err != nil {
		return nil, err
	}

	return backups, nil
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423027.html
func (self *SRegion) GetElasticCacheParameters(instanceId string) ([]SElasticcacheParameter, error) {
	params := make(map[string]string)
	params["instance_id"] = instanceId

	parameters := make([]SElasticcacheParameter, 0)
	err := doListAll(self.ecsClient.Elasticcache.ListParameters, params, &parameters)
	if err != nil {
		return nil, err
	}

	return parameters, nil
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423044.html
func (self *SRegion) GetElasticCaches() ([]SElasticcache, error) {
	params := make(map[string]string)
	caches := make([]SElasticcache, 0)
	err := doListAll(self.ecsClient.Elasticcache.List, params, &caches)
	if err != nil {
		return nil, errors.Wrap(err, "region.GetElasticCaches")
	}

	for i := range caches {
		cache, err := self.GetElasticCache(caches[i].GetId())
		if err != nil {
			return nil, err
		} else {
			caches[i] = *cache
		}

		caches[i].region = self
	}

	return caches, nil
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423020.html
func (self *SRegion) GetElasticCache(instanceId string) (*SElasticcache, error) {
	cache := SElasticcache{}
	err := DoGet(self.ecsClient.Elasticcache.Get, instanceId, nil, &cache)
	if err != nil {
		return nil, errors.Wrapf(err, "region.GetElasticCache %s", instanceId)
	}

	cache.region = self
	return &cache, nil
}

func (self *SRegion) GetIElasticcacheById(id string) (cloudprovider.ICloudElasticcache, error) {
	ec, err := self.GetElasticCache(id)
	if err != nil {
		return nil, errors.Wrap(err, "region.GetIElasticCacheById.GetElasticCache")
	}

	return ec, nil
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423047.html
func (self *SRegion) zoneNameToDcsZoneIds(zoneIds []string) ([]string, error) {
	type Z struct {
		ID                   string `json:"id"`
		Code                 string `json:"code"`
		Name                 string `json:"name"`
		Port                 string `json:"port"`
		ResourceAvailability string `json:"resource_availability"`
	}

	rs := []Z{}
	err := doListAll(self.ecsClient.DcsAvailableZone.List, nil, &rs)
	if err != nil {
		return nil, errors.Wrap(err, "region.zoneNameToDcsZoneIds")
	}

	zoneMap := map[string]string{}
	for i := range rs {
		if rs[i].ResourceAvailability == "true" {
			zoneMap[rs[i].Code] = rs[i].ID
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

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423019.html
func (self *SRegion) CreateIElasticcaches(ec *cloudprovider.SCloudElasticCacheInput) (cloudprovider.ICloudElasticcache, error) {
	params := jsonutils.NewDict()
	params.Set("name", jsonutils.NewString(ec.InstanceName))
	params.Set("engine", jsonutils.NewString(ec.Engine))
	params.Set("engine_version", jsonutils.NewString(ec.EngineVersion))
	params.Set("capacity", jsonutils.NewInt(ec.CapacityGB))
	params.Set("vpc_id", jsonutils.NewString(ec.VpcId))
	params.Set("security_group_id", jsonutils.NewString(ec.SecurityGroupId))
	params.Set("subnet_id", jsonutils.NewString(ec.NetworkId))
	params.Set("product_id", jsonutils.NewString(ec.InstanceType))
	zones, err := self.zoneNameToDcsZoneIds(ec.ZoneIds)
	if err != nil {
		return nil, err
	}
	params.Set("available_zones", jsonutils.NewStringArray(zones))

	if len(ec.Password) > 0 {
		params.Set("no_password_access", jsonutils.NewString("false"))
		params.Set("password", jsonutils.NewString(ec.Password))

		// todo: 这里换成常量
		if ec.Engine == "Memcache" {
			params.Set("access_user", jsonutils.NewString(ec.UserName))
		}
	} else {
		params.Set("no_password_access", jsonutils.NewString("true"))
	}

	if len(ec.EipId) > 0 {
		params.Set("enable_publicip", jsonutils.NewString("true"))
		params.Set("publicip_id", jsonutils.NewString(ec.EipId))
		// enable_ssl
	} else {
		params.Set("enable_publicip", jsonutils.NewString("false"))
	}

	if len(ec.PrivateIpAddress) > 0 {
		params.Set("private_ip", jsonutils.NewString(ec.PrivateIpAddress))
	}

	if len(ec.MaintainBegin) > 0 {
		params.Set("maintain_begin", jsonutils.NewString(ec.MaintainBegin))
		params.Set("maintain_end", jsonutils.NewString(ec.MaintainEnd))
	}

	if ec.ChargeType == billing_api.BILLING_TYPE_PREPAID && ec.BC != nil {
		bssParam := jsonutils.NewDict()
		bssParam.Set("charging_mode", jsonutils.NewString("prePaid"))
		bssParam.Set("is_auto_pay", jsonutils.NewString("true"))
		bssParam.Set("is_auto_renew", jsonutils.NewString("false"))
		if ec.BC.GetMonths() >= 1 && ec.BC.GetMonths() >= 9 {
			bssParam.Set("period_type", jsonutils.NewString("month"))
			bssParam.Set("period_num", jsonutils.NewInt(int64(ec.BC.GetMonths())))
		} else if ec.BC.GetYears() >= 1 && ec.BC.GetYears() <= 3 {
			bssParam.Set("period_type", jsonutils.NewString("year"))
			bssParam.Set("period_num", jsonutils.NewInt(int64(ec.BC.GetYears())))
		} else {
			return nil, fmt.Errorf("region.CreateIElasticcaches invalid billing cycle.reqired month (1~9) or  year(1~3)")
		}

		params.Set("bss_param", bssParam)
	}

	ret := &SElasticcache{}
	err = DoCreate(self.ecsClient.Elasticcache.Create, params, ret)
	if err != nil {
		return nil, errors.Wrap(err, "region.CreateIElasticcaches")
	}

	ret.region = self
	return ret, nil
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423030.html
func (self *SElasticcache) Restart() error {
	resp, err := self.region.ecsClient.Elasticcache.Restart(self.GetId())
	if err != nil {
		return errors.Wrap(err, "elasticcache.Restart")
	}

	rets, err := resp.GetArray("results")
	if err != nil {
		return errors.Wrap(err, "elasticcache.results")
	}

	for _, r := range rets {
		if ret, _ := r.GetString("result"); ret != "success" {
			return fmt.Errorf("elasticcache.Restart failed")
		}
	}

	return nil
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423022.html
func (self *SElasticcache) Delete() error {
	err := DoDelete(self.region.ecsClient.Elasticcache.Delete, self.GetId(), nil, nil)
	if err != nil {
		return errors.Wrap(err, "elasticcache.Delete")
	}

	return nil
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423024.html
func (self *SElasticcache) ChangeInstanceSpec(spec string) error {
	segs := strings.Split(spec, ":")
	if len(segs) < 2 {
		return fmt.Errorf("elasticcache.ChangeInstanceSpec invalid sku %s", spec)
	}

	if !strings.HasPrefix(segs[1], "m") || !strings.HasSuffix(segs[1], "g") {
		return fmt.Errorf("elasticcache.ChangeInstanceSpec sku %s memeory size is invalid.", spec)
	}

	newCapacity := segs[1][1 : len(segs[1])-1]
	_, err := self.region.ecsClient.Elasticcache.ChangeInstanceSpec(self.GetId(), newCapacity)
	if err != nil {
		return errors.Wrap(err, "elasticcache.ChangeInstanceSpec")
	}

	return nil
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423021.html
func (self *SElasticcache) SetMaintainTime(maintainStartTime, maintainEndTime string) error {
	params := jsonutils.NewDict()
	params.Set("maintain_begin", jsonutils.NewString(maintainStartTime))
	params.Set("maintain_end", jsonutils.NewString(maintainEndTime))
	err := DoUpdate(self.region.ecsClient.Elasticcache.Update, self.GetId(), params, nil)
	if err != nil {
		return errors.Wrap(err, "elasticcache.SetMaintainTime")
	}

	return nil
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
	params := jsonutils.NewDict()
	params.Set("redis_config", config)
	err := DoUpdateWithSpec(self.region.ecsClient.Elasticcache.UpdateInContextWithSpec, self.GetId(), "configs", params)
	if err != nil {
		return errors.Wrap(err, "elasticcache.UpdateInstanceParameters")
	}

	return nil
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423033.html
func (self *SElasticcache) CreateBackup() (cloudprovider.ICloudElasticcacheBackup, error) {
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
	params := jsonutils.NewDict()
	policy := jsonutils.NewDict()
	policy.Set("save_days", jsonutils.NewInt(int64(config.BackupReservedDays)))
	policy.Set("backup_type", jsonutils.NewString(config.BackupType))
	plan := jsonutils.NewDict()
	backTime := strings.ReplaceAll(config.PreferredBackupTime, "Z", "")
	backupPeriod := backupPeriodTrans(config)
	plan.Set("begin_at", jsonutils.NewString(backTime))
	plan.Set("period_type", jsonutils.NewString("weekly"))
	plan.Set("backup_at", backupPeriod)
	policy.Set("periodical_backup_plan", plan)
	params.Set("instance_backup_policy", policy)
	err := DoUpdateWithSpec(self.region.ecsClient.Elasticcache.UpdateInContextWithSpec, self.GetId(), "configs", params)
	if err != nil {
		return errors.Wrap(err, "elasticcache.UpdateInstanceParameters")
	}

	return nil
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423030.html
// 当前版本，只有DCS2.0实例支持清空数据功能，即flush操作。
func (self *SElasticcache) FlushInstance() error {
	resp, err := self.region.ecsClient.Elasticcache.Flush(self.GetId())
	if err != nil {
		return errors.Wrap(err, "elasticcache.FlushInstance")
	}

	rets, err := resp.GetArray("results")
	if err != nil {
		return errors.Wrap(err, "elasticcache.FlushInstance")
	}

	for _, r := range rets {
		if ret, _ := r.GetString("result"); ret != "success" {
			return fmt.Errorf("elasticcache.FlushInstance failed")
		}
	}

	return nil
}

// SElasticcacheAccount => ResetPassword
func (self *SElasticcache) UpdateAuthMode(noPwdAccess bool) error {
	return cloudprovider.ErrNotSupported
}

func (self *SElasticcache) GetAuthMode() string {
	switch self.NoPasswordAccess {
	case "true":
		return "on"
	default:
		return "off"
	}
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
