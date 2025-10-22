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

package oceanbase

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDBInstance struct {
	multicloud.SDBInstanceBase
	region *SRegion

	AvailableZones []string

	DeployMode      string
	DeployType      string
	NodeNum         int
	CreateTime      time.Time
	Status          string
	Vpc             string
	StopTime        time.Time
	PayType         string
	CloudProvider   string
	CPU             int
	Mem             int
	CPUArchitecture string
	DiskType        string
	Resource        struct {
		Cpu struct {
			TotalCpu float64
			UsedCpu  float64
		}
		Memory struct {
			TotalMemory float64
			UsedMemory  float64
		}
		Disk struct {
			TotalDataSize float64
			TotalDiskSize float64
			UsedDiskSize  float64
		}
	}
	InstanceName        string
	InstanceId          string
	InstanceType        string
	InstanceClass       string
	SaleChannel         string
	Series              string
	StandbyInstanceIds  []string
	State               string
	StorageArchitecture string
	TagList             []string
	UsedDiskSize        float64
	Version             string
	VpcId               string
	DiskSize            int
	Region              string
}

func (region *SRegion) GetDBInstances() ([]SDBInstance, error) {
	params := url.Values{}
	params.Set("pageSize", "10")
	pageNum := 1
	ret := []SDBInstance{}
	for {
		params.Set("pageNumber", fmt.Sprintf("%d", pageNum))
		resp, err := region.list("/api/v2/instances", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			Data struct {
				DataList []SDBInstance
				Total    int
			}
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Data.DataList...)
		if len(ret) >= part.Data.Total || len(part.Data.DataList) == 0 {
			break
		}
		pageNum++
	}
	return ret, nil
}

func (region *SRegion) GetDBInstance(id string) (*SDBInstance, error) {
	resp, err := region.list(fmt.Sprintf("/api/v2/instances/%s", id), url.Values{})
	if err != nil {
		return nil, err
	}
	dbinstance := &SDBInstance{region: region}
	err = resp.Unmarshal(&dbinstance, "data")
	if err != nil {
		return nil, err
	}
	return dbinstance, nil
}

func (region *SRegion) DeleteDBInstance(id string) error {
	_, err := region.delete("/api/v2/instances", map[string]interface{}{"instanceId": id})
	return err
}

func (region *SRegion) StartDBInstance(id string) error {
	_, err := region.put(fmt.Sprintf("/api/v2/instances/%s/startCluster", id), nil)
	return err
}

func (region *SRegion) StopDBInstance(id string) error {
	_, err := region.put(fmt.Sprintf("/api/v2/instances/%s/stopCluster", id), nil)
	return err
}

func (rds *SDBInstance) GetGlobalId() string {
	return rds.InstanceId
}

func (rds *SDBInstance) GetName() string {
	return rds.InstanceName
}

func (rds *SDBInstance) GetStatus() string {
	switch rds.Status {
	case "ONLINE":
		return api.DBINSTANCE_RUNNING
	case "PENDING_STOP", "STOPPED", "PENDING_START":
		return api.DBINSTANCE_REBOOTING
	case "PENDING_DELETE":
		return api.DBINSTANCE_DELETING
	case "PENDING_CREATE":
		return api.DBINSTANCE_DEPLOYING
	default:
		return api.DBINSTANCE_UNKNOWN
	}
}

func (rds *SDBInstance) GetCreatedAt() time.Time {
	return rds.CreateTime
}

func (rds *SDBInstance) GetExpiredAt() time.Time {
	return time.Time{}
}

func (rds *SDBInstance) GetBillingType() string {
	if rds.PayType == "POSTPAY" {
		return billing_api.BILLING_TYPE_POSTPAID
	}
	return billing_api.BILLING_TYPE_PREPAID
}

func (rds *SDBInstance) GetProjectId() string {
	return ""
}

// ICloudResource 接口方法
func (rds *SDBInstance) GetId() string {
	return rds.InstanceId
}

func (rds *SDBInstance) GetDescription() string {
	return ""
}

func (rds *SDBInstance) Refresh() error {
	instance, err := rds.region.GetDBInstance(rds.InstanceId)
	if err != nil {
		return err
	}
	return jsonutils.Update(rds, instance)
}

func (rds *SDBInstance) GetSysTags() map[string]string {
	return map[string]string{}
}

func (rds *SDBInstance) GetTags() (map[string]string, error) {
	return map[string]string{}, nil
}

func (rds *SDBInstance) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotSupported
}

