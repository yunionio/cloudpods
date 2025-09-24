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

package qcloud

import (
	"context"
	"fmt"
	"strings"
	"time"

	billingapi "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
)

/*
   {
     "Architecture": "SINGLE",
     "BackupCycle":
       [],
     "BackupCycleType": "",
     "BackupModel": "",
     "BackupSaveDays": 7,
     "BackupTime": "",
     "Collation": "Chinese_PRC_CI_AS",
     "Cpu": 2,
     "CreateTime": "2025-09-16 20:27:09",
     "CrossBackupEnabled": "disable",
     "CrossBackupSaveDays": 0,
     "CrossRegions":
       [],
     "DnsPodDomain": "",
     "EndTime": "0000-00-00 00:00:00",
     "HAFlag": "SINGLE",
     "InstanceId": "mssql-ezqm0pk4",
     "InstanceNote": "",
     "InstanceType": "SI",
     "IsDrZone": false,
     "IsolateOperator": "",
     "IsolateTime": "0000-00-00 00:00:00",
     "Memory": 4,
     "Model": 2,
     "MultiSlaveZones":
       [],
     "Name": "11b58192-1ddd-4dc8-8ae9-454ac919927f",
     "PayMode": 0,
     "Pid": 1003456,
     "ProjectId": 0,
     "ROFlag": "",
     "Region": "ap-beijing",
     "RegionId": 8,
     "RenewFlag": 0,
     "SlaveZones":
       {
         "SlaveZone": "",
         "SlaveZoneName": ""
       },
     "StartTime": "2025-09-16 20:27:09",
     "Status": 1,
     "Storage": 20,
     "Style": "EXCLUSIVE",
     "SubFlag": "",
     "SubnetId": 1344612,
     "TgwWanVPort": 0,
     "TimeZone": "China Standard Time",
     "Type": "CLOUD_BSSD",
     "Uid": "",
     "UniqSubnetId": "subnet-ej5fo5gd",
     "UniqVpcId": "vpc-dvnlj4aq",
     "UpdateTime": "2025-09-16 20:27:09",
     "UsedStorage": 0,
     "Version": "2019",
     "VersionName": "SQL Server 2019 Enterprise",
     "Vip": "",
     "VpcId": 3832347,
     "Vport": 0,
     "Zone": "ap-beijing-5",
     "ZoneId": 800005
   }
*/

type SSQLServer struct {
	multicloud.SDBInstanceBase
	QcloudTags

	Architecture        string
	BackupCycle         []string
	BackupCycleType     string
	BackupModel         string
	BackupSaveDays      int
	BackupTime          string
	Collation           string
	Cpu                 int
	CreateTime          time.Time
	CrossBackupEnabled  string
	CrossBackupSaveDays int
	CrossRegions        []string
	DnsPodDomain        string
	EndTime             time.Time
	HAFlag              string
	InstanceId          string
	InstanceNote        string
	InstanceType        string
	IsDrZone            bool
	IsolateOperator     string
	IsolateTime         string
	Memory              int
	Model               int
	MultiSlaveZones     []string
	Name                string
	PayMode             int
	Pid                 int
	ProjectId           string
	ROFlag              string
	Region              string
	RegionId            int
	RenewFlag           int
	SlaveZones          struct {
		SlaveZone     string
		SlaveZoneName string
	}
	StartTime    string
	Status       int
	Storage      int
	Style        string
	SubFlag      string
	SubnetId     int
	TgwWanVPort  int
	TimeZone     string
	Type         string
	Uid          string
	UniqSubnetId string
	UniqVpcId    string
	UpdateTime   string
	UsedStorage  int
	Version      string
	VersionName  string
	Vip          string
	VpcId        int
	Vport        int
	Zone         string
	ZoneId       int

	region *SRegion
}

func (mssql *SSQLServer) GetId() string {
	return mssql.InstanceId
}

func (mssql *SSQLServer) GetGlobalId() string {
	return mssql.InstanceId
}

func (mssql *SSQLServer) GetName() string {
	if len(mssql.Name) > 0 {
		return mssql.Name
	}
	return mssql.InstanceId
}

