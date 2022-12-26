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
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"

	billingapi "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SlaveInstanceInfo struct {
	Region string
	Vip    string
	VpcId  int
	Vport  int
	Zone   string
}

type SlaveInfo struct {
	First  SlaveInstanceInfo
	Second SlaveInstanceInfo
}

type SDrInfo struct {
	Status       int
	Zone         string
	InstanceId   string
	Region       string
	SyncStatus   string
	InstanceName string
	InstanceType string
}

type SMasterInfo struct {
	Region        string
	RegionId      int
	ZoneId        int
	Zone          string
	InstanceId    string
	ResourceId    string
	Status        int
	InstanceName  string
	InstanceType  int
	TaskStatus    int
	Memory        int
	Volume        int
	DeviceType    string
	Qps           int
	VpcId         int
	SubnetId      int
	ExClusterId   string
	ExClusterName string
}

type SRoGroup struct {
	RoGroupMode    string
	RoGroupId      string
	RoGroupName    string
	RoOfflineDelay int
	RoMaxDelayTime int
	MinRoInGroup   int
	WeightMode     string
	Weight         int
	// RoInstances
	Vip           string
	Vport         int
	UniqVpcId     string
	UniqSubnetId  string
	RoGroupRegion string
	RoGroupZone   string
}

type SRoVipInfo struct {
	RoVipStatus int
	RoSubnetId  int
	RoVpcId     int
	RoVport     int
	RoVip       string
}

type SMySQLInstance struct {
	region *SRegion
	multicloud.SDBInstanceBase
	QcloudTags

	AutoRenew        int
	CdbError         int
	Cpu              int
	CreateTime       time.Time
	DeadlineTime     string
	DeployGroupId    string
	DeployMode       int
	DeviceClass      string
	DeviceType       string
	DrInfo           []SDrInfo
	EngineVersion    string
	ExClusterId      string
	HourFeeStatus    int
	InitFlag         int
	InstanceId       string
	InstanceName     string
	InstanceType     int
	IsolateTime      string
	MasterInfo       SMasterInfo
	Memory           int
	OfflineTime      string
	PayType          int
	PhysicalId       string
	ProjectId        int
	ProtectMode      string
	Qps              int
	Region           string
	RegionId         string
	ResourceId       string
	RoGroups         []SRoGroup
	RoVipInfo        SRoVipInfo
	SecurityGroupIds []string
	SlaveInfo        SlaveInfo
	Status           int
	SubnetId         int
	//TagList": null,
	TaskStatus   int
	UniqSubnetId string
	UniqVpcId    string
	Vip          string
	Volume       int
	VpcId        int
	Vport        int
	WanDomain    string
	WanPort      int
	WanStatus    int
	Zone         string
	ZoneId       int
	ZoneName     string
}

func (self *SMySQLInstance) GetId() string {
	return self.InstanceId
}

func (self *SMySQLInstance) GetGlobalId() string {
	return self.InstanceId
}

func (self *SMySQLInstance) GetName() string {
	if len(self.InstanceName) > 0 {
		return self.InstanceName
	}
	return self.InstanceId
}

func (self *SMySQLInstance) GetDiskSizeGB() int {
	return self.Volume
}

func (self *SMySQLInstance) GetEngine() string {
	return api.DBINSTANCE_TYPE_MYSQL
}

func (self *SMySQLInstance) GetEngineVersion() string {
	return self.EngineVersion
}

func (self *SMySQLInstance) GetIVpcId() string {
	return self.UniqVpcId
}

func (self *SMySQLInstance) Refresh() error {
	rds, err := self.region.GetMySQLInstanceById(self.InstanceId)
	if err != nil {
		return errors.Wrapf(err, "GetMySQLInstanceById(%s)", self.InstanceId)
	}
	return jsonutils.Update(self, rds)
}

func (self *SMySQLInstance) GetInstanceType() string {
	return fmt.Sprintf("%d核%dMB", self.Cpu, self.Memory)
}

func (self *SMySQLInstance) GetMaintainTime() string {
	timeWindow, err := self.region.DescribeMySQLTimeWindow(self.InstanceId)
	if err != nil {
		log.Errorf("DescribeMySQLTimeWindow %s error: %v", self.InstanceId, err)
		return ""
	}
	return timeWindow.String()
}