// 资源相关方法
func (rds *SDBInstance) GetVcpuCount() int {
	return rds.CPU
}

func (rds *SDBInstance) GetVmemSizeMB() int {
	return rds.Mem * 1024
}

func (rds *SDBInstance) GetDiskSizeGB() int {
	return rds.DiskSize
}

func (rds *SDBInstance) GetDiskSizeUsedMB() int {
	return int(rds.UsedDiskSize * 1024)
}

func (rds *SDBInstance) GetInstanceType() string {
	return rds.InstanceClass
}

// 网络相关方法
func (rds *SDBInstance) GetPort() int {
	return 2881 // OceanBase 默认端口
}

func (rds *SDBInstance) GetConnectionStr() string {
	return ""
}

func (rds *SDBInstance) GetInternalConnectionStr() string {
	return ""
}

func (rds *SDBInstance) GetIVpcId() string {
	return rds.VpcId
}

// 可用区相关方法
func (rds *SDBInstance) GetZone1Id() string {
	if len(rds.AvailableZones) > 0 {
		return fmt.Sprintf("%s-%s", rds.Region, rds.AvailableZones[0])
	}
	return ""
}

func (rds *SDBInstance) GetZone2Id() string {
	if len(rds.AvailableZones) > 1 {
		return fmt.Sprintf("%s-%s", rds.Region, rds.AvailableZones[1])
	}
	return ""
}

func (rds *SDBInstance) GetZone3Id() string {
	if len(rds.AvailableZones) > 2 {
		return fmt.Sprintf("%s-%s", rds.Region, rds.AvailableZones[2])
	}
	return ""
}

// 引擎相关方法
func (rds *SDBInstance) GetEngine() string {
	return "OceanBase"
}

func (rds *SDBInstance) GetEngineVersion() string {
	return rds.Version
}

func (rds *SDBInstance) GetCategory() string {
	return strings.ToLower(rds.InstanceType)
}

func (rds *SDBInstance) GetStorageType() string {
	return rds.DiskType
}

func (rds *SDBInstance) GetMaintainTime() string {
	return ""
}

func (rds *SDBInstance) GetIops() int {
	return 0
}

// 操作相关方法
func (rds *SDBInstance) Reboot() error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "Reboot")
}

func (rds *SDBInstance) Delete() error {
	return rds.region.DeleteDBInstance(rds.InstanceId)
}

func (rds *SDBInstance) ChangeConfig(ctx context.Context, config *cloudprovider.SManagedDBInstanceChangeConfig) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "ChangeConfig")
}

func (rds *SDBInstance) Update(ctx context.Context, input cloudprovider.SDBInstanceUpdateOptions) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "Update")
}

// 管理相关方法
func (rds *SDBInstance) GetMasterInstanceId() string {
	return ""
}

func (rds *SDBInstance) GetSecurityGroupIds() ([]string, error) {
	return []string{}, nil
}

func (rds *SDBInstance) SetSecurityGroups(ids []string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "SetSecurityGroups")
}

func (rds *SDBInstance) GetDBNetworks() ([]cloudprovider.SDBInstanceNetwork, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetDBNetworks")
}

func (rds *SDBInstance) GetIDBInstanceParameters() ([]cloudprovider.ICloudDBInstanceParameter, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIDBInstanceParameters")
}

func (rds *SDBInstance) GetIDBInstanceDatabases() ([]cloudprovider.ICloudDBInstanceDatabase, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIDBInstanceDatabases")
}

func (rds *SDBInstance) GetIDBInstanceAccounts() ([]cloudprovider.ICloudDBInstanceAccount, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIDBInstanceAccounts")
}

func (rds *SDBInstance) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIDBInstanceBackups")
}

func (rds *SDBInstance) OpenPublicConnection() error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "OpenPublicConnection")
}

func (rds *SDBInstance) ClosePublicConnection() error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "ClosePublicConnection")
}

func (rds *SDBInstance) CreateDatabase(conf *cloudprovider.SDBInstanceDatabaseCreateConfig) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateDatabase")
}

func (rds *SDBInstance) CreateAccount(conf *cloudprovider.SDBInstanceAccountCreateConfig) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateAccount")
}

func (rds *SDBInstance) CreateIBackup(conf *cloudprovider.SDBInstanceBackupCreateConfig) (string, error) {
	return "", errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateIBackup")
}

func (rds *SDBInstance) RecoveryFromBackup(conf *cloudprovider.SDBInstanceRecoveryConfig) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RecoveryFromBackup")
}
