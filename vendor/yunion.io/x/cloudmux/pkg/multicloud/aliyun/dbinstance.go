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
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/rand"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SReadOnlyDBInstanceIds struct {
	ReadOnlyDBInstanceId []string
}

type SDBInstanceId struct {
	DBInstanceId []string
}

type SDBInstanceExtra struct {
	DBInstanceId SDBInstanceId
}

type SDBInstance struct {
	multicloud.SDBInstanceBase

	netInfo []SDBInstanceNetwork

	region *SRegion

	AccountMaxQuantity        int
	AccountType               string
	CanTempUpgrade            bool
	Category                  string
	AvailabilityValue         string
	DBInstanceDescription     string
	DBInstanceId              string
	ConnectionMode            string
	ConnectionString          string
	ConnectionDomain          string
	CurrentKernelVersion      string
	DBInstanceCPU             int
	CreateTime                time.Time
	DBInstanceClass           string
	DBInstanceClassType       string
	DBInstanceNetType         string
	DBInstanceStatus          string
	DBInstanceType            string
	DBInstanceDiskUsed        int64
	DBInstanceStorage         int
	DBInstanceStorageType     string
	MasterInstanceId          string
	DBInstanceMemory          int
	DBMaxQuantity             int
	IPType                    string
	LatestKernelVersion       string
	DispenseMode              string
	Engine                    string
	EngineVersion             string
	ExpireTime                time.Time
	InstanceNetworkType       string
	LockMode                  string
	LockReason                string
	MutriORsignle             bool
	MaintainTime              string
	MaxConnections            int
	MaxIOPS                   int
	Port                      int
	PayType                   TChargeType
	ReadOnlyDBInstanceIds     SReadOnlyDBInstanceIds
	RegionId                  string
	ResourceGroupId           string
	VSwitchId                 string
	VpcCloudInstanceId        string
	VpcId                     string
	ZoneId                    string
	Extra                     SDBInstanceExtra
	SecurityIPList            string
	SecurityIPMode            string
	SupportCreateSuperAccount string
	SupportUpgradeAccountType string
	TempUpgradeTimeEnd        time.Time
	TempUpgradeTimeStart      time.Time
}

func (rds *SDBInstance) GetName() string {
	if len(rds.DBInstanceDescription) > 0 {
		return rds.DBInstanceDescription
	}
	return rds.DBInstanceId
}

func (rds *SDBInstance) GetId() string {
	return rds.DBInstanceId
}

func (rds *SDBInstance) GetGlobalId() string {
	return rds.GetId()
}

// Creating	创建中
// Running	使用中
// Deleting	删除中
// Rebooting	重启中
// DBInstanceClassChanging	升降级中
// TRANSING	迁移中
// EngineVersionUpgrading	迁移版本中
// TransingToOthers	迁移数据到其他RDS中
// GuardDBInstanceCreating	生产灾备实例中
// Restoring	备份恢复中
// Importing	数据导入中
// ImportingFromOthers	从其他RDS实例导入数据中
// DBInstanceNetTypeChanging	内外网切换中
// GuardSwitching	容灾切换中
// INS_CLONING	实例克隆中
func (rds *SDBInstance) GetStatus() string {
	switch rds.DBInstanceStatus {
	case "Creating", "GuardDBInstanceCreating", "DBInstanceNetTypeChanging", "GuardSwitching", "NET_CREATING", "NET_DELETING":
		return api.DBINSTANCE_DEPLOYING
	case "DBInstanceClassChanging":
		return api.DBINSTANCE_CHANGE_CONFIG
	case "Running":
		return api.DBINSTANCE_RUNNING
	case "Deleting":
		return api.DBINSTANCE_DELETING
	case "Rebooting":
		return api.DBINSTANCE_REBOOTING
	case "TRANSING", "EngineVersionUpgrading", "TransingToOthers":
		return api.DBINSTANCE_MIGRATING
	case "Restoring":
		return api.DBINSTANCE_RESTORING
	case "Importing", "ImportingFromOthers":
		return api.DBINSTANCE_IMPORTING
	case "INS_CLONING":
		return api.DBINSTANCE_CLONING
	default:
		log.Errorf("Unknown dbinstance status %s", rds.DBInstanceStatus)
		return api.DBINSTANCE_UNKNOWN
	}
}