func (self *SMySQLInstance) GetDBNetworks() ([]cloudprovider.SDBInstanceNetwork, error) {
	return []cloudprovider.SDBInstanceNetwork{
		cloudprovider.SDBInstanceNetwork{NetworkId: self.UniqSubnetId, IP: self.Vip},
	}, nil
}

func (self *SMySQLInstance) GetConnectionStr() string {
	if self.WanStatus == 1 {
		return fmt.Sprintf("%s:%d", self.WanDomain, self.WanPort)
	}
	return ""
}

func (self *SMySQLInstance) GetInternalConnectionStr() string {
	return fmt.Sprintf("%s:%d", self.Vip, self.Vport)
}

func (self *SMySQLInstance) Reboot() error {
	return self.region.RebootMySQLInstance(self.InstanceId)
}

func (self *SMySQLInstance) ChangeConfig(ctx context.Context, opts *cloudprovider.SManagedDBInstanceChangeConfig) error {
	mb := self.GetVmemSizeMB()
	if len(opts.InstanceType) > 0 {
		re := regexp.MustCompile(`(\d{1,4})核(\d{1,20})MB$`)
		params := re.FindStringSubmatch(opts.InstanceType)
		if len(params) != 3 {
			return fmt.Errorf("invalid rds instance type %s", opts.InstanceType)
		}
		_mb, _ := strconv.Atoi(params[2])
		mb = int(_mb)
	}
	if opts.DiskSizeGB == 0 {
		opts.DiskSizeGB = self.GetDiskSizeGB()
	}
	return self.region.UpgradeMySQLDBInstance(self.InstanceId, mb, opts.DiskSizeGB)
}

func (self *SMySQLInstance) GetMasterInstanceId() string {
	return self.MasterInfo.InstanceId
}

func (self *SMySQLInstance) GetSecurityGroupIds() ([]string, error) {
	if len(self.SecurityGroupIds) > 0 {
		return self.SecurityGroupIds, nil
	}
	if self.DeviceType == "BASIC" {
		return []string{}, nil
	}
	secgroups, err := self.region.DescribeMySQLDBSecurityGroups(self.InstanceId)
	if err != nil {
		return []string{}, errors.Wrapf(err, "DescribeMySQLDBSecurityGroups")
	}
	ids := []string{}
	for i := range secgroups {
		ids = append(ids, secgroups[i].SecurityGroupId)
	}
	return ids, nil
}

func (self *SMySQLInstance) SetSecurityGroups(ids []string) error {
	return self.region.ModifyMySQLInstanceSecurityGroups(self.InstanceId, ids)
}

func (self *SRegion) ModifyMySQLInstanceSecurityGroups(rdsId string, secIds []string) error {
	params := map[string]string{
		"InstanceId": rdsId,
	}
	for idx, id := range secIds {
		params[fmt.Sprintf("SecurityGroupIds.%d", idx)] = id
	}
	_, err := self.cdbRequest("ModifyDBInstanceSecurityGroups", params)
	return err
}

func (self *SMySQLInstance) Renew(bc billing.SBillingCycle) error {
	month := bc.GetMonths()
	return self.region.RenewMySQLDBInstance(self.InstanceId, month)
}

func (self *SMySQLInstance) OpenPublicConnection() error {
	if self.WanStatus == 0 {
		return self.region.OpenMySQLWanService(self.InstanceId)
	}
	return nil
}

func (self *SMySQLInstance) ClosePublicConnection() error {
	if self.WanStatus == 1 {
		return self.region.CloseMySQLWanService(self.InstanceId)
	}
	return nil
}

func (self *SMySQLInstance) GetPort() int {
	return self.Vport
}

