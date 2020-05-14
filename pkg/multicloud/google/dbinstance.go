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

package google

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/billing"
)

var (
	EngineVersions = map[string]GoogleSQLVersion{
		"MYSQL_5_5":                 GoogleSQLVersion{Engine: api.DBINSTANCE_TYPE_MYSQL, Version: "5.5"},
		"MYSQL_5_6":                 GoogleSQLVersion{Engine: api.DBINSTANCE_TYPE_MYSQL, Version: "5.6"},
		"MYSQL_5_7":                 GoogleSQLVersion{Engine: api.DBINSTANCE_TYPE_MYSQL, Version: "5.7"},
		"POSTGRES_9_6":              GoogleSQLVersion{Engine: api.DBINSTANCE_TYPE_POSTGRESQL, Version: "9.6"},
		"POSTGRES_10":               GoogleSQLVersion{Engine: api.DBINSTANCE_TYPE_POSTGRESQL, Version: "10"},
		"POSTGRES_11":               GoogleSQLVersion{Engine: api.DBINSTANCE_TYPE_POSTGRESQL, Version: "11"},
		"POSTGRES_12":               GoogleSQLVersion{Engine: api.DBINSTANCE_TYPE_POSTGRESQL, Version: "12"},
		"SQLSERVER_2017_STANDARD":   GoogleSQLVersion{Engine: api.DBINSTANCE_TYPE_SQLSERVER, Version: "2017 Standard"},
		"SQLSERVER_2017_ENTERPRISE": GoogleSQLVersion{Engine: api.DBINSTANCE_TYPE_SQLSERVER, Version: "2017 Enterprise"},
		"SQLSERVER_2017_EXPRESS":    GoogleSQLVersion{Engine: api.DBINSTANCE_TYPE_SQLSERVER, Version: "2017 Express"},
		"SQLSERVER_2017_WEB":        GoogleSQLVersion{Engine: api.DBINSTANCE_TYPE_SQLSERVER, Version: "2017 Web"},
	}
	InstanceTypes = map[string]GoogleSQLType{
		"db-f1-micro": GoogleSQLType{VcpuCount: 1, VmemSizeMb: 614},
		"db-g1-small": GoogleSQLType{VcpuCount: 1, VmemSizeMb: 1740},
		"D0":          GoogleSQLType{VcpuCount: 1, VmemSizeMb: 512},
		"D1":          GoogleSQLType{VcpuCount: 1, VmemSizeMb: 1024},
		"D2":          GoogleSQLType{VcpuCount: 1, VmemSizeMb: 2048},
		"D4":          GoogleSQLType{VcpuCount: 1, VmemSizeMb: 5120},
		"D8":          GoogleSQLType{VcpuCount: 2, VmemSizeMb: 10240},
		"D16":         GoogleSQLType{VcpuCount: 4, VmemSizeMb: 10240},
		"D32":         GoogleSQLType{VcpuCount: 8, VmemSizeMb: 10240},
	}
)

type GoogleSQLType struct {
	VcpuCount  int
	VmemSizeMb int
}

type GoogleSQLVersion struct {
	Engine  string
	Version string
}

type SDBInstanceLocationPreference struct {
	Zone string
	Kind string
}

type SDBInstanceMaintenanceWindow struct {
	Kind string
	Hour int
	Day  int
}

type SDBInstanceBackupConfiguration struct {
	StartTime        string
	Kind             string
	Enabled          bool
	BinaryLogEnabled bool
}

type SAuthorizedNetwork struct {
	Value string
	Kind  string
	Name  string
}

type SDBInstanceSettingIpConfiguration struct {
	PrivateNetwork     string
	AuthorizedNetworks []SAuthorizedNetwork
	Ipv4Enabled        bool
}

type SDBInstanceSetting struct {
	AuthorizedGaeApplications []string
	Tier                      string
	Kind                      string
	AvailabilityType          string
	PricingPlan               string
	ReplicationType           string
	ActivationPolicy          string
	IpConfiguration           SDBInstanceSettingIpConfiguration
	LocationPreference        SDBInstanceLocationPreference
	DataDiskType              string
	MaintenanceWindow         SDBInstanceMaintenanceWindow
	BackupConfiguration       SDBInstanceBackupConfiguration
	SettingsVersion           string
	StorageAutoResizeLimit    string
	StorageAutoResize         bool
	DataDiskSizeGb            int
	DatabaseFlags             []SDBInstanceParameter
}

type SDBInstanceIpAddress struct {
	Type      string
	IpAddress string
}