func (rds *SDBInstance) GetBillingType() string {
	return convertChargeType(rds.PayType)
}

func (rds *SDBInstance) GetExpiredAt() time.Time {
	return rds.ExpireTime
}

func (rds *SDBInstance) GetCreatedAt() time.Time {
	return rds.CreateTime
}

func (rds *SDBInstance) GetStorageType() string {
	return rds.DBInstanceStorageType
}

func (rds *SDBInstance) GetEngine() string {
	switch rds.Engine {
	case "MySQL":
		return api.DBINSTANCE_TYPE_MYSQL
	case "SQLServer":
		return api.DBINSTANCE_TYPE_SQLSERVER
	case "PostgreSQL":
		return api.DBINSTANCE_TYPE_POSTGRESQL
	case "PPAS":
		return api.DBINSTANCE_TYPE_PPAS
	case "MariaDB":
		return api.DBINSTANCE_TYPE_MARIADB
	}
	return rds.Engine
}

func (rds *SDBInstance) GetEngineVersion() string {
	return rds.EngineVersion
}

func (rds *SDBInstance) GetInstanceType() string {
	return rds.DBInstanceClass
}

func (rds *SDBInstance) GetCategory() string {
	switch rds.Category {
	case "Basic":
		return api.ALIYUN_DBINSTANCE_CATEGORY_BASIC
	case "HighAvailability":
		return api.ALIYUN_DBINSTANCE_CATEGORY_HA
	case "AlwaysOn":
		return api.ALIYUN_DBINSTANCE_CATEGORY_ALWAYSON
	case "Finance":
		return api.ALIYUN_DBINSTANCE_CATEGORY_FINANCE
	}
	return rds.Category
}

func (rds *SDBInstance) GetVcpuCount() int {
	if rds.DBInstanceCPU == 0 {
		rds.Refresh()
	}
	return rds.DBInstanceCPU
}

func (rds *SDBInstance) GetVmemSizeMB() int {
	if rds.DBInstanceMemory == 0 {
		rds.Refresh()
	}
	return rds.DBInstanceMemory
}

func (rds *SDBInstance) GetDiskSizeGB() int {
	if rds.DBInstanceStorage == 0 {
		rds.Refresh()
	}
	return rds.DBInstanceStorage
}

func (rds *SDBInstance) GetDiskSizeUsedMB() int {
	if rds.DBInstanceDiskUsed == 0 {
		rds.Refresh()
	}
	return int(rds.DBInstanceDiskUsed / 1024 / 1024)
}

func (rds *SDBInstance) GetPort() int {
	if rds.Port == 0 {
		rds.Refresh()
	}
	return rds.Port
}

func (rds *SDBInstance) GetMaintainTime() string {
	return rds.MaintainTime
}

func (rds *SDBInstance) GetIVpcId() string {
	return rds.VpcId
}

func (rds *SDBInstance) Refresh() error {
	instance, err := rds.region.GetDBInstanceDetail(rds.DBInstanceId)
	if err != nil {
		return err
	}
	return jsonutils.Update(rds, instance)
}

func (rds *SDBInstance) getZoneId(index int) string {
	zoneId := rds.getZone(index)
	if len(zoneId) > 0 {
		zone, err := rds.region.getZoneById(zoneId)
		if err != nil {
			log.Errorf("failed to found zone %s for rds %s", zoneId, rds.GetName())
			return ""
		}
		return zone.GetGlobalId()
	}
	return ""
}

func (rds *SDBInstance) GetZone1Id() string {
	return rds.getZoneId(1)
}

func (rds *SDBInstance) GetZone2Id() string {
	return rds.getZoneId(2)
}

func (rds *SDBInstance) GetZone3Id() string {
	return rds.getZoneId(3)
}

func (rds *SDBInstance) GetIOPS() int {
	if rds.MaxIOPS == 0 {
		rds.Refresh()
	}
	return rds.MaxIOPS
}

func (rds *SDBInstance) GetNetworkAddress() string {
	return rds.ConnectionDomain
}