func (mssql *SSQLServer) GetDiskSizeGB() int {
	return mssql.Storage
}

func (mssql *SSQLServer) GetEngine() string {
	return api.DBINSTANCE_TYPE_SQLSERVER
}

func (mssql *SSQLServer) GetEngineVersion() string {
	return mssql.Version
}

func (mssql *SSQLServer) GetIVpcId() string {
	return mssql.UniqVpcId
}

func (mssql *SSQLServer) Refresh() error {
	rds, err := mssql.region.GetSQLServer(mssql.InstanceId)
	if err != nil {
		return errors.Wrapf(err, "GetSQLServer(%s)", mssql.InstanceId)
	}
	return jsonutils.Update(mssql, rds)
}

func (mssql *SSQLServer) GetInstanceType() string {
	return fmt.Sprintf("%d核%dMB", mssql.Cpu, mssql.Memory)
}

func (mssql *SSQLServer) GetMaintainTime() string {
	return ""
}

func (mssql *SSQLServer) GetDBNetworks() ([]cloudprovider.SDBInstanceNetwork, error) {
	return []cloudprovider.SDBInstanceNetwork{
		{NetworkId: mssql.UniqSubnetId, IP: mssql.Vip},
	}, nil
}

func (mssql *SSQLServer) GetConnectionStr() string {
	return ""
}

func (mssql *SSQLServer) GetInternalConnectionStr() string {
	return fmt.Sprintf("%s:%d", mssql.Vip, mssql.Vport)
}

func (mssql *SSQLServer) Reboot() error {
	return cloudprovider.ErrNotImplemented
}

func (mssql *SSQLServer) ChangeConfig(ctx context.Context, opts *cloudprovider.SManagedDBInstanceChangeConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (mssql *SSQLServer) GetMasterInstanceId() string {
	return ""
}

func (mssql *SSQLServer) GetSecurityGroupIds() ([]string, error) {
	return mssql.region.DescribeSQLServerDBSecurityGroups(mssql.InstanceId)
}

func (region *SRegion) DescribeSQLServerDBSecurityGroups(id string) ([]string, error) {
	params := map[string]string{
		"InstanceId": id,
	}
	resp, err := region.sqlserverRequest("DescribeDBSecurityGroups", params)
	if err != nil {
		return []string{}, errors.Wrapf(err, "DescribeDBSecurityGroups")
	}
	ret := struct {
		SecurityGroupSet []struct {
			SecurityGroupId string
		}
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	groups := []string{}
	for i := range ret.SecurityGroupSet {
		groups = append(groups, ret.SecurityGroupSet[i].SecurityGroupId)
	}

	return groups, nil
}

func (mssql *SSQLServer) SetSecurityGroups(ids []string) error {
	return mssql.region.ModifyDBInstanceSecurityGroups(mssql.InstanceId, ids)
}

func (region *SRegion) ModifyDBInstanceSecurityGroups(rdsId string, secIds []string) error {
	params := map[string]string{
		"InstanceId": rdsId,
	}
	for idx, id := range secIds {
		params[fmt.Sprintf("SecurityGroupIdSet.%d", idx)] = id
	}
	_, err := region.sqlserverRequest("ModifyDBInstanceSecurityGroups", params)
	return err
}

func (mssql *SSQLServer) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotImplemented
}

func (mssql *SSQLServer) OpenPublicConnection() error {
	return cloudprovider.ErrNotImplemented
}

func (mssql *SSQLServer) ClosePublicConnection() error {
	return cloudprovider.ErrNotImplemented
}

func (mssql *SSQLServer) GetPort() int {
	return mssql.Vport
}

func (mssql *SSQLServer) GetStatus() string {
	switch mssql.Status {
	case 1:
		return api.DBINSTANCE_DEPLOYING
	case 2, 3:
		return api.DBINSTANCE_RUNNING
	case 5, 6, 8:
		return api.DBINSTANCE_DELETING
	case 7, 9, 13, 14, 15, 16, 17:
		return api.DBINSTANCE_UPGRADING
	case 4:
		return api.DBINSTANCE_ISOLATE
	case 12:
		return api.DBINSTANCE_REBOOTING
	case 10:
		return api.DBINSTANCE_MIGRATING
	default:
		return api.DBINSTANCE_DEPLOYING
	}
}

func (mssql *SSQLServer) GetCategory() string {
	return strings.ToLower(mssql.Architecture)
}

func (mssql *SSQLServer) GetStorageType() string {
	return api.QCLOUD_DBINSTANCE_STORAGE_TYPE_CLOUD_SSD
}

func (mssql *SSQLServer) GetCreatedAt() time.Time {
	// 2019-12-25 09:00:43  #非UTC时间
	return mssql.CreateTime.Add(time.Hour * -8)
}

func (mssql *SSQLServer) GetBillingType() string {
	if mssql.PayMode == 1 {
		return billingapi.BILLING_TYPE_PREPAID
	}
	return billingapi.BILLING_TYPE_POSTPAID
}

func (mssql *SSQLServer) SetAutoRenew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotImplemented
}