type SDBInstanceCaCert struct {
	Kind             string
	CertSerialNumber string
	Cert             string
	CommonName       string
	Sha1Fingerprint  string
	Instance         string
	CreateTime       time.Time
	ExpirationTime   time.Time
}

type SDBInstance struct {
	multicloud.SDBInstanceBase
	region *SRegion

	Kind                       string
	State                      string
	DatabaseVersion            string
	Settings                   SDBInstanceSetting
	Etag                       string
	MasterInstanceName         string
	IpAddresses                []SDBInstanceIpAddress
	ServerCaCert               SDBInstanceCaCert
	InstanceType               string
	Project                    string
	ServiceAccountEmailAddress string
	BackendType                string
	SelfLink                   string
	ConnectionName             string
	Name                       string
	Region                     string
	GceZone                    string
}

func (region *SRegion) GetDBInstances(maxResults int, pageToken string) ([]SDBInstance, error) {
	instances := []SDBInstance{}
	params := map[string]string{"filter": "region=" + region.Name}
	err := region.RdsList("instances", params, maxResults, pageToken, &instances)
	if err != nil {
		return nil, errors.Wrap(err, "RdsList")
	}
	return instances, nil
}

func (region *SRegion) GetDBInstance(instanceId string) (*SDBInstance, error) {
	instance := SDBInstance{region: region}
	err := region.rdsGet(instanceId, &instance)
	if err != nil {
		return nil, errors.Wrap(err, "RdsGet")
	}
	return &instance, nil
}

func (self *SDBInstance) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (rds *SDBInstance) GetName() string {
	return rds.Name
}

func (rds *SDBInstance) GetId() string {
	return rds.SelfLink
}

func (rds *SDBInstance) GetGlobalId() string {
	return strings.TrimPrefix(rds.SelfLink, fmt.Sprintf("%s/%s/", GOOGLE_DBINSTANCE_DOMAIN, GOOGLE_DBINSTANCE_API_VERSION))
}

func (rds *SDBInstance) GetProjectId() string {
	return rds.region.GetProjectId()
}

func (rds *SDBInstance) IsEmulated() bool {
	return false
}

func (rds *SDBInstance) GetStatus() string {
	switch rds.State {
	case "RUNNABLE":
		return api.DBINSTANCE_RUNNING
	case "PENDING_CREATE":
		return api.DBINSTANCE_DEPLOYING
	case "MAINTENANCE":
		return api.DBINSTANCE_MAINTENANCE
	case "FAILED":
		return api.DBINSTANCE_CREATE_FAILED
	case "UNKNOWN_STATE", "SUSPENDED":
		return api.DBINSTANCE_UNKNOWN
	}
	return rds.State
}

func (rds *SDBInstance) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (rds *SDBInstance) GetCreatedAt() time.Time {
	return rds.ServerCaCert.CreateTime
}

func (rds *SDBInstance) GetExpiredAt() time.Time {
	return time.Time{}
}

func (rds *SDBInstance) GetMasterInstanceId() string {
	if len(rds.MasterInstanceName) > 0 {
		if master := strings.Split(rds.MasterInstanceName, ":"); len(master) == 2 {
			return fmt.Sprintf("projects/%s/instances/%s", master[0], master[1])
		}
	}
	return ""
}

func (rds *SDBInstance) GetSecurityGroupId() string {
	return ""
}

func (rds *SDBInstance) Refresh() error {
	instance, err := rds.region.GetDBInstance(rds.SelfLink)
	if err != nil {
		return errors.Wrapf(err, "GetDBInstance(%s)", rds.SelfLink)
	}
	return jsonutils.Update(rds, instance)
}

func (rds *SDBInstance) GetPort() int {
	switch rds.GetEngine() {
	case api.DBINSTANCE_TYPE_MYSQL:
		return 3306
	case api.DBINSTANCE_TYPE_POSTGRESQL:
		return 5432
	case api.DBINSTANCE_TYPE_SQLSERVER:
		return 1433
	default:
		return 0
	}
}

func (rds *SDBInstance) GetEngine() string {
	if e, ok := EngineVersions[rds.DatabaseVersion]; ok {
		return e.Engine
	}
	return rds.DatabaseVersion
}

func (rds *SDBInstance) GetEngineVersion() string {
	if e, ok := EngineVersions[rds.DatabaseVersion]; ok {
		return e.Version
	}
	return rds.DatabaseVersion
}

func (rds *SDBInstance) GetInstanceType() string {
	return rds.Settings.Tier
}

