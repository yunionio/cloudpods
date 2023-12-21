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
	"net/url"
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
	HuaweiTags
	region *SRegion

	flavorCache         []SDBInstanceFlavor
	BackupStrategy      SBackupStrategy
	Created             string //time.Time
	Datastore           SDatastore
	DbUserName          string
	DIskEncryptionId    string
	FlavorRef           string
	Ha                  SHa
	Id                  string
	MaintenanceWindow   string
	MaxIops             int
	Name                string
	Nodes               []SNonde
	Port                int
	PrivateIps          []string
	PublicIps           []string
	Region              string
	RelatedInstance     []SRelatedInstance
	SecurityGroupId     string
	Status              string
	SubnetId            string
	SwitchStrategy      string
	TimeZone            string
	Type                string
	Updated             string //time.Time
	Volume              SVolume
	VpcId               string
	EnterpriseProjectId string
}

func (region *SRegion) GetDBInstances(id string) ([]SDBInstance, error) {
	query := url.Values{}
	if len(id) > 0 {
		query.Set("id", id)
	}
	ret := []SDBInstance{}
	for {
		resp, err := region.list(SERVICE_RDS, "instances", query)
		if err != nil {
			return nil, errors.Wrapf(err, "list instances")
		}
		part := struct {
			Instances  []SDBInstance
			TotalCount int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Instances...)
		if len(ret) >= part.TotalCount || len(part.Instances) == 0 {
			break
		}
		query.Set("offset", fmt.Sprintf("%d", len(ret)))
	}
	return ret, nil
}

