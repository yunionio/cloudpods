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

package azure

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDBInstanceVirtualNetworkRule struct {
	ID         string                                  `json:"id"`
	Name       string                                  `json:"name"`
	Type       string                                  `json:"type"`
	Properties SDBInstanceVirtualNetworkRuleProperties `json:"properties"`
}

type SDBInstanceVirtualNetworkRuleProperties struct {
	IgnoreMissingVnetServiceEndpoint bool   `json:"ignoreMissingVnetServiceEndpoint"`
	State                            string `json:"state"`
	VirtualNetworkSubnetID           string `json:"virtualNetworkSubnetId"`
}

type SDBInstanceSku struct {
	Name     string `json:"name"`
	Tier     string `json:"tier"`
	Family   string `json:"family"`
	Capacity int    `json:"capacity"`
}

type SDBInstanceStorageProfile struct {
	StorageMB           int    `json:"storageMB"`
	BackupRetentionDays int    `json:"backupRetentionDays"`
	StorageIops         int    `json:"storageIops"`
	GeoRedundantBackup  string `json:"geoRedundantBackup"`
}

type SDBInstanceDelegatedSubnetArguments struct {
	SubnetArmResourceId string `json:"subnetArmResourceId"`
}

type SDBInstanceProperties struct {
	AdministratorLogin       string                    `json:"administratorLogin"`
	StorageProfile           SDBInstanceStorageProfile `json:"storageProfile"`
	Version                  string                    `json:"version"`
	SslEnforcement           string                    `json:"sslEnforcement"`
	UserVisibleState         string                    `json:"userVisibleState"`
	FullyQualifiedDomainName string                    `json:"fullyQualifiedDomainName"`
	EarliestRestoreDate      time.Time                 `json:"earliestRestoreDate"`

	ReplicationRole          string                              `json:"replicationRole"`
	MasterServerId           string                              `json:"masterServerId"`
	ReplicaCapacity          int                                 `json:"replicaCapacity"`
	DelegatedSubnetArguments SDBInstanceDelegatedSubnetArguments `json:"delegatedSubnetArguments"`
}

type SDBInstance struct {
	region *SRegion
	multicloud.SDBInstanceBase
	AzureTags
	Sku        SDBInstanceSku        `json:"sku"`
	Properties SDBInstanceProperties `json:"properties"`
	Location   string                `json:"location"`
	ID         string                `json:"id"`
	Name       string                `json:"name"`
	Type       string                `json:"type"`
}

func (self *SRegion) GetIDBInstances() ([]cloudprovider.ICloudDBInstance, error) {
	instanceTypes := []string{
		"Microsoft.DBForMariaDB/servers",
		"Microsoft.DBforMySQL/servers",
		"Microsoft.DBforMySQL/flexibleServers",
		"Microsoft.DBforPostgreSQL/servers",
		"Microsoft.DBforPostgreSQL/flexibleServers",
	}
	DBInstances := []SDBInstance{}
	for i := range instanceTypes {
		instances, err := self.ListDBInstance(instanceTypes[i])
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotFound {
				return nil, errors.Wrap(err, "self.ListDBInstance()")
			}
		}
		DBInstances = append(DBInstances, instances...)
	}
	result := []cloudprovider.ICloudDBInstance{}
	for i := range DBInstances {
		DBInstances[i].region = self
		result = append(result, &DBInstances[i])
	}
	sqlServers, err := self.ListSQLServer()
	if err != nil {
		return nil, errors.Wrapf(err, "ListSQLServer")
	}
	for i := range sqlServers {
		sqlServers[i].region = self
		result = append(result, &sqlServers[i])
	}
	managedSQLServers, err := self.ListManagedSQLServer()
	if err != nil {
		return nil, errors.Wrapf(err, "ListManagedSQLServer")
	}
	for i := range managedSQLServers {
		managedSQLServers[i].region = self
		result = append(result, &managedSQLServers[i])
	}
	return result, nil
}

func (self *SRegion) GetIDBInstanceById(instanceId string) (cloudprovider.ICloudDBInstance, error) {
	if strings.Index(strings.ToLower(instanceId), "microsoft.sql/servers") > 0 {
		return self.GetSQLServer(instanceId)
	}
	if strings.Index(strings.ToLower(instanceId), "microsoft.sql/managedinstances") > 0 {
		return self.GetManagedSQLServer(instanceId)
	}
	rds, err := self.GetDBInstanceById(instanceId)
	if err != nil {
		return nil, errors.Wrapf(err, "self.get(%s, url.Values{}, &newrds)", instanceId)
	}
	return rds, nil
}