func (rds *SDBInstance) GetVcpuCount() int {
	if t, ok := InstanceTypes[rds.Settings.Tier]; ok {
		return t.VcpuCount
	}
	numStr := ""
	if strings.HasPrefix(rds.Settings.Tier, "db-n1-standard-") {
		numStr = strings.TrimPrefix(rds.Settings.Tier, "db-n1-standard-")
	} else if strings.HasPrefix(rds.Settings.Tier, "db-n1-highmem-") {
		numStr = strings.TrimPrefix(rds.Settings.Tier, "db-n1-highmem-")
	} else {
		numStr = strings.TrimPrefix(rds.Settings.Tier, "db-custom-")
		numStr = strings.Split(numStr, "-")[0]
	}
	cpu, _ := strconv.ParseInt(numStr, 10, 32)
	return int(cpu)
}

func (rds *SDBInstance) GetVmemSizeMB() int {
	if t, ok := InstanceTypes[rds.Settings.Tier]; ok {
		return t.VmemSizeMb
	}
	if strings.HasPrefix(rds.Settings.Tier, "db-custom-") {
		info := strings.Split(rds.Settings.Tier, "-")
		numStr := info[len(info)-1]
		mem, _ := strconv.ParseInt(numStr, 10, 32)
		return int(mem)
	} else if strings.HasPrefix(rds.Settings.Tier, "db-n1-standard-") {
		return rds.GetVcpuCount() * 3840
	} else if strings.HasPrefix(rds.Settings.Tier, "db-n1-highmem-") {
		return rds.GetVcpuCount() * 3840 * 2
	}
	return 0
}

func (rds *SDBInstance) GetDiskSizeGB() int {
	return rds.Settings.DataDiskSizeGb
}

func (rds *SDBInstance) GetCategory() string {
	return rds.BackendType
}

func (rds *SDBInstance) GetStorageType() string {
	return rds.Settings.DataDiskType
}

func (rds *SDBInstance) GetMaintainTime() string {
	startTime := (rds.Settings.MaintenanceWindow.Hour + 8) % 24
	startDay := (rds.Settings.MaintenanceWindow.Day + 1) % 7
	return fmt.Sprintf("%s %02d:00 - %02d:00", time.Weekday(startDay).String(), startTime, startTime+1)
}

func (rds *SDBInstance) GetConnectionStr() string {
	for _, ip := range rds.IpAddresses {
		if ip.Type == "PRIMARY" {
			return ip.IpAddress
		}
	}
	return ""
}

func (rds *SDBInstance) GetInternalConnectionStr() string {
	ret := []string{rds.ConnectionName}
	for _, ip := range rds.IpAddresses {
		if ip.Type == "PRIVATE" {
			ret = append(ret, ip.IpAddress)
		}
	}
	return strings.Join(ret, ",")
}

func (rds *SDBInstance) GetZone1Id() string {
	zone, err := rds.region.GetZone(rds.GceZone)
	if err != nil {
		log.Errorf("failed to found rds %s zone %s", rds.Name, rds.GceZone)
		return ""
	}
	return zone.GetGlobalId()
}

func (rds *SDBInstance) GetZone2Id() string {
	return ""
}

func (rds *SDBInstance) GetZone3Id() string {
	return ""
}

func (rds *SDBInstance) GetIVpcId() string {
	if len(rds.Settings.IpConfiguration.PrivateNetwork) > 0 {
		globalnetwork, err := rds.region.client.GetGlobalNetwork(rds.Settings.IpConfiguration.PrivateNetwork)
		if err != nil {
			log.Errorf("failed to get global network %s error: %v", rds.Settings.IpConfiguration.PrivateNetwork, err)
			return ""
		}
		vpc := &SVpc{
			region:        rds.region,
			globalnetwork: globalnetwork,
		}
		return vpc.GetGlobalId()
	}
	return ""
}

func (rds *SDBInstance) GetDBNetwork() (*cloudprovider.SDBInstanceNetwork, error) {
	return nil, nil
}

func (rds *SDBInstance) GetIDBInstanceParameters() ([]cloudprovider.ICloudDBInstanceParameter, error) {
	ret := []cloudprovider.ICloudDBInstanceParameter{}
	for i := range rds.Settings.DatabaseFlags {
		rds.Settings.DatabaseFlags[i].rds = rds
		ret = append(ret, &rds.Settings.DatabaseFlags[i])
	}
	return ret, nil
}

