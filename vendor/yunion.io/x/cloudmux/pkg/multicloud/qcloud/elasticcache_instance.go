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

package qcloud

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/billing"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SElasticcache struct {
	multicloud.SElasticcacheBase
	QcloudTags
	region *SRegion

	ClientLimit      int
	ClientLimitMax   int
	ClientLimitMin   int
	MaintenanceTime  *MaintenanceTime `json:"maintenance_time"`
	NoAuth           *bool            `json:"no_auth"`
	NodeSet          []NodeSet        `json:"NodeSet"`
	Appid            int64            `json:"Appid"`
	AutoRenewFlag    int64            `json:"AutoRenewFlag"`
	BillingMode      int64            `json:"BillingMode"`
	CloseTime        string           `json:"CloseTime"`
	Createtime       time.Time        `json:"Createtime"`
	DeadlineTime     string           `json:"DeadlineTime"`
	Engine           string           `json:"Engine"`
	InstanceID       string           `json:"InstanceId"`
	InstanceName     string           `json:"InstanceName"`
	InstanceNode     []interface{}    `json:"InstanceNode"`
	InstanceTitle    string           `json:"InstanceTitle"`
	OfflineTime      string           `json:"OfflineTime"`
	Port             int              `json:"Port"`
	PriceID          int64            `json:"PriceId"`
	ProductType      string           `json:"ProductType"`
	ProjectID        int              `json:"ProjectId"`
	RedisReplicasNum int              `json:"RedisReplicasNum"`
	RedisShardNum    int64            `json:"RedisShardNum"`
	RedisShardSize   int64            `json:"RedisShardSize"` // MB
	DiskSize         int64            `json:"disk_size"`
	RegionID         int64            `json:"RegionId"`
	Size             int              `json:"Size"`
	SizeUsed         int64            `json:"SizeUsed"`
	SlaveReadWeight  int64            `json:"SlaveReadWeight"`
	Status           int              `json:"Status"`
	SubStatus        int64            `json:"SubStatus"`
	SubnetID         int64            `json:"SubnetId"`
	Type             int              `json:"Type"`
	UniqSubnetID     string           `json:"UniqSubnetId"`
	UniqVpcID        string           `json:"UniqVpcId"`
	VpcID            int64            `json:"VpcId"`
	WANIP            string           `json:"WanIp"`
	ZoneID           int              `json:"ZoneId"`
	WanAddress       string
}

func (self *SElasticcache) SetTags(tags map[string]string, replace bool) error {
	return self.region.SetResourceTags("redis", "instance", []string{self.InstanceID}, tags, replace)
}

type MaintenanceTime struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type NodeSet struct {
	NodeID   int64 `json:"NodeId"`
	NodeType int64 `json:"NodeType"`
	ZoneID   int64 `json:"ZoneId"`
}

var zoneMaps = map[int]string{
	100001: "ap-guangzhou-1",
	100002: "ap-guangzhou-2",
	100003: "ap-guangzhou-3",
	100004: "ap-guangzhou-4",
	100006: "ap-guangzhou-6",
	110001: "ap-shenzhen-fsi-1",
	110002: "ap-shenzhen-fsi-2",
	110003: "ap-shenzhen-fsi-3",
	200001: "ap-shanghai-1",
	200002: "ap-shanghai-2",
	200003: "ap-shanghai-3",
	200004: "ap-shanghai-4",
	200005: "ap-shanghai-5",
	200006: "ap-shanghai-6",
	200007: "ap-shanghai-7",
	700001: "ap-shanghai-fsi-1",
	700002: "ap-shanghai-fsi-2",
	700003: "ap-shanghai-fsi-3",
	330001: "ap-nanjing-1",
	330002: "ap-nanjing-2",
	330003: "ap-nanjing-3",
	800001: "ap-beijing-1",
	800002: "ap-beijing-2",
	800003: "ap-beijing-3",
	800004: "ap-beijing-4",
	800005: "ap-beijing-5",
	800006: "ap-beijing-6",
	800007: "ap-beijing-7",
	460001: "ap-beijing-fsi-1",
	360001: "ap-tianjin-1",
	360002: "ap-tianjin-2",
	160001: "ap-chengdu-1",
	160002: "ap-chengdu-2",
	190001: "ap-chongqing-1",
	300001: "ap-hongkong-1",
	300002: "ap-hongkong-2",
	300003: "ap-hongkong-3",
	390001: "ap-taipei-1",
	900001: "ap-singapore-1",
	230001: "ap-bangkok-1",
	210001: "ap-mumbai-1",
	210002: "ap-mumbai-2",
	180001: "ap-seoul-1",
	180002: "ap-seoul-2",
	250001: "ap-tokyo-1",
	150001: "na-siliconvalley-1",
	150002: "na-siliconvalley-2",
	220001: "na-ashburn-1",
	220002: "na-ashburn-2",
	400001: "na-toronto-1",
	170001: "eu-frankfurt-1",
	240001: "eu-moscow-1",
}