func (rds *SDBInstance) getZone(index int) string {
	zoneStr := strings.Replace(rds.ZoneId, ")", "", -1)
	zoneInfo := strings.Split(zoneStr, ",")
	if len(zoneInfo) < index {
		return ""
	}
	zone := zoneInfo[index-1]
	zoneCode := zone[len(zone)-1]
	if strings.HasPrefix(rds.ZoneId, fmt.Sprintf("%s-", rds.RegionId)) {
		return fmt.Sprintf("%s-%s", rds.RegionId, string(zoneCode))
	}
	return fmt.Sprintf("%s%s", rds.RegionId, string(zoneCode))
}

func (rds *SDBInstance) GetDBNetworks() ([]cloudprovider.SDBInstanceNetwork, error) {
	netInfo, err := rds.region.GetDBInstanceNetInfo(rds.DBInstanceId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDBInstanceNetInfo")
	}
	networks := []cloudprovider.SDBInstanceNetwork{}
	for _, net := range netInfo {
		if net.IPType == "Private" {
			network := cloudprovider.SDBInstanceNetwork{}
			network.IP = net.IPAddress
			network.NetworkId = net.VSwitchId
			networks = append(networks, network)
		}
	}
	return networks, nil
}

func (rds *SDBInstance) fetchNetInfo() error {
	if len(rds.netInfo) > 0 {
		return nil
	}
	netInfo, err := rds.region.GetDBInstanceNetInfo(rds.DBInstanceId)
	if err != nil {
		return errors.Wrap(err, "GetDBInstanceNetInfo")
	}
	rds.netInfo = netInfo
	return nil
}

func (rds *SDBInstance) GetInternalConnectionStr() string {
	err := rds.fetchNetInfo()
	if err != nil {
		log.Errorf("failed to fetch netInfo error: %v", err)
		return ""
	}

	str := ""
	for _, net := range rds.netInfo {
		if net.IPType != "Public" {
			if net.IPType == "Private" {
				return net.ConnectionString
			} else if net.IPType == "Inner" {
				str = net.ConnectionString
			}
		}
	}
	return str
}

func (rds *SDBInstance) GetConnectionStr() string {
	err := rds.fetchNetInfo()
	if err != nil {
		log.Errorf("failed to fetch netInfo error: %v", err)
		return ""
	}

	for _, net := range rds.netInfo {
		if net.IPType == "Public" {
			return net.ConnectionString
		}
	}
	return ""
}

func (region *SRegion) GetDBInstances(ids []string, offset int, limit int) ([]SDBInstance, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["RegionId"] = region.RegionId
	params["PageSize"] = fmt.Sprintf("%d", limit)
	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)

	body, err := region.rdsRequest("DescribeDBInstances", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "GetDBInstances")
	}
	instances := []SDBInstance{}
	err = body.Unmarshal(&instances, "Items", "DBInstance")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "GetDBInstances.Unmarshal")
	}
	total, _ := body.Int("TotalRecordCount")
	return instances, int(total), nil
}

func (region *SRegion) GetIDBInstanceById(instanceId string) (cloudprovider.ICloudDBInstance, error) {
	rds, err := region.GetDBInstanceDetail(instanceId)
	if err != nil {
		return nil, err
	}
	rds.region = region
	return rds, nil
}

func (region *SRegion) GetIDBInstances() ([]cloudprovider.ICloudDBInstance, error) {
	instances := []SDBInstance{}
	for {
		part, total, err := region.GetDBInstances([]string{}, len(instances), 50)
		if err != nil {
			return nil, err
		}
		instances = append(instances, part...)
		if len(instances) >= total {
			break
		}
	}
	idbinstances := []cloudprovider.ICloudDBInstance{}
	for i := 0; i < len(instances); i++ {
		instances[i].region = region
		idbinstances = append(idbinstances, &instances[i])
	}
	return idbinstances, nil
}

func (region *SRegion) GetDBInstanceDetail(instanceId string) (*SDBInstance, error) {
	if len(instanceId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["DBInstanceId"] = instanceId
	body, err := region.rdsRequest("DescribeDBInstanceAttribute", params)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDBInstanceDetail")
	}
	instances := []SDBInstance{}
	err = body.Unmarshal(&instances, "Items", "DBInstanceAttribute")
	if err != nil {
		return nil, errors.Wrapf(err, "GetDBInstanceDetail.Unmarshal")
	}
	if len(instances) == 1 {
		instances[0].region = region
		return &instances[0], nil
	}
	if len(instances) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	return nil, cloudprovider.ErrDuplicateId
}