func (rds *SDBInstance) GetIDBInstanceDatabases() ([]cloudprovider.ICloudDBInstanceDatabase, error) {
	databases, err := rds.region.GetDBInstanceDatabases(rds.Name)
	if err != nil {
		return nil, errors.Wrap(err, "GetDBInstanceDatabases")
	}
	ret := []cloudprovider.ICloudDBInstanceDatabase{}
	for i := range databases {
		databases[i].rds = rds
		ret = append(ret, &databases[i])
	}
	return ret, nil
}

func (rds *SDBInstance) GetIDBInstanceAccounts() ([]cloudprovider.ICloudDBInstanceAccount, error) {
	accounts, err := rds.region.GetDBInstanceAccounts(rds.Name)
	if err != nil {
		return nil, errors.Wrap(err, "GetDBInstanceAccounts")
	}
	ret := []cloudprovider.ICloudDBInstanceAccount{}
	for i := range accounts {
		accounts[i].rds = rds
		ret = append(ret, &accounts[i])
	}
	return ret, nil
}

func (rds *SDBInstance) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	backups, err := rds.region.GetDBInstanceBackups(rds.Name)
	if err != nil {
		return nil, errors.Wrap(err, "GetDBInstanceBackups")
	}
	ret := []cloudprovider.ICloudDBInstanceBackup{}
	for i := range backups {
		backups[i].rds = rds
		ret = append(ret, &backups[i])
	}
	return ret, nil
}

func (region *SRegion) ChangeDBInstanceConfig(instanceId string, diskSizeGb int, instanceType string) error {
	rds, err := region.GetDBInstance(instanceId)
	if err != nil {
		return errors.Wrapf(err, "GetDBInstance(%s)", instanceId)
	}
	body := map[string]interface{}{}
	settings := map[string]interface{}{}
	if len(instanceType) > 0 && instanceType != rds.GetInstanceType() {
		settings["tier"] = instanceType
	}
	if diskSizeGb > 0 && diskSizeGb != rds.GetDiskSizeGB() {
		settings["dataDiskSizeGb"] = diskSizeGb
	}
	if len(settings) == 0 {
		return nil
	}
	body["settings"] = settings
	return region.rdsPatch(rds.SelfLink, jsonutils.Marshal(body))
}

func (rds *SDBInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedDBInstanceChangeConfig) error {
	return rds.region.ChangeDBInstanceConfig(rds.SelfLink, config.DiskSizeGB, config.InstanceType)
}

func (rds *SDBInstance) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (region *SRegion) DBInstancePublicConnectionOperation(instanceId string, open bool) error {
	ipConfiguration := map[string]interface{}{
		"ipv4Enabled": open,
	}
	if open {
		ipConfiguration["authorizedNetworks"] = []map[string]string{
			map[string]string{
				"name":  "White list",
				"value": "0.0.0.0/0",
			},
		}
	}
	body := map[string]interface{}{
		"settings": map[string]interface{}{
			"ipConfiguration": ipConfiguration,
		},
	}
	return region.rdsPatch(instanceId, jsonutils.Marshal(body))
}

func (rds *SDBInstance) OpenPublicConnection() error {
	return rds.region.DBInstancePublicConnectionOperation(rds.SelfLink, true)
}

func (rds *SDBInstance) ClosePublicConnection() error {
	return rds.region.DBInstancePublicConnectionOperation(rds.SelfLink, false)
}

func (rds *SDBInstance) CreateDatabase(conf *cloudprovider.SDBInstanceDatabaseCreateConfig) error {
	return rds.region.CreateDatabase(rds.SelfLink, conf.Name, conf.CharacterSet)
}

func (rds *SDBInstance) CreateAccount(conf *cloudprovider.SDBInstanceAccountCreateConfig) error {
	return rds.region.CreateDBInstanceAccount(rds.SelfLink, conf.Name, conf.Password, "")
}

func (rds *SDBInstance) CreateIBackup(conf *cloudprovider.SDBInstanceBackupCreateConfig) (string, error) {
	err := rds.region.CreateDBInstanceBackup(rds.SelfLink, conf.Name, conf.Description)
	if err != nil {
		return "", errors.Wrap(err, "CreateIBackup")
	}
	return "", nil
}

func (region *SRegion) RecoverFromBackup(instanceId string, backupId string) error {
	backup, err := region.GetDBInstanceBackup(backupId)
	if err != nil {
		return errors.Wrap(err, "GetDBInstanceBackup")
	}
	body := map[string]interface{}{
		"restoreBackupContext": map[string]string{
			"backupRunId": backup.Id,
		},
	}
	return region.rdsDo(instanceId, "restoreBackup", nil, jsonutils.Marshal(body))
}

