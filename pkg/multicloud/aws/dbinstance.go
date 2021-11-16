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

package aws

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	billing "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SDBParameterGroup struct {
	DBParameterGroupName string `xml:"DBParameterGroupName"`
	ParameterApplyStatus string `xml:"ParameterApplyStatus"`
}

type SOptionGroupMembership struct {
	OptionGroupName string `xml:"OptionGroupName"`
	Status          string `xml:"Status"`
}

type SEndpoint struct {
	HostedZoneId string `xml:"HostedZoneId"`
	Address      string `xml:"Address"`
	Port         int    `xml:"Port"`
}

type SSubnetAvailabilityZone struct {
	Name string `xml:"Name"`
}

type SSubnet struct {
	SubnetIdentifier       string                  `xml:"SubnetIdentifier"`
	SubnetStatus           string                  `xml:"SubnetStatus"`
	SubnetAvailabilityZone SSubnetAvailabilityZone `xml:"SubnetAvailabilityZone"`
}

type SDBSubnetGroup struct {
	VpcId                    string    `xml:"VpcId"`
	Subnets                  []SSubnet `xml:"Subnets>Subnet"`
	SubnetGroupStatus        string    `xml:"SubnetGroupStatus"`
	DBSubnetGroupDescription string    `xml:"DBSubnetGroupDescription"`
	DBSubnetGroupName        string    `xml:"DBSubnetGroupName"`
}

type SVpcSecurityGroupMembership struct {
	VpcSecurityGroupId string `xml:"VpcSecurityGroupId"`
	Status             string `xml:"Status"`
}

type SVpcSecurityGroups struct {
	VpcSecurityGroupMembership SVpcSecurityGroupMembership `xml:"VpcSecurityGroupMembership"`
}

type SDBInstance struct {
	multicloud.SDBInstanceBase
	multicloud.AwsTags

	region *SRegion

	AllocatedStorage int `xml:"AllocatedStorage"`
	//AssociatedRoles     string             `xml:"AssociatedRoles"`
	DBParameterGroups   []SDBParameterGroup `xml:"DBParameterGroups>DBParameterGroup"`
	AvailabilityZone    string              `xml:"AvailabilityZone"`
	DBSecurityGroups    string              `xml:"DBSecurityGroups"`
	EngineVersion       string              `xml:"EngineVersion"`
	MasterUsername      string              `xml:"MasterUsername"`
	InstanceCreateTime  time.Time           `xml:"InstanceCreateTime"`
	DBInstanceClass     string              `xml:"DBInstanceClass"`
	HttpEndpointEnabled bool                `xml:"HttpEndpointEnabled"`
	//ReadReplicaDBInstanceIdentifiers string             `xml:"ReadReplicaDBInstanceIdentifiers"`
	MonitoringInterval               int                      `xml:"MonitoringInterval"`
	DBInstanceStatus                 string                   `xml:"DBInstanceStatus"`
	BackupRetentionPeriod            int                      `xml:"BackupRetentionPeriod"`
	OptionGroupMemberships           []SOptionGroupMembership `xml:"OptionGroupMemberships>OptionGroupMembership"`
	CACertificateIdentifier          string                   `xml:"CACertificateIdentifier"`
	DbInstancePort                   int                      `xml:"DbInstancePort"`
	DbiResourceId                    string                   `xml:"DbiResourceId"`
	PreferredBackupWindow            string                   `xml:"PreferredBackupWindow"`
	DeletionProtection               bool                     `xml:"DeletionProtection"`
	DBInstanceIdentifier             string                   `xml:"DBInstanceIdentifier"`
	DBInstanceArn                    string                   `xml:"DBInstanceArn"`
	Endpoint                         SEndpoint                `xml:"Endpoint"`
	Engine                           string                   `xml:"Engine"`
	PubliclyAccessible               bool                     `xml:"PubliclyAccessible"`
	IAMDatabaseAuthenticationEnabled bool                     `xml:"IAMDatabaseAuthenticationEnabled"`
	PerformanceInsightsEnabled       bool                     `xml:"PerformanceInsightsEnabled"`
	DBName                           string                   `xml:"DBName"`
	MultiAZ                          bool                     `xml:"MultiAZ"`
	//DomainMemberships                string                  `xml:"DomainMemberships"`
	StorageEncrypted           bool               `xml:"StorageEncrypted"`
	DBSubnetGroup              SDBSubnetGroup     `xml:"DBSubnetGroup"`
	VpcSecurityGroups          SVpcSecurityGroups `xml:"VpcSecurityGroups"`
	LicenseModel               string             `xml:"LicenseModel"`
	PreferredMaintenanceWindow string             `xml:"PreferredMaintenanceWindow"`
	StorageType                string             `xml:"StorageType"`
	AutoMinorVersionUpgrade    bool               `xml:"AutoMinorVersionUpgrade"`
	CopyTagsToSnapshot         bool               `xml:"CopyTagsToSnapshot"`
}