func (region *SRegion) DeleteDBInstance(instanceId string) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["DBInstanceId"] = instanceId
	_, err := region.rdsRequest("DeleteDBInstance", params)
	return err
}

type SDBInstanceWeight struct {
}

type SDBInstanceWeights struct {
	DBInstanceWeight []SDBInstanceWeight
}

type SsecurityIPGroup struct {
}

type SSecurityIPGroups struct {
	securityIPGroup []SsecurityIPGroup
}

type SDBInstanceNetwork struct {
	ConnectionString     string
	ConnectionStringType string
	DBInstanceWeights    SDBInstanceWeights
	IPAddress            string
	IPType               string
	Port                 int
	SecurityIPGroups     SSecurityIPGroups
	Upgradeable          string
	VPCId                string
	VSwitchId            string
}

func (network *SDBInstanceNetwork) GetGlobalId() string {
	return network.IPAddress
}

func (network *SDBInstanceNetwork) GetINetworkId() string {
	return network.VSwitchId
}

func (network *SDBInstanceNetwork) GetIP() string {
	return network.IPAddress
}

func (region *SRegion) GetDBInstanceNetInfo(instanceId string) ([]SDBInstanceNetwork, error) {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["DBInstanceId"] = instanceId
	body, err := region.rdsRequest("DescribeDBInstanceNetInfo", params)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDBInstanceNetwork")
	}
	networks := []SDBInstanceNetwork{}
	err = body.Unmarshal(&networks, "DBInstanceNetInfos", "DBInstanceNetInfo")
	if err != nil {
		return nil, err
	}
	return networks, nil
}

func (rds *SDBInstance) GetIDBInstanceParameters() ([]cloudprovider.ICloudDBInstanceParameter, error) {
	parameters, err := rds.region.GetDBInstanceParameters(rds.DBInstanceId)
	if err != nil {
		return nil, err
	}
	iparameters := []cloudprovider.ICloudDBInstanceParameter{}
	for i := 0; i < len(parameters); i++ {
		iparameters = append(iparameters, &parameters[i])
	}
	return iparameters, nil
}

func (self *SDBInstance) GetSecurityGroupIds() ([]string, error) {
	return self.region.GetRdsSecgroupIds(self.DBInstanceId)
}

func (self *SDBInstance) SetSecurityGroups(ids []string) error {
	return self.region.SetRdsSecgroups(self.DBInstanceId, ids)
}

func (self *SRegion) SetRdsSecgroups(rdsId string, secIds []string) error {
	params := map[string]string{
		"DBInstanceId":    rdsId,
		"SecurityGroupId": strings.Join(secIds, ","),
	}
	_, err := self.rdsRequest("ModifySecurityGroupConfiguration", params)
	if err != nil {
		return errors.Wrapf(err, "ModifySecurityGroupConfiguration")
	}
	return nil
}

func (self *SRegion) GetRdsSecgroupIds(rdsId string) ([]string, error) {
	params := map[string]string{
		"DBInstanceId": rdsId,
	}
	resp, err := self.rdsRequest("DescribeSecurityGroupConfiguration", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeSecurityGroupConfiguration")
	}
	items := []struct {
		NetworkType     string
		SecurityGroupId string
		RegionId        string
	}{}
	err = resp.Unmarshal(&items, "Items", "EcsSecurityGroupRelation")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	ids := []string{}
	for _, item := range items {
		ids = append(ids, item.SecurityGroupId)
	}
	return ids, nil
}

func (rds *SDBInstance) Reboot() error {
	return rds.region.RebootDBInstance(rds.DBInstanceId)
}

func (rds *SDBInstance) Delete() error {
	return rds.region.DeleteDBInstance(rds.DBInstanceId)
}