func (rds *SDBInstance) RecoveryFromBackup(conf *cloudprovider.SDBInstanceRecoveryConfig) error {
	return rds.region.RecoverFromBackup(rds.SelfLink, conf.BackupId)
}

func (rds *SDBInstance) Reboot() error {
	return rds.region.rdsDo(rds.SelfLink, "restart", nil, nil)
}

func (rds *SDBInstance) Delete() error {
	return rds.region.DeleteDBInstance(rds.SelfLink)
}

func (region *SRegion) DeleteDBInstance(id string) error {
	return region.rdsDelete(id)
}

func (region *SRegion) CreateRds(name, databaseVersion, category, instanceType, storageType string, diskSizeGb int, vpcId, zoneId, password string) (*SDBInstance, error) {
	settings := map[string]interface{}{
		"tier":              instanceType,
		"storageAutoResize": true,
		"dataDiskType":      storageType,
		"dataDiskSizeGb":    diskSizeGb,
	}
	ipConfiguration := map[string]interface{}{
		"ipv4Enabled": true,
	}
	ipConfiguration["authorizedNetworks"] = []map[string]string{
		map[string]string{
			"name":  "White list",
			"value": "0.0.0.0/0",
		},
	}
	settings["ipConfiguration"] = ipConfiguration
	body := map[string]interface{}{
		"databaseVersion": databaseVersion,
		"name":            name,
		"region":          region.Name,
		"settings":        settings,
		"backendType":     category,
	}
	if len(zoneId) > 0 {
		body["gceZone"] = zoneId
	}
	if len(password) > 0 {
		body["rootPassword"] = password
	}
	rds := SDBInstance{region: region}
	err := region.rdsInsert("instances", jsonutils.Marshal(body), &rds)
	if err != nil {
		return nil, errors.Wrap(err, "rdsInsert")
	}
	return &rds, nil
}

func (region *SRegion) CreateDBInstance(desc *cloudprovider.SManagedDBInstanceCreateConfig) (*SDBInstance, error) {
	desc.EngineVersion = strings.ToUpper(desc.EngineVersion)
	desc.EngineVersion = strings.Replace(desc.EngineVersion, ".", "_", -1)
	desc.EngineVersion = strings.Replace(desc.EngineVersion, " ", "_", -1)
	if desc.Engine == api.DBINSTANCE_TYPE_POSTGRESQL {
		desc.Engine = "POSTGRES"
	}
	databaseVersion := fmt.Sprintf("%s_%s", strings.ToUpper(desc.Engine), desc.EngineVersion)
	if _, ok := EngineVersions[databaseVersion]; !ok {
		return nil, fmt.Errorf("Unsupport %s version %s", desc.Engine, desc.EngineVersion)
	}
	var err error
	var rds *SDBInstance = nil
	if len(desc.InstanceType) > 0 {
		if len(desc.ZoneIds) == 0 {
			desc.ZoneIds = append(desc.ZoneIds, "")
		}
		for _, zoneId := range desc.ZoneIds {
			rds, err = region.CreateRds(desc.Name, databaseVersion, desc.Category, desc.InstanceType, desc.StorageType, desc.DiskSizeGB, desc.VpcId, zoneId, desc.Password)
			if err == nil {
				break
			} else {
				log.Errorf("failed to create dbinstance %s at %s error: %v", desc.Name, zoneId, err)
			}
		}
		if err != nil {
			return nil, errors.Wrap(err, "CreateRds")
		}
	} else if len(desc.InstanceTypes) > 0 {
		for _, spec := range desc.InstanceTypes {
			if len(spec.ZoneIds) == 0 {
				desc.ZoneIds = append(desc.ZoneIds, "")
			}
			for _, zoneId := range spec.ZoneIds {
				rds, err = region.CreateRds(desc.Name, databaseVersion, desc.Category, desc.InstanceType, desc.StorageType, desc.DiskSizeGB, desc.VpcId, zoneId, desc.Password)
				if err == nil {
					break
				} else {
					log.Errorf("failed to create dbinstance %s at %s error: %v", desc.Name, zoneId, err)
				}
			}
			if err == nil {
				break
			}
		}
		if err != nil {
			return nil, errors.Wrap(err, "CreateRds")
		}
	} else {
		return nil, fmt.Errorf("Missing instance type info")
	}
	return rds, nil
}