type SDBInstances struct {
	DBInstances []SDBInstance `xml:"DBInstances>DBInstance"`
	Marker      string        `xml:"Marker"`
}

func (rds *SDBInstance) GetName() string {
	return rds.DBInstanceIdentifier
}

func (rds *SDBInstance) GetId() string {
	return rds.DbiResourceId
}

func (rds *SDBInstance) GetGlobalId() string {
	return rds.GetId()
}

func (rds *SDBInstance) GetStatus() string {
	switch rds.DBInstanceStatus {
	case "creating":
		return api.DBINSTANCE_DEPLOYING
	case "available":
		return api.DBINSTANCE_RUNNING
	case "deleting":
		return api.DBINSTANCE_DELETING
	default:
		log.Errorf("Unknown db instance status: %s", rds.DBInstanceStatus)
		return api.DBINSTANCE_UNKNOWN
	}
}

func (rds *SDBInstance) GetBillingType() string {
	return billing.BILLING_TYPE_POSTPAID
}

func (rds *SDBInstance) GetExpiredAt() time.Time {
	return time.Time{}
}

func (rds *SDBInstance) GetCreatedAt() time.Time {
	return rds.InstanceCreateTime
}

func (rds *SDBInstance) GetStorageType() string {
	return rds.StorageType
}

func (rds *SDBInstance) GetEngine() string {
	return rds.Engine
}

func (rds *SDBInstance) GetEngineVersion() string {
	return rds.EngineVersion
}

func (rds *SDBInstance) GetInstanceType() string {
	return rds.DBInstanceClass
}

func (rds *SDBInstance) GetVcpuCount() int {
	if spec, ok := DBInstanceSpecs[rds.DBInstanceClass]; ok {
		return spec.VcpuCount
	}
	return 0
}

func (rds *SDBInstance) GetVmemSizeMB() int {
	if spec, ok := DBInstanceSpecs[rds.DBInstanceClass]; ok {
		return spec.VmemSizeMb
	}
	return 0
}

func (rds *SDBInstance) GetDiskSizeGB() int {
	return rds.AllocatedStorage
}

func (rds *SDBInstance) GetPort() int {
	return rds.Endpoint.Port
}

func (rds *SDBInstance) GetMaintainTime() string {
	return rds.PreferredMaintenanceWindow
}

func (rds *SDBInstance) GetIVpcId() string {
	return rds.DBSubnetGroup.VpcId
}

func (rds *SDBInstance) Refresh() error {
	instance, err := rds.region.GetDBInstance(rds.DbiResourceId)
	if err != nil {
		return err
	}
	return jsonutils.Update(rds, instance)
}

func (rds *SDBInstance) GetZone1Id() string {
	if len(rds.AvailabilityZone) > 0 {
		zone, err := rds.region.getZoneById(rds.AvailabilityZone)
		if err != nil {
			log.Errorf("rds.GetIZoneId %s error: %v", rds.DBInstanceIdentifier, err)
			return ""
		}
		return zone.GetGlobalId()
	}
	return ""
}

