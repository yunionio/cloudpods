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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	billing "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
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
	instance := SDBInstance{}
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
	return billing.BILLING_TYPE_POSTPAID
}

func (rds *SDBInstance) GetExpiredAt() time.Time {
	return time.Time{}
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
		return api.DBINSTANCE_CATEGORY_BASIC
	case "Ha":
		return api.DBINSTANCE_CATEGORY_HA
	case "Replica":
		return api.DBINSTANCE_CATEGORY_Replica
	}
	return rds.Type
}

func (rds *SDBInstance) GetVcpuCount() int {
	return 0
}

func (rds *SDBInstance) GetVmemSizeMB() int {
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
	if len(rds.Nodes) == 1 {
		zone, err := rds.region.getZoneById(rds.Nodes[0].AvailabilityZone)
		if err == nil {
			return zone.GetGlobalId()
		}
	}
	return ""
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

	iaccounts := []cloudprovider.ICloudDBInstanceAccount{}
	for i := 0; i < len(accounts); i++ {
		accounts[i].instance = rds
		iaccounts = append(iaccounts, &accounts[i])
	}
	return iaccounts, nil
}