func (region *SRegion) GetDBInstance(instanceId string) (*SDBInstance, error) {
	ret, err := region.GetDBInstances(instanceId)
	if err != nil {
		return nil, err
	}
	for i := range ret {
		if ret[i].Id == instanceId {
			ret[i].region = region
			return &ret[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, instanceId)
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
	orders, err := rds.region.client.GetOrderResources()
	if err != nil {
		return billing_api.BILLING_TYPE_PREPAID
	}
	_, ok := orders[rds.Id]
	if ok {
		return billing_api.BILLING_TYPE_POSTPAID
	}
	return billing_api.BILLING_TYPE_PREPAID
}

func (rds *SDBInstance) GetSecurityGroupIds() ([]string, error) {
	return []string{rds.SecurityGroupId}, nil
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
	orders, err := rds.region.client.GetOrderResources()
	if err != nil {
		return time.Time{}
	}
	order, ok := orders[rds.Id]
	if ok {
		return order.ExpireTime
	}
	return time.Time{}
}

func (rds *SDBInstance) GetStorageType() string {
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

func (rds *SDBInstance) GetProjectId() string {
	return rds.EnterpriseProjectId
}

func (rds *SDBInstance) Refresh() error {
	instance, err := rds.region.GetDBInstance(rds.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(rds, instance)
}

func (rds *SDBInstance) GetZone1Id() string {
	return rds.GetZoneIdByRole("master")
}

func (rds *SDBInstance) GetZoneIdByRole(role string) string {
	for _, node := range rds.Nodes {
		if node.Role == role {
			zone, err := rds.region.getZoneById(node.AvailabilityZone)
			if err != nil {
				log.Errorf("failed to found zone %s for rds %s error: %v", node.AvailabilityZone, rds.Name, err)
				return ""
			}
			return zone.GetGlobalId()
		}
	}
	return ""
}

func (rds *SDBInstance) GetZone2Id() string {
	return rds.GetZoneIdByRole("slave")
}

func (rds *SDBInstance) GetZone3Id() string {
	return ""
}

func (rds *SDBInstance) GetIops() int {
	return rds.MaxIops
}

type SRdsNetwork struct {
	SubnetId string
	IP       string
}

func (rds *SDBInstance) GetDBNetworks() ([]cloudprovider.SDBInstanceNetwork, error) {
	ret := []cloudprovider.SDBInstanceNetwork{}
	for _, ip := range rds.PrivateIps {
		network := cloudprovider.SDBInstanceNetwork{
			IP:        ip,
			NetworkId: rds.SubnetId,
		}
		ret = append(ret, network)
	}
	return ret, nil
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
	instances, err := region.GetDBInstances("")
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
	_, err := region.delete(SERVICE_RDS, "instances/"+instanceId)
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

	if len(desc.SecgroupIds) == 0 {
		return nil, fmt.Errorf("Missing secgroupId")
	}

	params := map[string]interface{}{
		"name": desc.Name,
		"datastore": map[string]string{
			"type":    desc.Engine,
			"version": desc.EngineVersion,
		},
		"password": desc.Password,
		"volume": map[string]interface{}{
			"type": desc.StorageType,
			"size": desc.DiskSizeGB,
		},
		"vpc_id":            desc.VpcId,
		"subnet_id":         desc.NetworkId,
		"security_group_id": desc.SecgroupIds[0],
	}

	if len(desc.ProjectId) > 0 {
		params["enterprise_project_id"] = desc.ProjectId
	}

	if len(desc.MasterInstanceId) > 0 {
		params["replica_of_id"] = desc.MasterInstanceId
		delete(params, "security_group_id")
	}

	if len(desc.RdsId) > 0 && len(desc.BackupId) > 0 {
		params["restore_point"] = map[string]interface{}{
			"backup_id":   desc.BackupId,
			"instance_id": desc.RdsId,
			"type":        "backup",
		}
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
	case api.HUAWEI_DBINSTANCE_CATEGORY_SINGLE:
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
	params["flavor_ref"] = desc.InstanceType
	params["availability_zone"] = desc.ZoneId
	resp, err := region.post(SERVICE_RDS, "instances", params)
	if err != nil {
		return nil, errors.Wrapf(err, "Create")
	}

	instance := &SDBInstance{region: region}
	err = resp.Unmarshal(instance, "instance")
	if err != nil {
		return nil, errors.Wrap(err, `resp.Unmarshal(&instance, "instance")`)
	}
	if jobId, _ := resp.GetString("job_id"); len(jobId) > 0 {
		err = cloudprovider.Wait(10*time.Second, 20*time.Minute, func() (bool, error) {
			job, err := region.GetDBInstanceJob(jobId)
			if err != nil {
				return false, nil
			}
			log.Debugf("create dbinstance job %s status: %s process: %s", jobId, job.Status, job.Process)
			if job.Status == "Completed" {
				return true, nil
			}
			if job.Status == "Failed" {
				return false, fmt.Errorf("create failed")
			}
			return false, nil
		})
	}
	return instance, err
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

type SDBInstanceJob struct {
	Status  string
	Process string
}

func (self *SRegion) GetDBInstanceJob(id string) (*SDBInstanceJob, error) {
	query := url.Values{}
	query.Set("id", id)
	resp, err := self.list(SERVICE_RDS, "jobs", query)
	if err != nil {
		return nil, err
	}
	ret := &SDBInstanceJob{}
	err = resp.Unmarshal(ret, "job")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}

func (region *SRegion) PublicConnectionAction(instanceId string, action string) error {
	resp, err := region.post(SERVICE_RDS, fmt.Sprintf("instances/%s/%s", instanceId, action), nil)
	if err != nil {
		return errors.Wrapf(err, "rds.%s", action)
	}

	if jobId, _ := resp.GetString("job_id"); len(jobId) > 0 {
		err = cloudprovider.WaitCreated(10*time.Second, 20*time.Minute, func() bool {
			job, err := region.GetDBInstanceJob(jobId)
			if err != nil {
				log.Errorf("failed to get job %s info error: %v", jobId, err)
				return false
			}
			if job.Status == "Completed" {
				return true
			}
			log.Debugf("%s dbinstance job %s status: %s process: %s", action, jobId, job.Status, job.Process)
			return false
		})
	}

	return nil

}

func (region *SRegion) RebootDBInstance(instanceId string) error {
	params := map[string]interface{}{
		"restart": map[string]string{},
	}
	resp, err := region.post(SERVICE_RDS, fmt.Sprintf("instances/%s/action", instanceId), params)
	if err != nil {
		return err
	}
	if jobId, _ := resp.GetString("job_id"); len(jobId) > 0 {
		err = cloudprovider.WaitCreated(10*time.Second, 20*time.Minute, func() bool {
			job, err := region.GetDBInstanceJob(jobId)
			if err != nil {
				return false
			}
			if job.Status == "Completed" {
				return true
			}
			log.Debugf("reboot dbinstance job %s status: %s process: %s", jobId, job.Status, job.Process)
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
	query := url.Values{}
	if len(version) > 0 {
		query.Set("version_name", version)
	}
	resp, err := region.list(SERVICE_RDS, "flavors/"+engine, query)
	if err != nil {
		return nil, err
	}
	flavors := []SDBInstanceFlavor{}
	err = resp.Unmarshal(&flavors, "flavors")
	if err != nil {
		return nil, err
	}
	return flavors, nil
}

func (rds *SDBInstance) CreateAccount(conf *cloudprovider.SDBInstanceAccountCreateConfig) error {
	return rds.region.CreateDBInstanceAccount(rds.Id, conf.Name, conf.Password)
}

func (region *SRegion) CreateDBInstanceAccount(instanceId, account, password string) error {
	params := map[string]interface{}{
		"name":     account,
		"password": password,
	}
	_, err := region.post(SERVICE_RDS, fmt.Sprintf("instances/%s/db_user", instanceId), params)
	return err
}

func (rds *SDBInstance) CreateDatabase(conf *cloudprovider.SDBInstanceDatabaseCreateConfig) error {
	return rds.region.CreateDBInstanceDatabase(rds.Id, conf.Name, conf.CharacterSet)
}

func (region *SRegion) CreateDBInstanceDatabase(instanceId, database, characterSet string) error {
	params := map[string]interface{}{
		"name":          database,
		"character_set": characterSet,
	}
	_, err := region.post(SERVICE_RDS, fmt.Sprintf("instances/%s/database", instanceId), params)
	return err
}

func (rds *SDBInstance) ChangeConfig(cxt context.Context, desc *cloudprovider.SManagedDBInstanceChangeConfig) error {
	return rds.region.ChangeDBInstanceConfig(rds.Id, desc.InstanceType, desc.DiskSizeGB)
}

func (region *SRegion) ChangeDBInstanceConfig(instanceId string, instanceType string, diskSizeGb int) error {
	instance, err := region.GetIDBInstanceById(instanceId)
	if err != nil {
		return errors.Wrapf(err, "region.GetIDBInstanceById(%s)", instanceId)
	}

	if len(instanceType) > 0 {
		params := map[string]interface{}{
			"resize_flavor": map[string]string{
				"spec_code": instanceType,
			},
		}
		_, err := region.post(SERVICE_RDS, fmt.Sprintf("instances/%s/action", instanceId), params)
		if err != nil {
			return errors.Wrap(err, "resize_flavor")
		}
		cloudprovider.WaitStatus(instance, api.DBINSTANCE_RUNNING, time.Second*5, time.Minute*30)
	}
	if diskSizeGb > 0 {
		params := map[string]interface{}{
			"enlarge_volume": map[string]int{
				"size": diskSizeGb,
			},
		}
		_, err := region.post(SERVICE_RDS, fmt.Sprintf("instances/%s/action", instanceId), params)
		if err != nil {
			return errors.Wrap(err, "enlarge_volume")
		}
		cloudprovider.WaitStatus(instance, api.DBINSTANCE_RUNNING, time.Second*5, time.Minute*30)
	}
	return nil
}

func (self *SDBInstance) SetTags(tags map[string]string, replace bool) error {
	existedTags, err := self.GetTags()
	if err != nil {
		return errors.Wrap(err, "self.GetTags()")
	}
	deleteTagsKey := []string{}
	for k := range existedTags {
		if replace {
			deleteTagsKey = append(deleteTagsKey, k)
		} else {
			if _, ok := tags[k]; ok {
				deleteTagsKey = append(deleteTagsKey, k)
			}
		}
	}
	if len(deleteTagsKey) > 0 {
		err := self.region.DeleteRdsTags(self.GetId(), deleteTagsKey)
		if err != nil {
			return errors.Wrapf(err, "DeleteRdsTags")
		}
	}
	if len(tags) > 0 {
		err := self.region.CreateRdsTags(self.GetId(), tags)
		if err != nil {
			return errors.Wrapf(err, "CreateRdsTags")
		}
	}
	return nil
}

func (self *SRegion) DeleteRdsTags(instanceId string, tagsKey []string) error {
	params := map[string]interface{}{
		"action": "delete",
	}
	tagsObj := []map[string]string{}
	for _, k := range tagsKey {
		tagsObj = append(tagsObj, map[string]string{"key": k})
	}
	params["tags"] = tagsObj
	_, err := self.post(SERVICE_RDS, fmt.Sprintf("instances/%s/tags/action", instanceId), params)
	return err
}

func (self *SRegion) CreateRdsTags(instanceId string, tags map[string]string) error {
	params := map[string]interface{}{
		"action": "create",
	}

	tagsObj := []map[string]string{}
	for k, v := range tags {
		tagsObj = append(tagsObj, map[string]string{"key": k, "value": v})
	}
	params["tags"] = tagsObj
	_, err := self.post(SERVICE_RDS, fmt.Sprintf("instances/%s/tags/action", instanceId), params)
	return err
}

func (rds *SDBInstance) RecoveryFromBackup(conf *cloudprovider.SDBInstanceRecoveryConfig) error {
	if len(conf.OriginDBInstanceExternalId) == 0 {
		conf.OriginDBInstanceExternalId = rds.Id
	}
	return rds.region.RecoveryDBInstanceFromBackup(rds.Id, conf.OriginDBInstanceExternalId, conf.BackupId, conf.Databases)
}

func (region *SRegion) RecoveryDBInstanceFromBackup(target, origin string, backupId string, databases map[string]string) error {
	source := map[string]interface{}{
		"type":      "backup",
		"backup_id": backupId,
	}
	if len(origin) > 0 {
		source["instance_id"] = origin
	}
	if len(databases) > 0 {
		source["database_name"] = databases
	}
	params := map[string]interface{}{
		"source": source,
		"target": map[string]string{
			"instance_id": target,
		},
	}
	_, err := region.post(SERVICE_RDS, "instances/recovery", params)
	if err != nil {
		return errors.Wrap(err, "dbinstance.recovery")
	}
	return nil
}

func (rds *SDBInstance) Renew(bc billing.SBillingCycle) error {
	return rds.region.RenewInstance(rds.Id, bc)
}

func (rds *SDBInstance) Update(ctx context.Context, input cloudprovider.SDBInstanceUpdateOptions) error {
	return rds.region.Update(rds.Id, input)
}

func (region *SRegion) Update(instanceId string, input cloudprovider.SDBInstanceUpdateOptions) error {
	if len(input.NAME) > 0 {
		err := region.ModifyDBInstanceName(instanceId, input.NAME)
		if err != nil {
			return errors.Wrap(err, "update dbinstance name")
		}
	}
	err := region.ModifyDBInstanceDesc(instanceId, input.Description)
	if err != nil {
		return errors.Wrap(err, "update dbinstance name")
	}
	return nil
}

func (region *SRegion) ModifyDBInstanceName(instanceId string, name string) error {
	params := map[string]interface{}{
		"name": name,
	}
	resource := fmt.Sprintf("instances/%s/name", instanceId)
	_, err := region.put(SERVICE_RDS, resource, params)
	return err
}

func (region *SRegion) ModifyDBInstanceDesc(instanceId string, desc string) error {
	params := map[string]interface{}{
		"alias": desc,
	}
	resource := fmt.Sprintf("instances/%s/alias", instanceId)
	_, err := region.put(SERVICE_RDS, resource, params)
	return err
}