func (self *SMySQLInstance) GetStatus() string {
	if self.InitFlag == 0 {
		return api.DBINSTANCE_INIT
	}
	switch self.Status {
	case 4:
		return api.DBINSTANCE_ISOLATING
	case 5:
		return api.DBINSTANCE_ISOLATE
	}
	switch self.TaskStatus {
	case 0:
		switch self.Status {
		case 0:
			return api.DBINSTANCE_DEPLOYING
		case 1:
			return api.DBINSTANCE_RUNNING
		}
	case 1:
		return api.DBINSTANCE_UPGRADING
	case 2: //数据导入中
		return api.DBINSTANCE_IMPORTING
	case 3, 4: //开放关闭外网地址
		return api.DBINSTANCE_DEPLOYING
	case 10:
		return api.DBINSTANCE_REBOOTING
	case 12:
		return api.DBINSTANCE_MIGRATING
	default:
		return api.DBINSTANCE_DEPLOYING
	}
	return api.DBINSTANCE_UNKNOWN
}

func (self *SMySQLInstance) GetCategory() string {
	category := strings.ToLower(self.DeviceType)
	if category == "universal" {
		category = "ha"
	}
	return category
}

func (self *SMySQLInstance) GetStorageType() string {
	switch self.DeviceType {
	case "BASIC":
		return api.QCLOUD_DBINSTANCE_STORAGE_TYPE_CLOUD_SSD
	default:
		return api.QCLOUD_DBINSTANCE_STORAGE_TYPE_LOCAL_SSD
	}
}

func (self *SMySQLInstance) GetCreatedAt() time.Time {
	// 2019-12-25 09:00:43  #非UTC时间
	return self.CreateTime.Add(time.Hour * -8)
}

func (self *SMySQLInstance) GetBillingType() string {
	if self.PayType == 0 {
		return billingapi.BILLING_TYPE_PREPAID
	}
	return billingapi.BILLING_TYPE_POSTPAID
}

func (self *SMySQLInstance) SetAutoRenew(bc billing.SBillingCycle) error {
	return self.region.ModifyMySQLAutoRenewFlag([]string{self.InstanceId}, bc.AutoRenew)
}

func (self *SMySQLInstance) IsAutoRenew() bool {
	return self.AutoRenew == 1
}

func (self *SMySQLInstance) GetExpiredAt() time.Time {
	offline, _ := timeutils.ParseTimeStr(self.OfflineTime)
	if !offline.IsZero() {
		return offline.Add(time.Hour * -8)
	}
	deadline, _ := timeutils.ParseTimeStr(self.DeadlineTime)
	if !deadline.IsZero() {
		return deadline.Add(time.Hour * -8)
	}
	return time.Time{}
}

func (self *SMySQLInstance) GetVcpuCount() int {
	return self.Cpu
}

func (self *SMySQLInstance) GetVmemSizeMB() int {
	return self.Memory
}

func (self *SMySQLInstance) GetZone1Id() string {
	return self.Zone
}

func (self *SMySQLInstance) GetZone2Id() string {
	return self.SlaveInfo.First.Zone
}

func (self *SMySQLInstance) GetZone3Id() string {
	return self.SlaveInfo.Second.Zone
}

func (self *SMySQLInstance) GetProjectId() string {
	return fmt.Sprintf("%d", self.ProjectId)
}

func (self *SMySQLInstance) Delete() error {
	err := self.region.IsolateMySQLDBInstance(self.InstanceId)
	if err != nil {
		return errors.Wrapf(err, "IsolateMySQLDBInstance")
	}
	return self.region.OfflineIsolatedMySQLInstances([]string{self.InstanceId})
}

func (self *SRegion) ListMySQLInstances(ids []string, offset, limit int) ([]SMySQLInstance, int, error) {
	if limit < 1 || limit > 50 {
		limit = 50
	}
	params := map[string]string{
		"Offset": fmt.Sprintf("%d", offset),
		"Limit":  fmt.Sprintf("%d", limit),
	}
	for idx, id := range ids {
		params[fmt.Sprintf("InstanceIds.%d", idx)] = id
	}
	resp, err := self.cdbRequest("DescribeDBInstances", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeDBInstances")
	}
	items := []SMySQLInstance{}
	err = resp.Unmarshal(&items, "Items")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	total, _ := resp.Float("TotalCount")
	return items, int(total), nil
}

type SAsyncRequestResult struct {
	Info   string
	Status string
}