func (mssql *SSQLServer) IsAutoRenew() bool {
	return mssql.RenewFlag == 1
}

func (mssql *SSQLServer) GetExpiredAt() time.Time {
	return mssql.EndTime
}

func (mssql *SSQLServer) GetVcpuCount() int {
	return mssql.Cpu
}

func (mssql *SSQLServer) GetVmemSizeMB() int {
	return mssql.Memory * 1024
}

func (mssql *SSQLServer) GetZone1Id() string {
	return mssql.Zone
}

func (mssql *SSQLServer) GetZone2Id() string {
	return mssql.SlaveZones.SlaveZone
}

func (mssql *SSQLServer) GetZone3Id() string {
	return mssql.SlaveZones.SlaveZone
}

func (mssql *SSQLServer) GetProjectId() string {
	return mssql.ProjectId
}

func (mssql *SSQLServer) Delete() error {
	err := mssql.region.DeleteSQLServer(mssql.InstanceId)
	if err != nil {
		return errors.Wrapf(err, "DeleteSQLServer")
	}
	return mssql.region.DeleteSQLServerInRecycleBin(mssql.InstanceId)
}

func (region *SRegion) GetSQLServers(id string) ([]SSQLServer, error) {
	params := map[string]string{}
	if len(id) > 0 {
		params["InstanceIdSet.0"] = id
	}
	offset := 0
	ret := []SSQLServer{}
	for {
		resp, err := region.sqlserverRequest("DescribeDBInstances", params)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeDBInstances")
		}
		part := struct {
			DBInstances []SSQLServer
			TotalCount  int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrapf(err, "resp.Unmarshal")
		}
		for i := range part.DBInstances {
			if part.DBInstances[i].Status == 4 { // 过滤掉回收站
				continue
			}
			part.DBInstances[i].region = region

			ret = append(ret, part.DBInstances[i])
		}
		if len(ret) >= part.TotalCount || len(part.DBInstances) == 0 {
			break
		}
		offset++
		params["Offset"] = fmt.Sprintf("%d", offset)
	}
	return ret, nil
}

func (region *SRegion) GetISQLServers() ([]cloudprovider.ICloudDBInstance, error) {
	servers, err := region.GetSQLServers("")
	if err != nil {
		return nil, errors.Wrapf(err, "GetSQLServers")
	}
	ret := []cloudprovider.ICloudDBInstance{}
	for i := range servers {
		servers[i].region = region
		ret = append(ret, &servers[i])
	}
	return ret, nil
}

func (region *SRegion) GetSQLServer(id string) (*SSQLServer, error) {
	vms, err := region.GetSQLServers(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetSQLServers")
	}
	for i := range vms {
		if vms[i].InstanceId == id {
			vms[i].region = region
			return &vms[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (region *SRegion) DeleteSQLServer(id string) error {
	params := map[string]string{
		"InstanceIdSet.0": id,
	}
	_, err := region.sqlserverRequest("TerminateDBInstance", params)
	return err
}

func (region *SRegion) DeleteSQLServerInRecycleBin(id string) error {
	params := map[string]string{
		"InstanceId": id,
	}
	_, err := region.sqlserverRequest("DeleteDBInstance", params)
	return err
}
