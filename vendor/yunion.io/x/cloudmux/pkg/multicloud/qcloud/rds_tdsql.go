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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type STDSQL struct {
	multicloud.SDBInstanceBase
	// multicloud.SBillingBase
	QcloudTags
	region *SRegion

	AppId             int           `json:"AppId"`
	AutoRenewFlag     int           `json:"AutoRenewFlag"`
	CPU               int           `json:"Cpu"`
	CreateTime        time.Time     `json:"CreateTime"`
	DbEngine          string        `json:"DbEngine"`
	DbVersion         string        `json:"DbVersion"`
	DcnDstNum         int           `json:"DcnDstNum"`
	DcnFlag           int           `json:"DcnFlag"`
	DcnStatus         int           `json:"DcnStatus"`
	ExclusterId       string        `json:"ExclusterId"`
	Id                int           `json:"Id"`
	InstanceId        string        `json:"InstanceId"`
	InstanceName      string        `json:"InstanceName"`
	InstanceType      int           `json:"InstanceType"`
	Ipv6Flag          int           `json:"Ipv6Flag"`
	IsAuditSupported  int           `json:"IsAuditSupported"`
	IsTmp             int           `json:"IsTmp"`
	IsolatedTimestamp string        `json:"IsolatedTimestamp"`
	Locker            int           `json:"Locker"`
	Memory            int           `json:"Memory"`
	NodeCount         int           `json:"NodeCount"`
	Paymode           string        `json:"Paymode"`
	PeriodEndTime     string        `json:"PeriodEndTime"`
	Pid               int           `json:"Pid"`
	ProjectId         int           `json:"ProjectId"`
	Region            string        `json:"Region"`
	ShardCount        int           `json:"ShardCount"`
	ShardDetail       []ShardDetail `json:"ShardDetail"`
	Status            int           `json:"Status"`
	StatusDesc        string        `json:"StatusDesc"`
	Storage           int           `json:"Storage"`
	SubnetId          int           `json:"SubnetId"`
	Uin               string        `json:"Uin"`
	UniqueSubnetId    string        `json:"UniqueSubnetId"`
	UniqueVpcId       string        `json:"UniqueVpcId"`
	UpdateTime        string        `json:"UpdateTime"`
	Vip               string        `json:"Vip"`
	Vipv6             string        `json:"Vipv6"`
	VpcId             int           `json:"VpcId"`
	Vport             int           `json:"Vport"`
	WanDomain         string        `json:"WanDomain"`
	WanPort           int           `json:"WanPort"`
	WanPortIpv6       int           `json:"WanPortIpv6"`
	WanStatus         int           `json:"WanStatus"`
	WanStatusIpv6     int           `json:"WanStatusIpv6"`
	WanVip            string        `json:"WanVip"`
	WanVipv6          string        `json:"WanVipv6"`
	Zone              string        `json:"Zone"`
}
type ShardDetail struct {
	CPU             int    `json:"Cpu"`
	Createtime      string `json:"Createtime"`
	Memory          int    `json:"Memory"`
	NodeCount       int    `json:"NodeCount"`
	Pid             int    `json:"Pid"`
	ShardId         int    `json:"ShardId"`
	ShardInstanceId string `json:"ShardInstanceId"`
	ShardSerialId   string `json:"ShardSerialId"`
	Status          int    `json:"Status"`
	Storage         int    `json:"Storage"`
}

func (self *STDSQL) GetName() string {
	return self.InstanceName
}

func (self *STDSQL) GetId() string {
	return self.InstanceId
}

func (self *STDSQL) GetGlobalId() string {
	return self.InstanceId
}

// 0 创建中，1 流程处理中， 2 运行中，3 实例未初始化，-1 实例已隔离，-2 实例已删除，4 实例初始化中，5 实例删除中，6 实例重启中，7 数据迁移中
func (self *STDSQL) GetStatus() string {
	switch self.Status {
	case 0, 1, 3, 4:
		return api.DBINSTANCE_DEPLOYING
	case 2:
		return api.DBINSTANCE_RUNNING
	case -1, -2, 5:
		return api.DBINSTANCE_DELETING
	case 6:
		return api.DBINSTANCE_REBOOTING
	case 7:
		return api.DBINSTANCE_MIGRATING
	default:
		return fmt.Sprintf("%d", self.Status)
	}
}