var zoneIdMaps = map[string]int{
	"ap-guangzhou-1":     100001,
	"ap-guangzhou-2":     100002,
	"ap-guangzhou-3":     100003,
	"ap-guangzhou-4":     100004,
	"ap-guangzhou-6":     100006,
	"ap-shenzhen-fsi-1":  110001,
	"ap-shenzhen-fsi-2":  110002,
	"ap-shenzhen-fsi-3":  110003,
	"ap-shanghai-1":      200001,
	"ap-shanghai-2":      200002,
	"ap-shanghai-3":      200003,
	"ap-shanghai-4":      200004,
	"ap-shanghai-5":      200005,
	"ap-shanghai-6":      200006,
	"ap-shanghai-7":      200007,
	"ap-shanghai-fsi-1":  700001,
	"ap-shanghai-fsi-2":  700002,
	"ap-shanghai-fsi-3":  700003,
	"ap-nanjing-1":       330001,
	"ap-nanjing-2":       330002,
	"ap-nanjing-3":       330003,
	"ap-beijing-1":       800001,
	"ap-beijing-2":       800002,
	"ap-beijing-3":       800003,
	"ap-beijing-4":       800004,
	"ap-beijing-5":       800005,
	"ap-beijing-6":       800006,
	"ap-beijing-7":       800007,
	"ap-beijing-fsi-1":   460001,
	"ap-tianjin-1":       360001,
	"ap-tianjin-2":       360002,
	"ap-chengdu-1":       160001,
	"ap-chengdu-2":       160002,
	"ap-chongqing-1":     190001,
	"ap-hongkong-1":      300001,
	"ap-hongkong-2":      300002,
	"ap-hongkong-3":      300003,
	"ap-taipei-1":        390001,
	"ap-singapore-1":     900001,
	"ap-bangkok-1":       230001,
	"ap-mumbai-1":        210001,
	"ap-mumbai-2":        210002,
	"ap-seoul-1":         180001,
	"ap-seoul-2":         180002,
	"ap-tokyo-1":         250001,
	"na-siliconvalley-1": 150001,
	"na-siliconvalley-2": 150002,
	"na-ashburn-1":       220001,
	"na-ashburn-2":       220002,
	"na-toronto-1":       400001,
	"eu-frankfurt-1":     170001,
	"eu-moscow-1":        240001,
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

// https://cloud.tencent.com/document/api/239/20022#InstanceSet
// 实例当前状态，0：待初始化；1：实例在流程中；2：实例运行中；-2：实例已隔离；-3：实例待删除
func (self *SElasticcache) GetStatus() string {
	switch self.Status {
	case 2:
		return api.ELASTIC_CACHE_STATUS_RUNNING
	case 0:
		return api.ELASTIC_CACHE_STATUS_DEPLOYING
	case -3:
		return api.ELASTIC_CACHE_STATUS_RELEASING
	case -2:
		return api.ELASTIC_CACHE_STATUS_UNAVAILABLE
	case 1:
		return api.ELASTIC_CACHE_STATUS_CHANGING
	}

	return ""
}

func (self *SElasticcache) GetConnections() int {
	return self.ClientLimit
}

func (self *SElasticcache) Refresh() error {
	cache, err := self.region.GetIElasticcacheById(self.GetId())
	if err != nil {
		return errors.Wrap(err, "Elasticcache.Refresh.GetElasticCache")
	}

	err = jsonutils.Update(self, cache)
	if err != nil {
		return errors.Wrap(err, "Elasticcache.Refresh.Update")
	}

	return nil
}

func (self *SElasticcache) IsEmulated() bool {
	return false
}

func (self *SElasticcache) GetProjectId() string {
	return strconv.Itoa(self.ProjectID)
}

func (self *SElasticcache) GetBillingType() string {
	// 计费模式：0-按量计费，1-包年包月
	if self.BillingMode == 1 {
		return billing_api.BILLING_TYPE_PREPAID
	} else {
		return billing_api.BILLING_TYPE_POSTPAID
	}
}

func (self *SElasticcache) GetCreatedAt() time.Time {
	return self.Createtime
}

func (self *SElasticcache) GetExpiredAt() time.Time {
	if self.DeadlineTime == "0000-00-00 00:00:00" || len(self.DeadlineTime) == 0 {
		return time.Time{}
	}

	t, err := time.Parse("2006-01-02 15:04:05", self.DeadlineTime)
	if err != nil {
		log.Debugf("GetExpiredAt.Parse %s", err)
		return time.Time{}
	}

	return t
}

// https://cloud.tencent.com/document/product/239/31785
func (self *SElasticcache) SetAutoRenew(bc billing.SBillingCycle) error {
	params := map[string]string{}
	params["Operation"] = "modifyAutoRenew"
	params["InstanceIds.0"] = self.GetId()
	if bc.AutoRenew {
		params["AutoRenews.0"] = "1"
	} else {
		params["AutoRenews.0"] = "0"
	}

	_, err := self.region.redisRequest("ModifyInstance", params)
	if err != nil {
		return errors.Wrap(err, "ModifyInstance")
	}

	return nil
}

func (self *SElasticcache) IsAutoRenew() bool {
	// 实例是否设置自动续费标识，1：设置自动续费；0：未设置自动续费
	return self.AutoRenewFlag == 1
}

//  redis:master:s1:r5:m1g:v4.0
// 实例类型：2 – Redis2.8内存版（标准架构），3 – CKV 3.2内存版(标准架构)，4 – CKV 3.2内存版(集群架构)
// ，6 – Redis4.0内存版（标准架构），7 – Redis4.0内存版（集群架构），
//  8 – Redis5.0内存版（标准架构），9 – Redis5.0内存版（集群架构）
func (self *SElasticcache) GetInstanceType() string {
	segs := make([]string, 6)
	segs[0] = "redis"
	switch self.Type {
	case 2, 3, 6, 8:
		segs[1] = "master"
	case 4, 7, 9, 10:
		segs[1] = "cluster"
	case 5:
		segs[1] = "single"
	default:
		segs[1] = strconv.Itoa(self.Type)
	}

	segs[2] = fmt.Sprintf("s%d", self.RedisShardNum)
	segs[3] = fmt.Sprintf("r%d", self.RedisReplicasNum)
	if self.DiskSize > 0 {
		segs[4] = fmt.Sprintf("m%dg-d%dg", self.Size/1024, self.DiskSize)
	} else {
		segs[4] = fmt.Sprintf("m%dg", self.Size/1024)
	}
	segs[5] = fmt.Sprintf("v%s", self.GetEngineVersion())

	return strings.Join(segs, ":")
}

func (self *SElasticcache) GetCapacityMB() int {
	return self.Size
}

func (self *SElasticcache) GetArchType() string {
	// 产品类型：standalone – 标准版，cluster – 集群版
	switch self.ProductType {
	case "standalone":
		return api.ELASTIC_CACHE_ARCH_TYPE_MASTER
	case "cluster":
		return api.ELASTIC_CACHE_ARCH_TYPE_CLUSTER
	}

	return ""
}

func (self *SElasticcache) GetNodeType() string {
	// single（单副本） | double（双副本)
	switch self.RedisReplicasNum {
	case 1:
		return "single"
	case 2:
		return "double"
	case 3:
		return "three"
	case 4:
		return "four"
	case 5:
		return "five"
	case 6:
		return "six"
	}

	return strconv.Itoa(self.RedisReplicasNum)
}

func (self *SElasticcache) GetEngine() string {
	return api.ELASTIC_CACHE_ENGINE_REDIS
}

func (self *SElasticcache) GetEngineVersion() string {
	switch self.Type {
	case 1, 2, 5:
		return "2.8"
	case 3, 4:
		return "3.2"
	case 6, 7, 10:
		return "4.0"
	case 8, 9:
		return "5.0"
	}

	return "unknown"
}

func (self *SElasticcache) GetVpcId() string {
	return self.UniqVpcID
}

// https://cloud.tencent.com/document/product/239/4106
func (self *SElasticcache) GetZoneId() string {
	zone := zoneMaps[self.ZoneID]
	if len(zone) == 0 {
		net, err := self.region.GetNetwork(self.UniqSubnetID)
		if err != nil {
			log.Warningf("zone %d not found. zone map needed update.", self.ZoneID)
			log.Debugf("GetNetwork %s %s", self.UniqSubnetID, err)
			return ""
		}

		zone = net.Zone
	}

	z, err := self.region.getZoneById(zone)
	if err != nil {
		log.Debugf("getZoneById %s", err)
		return ""
	}

	return z.GetGlobalId()
}

func (self *SElasticcache) GetNetworkType() string {
	if len(self.UniqVpcID) > 0 {
		return api.LB_NETWORK_TYPE_VPC
	} else {
		return api.LB_NETWORK_TYPE_CLASSIC
	}
}

func (self *SElasticcache) GetNetworkId() string {
	return self.UniqSubnetID
}

func (self *SElasticcache) GetPrivateDNS() string {
	return self.WANIP
}

func (self *SElasticcache) GetPrivateIpAddr() string {
	return self.WANIP
}

func (self *SElasticcache) GetPrivateConnectPort() int {
	return self.Port
}

func (self *SElasticcache) GetPublicDNS() string {
	if len(self.WanAddress) > 0 {
		if idx := strings.Index(self.WanAddress, ":"); idx > 0 {
			return self.WanAddress[:idx]
		}
		return self.WanAddress
	}
	return ""
}

func (self *SElasticcache) GetPublicIpAddr() string {
	return ""
}

func (self *SElasticcache) GetPublicConnectPort() int {
	if idx := strings.Index(self.WanAddress, ":"); idx > 0 {
		port, _ := strconv.Atoi(self.WanAddress[idx+1:])
		if port > 0 {
			return port
		}
	}
	return self.Port
}

// https://cloud.tencent.com/document/api/239/46336
func (self *SElasticcache) getMaintenanceTime() error {
	params := map[string]string{}
	params["InstanceId"] = self.GetId()
	params["Region"] = self.region.GetId()
	resp, err := self.region.client.redisRequest("DescribeMaintenanceWindow", params)
	if err != nil {
		return errors.Wrap(err, "DescribeMaintenanceWindow")
	}

	self.MaintenanceTime = &MaintenanceTime{}
	err = resp.Unmarshal(self.MaintenanceTime)
	if err != nil {
		return errors.Wrap(err, "err.MaintenanceTime")
	}

	return nil
}

func (self *SElasticcache) GetMaintainStartTime() string {
	if self.MaintenanceTime == nil {
		if err := self.getMaintenanceTime(); err != nil {
			log.Debugf("getMaintenanceTime %s", err)
			return ""
		}
	}

	return self.MaintenanceTime.StartTime
}

func (self *SElasticcache) GetMaintainEndTime() string {
	if self.MaintenanceTime == nil {
		if err := self.getMaintenanceTime(); err != nil {
			log.Debugf("getMaintenanceTime %s", err)
			return ""
		}
	}

	return self.MaintenanceTime.EndTime
}

func (self *SElasticcache) GetAuthMode() string {
	if self.NoAuth != nil && *self.NoAuth == false {
		return "on"
	}

	return "off"
}

func (self *SElasticcache) GetSecurityGroupIds() ([]string, error) {
	ss, err := self.region.GetCloudElasticcacheSecurityGroups(self.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudElasticcacheSecurityGroups")
	}

	ret := make([]string, len(ss))
	for i := range ss {
		ret[i] = ss[i].SecurityGroupID
	}

	return ret, nil
}

func (self *SElasticcache) GetICloudElasticcacheAccounts() ([]cloudprovider.ICloudElasticcacheAccount, error) {
	accounts, err := self.getCloudElasticcacheAccounts()
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudElasticcacheAccounts")
	}

	iaccounts := make([]cloudprovider.ICloudElasticcacheAccount, len(accounts))
	for i := range accounts {
		account := accounts[i]
		account.cacheDB = self
		iaccounts[i] = &account
	}

	return iaccounts, nil
}

