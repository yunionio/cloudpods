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
	"context"
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
	"yunion.io/x/onecloud/pkg/util/billing"
)

type SBackupStrategy struct {
	KeepDays  int
	StartTime string
}

type SDatastore struct {
	Type    string
	Version string
}

type SHa struct {
	ReplicationMode string
}

type SNonde struct {
	AvailabilityZone string
	Id               string
	Name             string
	Role             string
	Staus            string
}

type SVolume struct {
	Size int
	Type string
}

type SRelatedInstance struct {
	Id   string
	Type string
}

type SDBInstance struct {
	multicloud.SDBInstanceBase
	region *SRegion

	flavorCache []SDBInstanceFlavor

	BackupStrategy    SBackupStrategy
	Created           string //time.Time
	Datastore         SDatastore
	DbUserName        string
	DIskEncryptionId  string
	FlavorRef         string
	Ha                SHa
	Id                string
	MaintenanceWindow string
	Name              string
	Nodes             []SNonde
	Port              int
	PrivateIps        []string
	PublicIps         []string
	Region            string
	RelatedInstance   []SRelatedInstance
	SecurityGroupId   string
	Status            string
	SubnetId          string
	SwitchStrategy    string
	TimeZone          string
	Type              string
	Updated           string //time.Time
	Volume            SVolume
	VpcId             string
}

func (region *SRegion) GetDBInstances() ([]SDBInstance, error) {
	params := map[string]string{}
	dbinstances := []SDBInstance{}
	err := doListAllWithOffset(region.ecsClient.DBInstance.List, params, &dbinstances)
	return dbinstances, err
}

func (region *SRegion) GetDBInstance(instanceId string) (*SDBInstance, error) {
	if len(instanceId) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	instance := SDBInstance{region: region}
	err := DoGet(region.ecsClient.DBInstance.Get, instanceId, nil, &instance)
	return &instance, err
}

func (rds *SDBInstance) GetName() string {
	return rds.Name
}

func (rds *SDBInstance) GetId() string {
	return rds.Id
}

func (rds *SDBInstance) GetGlobalId() string {
	return rds.GetId()
}

// 值为“BUILD”，表示实例正在创建。
// 值为“ACTIVE”，表示实例正常。
// 值为“FAILED”，表示实例异常。
// 值为“FROZEN”，表示实例冻结。
// 值为“MODIFYING”，表示实例正在扩容。
// 值为“REBOOTING”，表示实例正在重启。
// 值为“RESTORING”，表示实例正在恢复。
// 值为“MODIFYING INSTANCE TYPE”，表示实例正在转主备。
// 值为“SWITCHOVER”，表示实例正在主备切换。
// 值为“MIGRATING”，表示实例正在迁移。
// 值为“BACKING UP”，表示实例正在进行备份。
// 值为“MODIFYING DATABASE PORT”，表示实例正在修改数据库端口。
// 值为“STORAGE FULL”，表示实例磁盘空间满。

func (rds *SDBInstance) GetStatus() string {
	switch rds.Status {
	case "BUILD", "MODIFYING", "MODIFYING INSTANCE TYPE", "SWITCHOVER", "MODIFYING DATABASE PORT":
		return api.DBINSTANCE_DEPLOYING
	case "ACTIVE":
		return api.DBINSTANCE_RUNNING
	case "FAILED", "FROZEN", "STORAGE FULL":
		return api.DBINSTANCE_UNKNOWN
	case "REBOOTING":
		return api.DBINSTANCE_REBOOTING
	case "RESTORING":
		return api.DBINSTANCE_RESTORING
	case "MIGRATING":
		return api.DBINSTANCE_MIGRATING
	case "BACKING UP":
		return api.DBINSTANCE_BACKING_UP
	}
	return rds.Status
}

func (rds *SDBInstance) GetBillingType() string {
	_, err := rds.region.GetOrderResourceDetail(fmt.Sprintf("%s.vm", rds.Id))
	if err != nil {
		return billing_api.BILLING_TYPE_POSTPAID
	}
	return billing_api.BILLING_TYPE_PREPAID
}