func (region *SRegion) RebootDBInstance(instanceId string) error {
	params := map[string]string{}
	params["RegionId"] = region.RegionId
	params["DBInstanceId"] = instanceId
	_, err := region.rdsRequest("RestartDBInstance", params)
	return err
}

func (rds *SDBInstance) GetIDBInstanceDatabases() ([]cloudprovider.ICloudDBInstanceDatabase, error) {
	databases := []SDBInstanceDatabase{}
	for {
		parts, total, err := rds.region.GetDBInstanceDatabases(rds.DBInstanceId, "", len(databases), 500)
		if err != nil {
			return nil, err
		}
		databases = append(databases, parts...)
		if len(databases) >= total {
			break
		}
	}

	idatabase := []cloudprovider.ICloudDBInstanceDatabase{}
	for i := 0; i < len(databases); i++ {
		databases[i].instance = rds
		idatabase = append(idatabase, &databases[i])
	}
	return idatabase, nil
}

func (rds *SDBInstance) GetIDBInstanceAccounts() ([]cloudprovider.ICloudDBInstanceAccount, error) {
	accounts := []SDBInstanceAccount{}
	for {
		parts, total, err := rds.region.GetDBInstanceAccounts(rds.DBInstanceId, len(accounts), 50)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, parts...)
		if len(accounts) >= total {
			break
		}
	}

	iaccounts := []cloudprovider.ICloudDBInstanceAccount{}
	for i := 0; i < len(accounts); i++ {
		accounts[i].instance = rds
		iaccounts = append(iaccounts, &accounts[i])
	}
	return iaccounts, nil
}

func (rds *SDBInstance) ChangeConfig(cxt context.Context, desc *cloudprovider.SManagedDBInstanceChangeConfig) error {
	return rds.region.ChangeDBInstanceConfig(rds.DBInstanceId, string(rds.PayType), desc)
}

func (region *SRegion) ChangeDBInstanceConfig(instanceId, payType string, desc *cloudprovider.SManagedDBInstanceChangeConfig) error {
	params := map[string]string{
		"RegionId":     region.RegionId,
		"DBInstanceId": instanceId,
		"PayType":      payType,
	}
	if len(desc.InstanceType) > 0 {
		params["DBInstanceClass"] = desc.InstanceType
	}
	if desc.DiskSizeGB > 0 {
		params["DBInstanceStorage"] = fmt.Sprintf("%d", desc.DiskSizeGB)
	}

	_, err := region.rdsRequest("ModifyDBInstanceSpec", params)
	if err != nil {
		return errors.Wrap(err, "region.rdsRequest.ModifyDBInstanceSpec")
	}
	return nil
}

func (region *SRegion) CreateIDBInstance(desc *cloudprovider.SManagedDBInstanceCreateConfig) (cloudprovider.ICloudDBInstance, error) {
	params := map[string]string{
		"RegionId":              region.RegionId,
		"Engine":                desc.Engine,
		"EngineVersion":         desc.EngineVersion,
		"DBInstanceStorage":     fmt.Sprintf("%d", desc.DiskSizeGB),
		"DBInstanceNetType":     "Intranet",
		"PayType":               "Postpaid",
		"SecurityIPList":        "0.0.0.0/0",
		"DBInstanceDescription": desc.Name,
		"InstanceNetworkType":   "VPC",
		"VPCId":                 desc.VpcId,
		"VSwitchId":             desc.NetworkId,
		"DBInstanceStorageType": desc.StorageType,
		"DBInstanceClass":       desc.InstanceType,
		"ZoneId":                desc.ZoneId,
		"ClientToken":           utils.GenRequestId(20),
	}
	switch desc.Category {
	case api.ALIYUN_DBINSTANCE_CATEGORY_HA:
		params["Category"] = "HighAvailability"
	case api.ALIYUN_DBINSTANCE_CATEGORY_BASIC:
		params["Category"] = "Basic"
	case api.ALIYUN_DBINSTANCE_CATEGORY_ALWAYSON:
		params["Category"] = "AlwaysOn"
	case api.ALIYUN_DBINSTANCE_CATEGORY_FINANCE:
		params["Category"] = "Finance"
	}

	if len(desc.Address) > 0 {
		params["PrivateIpAddress"] = desc.Address
	}
	if len(desc.ProjectId) > 0 {
		params["ResourceGroupId"] = desc.ProjectId
	}
	if desc.BillingCycle != nil {
		params["PayType"] = "Prepaid"
		if desc.BillingCycle.GetYears() > 0 {
			params["Period"] = "Year"
			params["UsedTime"] = fmt.Sprintf("%d", desc.BillingCycle.GetYears())
		} else {
			params["Period"] = "Month"
			params["UsedTime"] = fmt.Sprintf("%d", desc.BillingCycle.GetMonths())
		}
		params["AutoRenew"] = "False"
		if desc.BillingCycle.AutoRenew {
			params["AutoRenew"] = "True"
		}
	}

	action := "CreateDBInstance"
	if len(desc.MasterInstanceId) > 0 {
		action = "CreateReadOnlyDBInstance"
		params["DBInstanceId"] = desc.MasterInstanceId
	}

	resp, err := region.rdsRequest(action, params)
	if err != nil {
		return nil, errors.Wrapf(err, "rdsRequest")
	}
	instanceId, err := resp.GetString("DBInstanceId")
	if err != nil {
		return nil, errors.Wrap(err, `resp.GetString("DBInstanceId")`)
	}
	region.SetResourceTags(ALIYUN_SERVICE_RDS, "INSTANCE", instanceId, desc.Tags, false)
	return region.GetIDBInstanceById(instanceId)
}

