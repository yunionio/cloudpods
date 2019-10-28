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
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
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
	case "Creating", "DBInstanceClassChanging", "GuardDBInstanceCreating", "DBInstanceNetTypeChanging", "GuardSwitching":
		return api.DBINSTANCE_DEPLOYING
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
		return api.DBINSTANCE_CATEGORY_BASIC
	case "HighAvailability":
		return api.DBINSTANCE_CATEGORY_HA
	case "AlwaysOn":
		return api.DBINSTANCE_CATEGORY_ALWAYSON
	case "Finance":
		return api.DBINSTANCE_CATEGORY_FINANCE
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

func (rds *SDBInstance) GetIZoneId() string {
	if len(rds.ZoneId) > 0 {
		zone, err := rds.region.getZoneById(rds.ZoneId)
		if err != nil {
			log.Errorf("SDBInstances.getZoneById %s error: %v", rds.ZoneId, err)
			return ""
		}
		return zone.GetGlobalId()
	}
	return ""
}

func (rds *SDBInstance) GetDBNetwork() (*cloudprovider.SDBInstanceNetwork, error) {
	netInfo, err := rds.region.GetDBInstanceNetInfo(rds.DBInstanceId)
	if err != nil {
		return nil, err
	}
	network := &cloudprovider.SDBInstanceNetwork{}
	for _, net := range netInfo {
		if net.IPType == "Private" {
			network.IP = net.IPAddress
			network.NetworkId = net.VSwitchId
			return network, nil
		}
	}
	return nil, fmt.Errorf("failed to found network for aliyun rds %s", rds.DBInstanceId)
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

	for _, net := range rds.netInfo {
		if net.IPType != "Public" {
			return net.ConnectionString
		}
	}
	return ""
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