func (self *SRegion) GetDBInstanceById(instanceId string) (*SDBInstance, error) {
	rds := SDBInstance{}
	err := self.get(instanceId, url.Values{}, &rds)
	if err != nil {
		return nil, errors.Wrapf(err, "self.get(%s, url.Values{}, &newrds)", instanceId)
	}
	rds.region = self
	return &rds, nil
}

func (self *SRegion) ListDBInstance(instanceType string) ([]SDBInstance, error) {
	result := []SDBInstance{}
	err := self.list(instanceType, url.Values{}, &result)
	if err != nil {
		return nil, errors.Wrapf(err, "list(%s)", instanceType)
	}
	for i := range result {
		result[i].region = self
	}
	return result, nil
}

func (self *SRegion) ListDBInstanceReplica(Id string) ([]SDBInstance, error) {
	type replicas struct {
		Value []SDBInstance
	}
	result := replicas{}
	err := self.get(fmt.Sprintf("%s/replicas", Id), url.Values{}, &result)
	if err != nil {
		return nil, errors.Wrapf(err, "get(%s)", Id)
	}
	for i := range result.Value {
		result.Value[i].region = self
	}
	return result.Value, nil
}

func (self *SRegion) ListDBInstanceVirtualNetworkRule(Id string) ([]SDBInstanceVirtualNetworkRule, error) {
	type networksRules struct {
		Value []SDBInstanceVirtualNetworkRule
	}
	result := networksRules{}
	err := self.get(fmt.Sprintf("%s/virtualNetworkRules", Id), url.Values{}, &result)
	if err != nil {
		return nil, errors.Wrapf(err, "get(%s)", Id)
	}
	return result.Value, nil
}

func (rds *SDBInstance) GetName() string {
	return rds.Name
}

func (rds *SDBInstance) GetId() string {
	return strings.ToLower(rds.ID)
}

func (rds *SDBInstance) GetGlobalId() string {
	return rds.GetId()
}

func (rds *SDBInstance) GetStatus() string {
	return api.DBINSTANCE_RUNNING
}

func (self *SDBInstance) GetProjectId() string {
	return getResourceGroup(self.ID)
}

func (rds *SDBInstance) GetExpiredAt() time.Time {
	return time.Time{}
}

func (rds *SDBInstance) GetStorageType() string {
	switch rds.Sku.Tier {
	case "Basic":
		return api.STORAGE_AZURE_BASIC
	case "General Purpose":
		return api.STORAGE_AZURE_GENERAL_PURPOSE
	case "Memory Optimized":
		return api.STORAGE_AZURE_GENERAL_PURPOSE
	default:
		return api.STORAGE_AZURE_BASIC
	}
}

func (rds *SDBInstance) Refresh() error {
	newrds := SDBInstance{}
	err := rds.region.get(rds.ID, url.Values{}, &newrds)
	if err != nil {
		return errors.Wrapf(err, "rds.region.get(%s, url.Values{}, &newdb)", rds.ID)
	}

	err = jsonutils.Update(rds, newrds)
	if err != nil {
		return err
	}
	rds.Tags = newrds.Tags
	return nil
}

func (rds *SDBInstance) GetMasterInstanceId() string {
	if len(rds.Properties.MasterServerId) > 0 {
		return strings.ToLower(rds.Properties.MasterServerId)
	}
	return ""
}

func (rds *SDBInstance) GetPort() int {
	switch rds.GetEngine() {
	case api.DBINSTANCE_TYPE_POSTGRESQL:
		return 5432
	case api.DBINSTANCE_TYPE_MYSQL:
		return 3306
	case api.DBINSTANCE_TYPE_MARIADB:
		return 3306
	default:
		return 0
	}
}

func (rds *SDBInstance) GetEngine() string {
	databaseType := strings.Split(rds.Type, "/")
	switch databaseType[0] {
	case "Microsoft.DBforPostgreSQL":
		return api.DBINSTANCE_TYPE_POSTGRESQL
	case "Microsoft.DBforMySQL":
		return api.DBINSTANCE_TYPE_MYSQL
	case "Microsoft.DBforMariaDB":
		return api.DBINSTANCE_TYPE_MARIADB
	default:
		return ""
	}
}