func (self *SRegion) DescribeMySQLAsyncRequestInfo(id string) (*SAsyncRequestResult, error) {
	resp, err := self.cdbRequest("DescribeAsyncRequestInfo", map[string]string{"AsyncRequestId": id})
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeAsyncRequestInfo")
	}
	result := SAsyncRequestResult{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return &result, nil
}

func (self *SRegion) waitAsyncAction(action string, resId, asyncRequestId string) error {
	if len(asyncRequestId) == 0 {
		return errors.Error("Missing AsyncRequestId")
	}
	return cloudprovider.Wait(time.Second*10, time.Minute*20, func() (bool, error) {
		result, err := self.DescribeMySQLAsyncRequestInfo(asyncRequestId)
		if err != nil {
			return false, errors.Wrapf(err, action)
		}
		log.Debugf("task %s(%s) for mysql instance %s status: %s", action, asyncRequestId, resId, result.Status)
		switch result.Status {
		case "FAILED", "KILLED", "REMOVED", "PAUSED":
			return true, errors.Errorf(result.Info)
		case "SUCCESS":
			return true, nil
		default:
			return false, nil
		}
	})
}

func (self *SRegion) RebootMySQLInstance(id string) error {
	resp, err := self.cdbRequest("RestartDBInstances", map[string]string{"InstanceIds.0": id})
	if err != nil {
		return errors.Wrapf(err, "RestartDBInstances")
	}
	asyncRequestId, _ := resp.GetString("AsyncRequestId")
	return self.waitAsyncAction("RestartDBInstances", id, asyncRequestId)
}

func (self *SRegion) DescribeMySQLDBInstanceInfo(id string) (*SMySQLInstance, error) {
	resp, err := self.cdbRequest("DescribeDBInstanceInfo", map[string]string{"InstanceId": id})
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeDBInstanceInfo")
	}
	result := SMySQLInstance{region: self}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return &result, nil
}

func (self *SRegion) RenewMySQLDBInstance(id string, month int) error {
	params := map[string]string{
		"InstanceId": id,
		"TimeSpan":   fmt.Sprintf("%d", month),
	}
	_, err := self.cdbRequest("RenewDBInstance", params)
	if err != nil {
		return errors.Wrapf(err, "RenewDBInstance")
	}
	return nil
}

func (self *SRegion) OfflineIsolatedMySQLInstances(ids []string) error {
	params := map[string]string{}
	for idx, id := range ids {
		params[fmt.Sprintf("InstanceIds.%d", idx)] = id
	}
	_, err := self.cdbRequest("OfflineIsolatedInstances", params)
	if err != nil {
		return errors.Wrapf(err, "OfflineIsolatedInstances")
	}
	return nil
}

func (self *SRegion) ReleaseIsolatedMySQLDBInstances(ids []string) error {
	params := map[string]string{}
	for idx, id := range ids {
		params[fmt.Sprintf("InstanceIds.%d", idx)] = id
	}
	resp, err := self.cdbRequest("ReleaseIsolatedDBInstances", params)
	if err != nil {
		return errors.Wrapf(err, "ReleaseIsolatedDBInstances")
	}
	result := []struct {
		InstanceId string
		Code       int
		Message    string
	}{}
	err = resp.Unmarshal(&result, "Items")
	if err != nil {
		return errors.Wrapf(err, "resp.Unmarshal")
	}
	msg := []string{}
	for i := range result {
		if result[i].Code != 0 {
			msg = append(msg, fmt.Sprintf("instance %s release isolate error: %s", result[i].InstanceId, result[i].Message))
		}
	}
	if len(msg) > 0 {
		return errors.Error(strings.Join(msg, " "))
	}
	return cloudprovider.Wait(time.Second, time.Minute*10, func() (bool, error) {
		instances, _, err := self.ListMySQLInstances(ids, 0, len(ids))
		if err != nil {
			return false, errors.Wrapf(err, "ListMySQLInstances")
		}
		for i := range instances {
			if instances[i].Status == 4 || instances[i].Status == 5 {
				log.Debugf("mysql instance %s(%s) current be isolate", instances[i].InstanceName, instances[i].InstanceId)
				return false, nil
			}
		}
		return true, nil
	})
}