func (rds *SDBInstance) GetMasterInstanceId() string {
	if len(rds.MasterInstanceId) > 0 {
		return rds.MasterInstanceId
	}
	rds.Refresh()
	return rds.MasterInstanceId
}

func (region *SRegion) OpenPublicConnection(instanceId string) error {
	rds, err := region.GetDBInstanceDetail(instanceId)
	if err != nil {
		return err
	}
	params := map[string]string{
		"RegionId":               region.RegionId,
		"ConnectionStringPrefix": rds.DBInstanceId + rand.String(3),
		"DBInstanceId":           rds.DBInstanceId,
		"Port":                   fmt.Sprintf("%d", rds.Port),
	}
	_, err = rds.region.rdsRequest("AllocateInstancePublicConnection", params)
	if err != nil {
		return errors.Wrap(err, "rdsRequest(AllocateInstancePublicConnection)")
	}
	return nil
}

func (rds *SDBInstance) OpenPublicConnection() error {
	if url := rds.GetConnectionStr(); len(url) == 0 {
		err := rds.region.OpenPublicConnection(rds.DBInstanceId)
		if err != nil {
			return err
		}
		rds.netInfo = []SDBInstanceNetwork{}
	}
	return nil
}

func (region *SRegion) ClosePublicConnection(instanceId string) error {
	netInfo, err := region.GetDBInstanceNetInfo(instanceId)
	if err != nil {
		return errors.Wrap(err, "GetDBInstanceNetInfo")
	}

	for _, net := range netInfo {
		if net.IPType == "Public" {
			params := map[string]string{
				"RegionId":                region.RegionId,
				"CurrentConnectionString": net.ConnectionString,
				"DBInstanceId":            instanceId,
			}
			_, err = region.rdsRequest("ReleaseInstancePublicConnection", params)
			if err != nil {
				return errors.Wrap(err, "rdsRequest(ReleaseInstancePublicConnection)")
			}

		}
	}
	return nil

}

func (rds *SDBInstance) ClosePublicConnection() error {
	return rds.region.ClosePublicConnection(rds.DBInstanceId)
}

func (rds *SDBInstance) RecoveryFromBackup(conf *cloudprovider.SDBInstanceRecoveryConfig) error {
	if len(conf.OriginDBInstanceExternalId) == 0 {
		conf.OriginDBInstanceExternalId = rds.DBInstanceId
	}
	return rds.region.RecoveryDBInstanceFromBackup(conf.OriginDBInstanceExternalId, rds.DBInstanceId, conf.BackupId, conf.Databases)
}