func (self *SElasticcache) GetICloudElasticcacheAcls() ([]cloudprovider.ICloudElasticcacheAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SElasticcache) GetICloudElasticcacheBackups() ([]cloudprovider.ICloudElasticcacheBackup, error) {
	backups, err := self.region.GetCloudElasticcacheBackups(self.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudElasticcacheBackups")
	}

	ibackups := make([]cloudprovider.ICloudElasticcacheBackup, len(backups))
	for i := range backups {
		backup := backups[i]
		backup.cacheDB = self
		ibackups[i] = &backup
	}

	return ibackups, nil
}

func (self *SElasticcache) GetICloudElasticcacheParameters() ([]cloudprovider.ICloudElasticcacheParameter, error) {
	parameters, err := self.region.GetCloudElasticcacheParameters(self.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudElasticcacheParameters")
	}

	ret := []cloudprovider.ICloudElasticcacheParameter{}
	for i := range parameters {
		parameter := parameters[i]
		parameter.cacheDB = self
		ret = append(ret, &parameter)
	}

	return ret, nil
}

func (self *SElasticcache) GetICloudElasticcacheAccount(accountId string) (cloudprovider.ICloudElasticcacheAccount, error) {
	accounts, err := self.getCloudElasticcacheAccounts()
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudElasticcacheAccounts")
	}

	for i := range accounts {
		account := accounts[i]
		if account.GetGlobalId() == accountId {
			account.cacheDB = self
			return &account, nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func (self *SElasticcache) GetICloudElasticcacheAcl(aclId string) (cloudprovider.ICloudElasticcacheAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SElasticcache) GetICloudElasticcacheBackup(backupId string) (cloudprovider.ICloudElasticcacheBackup, error) {
	backups, err := self.region.GetCloudElasticcacheBackups(self.InstanceID)
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudElasticcacheBackups")
	}

	for i := range backups {
		backup := backups[i]
		if backup.GetGlobalId() == backupId {
			backup.cacheDB = self
			return &backup, nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func (self *SElasticcache) GetICloudElasticcacheBackupByReadmark(readmark string) (cloudprovider.ICloudElasticcacheBackup, error) {
	backups, err := self.region.GetCloudElasticcacheBackups(self.InstanceID)
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudElasticcacheBackups")
	}

	for i := range backups {
		backup := backups[i]
		if backup.Remark == readmark {
			backup.cacheDB = self
			return &backup, nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

func (self *SElasticcache) Restart() error {
	return errors.Wrap(cloudprovider.ErrNotSupported, "Restart")
}

// https://cloud.tencent.com/document/product/239/34440
func (self *SElasticcache) Delete() error {
	requiredStatus := ""
	if self.GetBillingType() == billing_api.BILLING_TYPE_POSTPAID {
		if err := self.DestroyPostpaidInstance(); err != nil {
			return errors.Wrap(err, "DestroyPostpaidInstance")
		}

		requiredStatus = api.ELASTIC_CACHE_STATUS_RELEASING
	} else {
		if err := self.DestroyPrepaidInstance(); err != nil {
			return errors.Wrap(err, "DestroyPrepaidInstance")
		}

		requiredStatus = api.ELASTIC_CACHE_STATUS_UNAVAILABLE
	}

	err := cloudprovider.WaitStatus(self, requiredStatus, 5*time.Second, 300*time.Second)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("WaitStatus.%s", requiredStatus))
	}

	err = self.CleanupInstance()
	if err != nil {
		return errors.Wrap(err, "CleanupInstance")
	}

	return nil
}

// https://cloud.tencent.com/document/product/239/34439
func (self *SElasticcache) DestroyPrepaidInstance() error {
	if self.GetStatus() == api.ELASTIC_CACHE_STATUS_UNAVAILABLE {
		return nil
	}

	params := map[string]string{}
	params["InstanceId"] = self.GetId()
	_, err := self.region.redisRequest("DestroyPrepaidInstance", params)
	if err != nil {
		return errors.Wrap(err, "DestroyPrepaidInstance")
	}

	return nil
}

// https://cloud.tencent.com/document/product/239/34439
func (self *SElasticcache) DestroyPostpaidInstance() error {
	if self.GetStatus() == api.ELASTIC_CACHE_STATUS_RELEASING {
		return nil
	}

	params := map[string]string{}
	params["InstanceId"] = self.GetId()
	_, err := self.region.redisRequest("DestroyPostpaidInstance", params)
	if err != nil {
		return errors.Wrap(err, "DestroyPostpaidInstance")
	}

	return nil
}

// https://cloud.tencent.com/document/product/239/34442
func (self *SElasticcache) CleanupInstance() error {
	params := map[string]string{}
	params["InstanceId"] = self.GetId()
	_, err := self.region.redisRequest("CleanUpInstance", params)
	if err != nil {
		return errors.Wrap(err, "CleanUpInstance")
	}

	return nil
}

// https://cloud.tencent.com/document/product/239/20013
func (self *SElasticcache) ChangeInstanceSpec(spec string) error {
	s, err := parseLocalInstanceSpec(spec)
	if err != nil {
		return errors.Wrap(err, "parseLocalInstanceSpec")
	}

	params := map[string]string{}
	params["InstanceId"] = self.GetId()
	params["MemSize"] = fmt.Sprintf("%d", s.MemSizeMB)
	params["RedisShardNum"] = s.RedisShardNum
	params["RedisReplicasNum"] = s.RedisReplicasNum
	_, err = self.region.redisRequest("UpgradeInstance", params)
	if err != nil {
		return errors.Wrap(err, "UpgradeInstance")
	}

	return nil
}

// https://cloud.tencent.com/document/product/239/46335
func (self *SElasticcache) SetMaintainTime(maintainStartTime, maintainEndTime string) error {
	params := map[string]string{}
	params["InstanceId"] = self.GetId()
	params["StartTime"] = maintainStartTime
	params["EndTime"] = maintainEndTime
	_, err := self.region.redisRequest("ModifyMaintenanceWindow", params)
	if err != nil {
		return errors.Wrap(err, "ModifyMaintenanceWindow")
	}

	return nil
}

func (self *SElasticcache) AllocatePublicConnection(port int) (string, error) {
	return "", errors.Wrap(cloudprovider.ErrNotSupported, "AllocatePublicConnection")
}

func (self *SElasticcache) ReleasePublicConnection() error {
	return errors.Wrap(cloudprovider.ErrNotSupported, "ReleasePublicConnection")
}

// https://cloud.tencent.com/document/product/239/38926
func (self *SElasticcache) CreateAccount(account cloudprovider.SCloudElasticCacheAccountInput) (cloudprovider.ICloudElasticcacheAccount, error) {
	params := map[string]string{}
	params["InstanceId"] = self.GetId()
	params["AccountName"] = account.AccountName
	params["AccountPassword"] = account.AccountPassword
	params["Privilege"] = account.AccountPrivilege
	params["ReadonlyPolicy.0"] = "master"
	if len(account.Description) > 0 {
		params["Remark"] = account.Description
	}

	_, err := self.region.redisRequest("CreateInstanceAccount", params)
	if err != nil {
		return nil, errors.Wrap(err, "CreateInstanceAccount")
	}

	accountId := ""
	cloudprovider.Wait(5*time.Second, 60*time.Second, func() (bool, error) {
		accounts, err := self.GetICloudElasticcacheAccounts()
		if err != nil {
			log.Debugf("GetICloudElasticcacheAccounts %s", err)
			return false, nil
		}

		for i := range accounts {
			if accounts[i].GetName() == account.AccountName {
				accountId = accounts[i].GetGlobalId()
				return true, nil
			}
		}

		return false, nil
	})

	return self.GetICloudElasticcacheAccount(accountId)
}

func (self *SElasticcache) CreateAcl(aclName, securityIps string) (cloudprovider.ICloudElasticcacheAcl, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotSupported, "CreateAcl")
}

// https://cloud.tencent.com/document/product/239/20010
func (self *SElasticcache) CreateBackup(desc string) (cloudprovider.ICloudElasticcacheBackup, error) {
	readmark := strings.Join([]string{desc, fmt.Sprintf("%d", time.Now().Unix())}, "@")
	params := map[string]string{}
	params["InstanceId"] = self.GetId()
	params["Remark"] = readmark
	resp, err := self.region.redisRequest("ManualBackupInstance", params)
	if err != nil {
		return nil, errors.Wrap(err, "ManualBackupInstance")
	}

	taskId := 0
	err = resp.Unmarshal(&taskId, "TaskId")
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}

	var task *SElasticcacheTask
	err = cloudprovider.Wait(5*time.Second, 900*time.Second, func() (bool, error) {
		task, err = self.region.DescribeTaskInfo(fmt.Sprintf("%d", taskId))
		if err != nil {
			return false, nil
		}

		if task != nil {
			if task.Status == "failed" || task.Status == "error" {
				return false, fmt.Errorf("CreateBackup failed %#v", task)
			}

			if task.Status == "succeed" {
				return true, nil
			}
		}

		return false, nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "Wait.DescribeTaskInfo")
	}

	if task == nil {
		return nil, fmt.Errorf("CreateBackup failed task is empty")
	}

	return self.GetICloudElasticcacheBackupByReadmark(readmark)
}

// https://cloud.tencent.com/document/product/239/20021
func (self *SElasticcache) FlushInstance(input cloudprovider.SCloudElasticCacheFlushInstanceInput) error {
	params := map[string]string{}
	params["InstanceId"] = self.GetId()
	if self.NoAuth != nil && *self.NoAuth == false {
		if len(input.Password) > 0 {
			params["Password"] = input.Password
		} else {
			return fmt.Errorf("password required on auth mode while flush elastich cache instance")
		}
	}
	_, err := self.region.redisRequest("ClearInstance", params)
	if err != nil {
		return errors.Wrap(err, "ClearInstance")
	}

	return nil
}

// https://cloud.tencent.com/document/product/239/38923
// https://cloud.tencent.com/document/product/239/20014
// true表示将主账号切换为免密账号，这里只适用于主账号，子账号不可免密
func (self *SElasticcache) UpdateAuthMode(noPasswordAccess bool, password string) error {
	params := map[string]string{}
	params["InstanceId"] = self.GetId()
	if noPasswordAccess {
		params["NoAuth"] = "true"
	} else {
		params["NoAuth"] = "false"
		params["Password"] = password
	}

	_, err := self.region.redisRequest("ResetPassword", params)
	if err != nil {
		return errors.Wrap(err, "ResetPassword")
	}

	return nil
}

// https://cloud.tencent.com/document/product/239/34445
// todo: finish me
func (self *SElasticcache) UpdateInstanceParameters(config jsonutils.JSONObject) error {
	params := map[string]string{}
	params["InstanceId"] = self.GetId()
	params["InstanceParams.0"] = config.String()
	_, err := self.region.redisRequest("ModifyInstanceParams", params)
	if err != nil {
		return errors.Wrap(err, "ModifyInstanceParams")
	}

	return nil
}

func (self *SElasticcache) UpdateBackupPolicy(config cloudprovider.SCloudElasticCacheBackupPolicyUpdateInput) error {
	panic("implement me")
}

func (self *SElasticcache) getCloudElasticcacheAccounts() ([]SElasticcacheAccount, error) {
	if self.GetEngineVersion() == "2.8" || (self.NodeSet != nil && len(self.NodeSet) > 0) {
		account := SElasticcacheAccount{}
		account.cacheDB = self
		account.AccountName = "root"
		account.InstanceID = self.GetId()
		account.Privilege = "rw"
		account.Status = 2
		account.IsEmulate = true

		return []SElasticcacheAccount{account}, nil
	}

	accounts, err := self.region.GetCloudElasticcacheAccounts(self.GetId())
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudElasticcacheAccounts")
	}

	for i := range accounts {
		accounts[i].cacheDB = self
	}

	return accounts, nil
}

// https://cloud.tencent.com/document/api/239/38924
func (self *SElasticcache) getCloudElasticcacheAccount(accountName string) (*SElasticcacheAccount, error) {
	accounts, err := self.getCloudElasticcacheAccounts()
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudElasticcacheAccounts")
	}

	for i := range accounts {
		if accounts[i].AccountName == accountName {
			accounts[i].cacheDB = self
			return &accounts[i], nil
		}
	}

	return nil, cloudprovider.ErrNotFound
}