func (self *SRegion) IsolateMySQLDBInstance(id string) error {
	params := map[string]string{"InstanceId": id}
	resp, err := self.cdbRequest("IsolateDBInstance", params)
	if err != nil {
		return errors.Wrapf(err, "IsolateDBInstance")
	}
	asyncRequestId, _ := resp.GetString("AsyncRequestId")
	if len(asyncRequestId) > 0 {
		return self.waitAsyncAction("IsolateDBInstance", id, asyncRequestId)
	}
	return cloudprovider.Wait(time.Second*10, time.Minute*5, func() (bool, error) {
		instances, _, err := self.ListMySQLInstances([]string{id}, 0, 1)
		if err != nil {
			return false, errors.Wrapf(err, "ListMySQLInstances(%s)", id)
		}
		statusMap := map[int]string{0: "创建中", 1: "运行中", 4: "隔离中", 5: "已隔离"}
		for _, rds := range instances {
			status, _ := statusMap[rds.Status]
			log.Debugf("instance %s(%s) status %d(%s)", rds.InstanceName, rds.InstanceId, rds.Status, status)
			if rds.Status != 5 {
				return false, nil
			}
		}
		return true, nil
	})
}

func (self *SRegion) CloseMySQLWanService(id string) error {
	params := map[string]string{"InstanceId": id}
	resp, err := self.cdbRequest("CloseWanService", params)
	if err != nil {
		return errors.Wrapf(err, "CloseWanService")
	}
	asyncRequestId, _ := resp.GetString("AsyncRequestId")
	return self.waitAsyncAction("CloseWanService", id, asyncRequestId)
}

func (self *SRegion) OpenMySQLWanService(id string) error {
	params := map[string]string{"InstanceId": id}
	resp, err := self.cdbRequest("OpenWanService", params)
	if err != nil {
		return errors.Wrapf(err, "OpenWanService")
	}
	asyncRequestId, _ := resp.GetString("AsyncRequestId")
	return self.waitAsyncAction("OpenWanService", id, asyncRequestId)
}