func (region *SRegion) RecoveryDBInstanceFromBackup(srcId, destId string, backupId string, databases map[string]string) error {
	params := map[string]string{
		"RegionId":           region.RegionId,
		"DBInstanceId":       srcId,
		"TargetDBInstanceId": destId,
		"BackupId":           backupId,
		"DbNames":            jsonutils.Marshal(databases).String(),
	}
	_, err := region.rdsRequest("RecoveryDBInstance", params)
	if err != nil {
		return errors.Wrap(err, "rdsRequest.RecoveryDBInstance")
	}
	return nil
}

func (rds *SDBInstance) GetProjectId() string {
	return rds.ResourceGroupId
}

func (rds *SDBInstance) CreateDatabase(conf *cloudprovider.SDBInstanceDatabaseCreateConfig) error {
	return rds.region.CreateDBInstanceDatabae(rds.DBInstanceId, conf.CharacterSet, conf.Name, conf.Description)
}

func (rds *SDBInstance) CreateAccount(conf *cloudprovider.SDBInstanceAccountCreateConfig) error {
	return rds.region.CreateDBInstanceAccount(rds.DBInstanceId, conf.Name, conf.Password, conf.Description)
}

func (rds *SDBInstance) Renew(bc billing.SBillingCycle) error {
	return rds.region.RenewInstance(rds.DBInstanceId, bc)
}

func (rds *SDBInstance) SetAutoRenew(bc billing.SBillingCycle) error {
	month := 1
	if bc.GetMonths() > 0 {
		month = bc.GetMonths()
	}
	return rds.region.ModifyInstanceAutoRenewalAttribute(rds.DBInstanceId, month, bc.AutoRenew)
}

func (region *SRegion) ModifyInstanceAutoRenewalAttribute(rdsId string, month int, autoRenew bool) error {
	params := map[string]string{
		"RegionId":     region.RegionId,
		"DBInstanceId": rdsId,
		"AutoRenew":    "False",
		"ClientToken":  utils.GenRequestId(20),
	}
	if autoRenew {
		params["AutoRenew"] = "True"
		params["Duration"] = fmt.Sprintf("%d", month)
	}
	_, err := region.rdsRequest("ModifyInstanceAutoRenewalAttribute", params)
	if err != nil {
		return errors.Wrap(err, "ModifyInstanceAutoRenewalAttribute")
	}
	return nil
}

func (rds *SDBInstance) Update(ctx context.Context, input cloudprovider.SDBInstanceUpdateOptions) error {
	return rds.region.ModifyDBInstanceName(rds.DBInstanceId, input.NAME)
}

func (region *SRegion) ModifyDBInstanceName(id, name string) error {
	params := map[string]string{
		"DBInstanceId":          id,
		"DBInstanceDescription": name,
	}
	_, err := region.rdsRequest("ModifyDBInstanceDescription", params)
	if err != nil {
		return errors.Wrap(err, "ModifyDBInstanceDescription")
	}
	return nil
}

func (region *SRegion) RenewDBInstance(instanceId string, bc billing.SBillingCycle) error {
	params := map[string]string{
		"DBInstanceId": instanceId,
		"Period":       fmt.Sprintf("%d", bc.GetMonths()),
		"ClientToken":  utils.GenRequestId(20),
	}
	_, err := region.rdsRequest("RenewInstance", params)
	return err
}

func (self *SDBInstance) GetTags() (map[string]string, error) {
	_, tags, err := self.region.ListSysAndUserTags(ALIYUN_SERVICE_RDS, "INSTANCE", self.DBInstanceId)
	if err != nil {
		return nil, errors.Wrapf(err, "ListTags")
	}
	tagMaps := map[string]string{}
	for k, v := range tags {
		tagMaps[strings.ToLower(k)] = v
	}
	return tagMaps, nil
}

func (self *SDBInstance) GetSysTags() map[string]string {
	tags, _, err := self.region.ListSysAndUserTags(ALIYUN_SERVICE_RDS, "INSTANCE", self.DBInstanceId)
	if err != nil {
		return nil
	}
	tagMaps := map[string]string{}
	for k, v := range tags {
		tagMaps[strings.ToLower(k)] = v
	}
	return tagMaps
}

func (rds *SDBInstance) SetTags(tags map[string]string, replace bool) error {
	return rds.region.SetResourceTags(ALIYUN_SERVICE_RDS, "INSTANCE", rds.GetId(), tags, replace)
}