func (rds *SDBInstance) GetSecurityGroupId() string {
	return rds.SecurityGroupId
}

func (rds *SDBInstance) fetchFlavor() error {
	if len(rds.flavorCache) > 0 {
		return nil
	}
	flavors, err := rds.region.GetDBInstanceFlavors(rds.Datastore.Type, rds.Datastore.Version)
	if err != nil {
		return err
	}
	rds.flavorCache = flavors
	return nil
}

func (rds *SDBInstance) GetExpiredAt() time.Time {
	order, err := rds.region.GetOrderResourceDetail(fmt.Sprintf("%s.vm", rds.Id))
	if err != nil {
		return time.Time{}
	}
	return order.ExpireTime
}

func (rds *SDBInstance) GetStorageType() string {
	switch rds.Volume.Type {
	case "COMMON":
		return api.STORAGE_HUAWEI_SATA
	case "HIGH":
		return api.STORAGE_HUAWEI_SAS
	case "ULTRAHIGH":
		return api.STORAGE_HUAWEI_SSD
	}
	return rds.Volume.Type
}

func (rds *SDBInstance) GetCreatedAt() time.Time {
	t, err := time.Parse("2006-01-02T15:04:05Z0700", rds.Created)
	if err != nil {
		return time.Time{}
	}
	return t
}

func (rds *SDBInstance) GetEngine() string {
	return rds.Datastore.Type
}

func (rds *SDBInstance) GetEngineVersion() string {
	return rds.Datastore.Version
}

func (rds *SDBInstance) GetInstanceType() string {
	return rds.FlavorRef
}

func (rds *SDBInstance) GetCategory() string {
	switch rds.Type {
	case "Single":
		return api.HUAWEI_DBINSTANCE_CATEGORY_SINGLE
	case "Ha":
		return api.HUAWEI_DBINSTANCE_CATEGORY_HA
	case "Replica":
		return api.HUAWEI_DBINSTANCE_CATEGORY_REPLICA
	}
	return rds.Type
}

func (rds *SDBInstance) GetVcpuCount() int {
	err := rds.fetchFlavor()
	if err != nil {
		log.Errorf("failed to fetch flavors: %v", err)
		return 0
	}
	for _, flavor := range rds.flavorCache {
		if flavor.SpecCode == rds.FlavorRef {
			return flavor.Vcpus
		}
	}
	return 0
}

func (rds *SDBInstance) GetVmemSizeMB() int {
	err := rds.fetchFlavor()
	if err != nil {
		log.Errorf("failed to fetch flavors: %v", err)
		return 0
	}
	for _, flavor := range rds.flavorCache {
		if flavor.SpecCode == rds.FlavorRef {
			return flavor.Ram * 1024
		}
	}
	return 0
}

func (rds *SDBInstance) GetDiskSizeGB() int {
	return rds.Volume.Size
}

func (rds *SDBInstance) GetPort() int {
	return rds.Port
}

func (rds *SDBInstance) GetMaintainTime() string {
	return rds.MaintenanceWindow
}

func (rds *SDBInstance) GetIVpcId() string {
	return rds.VpcId
}