func (rds *SDBInstance) GetEngineVersion() string {
	return rds.Properties.Version
}

func (rds *SDBInstance) GetInstanceType() string {
	return rds.Sku.Name
}

func (rds *SDBInstance) GetVcpuCount() int {
	return rds.Sku.Capacity
}

func (rds *SDBInstance) GetVmemSizeMB() int {
	if strings.Contains(rds.Type, "server") {
		switch rds.Sku.Tier {
		case "Basic":
			return rds.Sku.Capacity * 2 * 1024
		case "General Purpose":
			return rds.Sku.Capacity * 5 * 1024
		case "GeneralPurpose":
			return int(float32(rds.Sku.Capacity) * 5.2 * 1024)
		case "Memory Optimized":
			return rds.Sku.Capacity * 10 * 1024
		default:
			return 0
		}
	}
	return 0
}

func (rds *SDBInstance) GetDiskSizeGB() int {
	return rds.Properties.StorageProfile.StorageMB / 1024
}

func (rds *SDBInstance) GetCategory() string {
	return rds.Sku.Tier
}

func (rds *SDBInstance) GetMaintainTime() string {
	return ""
}

func (rds *SDBInstance) GetConnectionStr() string {
	return rds.Properties.FullyQualifiedDomainName
}

// func (rds *SDBInstance) GetInternalConnectionStr() string

func (rds *SDBInstance) GetIVpcId() string {
	splited := strings.Split(rds.Properties.DelegatedSubnetArguments.SubnetArmResourceId, "/subnets")
	return splited[0]
}

func (rds *SDBInstance) GetDBNetworks() ([]cloudprovider.SDBInstanceNetwork, error) {
	result := []cloudprovider.SDBInstanceNetwork{}
	delegateNet := cloudprovider.SDBInstanceNetwork{NetworkId: rds.Properties.DelegatedSubnetArguments.SubnetArmResourceId}
	result = append(result, delegateNet)
	return result, nil
}

func (rds *SDBInstance) GetZone1Id() string {
	return ""
}

func (rds *SDBInstance) GetZone2Id() string {
	return ""
}

func (rds *SDBInstance) GetZone3Id() string {
	return ""
}

func (rds *SDBInstance) GetIDBInstanceParameters() ([]cloudprovider.ICloudDBInstanceParameter, error) {
	configs, err := rds.region.ListDBInstanceConfiguration(rds.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "rds.region.ListDBInstanceConfiguration(%s)", rds.ID)
	}
	result := []cloudprovider.ICloudDBInstanceParameter{}
	for i := range configs {
		result = append(result, &configs[i])
	}
	return result, nil
}

func (rds *SDBInstance) GetIDBInstanceDatabases() ([]cloudprovider.ICloudDBInstanceDatabase, error) {
	db, err := rds.region.ListSDBInstanceDatabase(rds.ID)
	if err != nil {
		return nil, errors.Wrapf(err, "rds.region.ListSDBInstanceDatabase(%s)", rds.ID)
	}
	result := []cloudprovider.ICloudDBInstanceDatabase{}
	for i := range db {
		result = append(result, &db[i])
	}
	return result, nil
}

func (rds *SDBInstance) GetIDBInstanceAccounts() ([]cloudprovider.ICloudDBInstanceAccount, error) {
	accounts := []cloudprovider.ICloudDBInstanceAccount{}
	if len(rds.Properties.AdministratorLogin) > 0 {
		account := &SDBInstanceAccount{instance: rds, AccountName: rds.Properties.AdministratorLogin}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func (rds *SDBInstance) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	return []cloudprovider.ICloudDBInstanceBackup{}, nil
}

func (rds *SDBInstance) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDBInstance) SetTags(tags map[string]string, replace bool) error {
	if !replace {
		for k, v := range self.Tags {
			if _, ok := tags[k]; !ok {
				tags[k] = v
			}
		}
	}
	_, err := self.region.client.SetTags(self.ID, tags)
	if err != nil {
		return errors.Wrapf(err, "self.region.client.SetTags(%s,%s)", self.ID, jsonutils.Marshal(tags).String())
	}
	return nil
}