func (rds *SDBInstance) GetZone2Id() string {
	return ""
}

func (rds *SDBInstance) GetZone3Id() string {
	return ""
}

func (rds *SDBInstance) GetIDBInstanceAccounts() ([]cloudprovider.ICloudDBInstanceAccount, error) {
	accounts := []cloudprovider.ICloudDBInstanceAccount{}
	if len(rds.MasterUsername) > 0 {
		account := &SDBInstanceAccount{instance: rds, AccountName: rds.MasterUsername}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func (rds *SDBInstance) GetDBNetworks() ([]cloudprovider.SDBInstanceNetwork, error) {
	return []cloudprovider.SDBInstanceNetwork{}, nil
}

func (rds *SDBInstance) GetInternalConnectionStr() string {
	return rds.Endpoint.Address
}

func (rds *SDBInstance) GetConnectionStr() string {
	if rds.PubliclyAccessible {
		return rds.Endpoint.Address
	}
	return ""
}

func (rds *SDBInstance) GetIDBInstanceParameters() ([]cloudprovider.ICloudDBInstanceParameter, error) {
	parameters, err := rds.region.GetDBInstanceParameters(rds.DBParameterGroups[0].DBParameterGroupName)
	if err != nil {
		return nil, errors.Wrap(err, "GetDBInstanceParameters")
	}
	iparams := []cloudprovider.ICloudDBInstanceParameter{}
	for i := 0; i < len(parameters); i++ {
		parameters[i].instance = rds
		iparams = append(iparams, &parameters[i])
	}
	return iparams, nil
}

func (rds *SDBInstance) GetIDBInstanceDatabases() ([]cloudprovider.ICloudDBInstanceDatabase, error) {
	idatabases := []cloudprovider.ICloudDBInstanceDatabase{}
	if len(rds.DBName) > 0 {
		database := &SDBInstanceDatabase{DBName: rds.DBName}
		idatabases = append(idatabases, database)
	}
	return idatabases, nil
}

func (region *SRegion) GetDBInstances(instanceId string) ([]SDBInstance, error) {
	params := map[string]string{}
	idx := 1
	if len(instanceId) > 0 {
		params[fmt.Sprintf("Filters.Filter.%d.Name", idx)] = "dbi-resource-id"
		params[fmt.Sprintf("Filters.Filter.%d.Values.Value.1", idx)] = instanceId
	}
	ret := []SDBInstance{}
	for {
		result := SDBInstances{}
		err := region.rdsRequest("DescribeDBInstances", params, &result)
		if err != nil {
			return nil, errors.Wrap(err, "DescribeDBInstances")
		}
		ret = append(ret, result.DBInstances...)
		if len(result.Marker) == 0 || len(result.DBInstances) == 0 {
			break
		}
		params["Marker"] = result.Marker
	}
	return ret, nil
}

func (region *SRegion) GetIDBInstances() ([]cloudprovider.ICloudDBInstance, error) {
	instances, err := region.GetDBInstances("")
	if err != nil {
		return nil, errors.Wrap(err, "GetDBInstances")
	}
	idbinstances := []cloudprovider.ICloudDBInstance{}
	for i := 0; i < len(instances); i++ {
		instances[i].region = region
		idbinstances = append(idbinstances, &instances[i])
	}
	return idbinstances, nil
}

func (self *SRegion) GetDBInstance(id string) (*SDBInstance, error) {
	instances, err := self.GetDBInstances(id)
	if err != nil {
		return nil, errors.Wrap(err, "GetDBInstances")
	}

	for i := range instances {
		if instances[i].GetGlobalId() == id {
			instances[i].region = self
			return &instances[i], nil
		}
	}

	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) GetIDBInstanceById(id string) (cloudprovider.ICloudDBInstance, error) {
	return self.GetDBInstance(id)
}