// https://cloud.tencent.com/document/api/239/41256
func (self *SElasticcache) UpdateSecurityGroups(secgroupIds []string) error {
	params := map[string]string{}
	params["InstanceId"] = self.GetId()
	params["Product"] = "redis"
	for i := range secgroupIds {
		params[fmt.Sprintf("SecurityGroupIds.%d", i)] = secgroupIds[i]
	}

	_, err := self.region.redisRequest("ModifyDBInstanceSecurityGroups", params)
	if err != nil {
		return errors.Wrap(err, "ModifyDBInstanceSecurityGroups")
	}

	return nil
}

func (self *SElasticcache) Renew(bc billing.SBillingCycle) error {
	month := bc.GetMonths()
	if month <= 0 {
		return errors.Wrap(fmt.Errorf("month should great than 0"), "GetMonths")
	}

	params := map[string]string{}
	params["InstanceId"] = self.GetId()
	params["Period"] = fmt.Sprintf("%d", month)
	_, err := self.region.redisRequest("RenewInstance", params)
	if err != nil {
		return errors.Wrap(err, "RenewInstance")
	}

	return nil
}

// https://cloud.tencent.com/document/api/239/38924
func (self *SRegion) GetCloudElasticcacheAccounts(instanceId string) ([]SElasticcacheAccount, error) {
	params := map[string]string{}
	params["Region"] = self.GetId()
	params["InstanceId"] = instanceId
	params["Limit"] = "20"
	params["Offset"] = "0"

	ret := []SElasticcacheAccount{}
	offset := 0
	for {
		resp, err := self.client.redisRequest("DescribeInstanceAccount", params)
		if err != nil {
			return nil, errors.Wrap(err, "DescribeInstanceAccount")
		}

		_ret := []SElasticcacheAccount{}
		err = resp.Unmarshal(&_ret, "Accounts")
		if err != nil {
			return nil, errors.Wrap(err, "Unmarshal")
		} else {
			ret = append(ret, _ret...)
		}

		if len(_ret) < 20 {
			break
		} else {
			offset += 20
			params["Offset"] = strconv.Itoa(offset)
		}
	}

	return ret, nil
}