func (rds *SDBInstance) Refresh() error {
	instance, err := rds.region.GetDBInstance(rds.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(rds, instance)
}

func (rds *SDBInstance) GetIZoneId() string {
	zones := []string{}
	for _, node := range rds.Nodes {
		if node.Role == "master" {
			zones = append([]string{node.AvailabilityZone}, zones...)
		} else if node.Role == "slave" {
			zones = append(zones, node.AvailabilityZone)
		}
	}
	return strings.Join(zones, ",")
}

type SRdsNetwork struct {
	SubnetId string
	IP       string
}

func (rds *SDBInstance) GetDBNetwork() (*cloudprovider.SDBInstanceNetwork, error) {
	for _, ip := range rds.PrivateIps {
		inetwork := &cloudprovider.SDBInstanceNetwork{
			IP:        ip,
			NetworkId: rds.SubnetId,
		}
		return inetwork, nil
	}

	return nil, fmt.Errorf("failed to found network for huawei rds %s", rds.Name)
}

func (rds *SDBInstance) GetInternalConnectionStr() string {
	for _, ip := range rds.PrivateIps {
		return ip
	}
	return ""
}

func (rds *SDBInstance) GetConnectionStr() string {
	for _, ip := range rds.PublicIps {
		return ip
	}
	return ""
}

func (region *SRegion) GetIDBInstanceById(instanceId string) (cloudprovider.ICloudDBInstance, error) {
	dbinstance, err := region.GetDBInstance(instanceId)
	if err != nil {
		log.Errorf("failed to get dbinstance by id %s error: %v", instanceId, err)
	}
	return dbinstance, err
}

func (region *SRegion) GetIDBInstances() ([]cloudprovider.ICloudDBInstance, error) {
	instances, err := region.GetDBInstances()
	if err != nil {
		return nil, errors.Wrapf(err, "region.GetDBInstances()")
	}
	idbinstances := []cloudprovider.ICloudDBInstance{}
	for i := 0; i < len(instances); i++ {
		instances[i].region = region
		idbinstances = append(idbinstances, &instances[i])
	}
	return idbinstances, nil
}

func (rds *SDBInstance) GetIDBInstanceParameters() ([]cloudprovider.ICloudDBInstanceParameter, error) {
	parameters, err := rds.region.GetDBInstanceParameters(rds.Id)
	if err != nil {
		return nil, err
	}
	iparameters := []cloudprovider.ICloudDBInstanceParameter{}
	for i := 0; i < len(parameters); i++ {
		iparameters = append(iparameters, &parameters[i])
	}
	return iparameters, nil
}

func (rds *SDBInstance) GetIDBInstanceDatabases() ([]cloudprovider.ICloudDBInstanceDatabase, error) {
	databases, err := rds.region.GetDBInstanceDatabases(rds.Id)
	if err != nil {
		return nil, errors.Wrap(err, "rds.region.GetDBInstanceDatabases(rds.Id)")
	}

	idatabase := []cloudprovider.ICloudDBInstanceDatabase{}
	for i := 0; i < len(databases); i++ {
		databases[i].instance = rds
		idatabase = append(idatabase, &databases[i])
	}
	return idatabase, nil
}

func (rds *SDBInstance) GetIDBInstanceAccounts() ([]cloudprovider.ICloudDBInstanceAccount, error) {
	accounts, err := rds.region.GetDBInstanceAccounts(rds.Id)
	if err != nil {
		return nil, errors.Wrap(err, "rds.region.GetDBInstanceAccounts(rds.Id)")
	}

	user := "root"
	if rds.GetEngine() == api.DBINSTANCE_TYPE_SQLSERVER {
		user = "rduser"
	}

	accounts = append(accounts, SDBInstanceAccount{
		Name:     user,
		instance: rds,
	})

	iaccounts := []cloudprovider.ICloudDBInstanceAccount{}
	for i := 0; i < len(accounts); i++ {
		accounts[i].instance = rds
		iaccounts = append(iaccounts, &accounts[i])
	}
	return iaccounts, nil
}

func (rds *SDBInstance) Delete() error {
	return rds.region.DeleteDBInstance(rds.Id)
}

func (region *SRegion) DeleteDBInstance(instanceId string) error {
	_, err := region.ecsClient.DBInstance.Delete(instanceId, nil)
	return err
}

func (region *SRegion) CreateIDBInstance(desc *cloudprovider.SManagedDBInstanceCreateConfig) (cloudprovider.ICloudDBInstance, error) {
	zoneIds := []string{}
	zones, err := region.GetIZones()
	if err != nil {
		return nil, err
	}
	for _, zone := range zones {
		zoneIds = append(zoneIds, zone.GetId())
	}
	storageType := ""
	switch desc.StorageType {
	case api.STORAGE_HUAWEI_SAS:
		storageType = "HIGH"
	case api.STORAGE_HUAWEI_SSD:
		storageType = "ULTRAHIGH"
	case api.STORAGE_HUAWEI_SATA:
		storageType = "COMMON"
	default:
		return nil, fmt.Errorf("Unknown storage type %s", desc.StorageType)
	}

	params := map[string]interface{}{
		"region": region.ID,
		"name":   desc.Name,
		"datastore": map[string]string{
			"type":    desc.Engine,
			"version": desc.EngineVersion,
		},
		"password": desc.Password,
		"volume": map[string]interface{}{
			"type": storageType,
			"size": desc.DiskSizeGB,
		},
		"vpc_id":            desc.VpcId,
		"subnet_id":         desc.NetworkId,
		"security_group_id": desc.SecgroupId,
	}

	if len(desc.MasterInstanceId) > 0 {
		params["replica_of_id"] = desc.MasterInstanceId
	}

	switch desc.Category {
	case api.HUAWEI_DBINSTANCE_CATEGORY_HA:
		switch desc.Engine {
		case api.DBINSTANCE_TYPE_MYSQL, api.DBINSTANCE_TYPE_POSTGRESQL:
			params["ha"] = map[string]string{
				"mode":             "Ha",
				"replication_mode": "async",
			}
		case api.DBINSTANCE_TYPE_SQLSERVER:
			params["ha"] = map[string]string{
				"mode":             "Ha",
				"replication_mode": "sync",
			}
		}
		if len(desc.ZoneIds) == 0 {
			for _, masterZoneId := range zoneIds {
				for _, slaveZoneId := range zoneIds {
					desc.ZoneIds = append(desc.ZoneIds, fmt.Sprintf("%s,%s", masterZoneId, slaveZoneId))
				}
			}
		}
	case api.HUAWEI_DBINSTANCE_CATEGORY_SINGLE:
		if len(desc.ZoneIds) == 0 {
			desc.ZoneIds = zoneIds
		}
	case api.HUAWEI_DBINSTANCE_CATEGORY_REPLICA:
	}

	if desc.BillingCycle != nil {
		periodType := "month"
		periodNum := desc.BillingCycle.GetMonths()
		if desc.BillingCycle.GetYears() > 0 {
			periodType = "year"
			periodNum = desc.BillingCycle.GetYears()
		}
		params["charge_info"] = map[string]interface{}{
			"charge_mode":   "prePaid",
			"period_type":   periodType,
			"period_num":    periodNum,
			"is_auto_renew": false,
		}
	}
	var resp jsonutils.JSONObject = nil

	if len(desc.InstanceType) > 0 {
		params["flavor_ref"] = desc.InstanceType
		for _, zoneId := range desc.ZoneIds {
			params["availability_zone"] = zoneId
			resp, err = region.ecsClient.DBInstance.Create(jsonutils.Marshal(params))
			if err == nil {
				break
			}
		}
		if err != nil {
			log.Debugf("params: %s", jsonutils.Marshal(params).PrettyString())
			return nil, errors.Wrap(err, "DBInstance.Create")
		}
	} else {
		for _, spec := range desc.InstanceTypes {
			params["flavor_ref"] = spec.InstanceType
			for _, zoneId := range spec.ZoneIds {
				params["availability_zone"] = zoneId
				resp, err = region.ecsClient.DBInstance.Create(jsonutils.Marshal(params))
				if err == nil {
					break
				}
			}
			if err == nil {
				break
			}
		}
		if err != nil {
			log.Debugf("params: %s", jsonutils.Marshal(params).PrettyString())
			return nil, errors.Wrap(err, "region.ecsClient.DBInstance.Create")
		}
		if resp == nil {
			return nil, fmt.Errorf("dbinstance type %dC%dMB not avaiable", desc.VcpuCount, desc.VmemSizeMb)
		}
	}

	instance := &SDBInstance{region: region}
	err = resp.Unmarshal(instance, "instance")
	if err != nil {
		return nil, errors.Wrap(err, `resp.Unmarshal(&instance, "instance")`)
	}
	if jobId, _ := resp.GetString("job_id"); len(jobId) > 0 {
		err = cloudprovider.WaitCreated(10*time.Second, 20*time.Minute, func() bool {
			job, err := region.ecsClient.DBInstanceJob.Get(jobId, map[string]string{"id": jobId})
			if err != nil {
				log.Errorf("failed to get job %s info error: %v", jobId, err)
				return false
			}
			status, _ := job.GetString("status")
			process, _ := job.GetString("process")
			if status == "Completed" {
				return true
			}
			log.Debugf("create dbinstance job %s status: %s process: %s", jobId, status, process)
			return false
		})
	}
	return instance, nil
}

func (rds *SDBInstance) Reboot() error {
	return rds.region.RebootDBInstance(rds.Id)
}

func (rds *SDBInstance) OpenPublicConnection() error {
	return fmt.Errorf("Huawei current not support this operation")
	//return rds.region.PublicConnectionAction(rds.Id, "openRC")
}

func (rds *SDBInstance) ClosePublicConnection() error {
	return fmt.Errorf("Huawei current not support this operation")
	//return rds.region.PublicConnectionAction(rds.Id, "closeRC")
}

func (region *SRegion) PublicConnectionAction(instanceId string, action string) error {
	resp, err := region.ecsClient.DBInstance.PerformAction2(action, instanceId, nil, "")
	if err != nil {
		return errors.Wrapf(err, "rds.%s", action)
	}

	if jobId, _ := resp.GetString("job_id"); len(jobId) > 0 {
		err = cloudprovider.WaitCreated(10*time.Second, 20*time.Minute, func() bool {
			job, err := region.ecsClient.DBInstanceJob.Get(jobId, map[string]string{"id": jobId})
			if err != nil {
				log.Errorf("failed to get job %s info error: %v", jobId, err)
				return false
			}
			status, _ := job.GetString("status")
			process, _ := job.GetString("process")
			if status == "Completed" {
				return true
			}
			log.Debugf("%s dbinstance job %s status: %s process: %s", action, jobId, status, process)
			return false
		})
	}

	return nil

}

func (region *SRegion) RebootDBInstance(instanceId string) error {
	params := jsonutils.Marshal(map[string]interface{}{
		"restart": map[string]string{},
	})
	resp, err := region.ecsClient.DBInstance.PerformAction2("action", instanceId, params, "")
	if err != nil {
		return err
	}
	if jobId, _ := resp.GetString("job_id"); len(jobId) > 0 {
		err = cloudprovider.WaitCreated(10*time.Second, 20*time.Minute, func() bool {
			job, err := region.ecsClient.DBInstanceJob.Get(jobId, map[string]string{"id": jobId})
			if err != nil {
				log.Errorf("failed to get job %s info error: %v", jobId, err)
				return false
			}
			status, _ := job.GetString("status")
			process, _ := job.GetString("process")
			if status == "Completed" {
				return true
			}
			log.Debugf("reboot dbinstance job %s status: %s process: %s", jobId, status, process)
			return false
		})
	}
	return err
}

type SDBInstanceFlavor struct {
	Vcpus        int
	Ram          int //单位GB
	SpecCode     string
	InstanceMode string //实例模型
}

func (region *SRegion) GetDBInstanceFlavors(engine string, version string) ([]SDBInstanceFlavor, error) {
	flavors := []SDBInstanceFlavor{}
	resp, err := region.ecsClient.DBInstanceFlavor.ListInContextWithSpec(nil, engine, map[string]string{"version_name": version}, "flavors")
	if err != nil {
		return nil, err
	}
	return flavors, jsonutils.Update(&flavors, resp.Data)
}

func (rds *SDBInstance) CreateAccount(conf *cloudprovider.SDBInstanceAccountCreateConfig) error {
	return rds.region.CreateDBInstanceAccount(rds.Id, conf.Name, conf.Password)
}

func (region *SRegion) CreateDBInstanceAccount(instanceId, account, password string) error {
	params := map[string]string{
		"name":     account,
		"password": password,
	}
	_, err := region.ecsClient.DBInstance.CreateInContextWithSpec(nil, fmt.Sprintf("%s/db_user", instanceId), jsonutils.Marshal(params), "")
	return err
}

func (rds *SDBInstance) CreateDatabase(conf *cloudprovider.SDBInstanceDatabaseCreateConfig) error {
	return rds.region.CreateDBInstanceDatabase(rds.Id, conf.Name, conf.CharacterSet)
}

func (region *SRegion) CreateDBInstanceDatabase(instanceId, database, characterSet string) error {
	params := map[string]string{
		"name":          database,
		"character_set": characterSet,
	}
	_, err := region.ecsClient.DBInstance.CreateInContextWithSpec(nil, fmt.Sprintf("%s/database", instanceId), jsonutils.Marshal(params), "")
	return err
}

func (rds *SDBInstance) ChangeConfig(cxt context.Context, desc *cloudprovider.SManagedDBInstanceChangeConfig) error {
	if rds.GetInstanceType() == desc.InstanceType {
		desc.InstanceType = ""
	}

	if rds.GetDiskSizeGB() >= desc.DiskSizeGB {
		desc.DiskSizeGB = 0
	}

	return rds.region.ChangeDBInstanceConfig(rds.Id, desc.InstanceType, desc.DiskSizeGB)
}

func (region *SRegion) ChangeDBInstanceConfig(instanceId string, instanceType string, diskSizeGb int) error {
	instance, err := region.GetIDBInstanceById(instanceId)
	if err != nil {
		return errors.Wrapf(err, "region.GetIDBInstanceById(%s)", instanceId)
	}

	if len(instanceType) > 0 {
		params := map[string]map[string]string{
			"resize_flavor": map[string]string{
				"spec_code": instanceType,
			},
		}
		_, err := region.ecsClient.DBInstance.PerformAction("action", instanceId, jsonutils.Marshal(params))
		if err != nil {
			return errors.Wrap(err, "resize_flavor")
		}
		cloudprovider.WaitStatus(instance, api.DBINSTANCE_RUNNING, time.Second*5, time.Minute*30)
	}
	if diskSizeGb > 0 {
		params := map[string]map[string]int{
			"enlarge_volume": map[string]int{
				"size": diskSizeGb,
			},
		}
		_, err := region.ecsClient.DBInstance.PerformAction("action", instanceId, jsonutils.Marshal(params))
		if err != nil {
			return errors.Wrap(err, "enlarge_volume")
		}
		cloudprovider.WaitStatus(instance, api.DBINSTANCE_RUNNING, time.Second*5, time.Minute*30)
	}
	return nil
}

func (rds *SDBInstance) RecoveryFromBackup(conf *cloudprovider.SDBInstanceRecoveryConfig) error {
	return rds.region.RecoveryDBInstanceFromBackup(rds.Id, conf.BackupId, conf.Databases)
}

func (region *SRegion) RecoveryDBInstanceFromBackup(instanceId string, backupId string, databases map[string]string) error {
	params := map[string]interface{}{
		"source": map[string]interface{}{
			"instance_id":   instanceId,
			"type":          "backup",
			"backup_id":     backupId,
			"database_name": databases,
		},
		"target": map[string]string{
			"instance_id": instanceId,
		},
	}
	_, err := region.ecsClient.DBInstance.PerformAction("", "recovery", jsonutils.Marshal(params))
	if err != nil {
		return errors.Wrap(err, "dbinstance.recovery")
	}
	return nil
}

func (rds *SDBInstance) Renew(bc billing.SBillingCycle) error {
	return rds.region.RenewInstance(rds.Id, bc)
}
