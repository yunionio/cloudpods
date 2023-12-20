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

type SElasticcache struct {
	multicloud.SElasticcacheBase
	HuaweiTags

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

func (self *SElasticcache) GetProjectId() string {
	return self.EnterpriseProjectID
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
	orders, err := self.region.client.GetOrderResources()
	if err != nil {
		return time.Time{}
	}
	order, ok := orders[self.InstanceID]
	if ok {
		return order.ExpireTime
	}
	return time.Time{}
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
	if strings.Contains(self.ResourceSpecCode, "ha") {
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

func (self *SRegion) GetElasticCacheBackups(instanceId, startTime, endTime string) ([]SElasticcacheBackup, error) {
	params := url.Values{}
	params.Set("begin_time", startTime)
	params.Set("end_time", endTime)

	ret := make([]SElasticcacheBackup, 0)
	for {
		resp, err := self.list(SERVICE_DCS, fmt.Sprintf("instances/%s/backups", instanceId), params)
		if err != nil {
			return nil, err
		}
		part := struct {
			BackupRecordResponse []SElasticcacheBackup
			TotalNum             int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.BackupRecordResponse...)
		if len(ret) >= part.TotalNum || len(part.BackupRecordResponse) == 0 {
			break
		}
		params.Set("offset", fmt.Sprintf("%d", len(ret)))
	}
	return ret, nil
}

func (self *SRegion) GetElasticCacheParameters(instanceId string) ([]SElasticcacheParameter, error) {
	resp, err := self.list(SERVICE_DCS, fmt.Sprintf("instances/%s/configs", instanceId), nil)
	if err != nil {
		return nil, err
	}
	ret := make([]SElasticcacheParameter, 0)
	err = resp.Unmarshal(&ret, "redis_config")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SRegion) GetElasticCaches() ([]SElasticcache, error) {
	ret := make([]SElasticcache, 0)
	query := url.Values{}
	for {
		resp, err := self.list(SERVICE_DCS, "instances", query)
		if err != nil {
			return nil, err
		}
		part := struct {
			Instances   []SElasticcache
			InstanceNum int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Instances...)
		if len(ret) >= part.InstanceNum || len(part.Instances) == 0 {
			break
		}
		query.Set("offset", fmt.Sprintf("%d", len(ret)))
	}
	return ret, nil
}

func (self *SRegion) GetElasticCache(instanceId string) (*SElasticcache, error) {
	resp, err := self.list(SERVICE_DCS, "instances/"+instanceId, nil)
	if err != nil {
		return nil, err
	}
	ret := &SElasticcache{region: self}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SRegion) GetIElasticcacheById(id string) (cloudprovider.ICloudElasticcache, error) {
	ec, err := self.GetElasticCache(id)
	if err != nil {
		return nil, errors.Wrap(err, "region.GetIElasticCacheById.GetElasticCache")
	}

	return ec, nil
}

func (self *SRegion) CreateIElasticcaches(opts *cloudprovider.SCloudElasticCacheInput) (cloudprovider.ICloudElasticcache, error) {
	ret, err := self.CreateElasticcache(opts)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SRegion) CreateElasticcache(opts *cloudprovider.SCloudElasticCacheInput) (cloudprovider.ICloudElasticcache, error) {
	params := map[string]interface{}{
		"name":               opts.InstanceName,
		"engine":             opts.Engine,
		"engine_version":     opts.EngineVersion,
		"capacity":           opts.CapacityGB,
		"vpc_id":             opts.VpcId,
		"subnet_id":          opts.NetworkId,
		"product_id":         opts.InstanceType,
		"zone_codes":         opts.ZoneIds,
		"no_password_access": true,
		"enable_publicip":    false,
	}
	if len(opts.SecurityGroupIds) > 0 {
		params["security_group_id"] = opts.SecurityGroupIds[0]
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

	if opts.BillingCycle != nil {
		bssParam := map[string]interface{}{
			"charging_mode": "prePaid",
			"is_auto_pay":   true,
			"is_auto_renew": opts.BillingCycle.AutoRenew,
		}
		if opts.BillingCycle.GetMonths() >= 1 && opts.BillingCycle.GetMonths() >= 9 {
			bssParam["period_type"] = "month"
			bssParam["period_num"] = opts.BillingCycle.GetMonths()
		} else if opts.BillingCycle.GetYears() >= 1 && opts.BillingCycle.GetYears() <= 3 {
			bssParam["period_type"] = "year"
			bssParam["period_num"] = opts.BillingCycle.GetYears()
		} else {
			return nil, fmt.Errorf("region.CreateIElasticcaches invalid billing cycle.reqired month (1~9) or  year(1~3)")
		}
		params["bss_param"] = bssParam
	}
	resp, err := self.post(SERVICE_DCS, "instances", params)
	if err != nil {
		return nil, err
	}
	ret := struct {
		OrderId   string
		Instances []SElasticcache
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	for i := range ret.Instances {
		ret.Instances[i].region = self
		return &ret.Instances[i], nil
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after created %s", resp.String())
}

func (self *SElasticcache) Restart() error {
	params := map[string]interface{}{
		"instances": []string{self.InstanceID},
		"action":    "restart",
	}
	_, err := self.region.put(SERVICE_DCS, "instances/status", params)
	return err
}

func (self *SElasticcache) Delete() error {
	_, err := self.region.delete(SERVICE_DCS, "instances/"+self.GetId())
	return err
}

func (self *SElasticcache) ChangeInstanceSpec(spec string) error {
	params := map[string]interface{}{
		"spec_code":    spec,
		"new_capacity": self.CapacityGB,
	}
	_, err := self.region.post(SERVICE_DCS, fmt.Sprintf("instances/%s/resize", self.InstanceID), params)
	return err
}

func (self *SElasticcache) SetMaintainTime(maintainStartTime, maintainEndTime string) error {
	params := map[string]interface{}{
		"maintain_begin": maintainStartTime,
		"maintain_end":   maintainEndTime,
	}
	_, err := self.region.put(SERVICE_DCS, "instances/"+self.InstanceID, params)
	return err
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

func (self *SElasticcache) CreateAcl(aclName, securityIps string) (cloudprovider.ICloudElasticcacheAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SElasticcache) UpdateInstanceParameters(config jsonutils.JSONObject) error {
	params := map[string]interface{}{
		"redis_config": config,
	}
	_, err := self.region.put(SERVICE_DCS, fmt.Sprintf("instances/%s/async-configs", self.InstanceID), params)
	return err
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

func (self *SElasticcache) UpdateBackupPolicy(opts cloudprovider.SCloudElasticCacheBackupPolicyUpdateInput) error {
	params := map[string]interface{}{
		"save_days":   opts.BackupReservedDays,
		"backup_type": opts.BackupType,
		"periodical_backup_plan": map[string]interface{}{
			"begin_at":    strings.ReplaceAll(opts.PreferredBackupTime, "Z", ""),
			"period_type": "weekly",
			"backup_at":   backupPeriodTrans(opts),
		},
	}
	_, err := self.region.put(SERVICE_DCS, "instances/"+self.InstanceID, map[string]interface{}{"instance_backup_policy": params})
	return err
}

func (self *SElasticcache) FlushInstance(input cloudprovider.SCloudElasticCacheFlushInstanceInput) error {
	params := map[string]interface{}{
		"instances": []string{self.InstanceID},
		"action":    "flush",
	}
	_, err := self.region.put(SERVICE_DCS, "instances/status", params)
	return err
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