func (self *STDSQL) GetPort() int {
	return self.Vport
}

func (self *STDSQL) GetVmemSizeMB() int {
	return self.Memory * 1024
}

func (self *STDSQL) GetDiskSizeGB() int {
	return self.Storage
}

func (self *STDSQL) GetVcpuCount() int {
	return self.CPU
}

func (self *STDSQL) GetCreatedAt() time.Time {
	return self.CreateTime.Add(time.Hour * -8)
}

func (self *STDSQL) GetBillingType() string {
	return self.Paymode
}

func (self *STDSQL) GetProjectId() string {
	return fmt.Sprintf("%d", self.ProjectId)
}

func (self *STDSQL) Refresh() error {
	sql, err := self.region.GetTDSQL(self.InstanceId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, sql)
}

func (self *STDSQL) Reboot() error {
	return cloudprovider.ErrNotSupported
}

func (self *STDSQL) GetMasterInstanceId() string {
	return ""
}

func (self *STDSQL) GetSecurityGroupIds() ([]string, error) {
	ret := []string{}
	groups, err := self.region.GetTDSQLSecurityGroups(self.InstanceId)
	if err != nil {
		return ret, err
	}
	for i := range groups {
		ret = append(ret, groups[i].SecurityGroupId)
	}
	return ret, nil
}

func (self *STDSQL) SetSecurityGroups(ids []string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *STDSQL) GetEngine() string {
	return self.DbEngine
}

func (self *STDSQL) GetEngineVersion() string {
	return self.DbVersion
}

func (self *STDSQL) GetInstanceType() string {
	return fmt.Sprintf("%dC%dG", self.GetVcpuCount(), self.GetVmemSizeMB()/1024)
}