// https://cloud.tencent.com/document/api/239/20011
func (self *SRegion) GetCloudElasticcacheBackups(instanceId string) ([]SElasticcacheBackup, error) {
	params := map[string]string{}
	params["Region"] = self.GetId()
	params["InstanceId"] = instanceId
	params["Limit"] = "20"
	params["Offset"] = "0"

	ret := []SElasticcacheBackup{}
	offset := 0
	for {
		resp, err := self.client.redisRequest("DescribeInstanceBackups", params)
		if err != nil {
			return nil, errors.Wrap(err, "DescribeInstanceBackups")
		}

		_ret := []SElasticcacheBackup{}
		err = resp.Unmarshal(&_ret, "BackupSet")
		if err != nil {
			return nil, errors.Wrap(err, "Unmarshal")
		} else {
			ret = append(ret, _ret...)
		}

		if len(_ret) < 20 {
			break
		} else {
			offset += 20
			params["Offset"] = strconv.Itoa(offset)
		}
	}

	return ret, nil
}

// https://cloud.tencent.com/document/api/239/41259
func (self *SRegion) GetCloudElasticcacheSecurityGroups(instanceId string) ([]SElasticcacheSecgroup, error) {
	params := map[string]string{}
	params["Region"] = self.GetId()
	params["InstanceId"] = instanceId
	params["Product"] = "redis"
	resp, err := self.client.redisRequest("DescribeDBSecurityGroups", params)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeDBSecurityGroups")
	}

	ret := []SElasticcacheSecgroup{}
	err = resp.Unmarshal(&ret, "Groups")
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}

	return ret, nil
}

