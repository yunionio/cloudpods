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
	"context"
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	billing "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
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
	AwsTags

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

// https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/accessing-monitoring.html#Overview.DBInstance.Status
func (rds *SDBInstance) GetStatus() string {
	switch rds.DBInstanceStatus {
	case "creating", "backing-up":
		return api.DBINSTANCE_DEPLOYING
	case "available":
		return api.DBINSTANCE_RUNNING
	case "deleting":
		return api.DBINSTANCE_DELETING
	case "rebooting":
		return api.DBINSTANCE_REBOOTING
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

func (rds *SDBInstance) Reboot() error {
	return rds.region.RebootDBInstance(rds.DBInstanceIdentifier)
}

func (self *SDBInstance) GetCategory() string {
	switch self.Engine {
	case "aurora", "aurora-mysql":
		return api.DBINSTANCE_TYPE_MYSQL
	case "aurora-postgresql":
		return api.DBINSTANCE_TYPE_POSTGRESQL
	case "oracle-ee", "sqlserver-ee":
		return api.AWS_DBINSTANCE_CATEGORY_ENTERPRISE_EDITION
	case "oracle-se2":
		return api.AWS_DBINSTANCE_CATEGORY_STANDARD_EDITION_TWO
	case "sqlserver-se":
		return api.AWS_DBINSTANCE_CATEGORY_STANDARD_EDITION
	case "sqlserver-ex":
		return api.AWS_DBINSTANCE_CATEGORY_EXPRESS_EDITION
	case "sqlserver-web":
		return api.AWS_DBINSTANCE_CATEGORY_WEB_EDITION
	default:
		if strings.HasPrefix(self.DBInstanceClass, "db.r") || strings.HasPrefix(self.DBInstanceClass, "db.x") || strings.HasPrefix(self.DBInstanceClass, "db.d") {
			return api.AWS_DBINSTANCE_CATEGORY_MEMORY_OPTIMIZED
		}
		return api.AWS_DBINSTANCE_CATEGORY_GENERAL_PURPOSE
	}
}

func (rds *SDBInstance) GetStorageType() string {
	return rds.StorageType
}

func (rds *SDBInstance) GetEngine() string {
	if strings.Contains(rds.Engine, "aurora") {
		return api.DBINSTANCE_TYPE_AURORA
	}
	if strings.Contains(rds.Engine, "oracle") {
		return api.DBINSTANCE_TYPE_ORACLE
	}
	if strings.Contains(rds.Engine, "sqlserver") {
		return api.DBINSTANCE_TYPE_SQLSERVER
	}
	for k, v := range map[string]string{
		"mariadb":  api.DBINSTANCE_TYPE_MARIADB,
		"mysql":    api.DBINSTANCE_TYPE_MYSQL,
		"postgres": api.DBINSTANCE_TYPE_POSTGRESQL,
	} {
		if rds.Engine == k {
			return v
		}
	}
	return rds.Engine
}

func (rds *SDBInstance) GetEngineVersion() string {
	return rds.EngineVersion
}

func (rds *SDBInstance) GetInstanceType() string {
	return rds.DBInstanceClass
}

func (rds *SDBInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedDBInstanceChangeConfig) error {
	params := map[string]string{
		"DBInstanceIdentifier": rds.DBInstanceIdentifier,
		"ApplyImmediately":     "true",
	}
	if config.DiskSizeGB > 0 && rds.GetEngine() != api.DBINSTANCE_TYPE_AURORA {
		params["AllocatedStorage"] = fmt.Sprintf("%d", config.DiskSizeGB)
	}
	if len(config.InstanceType) > 0 {
		params["DBInstanceClass"] = config.InstanceType
	}
	return rds.region.rdsRequest("ModifyDBInstance", params, nil)
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

func (rds *SDBInstance) GetDescription() string {
	return rds.AwsTags.GetDescription()
}

func (rds *SDBInstance) Update(ctx context.Context, input cloudprovider.SDBInstanceUpdateOptions) error {
	return rds.SetTags(map[string]string{"Description": input.Description}, false)
}

func (region *SRegion) Update(instanceId string, input cloudprovider.SDBInstanceUpdateOptions) error {
	dbinstance, err := region.GetDBInstance(instanceId)
	if err != nil {
		return errors.Wrap(err, "GetDBInstance")
	}
	return dbinstance.SetTags(map[string]string{"Description": input.Description}, false)
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
	rds.AwsTags = instance.AwsTags
	return jsonutils.Update(rds, instance)
}

func (region *SRegion) GetDBInstance(instanceId string) (*SDBInstance, error) {
	instances, _, err := region.GetDBInstances(instanceId, "")
	if err != nil {
		return nil, errors.Wrap(err, "GetDBInstances")
	}

	if len(instances) == 1 {
		if instances[0].DbiResourceId == instanceId {
			instances[0].region = region
			return &instances[0], nil
		}
		return nil, cloudprovider.ErrNotFound
	}

	if len(instances) == 0 {
		return nil, cloudprovider.ErrNotFound
	}

	return nil, cloudprovider.ErrDuplicateId
}

func (rds *SDBInstance) GetZone1Id() string {
	return rds.AvailabilityZone
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

func (rds *SDBInstance) OpenPublicConnection() error {
	params := map[string]string{
		"DBInstanceIdentifier": rds.DBInstanceIdentifier,
		"PubliclyAccessible":   "true",
	}
	return rds.region.rdsRequest("ModifyDBInstance", params, nil)
}

func (rds *SDBInstance) ClosePublicConnection() error {
	params := map[string]string{
		"DBInstanceIdentifier": rds.DBInstanceIdentifier,
		"PubliclyAccessible":   "false",
	}
	return rds.region.rdsRequest("ModifyDBInstance", params, nil)
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

func (rds *SDBInstance) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	backups, err := rds.region.GetDBInstanceSnapshots(rds.DBInstanceIdentifier, "")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudDBInstanceBackup{}
	for i := range backups {
		backups[i].region = rds.region
		ret = append(ret, &backups[i])
	}
	return ret, nil
}

func (rds *SDBInstance) CreateIBackup(conf *cloudprovider.SDBInstanceBackupCreateConfig) (string, error) {
	params := map[string]string{
		"DBInstanceIdentifier": rds.DBInstanceIdentifier,
		"DBSnapshotIdentifier": conf.Name,
	}
	ret := struct {
		DBSnapshot SDBInstanceSnapshot `xml:"DBSnapshot"`
	}{}
	err := rds.region.rdsRequest("CreateDBSnapshot", params, &ret)
	if err != nil {
		return "", err
	}
	ret.DBSnapshot.region = rds.region
	cloudprovider.WaitStatus(&ret.DBSnapshot, api.DBINSTANCE_BACKUP_READY, time.Second*10, time.Hour*2)
	return ret.DBSnapshot.GetGlobalId(), nil
}

func (region *SRegion) GetDBInstances(instanceId, marker string) ([]SDBInstance, string, error) {
	instances := SDBInstances{}
	params := map[string]string{}
	idx := 1
	if len(instanceId) > 0 {
		params[fmt.Sprintf("Filters.Filter.%d.Name", idx)] = "dbi-resource-id"
		params[fmt.Sprintf("Filters.Filter.%d.Values.Value.1", idx)] = instanceId
	}

	if len(marker) > 0 {
		params["Marker"] = marker
	}

	err := region.rdsRequest("DescribeDBInstances", params, &instances)
	if err != nil {
		return nil, "", errors.Wrap(err, "DescribeDBInstances")
	}
	return instances.DBInstances, instances.Marker, nil
}

func (region *SRegion) GetIDBInstances() ([]cloudprovider.ICloudDBInstance, error) {
	idbinstances := []cloudprovider.ICloudDBInstance{}
	instances, marker, err := region.GetDBInstances("", "")
	if err != nil {
		return nil, errors.Wrap(err, "GetDBInstances")
	}
	for i := 0; i < len(instances); i++ {
		instances[i].region = region
		idbinstances = append(idbinstances, &instances[i])
	}
	for len(marker) > 0 {
		instances, marker, err = region.GetDBInstances("", marker)
		if err != nil {
			return nil, errors.Wrap(err, "GetDBInstances")
		}
		for i := 0; i < len(instances); i++ {
			instances[i].region = region
			idbinstances = append(idbinstances, &instances[i])
		}
	}
	return idbinstances, nil
}

func (self *SRegion) GetIDBInstanceById(id string) (cloudprovider.ICloudDBInstance, error) {
	instances, _, err := self.GetDBInstances(id, "")
	if err != nil {
		return nil, errors.Wrap(err, "GetDBInstances")
	}

	if len(instances) > 1 {
		return nil, errors.Wrapf(cloudprovider.ErrDuplicateId, id)
	}

	if len(instances) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
	}

	instances[0].region = self
	return &instances[0], nil
}

func (self *SRegion) CreateIDBInstance(desc *cloudprovider.SManagedDBInstanceCreateConfig) (cloudprovider.ICloudDBInstance, error) {
	params := map[string]string{
		"DBInstanceClass":      desc.InstanceType,
		"DBInstanceIdentifier": desc.Name,
		"EngineVersion":        desc.EngineVersion,
		"MasterUsername":       "admin",
	}
	if len(desc.Password) > 0 {
		params["MasterUserPassword"] = desc.Password
	}

	if desc.Engine != api.DBINSTANCE_TYPE_AURORA {
		params["StorageType"] = desc.StorageType
		params["AllocatedStorage"] = fmt.Sprintf("%d", desc.DiskSizeGB)
		for i, sec := range desc.SecgroupIds {
			params[fmt.Sprintf("VpcSecurityGroupIds.VpcSecurityGroupId.%d", i+1)] = sec
		}
	}
	if desc.MultiAz {
		params["MultiAZ"] = "true"
	}
	if desc.StorageType == api.STORAGE_IO1_SSD {
		params["Iops"] = "3000"
	}
	switch desc.Engine {
	case api.DBINSTANCE_TYPE_MYSQL:
		params["Engine"] = "mysql"
	case api.DBINSTANCE_TYPE_POSTGRESQL:
		params["Engine"] = "postgres"
		params["MasterUsername"] = "postgres"
	case api.DBINSTANCE_TYPE_MARIADB:
		params["Engine"] = "mariadb"
	case api.DBINSTANCE_TYPE_SQLSERVER:
		params["LicenseModel"] = "license-included"
		switch desc.Category {
		case api.AWS_DBINSTANCE_CATEGORY_ENTERPRISE_EDITION:
			params["Engine"] = "sqlserver-ee"
		case api.AWS_DBINSTANCE_CATEGORY_EXPRESS_EDITION:
			params["Engine"] = "sqlserver-ex"
			params["MultiAZ"] = "false"
		case api.AWS_DBINSTANCE_CATEGORY_STANDARD_EDITION:
			params["Engine"] = "sqlserver-se"
		case api.AWS_DBINSTANCE_CATEGORY_WEB_EDITION:
			params["Engine"] = "sqlserver-web"
			params["MultiAZ"] = "false"
		default:
			return nil, fmt.Errorf("invalid category %s for engine %s", desc.Category, desc.Engine)
		}
	case api.DBINSTANCE_TYPE_AURORA:
		delete(params, "MultiAZ")
		switch desc.Category {
		case api.DBINSTANCE_TYPE_MYSQL:
			params["Engine"] = "aurora"
			if !strings.HasPrefix(desc.EngineVersion, "5.6") {
				params["Engine"] = "aurora-mysql"
			}
		case api.DBINSTANCE_TYPE_POSTGRESQL:
			params["Engine"] = "aurora-postgresql"
			params["MasterUsername"] = "postgres"
		default:
			return nil, fmt.Errorf("invalid category %s for engine %s", desc.Category, desc.Engine)
		}
	case api.DBINSTANCE_TYPE_ORACLE:
		params["LicenseModel"] = "bring-your-own-license"
		switch desc.Category {
		case api.AWS_DBINSTANCE_CATEGORY_ENTERPRISE_EDITION:
			params["Engine"] = "oracle-ee"
		case api.AWS_DBINSTANCE_CATEGORY_STANDARD_EDITION_TWO:
			params["Engine"] = "oracle-se2"
		default:
			return nil, fmt.Errorf("invalid category %s for engine %s", desc.Category, desc.Engine)
		}
	}
	i := 1
	for k, v := range desc.Tags {
		params[fmt.Sprintf("Tags.Tag.%d.Key", i)] = k
		params[fmt.Sprintf("Tags.Tag.%d.Value", i)] = v
		i++
	}
	result := struct {
		DBInstance SDBInstance `xml:"DBInstance"`
	}{}
	result.DBInstance.region = self
	return &result.DBInstance, self.rdsRequest("CreateDBInstance", params, &result)
}

func (self *SDBInstance) Delete() error {
	params := map[string]string{
		"DBInstanceIdentifier": self.DBInstanceIdentifier,
		"SkipFinalSnapshot":    "true",
	}
	return self.region.rdsRequest("DeleteDBInstance", params, nil)
}

func (self *SRegion) RebootDBInstance(id string) error {
	params := map[string]string{
		"DBInstanceIdentifier": id,
	}
	return self.rdsRequest("RebootDBInstance", params, nil)
}

func (self *SDBInstance) SetTags(tags map[string]string, replace bool) error {
	oldTags, err := self.region.ListRdsResourceTags(self.DBInstanceArn)
	if err != nil {
		return errors.Wrapf(err, "ListRdsResourceTags")
	}
	added, removed := map[string]string{}, map[string]string{}
	for k, v := range tags {
		oldValue, ok := oldTags[k]
		if !ok {
			added[k] = v
		} else if oldValue != v {
			removed[k] = oldValue
			added[k] = v
		}
	}
	if replace {
		for k, v := range oldTags {
			newValue, ok := tags[k]
			if !ok {
				removed[k] = v
			} else if v != newValue {
				added[k] = newValue
				removed[k] = v
			}
		}
	}
	if len(removed) > 0 {
		err = self.region.RemoveRdsTagsFromResource(self.DBInstanceArn, removed)
		if err != nil {
			return errors.Wrapf(err, "RemoveRdsTagsFromResource %s", removed)
		}
	}
	if len(added) > 0 {
		return self.region.AddRdsTagsToResource(self.DBInstanceArn, added)
	}
	return nil
}

func (self *SRegion) ListRdsResourceTags(arn string) (map[string]string, error) {
	params := map[string]string{
		"ResourceName": arn,
	}
	tags := AwsTags{}
	err := self.rdsRequest("ListTagsForResource", params, &tags)
	if err != nil {
		return nil, errors.Wrapf(err, "ListTagsForResource")
	}
	return tags.GetTags()
}

func (self *SRegion) AddRdsTagsToResource(arn string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}
	params := map[string]string{
		"ResourceName": arn,
	}
	i := 1
	for k, v := range tags {
		params[fmt.Sprintf("Tags.member.%d.Key", i)] = k
		params[fmt.Sprintf("Tags.member.%d.Value", i)] = v
		i++
	}
	return self.rdsRequest("AddTagsToResource", params, nil)
}

func (self *SRegion) RemoveRdsTagsFromResource(arn string, tags map[string]string) error {
	if len(tags) == 0 {
		return nil
	}
	params := map[string]string{
		"ResourceName": arn,
	}
	i := 1
	for k := range tags {
		params[fmt.Sprintf("TagKeys.member.%d", i)] = k
		i++
	}
	return self.rdsRequest("RemoveTagsFromResource", params, nil)
}