func (self *SRegion) InitMySQLDBInstances(ids []string, password string, parameters map[string]string, vport int) error {
	params := map[string]string{"NewPassword": password}
	for idx, id := range ids {
		params[fmt.Sprintf("InstanceIds.%d", idx)] = id
	}
	i := 0
	for k, v := range parameters {
		params[fmt.Sprintf("Parameters.%d.name", i)] = k
		params[fmt.Sprintf("Parameters.%d.value", i)] = v
		i++
	}
	if vport >= 1024 && vport <= 65535 {
		params["Vport"] = fmt.Sprintf("%d", vport)
	}
	resp, err := self.cdbRequest("InitDBInstances", params)
	if err != nil {
		return errors.Wrapf(err, "InitDBInstances")
	}
	asyncRequestIds := []string{}
	err = resp.Unmarshal(&asyncRequestIds, "AsyncRequestIds")
	if err != nil {
		return errors.Wrapf(err, "resp.Unmarshal")
	}
	for idx, requestId := range asyncRequestIds {
		err = self.waitAsyncAction("InitDBInstances", fmt.Sprintf("%d", idx), requestId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SRegion) UpgradeMySQLDBInstance(id string, memoryMb int, volumeGb int) error {
	params := map[string]string{
		"InstanceId": id,
		"Memory":     fmt.Sprintf("%d", memoryMb),
		"Volume":     fmt.Sprintf("%d", volumeGb),
	}
	resp, err := self.cdbRequest("UpgradeDBInstance", params)
	if err != nil {
		return errors.Wrapf(err, "UpgradeDBInstance")
	}
	asyncRequestId, _ := resp.GetString("AsyncRequestId")
	return self.waitAsyncAction("UpgradeDBInstance", id, asyncRequestId)
}

func (self *SRegion) ModifyMySQLAutoRenewFlag(ids []string, autoRenew bool) error {
	params := map[string]string{}
	for idx, id := range ids {
		params[fmt.Sprintf("InstanceIds.%d", idx)] = id
	}
	params["AutoRenew"] = "0"
	if autoRenew {
		params["AutoRenew"] = "1"
	}
	_, err := self.cdbRequest("ModifyAutoRenewFlag", params)
	return err
}

type SMaintenanceTime struct {
	Monday    []string
	Tuesday   []string
	Wednesday []string
	Thursday  []string
	Friday    []string
	Saturday  []string
	Sunday    []string
}

func (w SMaintenanceTime) String() string {
	windows := []string{}
	for k, v := range map[string][]string{
		"Monday":    w.Monday,
		"Tuesday":   w.Tuesday,
		"Wednesday": w.Wednesday,
		"Thursday":  w.Thursday,
		"Friday":    w.Friday,
		"Saturday":  w.Saturday,
		"Sunday":    w.Sunday,
	} {
		if len(v) > 0 {
			windows = append(windows, fmt.Sprintf("%s: %s", k, strings.Join(v, " ")))
		}
	}
	return strings.Join(windows, "\n")
}

func (self *SRegion) DescribeMySQLTimeWindow(id string) (*SMaintenanceTime, error) {
	params := map[string]string{"InstanceId": id}
	resp, err := self.cdbRequest("DescribeTimeWindow", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeTimeWindow")
	}
	timeWindow := &SMaintenanceTime{}
	err = resp.Unmarshal(timeWindow)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return timeWindow, nil
}

type SDBSecgroup struct {
	ProjectId           int
	CreateTime          time.Time
	SecurityGroupId     string
	SecurityGroupName   string
	SecurityGroupRemark string
}

func (self *SRegion) DescribeMySQLDBSecurityGroups(instanceId string) ([]SDBSecgroup, error) {
	params := map[string]string{
		"InstanceId": instanceId,
	}
	resp, err := self.cdbRequest("DescribeDBSecurityGroups", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeDBSecurityGroups")
	}
	result := []SDBSecgroup{}
	err = resp.Unmarshal(&result, "Groups")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return result, nil
}

func (self *SRegion) CreateMySQLDBInstance(opts *cloudprovider.SManagedDBInstanceCreateConfig) (*SMySQLInstance, error) {
	params := map[string]string{
		"InstanceName":  opts.Name,
		"GoodsNum":      "1",
		"Memory":        fmt.Sprintf("%d", opts.VmemSizeMb),
		"Volume":        fmt.Sprintf("%d", opts.DiskSizeGB),
		"EngineVersion": opts.EngineVersion,
	}
	if len(opts.VpcId) > 0 {
		params["UniqVpcId"] = opts.VpcId
	}
	if len(opts.NetworkId) > 0 {
		params["UniqSubnetId"] = opts.NetworkId
	}
	if len(opts.ProjectId) > 0 {
		params["ProjectId"] = opts.ProjectId
	}
	if opts.Port > 1024 && opts.Port < 65535 {
		params["Port"] = fmt.Sprintf("%d", opts.Port)
	}
	if len(opts.Password) > 0 {
		params["Password"] = opts.Password
	}
	for i, secId := range opts.SecgroupIds {
		params[fmt.Sprintf("SecurityGroup.%d", i)] = secId
	}
	action := "CreateDBInstanceHour"
	if opts.BillingCycle != nil {
		params["Period"] = fmt.Sprintf("%d", opts.BillingCycle.GetMonths())
		params["AutoRenewFlag"] = "0"
		if opts.BillingCycle.AutoRenew {
			params["AutoRenewFlag"] = "1"
		}
		action = "CreateDBInstance"
	}
	if len(opts.Zone1) > 0 {
		params["Zone"] = opts.Zone1
	}
	params["DeployMode"] = "0"
	switch opts.Category {
	case api.QCLOUD_DBINSTANCE_CATEGORY_BASIC:
		params["DeviceType"] = strings.ToUpper(opts.Category)
	case api.QCLOUD_DBINSTANCE_CATEGORY_HA:
		params["DeviceType"] = strings.ToUpper(opts.Category)
		if len(opts.Zone2) > 0 {
			params["SlaveZone"] = opts.Zone2
		}
	case api.QCLOUD_DBINSTANCE_CATEGORY_FINANCE:
		params["DeviceType"] = "HA"
		params["ProtectMode"] = "2"
		if len(opts.Zone2) > 0 {
			params["SlaveZone"] = opts.Zone2
		}
		if len(opts.Zone3) > 0 {
			params["BackupZone"] = opts.Zone3
		}
	}
	if len(opts.Zone1) > 0 && len(opts.Zone2) > 0 && opts.Zone1 != opts.Zone2 {
		params["DeployMode"] = "1"
	}
	params["ClientToken"] = utils.GenRequestId(20)

	i := 0
	for k, v := range opts.Tags {
		params[fmt.Sprintf("ResourceTags.%d.TagKey", i)] = k
		params[fmt.Sprintf("ResourceTags.%d.TagValue", i)] = v
		i++
	}

	var create = func(action string, params map[string]string) (jsonutils.JSONObject, error) {
		startTime := time.Now()
		var resp jsonutils.JSONObject
		var err error
		for time.Now().Sub(startTime) < time.Minute*10 {
			resp, err = self.cdbRequest(action, params)
			if err != nil {
				if strings.Contains(err.Error(), "OperationDenied.OtherOderInProcess") || strings.Contains(err.Error(), "Message=请求已经在处理中") {
					time.Sleep(time.Second * 20)
					continue
				}
				return nil, errors.Wrapf(err, "cdbRequest")
			}
			return resp, nil
		}
		return resp, err
	}

	resp, err := create(action, params)
	if err != nil {
		return nil, errors.Wrapf(err, "cdbRequest")
	}
	instanceIds := []string{}
	err = resp.Unmarshal(&instanceIds, "InstanceIds")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	if len(instanceIds) == 0 {
		return nil, fmt.Errorf("%s not return InstanceIds", action)
	}
	err = cloudprovider.Wait(time.Second*10, time.Minute*20, func() (bool, error) {
		instances, _, err := self.ListMySQLInstances(instanceIds, 0, 1)
		if err != nil {
			return false, errors.Wrapf(err, "ListMySQLInstances(%s)", instanceIds)
		}
		for _, rds := range instances {
			log.Debugf("instance %s(%s) task status: %d", rds.InstanceName, rds.InstanceId, rds.TaskStatus)
			if rds.TaskStatus == 1 {
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		return nil, errors.Wrapf(err, "cloudprovider.Wait After create")
	}
	return self.GetMySQLInstanceById(instanceIds[0])
}

func (self *SRegion) GetMySQLInstanceById(id string) (*SMySQLInstance, error) {
	part, total, err := self.ListMySQLInstances([]string{id}, 0, 20)
	if err != nil {
		return nil, errors.Wrapf(err, "ListMySQLInstances")
	}
	if total > 1 {
		return nil, errors.Wrapf(cloudprovider.ErrDuplicateId, "id: [%s]", id)
	}
	if total < 1 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
	}
	part[0].region = self
	return &part[0], nil
}

func (self *SMySQLInstance) CreateDatabase(opts *cloudprovider.SDBInstanceDatabaseCreateConfig) error {
	return cloudprovider.ErrNotSupported
}

func (self *SMySQLInstance) CreateAccount(opts *cloudprovider.SDBInstanceAccountCreateConfig) error {
	return self.region.CreateMySQLAccount(self.InstanceId, opts)
}

func (self *SMySQLInstance) CreateIBackup(opts *cloudprovider.SDBInstanceBackupCreateConfig) (string, error) {
	tables := map[string]string{}
	for _, d := range opts.Databases {
		tables[d] = ""
	}
	return self.region.CreateMySQLBackup(self.InstanceId, tables)
}

func (self *SMySQLInstance) SetTags(tags map[string]string, replace bool) error {
	return self.region.SetResourceTags("cdb", "instanceId", []string{self.InstanceId}, tags, replace)
}

func (self *SRegion) GetIMySQLs() ([]cloudprovider.ICloudDBInstance, error) {
	ret := []cloudprovider.ICloudDBInstance{}
	mysql := []SMySQLInstance{}
	for {
		part, total, err := self.ListMySQLInstances([]string{}, len(mysql), 50)
		if err != nil {
			return nil, errors.Wrapf(err, "ListMySQLInstances")
		}
		mysql = append(mysql, part...)
		if len(mysql) >= total {
			break
		}
	}
	for i := range mysql {
		mysql[i].region = self
		ret = append(ret, &mysql[i])
	}
	return ret, nil
}