// https://cloud.tencent.com/document/api/239/34448
func (self *SRegion) GetCloudElasticcacheParameters(instanceId string) ([]SElasticcacheParameter, error) {
	params := map[string]string{}
	params["Region"] = self.GetId()
	params["InstanceId"] = instanceId
	resp, err := self.client.redisRequest("DescribeInstanceParams", params)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeInstanceParams")
	}

	ret1 := []SElasticcacheParameter{}
	err = resp.Unmarshal(&ret1, "InstanceEnumParam")
	if err != nil {
		return nil, errors.Wrap(err, "InstanceEnumParam")
	}

	ret2 := []SElasticcacheParameter{}
	err = resp.Unmarshal(&ret2, "InstanceIntegerParam")
	if err != nil {
		return nil, errors.Wrap(err, "InstanceIntegerParam")
	}

	ret3 := []SElasticcacheParameter{}
	err = resp.Unmarshal(&ret3, "InstanceMultiParam")
	if err != nil {
		return nil, errors.Wrap(err, "InstanceMultiParam")
	}

	ret4 := []SElasticcacheParameter{}
	err = resp.Unmarshal(&ret4, "InstanceTextParam")
	if err != nil {
		return nil, errors.Wrap(err, "InstanceTextParam")
	}

	ret := []SElasticcacheParameter{}
	ret = append(ret, ret1...)
	ret = append(ret, ret2...)
	ret = append(ret, ret3...)
	ret = append(ret, ret4...)
	return ret, nil
}