func (self *STDSQL) ChangeConfig(ctx context.Context, opts *cloudprovider.SManagedDBInstanceChangeConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (self *STDSQL) ClosePublicConnection() error {
	return cloudprovider.ErrNotImplemented
}

func (self *STDSQL) OpenPublicConnection() error {
	return cloudprovider.ErrNotImplemented
}

func (self *STDSQL) CreateAccount(opts *cloudprovider.SDBInstanceAccountCreateConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (self *STDSQL) CreateDatabase(opts *cloudprovider.SDBInstanceDatabaseCreateConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (self *STDSQL) CreateIBackup(opts *cloudprovider.SDBInstanceBackupCreateConfig) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *STDSQL) GetMaintainTime() string {
	return ""
}

func (self *STDSQL) GetStorageType() string {
	return api.QCLOUD_DBINSTANCE_STORAGE_TYPE_LOCAL_SSD
}

func (self *STDSQL) GetIVpcId() string {
	return self.UniqueVpcId
}

func (self *STDSQL) Delete() error {
	if self.GetBillingType() == billing_api.BILLING_TYPE_PREPAID {
		return self.region.DeletePrepaidTDSQL(self.InstanceId)
	}
	return self.region.DeletePostpaidTDSQL(self.InstanceId)
}

func (self *STDSQL) GetCategory() string {
	return api.QCLOUD_DBINSTANCE_CATEGORY_TDSQL
}

func (self *STDSQL) GetConnectionStr() string {
	if len(self.WanDomain) > 0 {
		return fmt.Sprintf("%s:%d", self.WanDomain, self.WanPort)
	}
	return ""
}

func (self *STDSQL) GetInternalConnectionStr() string {
	if len(self.Vip) > 0 {
		return fmt.Sprintf("%s:%d", self.Vip, self.Vport)
	}
	return ""
}

func (self *STDSQL) GetZone1Id() string {
	return self.Zone
}

func (self *STDSQL) GetZone2Id() string {
	return ""
}

func (self *STDSQL) GetZone3Id() string {
	return ""
}

func (self *STDSQL) RecoveryFromBackup(conf *cloudprovider.SDBInstanceRecoveryConfig) error {
	return cloudprovider.ErrNotImplemented
}

func (self *STDSQL) GetDBNetworks() ([]cloudprovider.SDBInstanceNetwork, error) {
	ret := []cloudprovider.SDBInstanceNetwork{}
	if len(self.Vip) > 0 && len(self.UniqueSubnetId) > 0 {
		ret = append(ret, cloudprovider.SDBInstanceNetwork{NetworkId: self.UniqueSubnetId, IP: self.Vip})
	}
	return ret, nil
}

func (self *SRegion) GetTDSQL(id string) (*STDSQL, error) {
	sqls, _, err := self.GetTDSQLs([]string{id}, 1, 0)
	if err != nil {
		return nil, errors.Wrapf(err, "GetTDSQLs")
	}
	for i := range sqls {
		if sqls[i].InstanceId == id {
			sqls[i].region = self
			return &sqls[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "id: [%s]", id)
}

func (self *SRegion) GetTDSQLs(ids []string, limit, offset int) ([]STDSQL, int, error) {
	if limit < 1 || limit > 100 {
		limit = 100
	}
	params := map[string]string{
		"Limit":  fmt.Sprintf("%d", limit),
		"Offset": fmt.Sprintf("%d", offset),
	}
	for idx, id := range ids {
		params[fmt.Sprintf("InstanceIds.%d", idx)] = id
	}
	resp, err := self.dcdbRequest("DescribeDCDBInstances", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeDCDBInstances")
	}
	ret := []STDSQL{}
	err = resp.Unmarshal(&ret, "Instances")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	totalCount, _ := resp.Float("TotalCount")
	return ret, int(totalCount), nil
}

func (self *SRegion) GetITDSQLs() ([]cloudprovider.ICloudDBInstance, error) {
	ret := []cloudprovider.ICloudDBInstance{}
	for {
		part, total, err := self.GetTDSQLs(nil, 100, len(ret))
		if err != nil {
			return nil, errors.Wrapf(err, "GetTDSQLs")
		}
		for i := range part {
			part[i].region = self
			ret = append(ret, &part[i])
		}
		if len(ret) >= total {
			break
		}
	}
	return ret, nil
}

type STDSQLSecurityGroup struct {
	CreateTime          string `json:"CreateTime"`
	ProjectID           int    `json:"ProjectId"`
	SecurityGroupId     string `json:"SecurityGroupId"`
	SecurityGroupName   string `json:"SecurityGroupName"`
	SecurityGroupRemark string `json:"SecurityGroupRemark"`
}

func (self *SRegion) GetTDSQLSecurityGroups(id string) ([]STDSQLSecurityGroup, error) {
	params := map[string]string{
		"Product":    "dcdb",
		"InstanceId": id,
	}
	resp, err := self.dcdbRequest("DescribeDBSecurityGroups", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeDBSecurityGroups")
	}
	ret := []STDSQLSecurityGroup{}
	err = resp.Unmarshal(&ret, "Groups")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return ret, nil
}

func (self *STDSQL) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	return []cloudprovider.ICloudDBInstanceBackup{}, nil
}

func (self *STDSQL) GetIDBInstanceParameters() ([]cloudprovider.ICloudDBInstanceParameter, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) DeletePostpaidTDSQL(id string) error {
	params := map[string]string{
		"InstanceId": id,
	}
	_, err := self.dcdbRequest("DestroyHourDCDBInstance", params)
	return errors.Wrapf(err, "DestroyHourDCDBInstance")
}

func (self *SRegion) DeletePrepaidTDSQL(id string) error {
	params := map[string]string{
		"InstanceId": id,
	}
	_, err := self.dcdbRequest("DestroyDCDBInstance", params)
	return errors.Wrapf(err, "DestroyDCDBInstance")
}