// https://cloud.tencent.com/document/api/239/20018
func (self *SRegion) GetCloudElasticcaches(instanceId string) ([]SElasticcache, error) {
	params := map[string]string{}
	params["Region"] = self.GetId()
	params["Limit"] = "20"
	params["Offset"] = "0"

	if len(instanceId) > 0 {
		params["InstanceId"] = instanceId
	}

	ret := []SElasticcache{}
	offset := 0
	for {
		resp, err := self.client.redisRequest("DescribeInstances", params)
		if err != nil {
			return nil, errors.Wrap(err, "DescribeInstances")
		}

		_ret := []SElasticcache{}
		err = resp.Unmarshal(&_ret, "InstanceSet")
		if err != nil {
			return nil, errors.Wrap(err, "Unmarshal")
		} else {
			ret = append(ret, _ret...)
		}

		if len(_ret) < 20 {
			break
		} else {
			offset += 20
			params["Offset"] = strconv.Itoa(offset)
		}
	}

	return ret, nil
}

type elasticcachInstanceSpec struct {
	TypeId           string `json:"type_id"`
	MemSizeMB        int    `json:"mem_size_mb"`
	DiskSizeGB       int    `json:"disk_size_gb"`
	RedisShardNum    string `json:"redis_shard_num"`
	RedisReplicasNum string `json:"redis_replicas_num"`
}

//  redis:master:s1:r5:m1g:v4.0
func parseLocalInstanceSpec(s string) (elasticcachInstanceSpec, error) {
	ret := elasticcachInstanceSpec{}
	segs := strings.Split(s, ":")
	if len(segs) != 6 {
		return ret, fmt.Errorf("invalid instance spec %s", s)
	}
	if segs[1] == "master" {
		switch segs[5] {
		case "v2.8":
			ret.TypeId = "2"
		case "v3.2":
			ret.TypeId = "3"
		case "v4.0":
			ret.TypeId = "6"
		case "v5.0":
			ret.TypeId = "8"
		default:
			return ret, fmt.Errorf("unknown master elastic cache version %s", segs[1])
		}
	} else if segs[1] == "cluster" {
		switch segs[5] {
		case "v3.0":
			ret.TypeId = "4"
		case "v4.0":
			ret.TypeId = "7"
		case "v5.0":
			if strings.Contains(segs[4], "-d") {
				ret.TypeId = "10"
			} else {
				ret.TypeId = "9"
			}

		default:
			return ret, fmt.Errorf("unknown cluster elastic cache version %s", segs[1])
		}
	} else if segs[1] == "single" {
		switch segs[5] {
		case "v2.8":
			ret.TypeId = "5"
		default:
			return ret, fmt.Errorf("unknown single elastic cache version %s", segs[1])
		}
	} else {
		return ret, fmt.Errorf("unknown elastic cache type %s", segs[1])
	}

	//
	sizes := strings.Split(segs[4], "-")
	if len(sizes) == 2 {
		ms, err := strconv.Atoi(strings.Trim(sizes[1], "dg"))
		if err != nil {
			return ret, errors.Wrap(err, "Atoi")
		}

		ret.DiskSizeGB = ms
	}

	sn, err := strconv.Atoi(strings.Trim(segs[2], "s"))
	if err != nil {
		return ret, errors.Wrap(err, "RedisShardNum")
	}

	ms, err := strconv.Atoi(strings.Trim(sizes[0], "mdg"))
	if err != nil {
		return ret, errors.Wrap(err, "Atoi")
	}
	if strings.HasSuffix(sizes[0], "g") {
		ret.MemSizeMB = ms * 1024 / sn
	} else if strings.HasSuffix(sizes[0], "m") {
		ret.MemSizeMB = ms / sn
	}

	ret.RedisShardNum = strings.Trim(segs[2], "s")
	ret.RedisReplicasNum = strings.Trim(segs[3], "r")
	return ret, nil
}
