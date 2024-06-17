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
	"fmt"
	"strings"
	"time"

	sdkerrors "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SMongoDB struct {
	QcloudTags
	multicloud.SVirtualResourceBase
	multicloud.SBillingBase
	multicloud.SMongodbBase

	region *SRegion

	IOPS                int
	NetworkAddress      string
	AutoRenewFlag       int           `json:"AutoRenewFlag"`
	CloneInstances      []interface{} `json:"CloneInstances"`
	ClusterType         int           `json:"ClusterType"`
	ClusterVer          int           `json:"ClusterVer"`
	ConfigServerCpuNum  int           `json:"ConfigServerCpuNum"`
	ConfigServerMemory  int           `json:"ConfigServerMemory"`
	ConfigServerNodeNum int           `json:"ConfigServerNodeNum"`
	ConfigServerVolume  int           `json:"ConfigServerVolume"`
	CpuNum              int           `json:"CpuNum"`
	CreateTime          time.Time     `json:"CreateTime"`
	DeadLine            string        `json:"DeadLine"`
	InstanceId          string        `json:"InstanceId"`
	InstanceName        string        `json:"InstanceName"`
	InstanceStatusDesc  string        `json:"InstanceStatusDesc"`
	InstanceTaskDesc    string        `json:"InstanceTaskDesc"`
	InstanceTaskId      int           `json:"InstanceTaskId"`
	InstanceType        int           `json:"InstanceType"`
	InstanceVer         int           `json:"InstanceVer"`
	MachineType         string        `json:"MachineType"`
	MaintenanceEnd      string        `json:"MaintenanceEnd"`
	MaintenanceStart    string        `json:"MaintenanceStart"`
	Memory              int           `json:"Memory"`
	MongoVersion        string        `json:"MongoVersion"`
	MongosCpuNum        int           `json:"MongosCpuNum"`
	MongosMemory        int           `json:"MongosMemory"`
	MongosNodeNum       int           `json:"MongosNodeNum"`
	NetType             int           `json:"NetType"`
	PayMode             int           `json:"PayMode"`
	ProjectId           int           `json:"ProjectId"`
	Protocol            int           `json:"Protocol"`
	Readonlyinstances   []interface{} `json:"ReadonlyInstances"`
	RealInstanceId      string        `json:"RealInstanceId"`
	Region              string        `json:"Region"`
	Relatedinstance     struct {
		InstanceId string `json:"InstanceId"`
		Region     string `json:"Region"`
	} `json:"RelatedInstance"`
	Replicasets []struct {
		Memory           int    `json:"Memory"`
		OplogSize        int    `json:"OplogSize"`
		RealReplicasetId string `json:"RealReplicaSetId"`
		ReplicaSetId     string `json:"ReplicaSetId"`
		ReplicaSetName   string `json:"ReplicaSetName"`
		SecondaryNum     int    `json:"SecondaryNum"`
		UsedVolume       int    `json:"UsedVolume"`
		Volume           int    `json:"Volume"`
	} `json:"ReplicaSets"`
	ReplicationSetNum int           `json:"ReplicationSetNum"`
	SecondaryNum      int           `json:"SecondaryNum"`
	StandbyInstances  []interface{} `json:"StandbyInstances"`
	Status            int           `json:"Status"`
	SubnetId          string        `json:"SubnetId"`
	UsedVolume        int           `json:"UsedVolume"`
	Vip               string        `json:"Vip"`
	Volume            int           `json:"Volume"`
	VpcId             string        `json:"VpcId"`
	Vport             int           `json:"Vport"`
	Zone              string        `json:"Zone"`
}

func (self *SMongoDB) GetGlobalId() string {
	return self.InstanceId
}

func (self *SMongoDB) GetId() string {
	return self.InstanceId
}

func (self *SMongoDB) GetName() string {
	return self.InstanceName
}

func (self *SMongoDB) GetStatus() string {
	switch self.Status {
	case 0:
		return api.MONGO_DB_STATUS_CREATING
	case 1:
		return api.MONGO_DB_STATUS_PROCESSING
	case 2:
		return api.MONGO_DB_STATUS_RUNNING
	case -2, -3:
		return api.MONGO_DB_STATUS_DELETING
	}
	return fmt.Sprintf("%d", self.Status)
}

func (self *SMongoDB) Refresh() error {
	ins, err := self.region.GetMongoDB(self.InstanceId)
	if err != nil {
		return errors.Wrapf(err, "GetMongoDB")
	}
	return jsonutils.Update(self, ins)
}

func (self *SMongoDB) GetProjectId() string {
	return fmt.Sprintf("%d", self.ProjectId)
}

func (self *SMongoDB) GetVpcId() string {
	return self.VpcId
}

func (self *SMongoDB) GetNetworkId() string {
	return self.SubnetId
}

func (self *SMongoDB) GetCreatedAt() time.Time {
	return self.CreateTime.Add(time.Hour * -8)
}

func (self *SMongoDB) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SMongoDB) GetIpAddr() string {
	return self.Vip
}

func (self *SMongoDB) GetVcpuCount() int {
	return self.CpuNum
}

func (self *SMongoDB) GetVmemSizeMb() int {
	return self.Memory
}

func (self *SMongoDB) GetReplicationNum() int {
	switch self.GetCategory() {
	case api.MONGO_DB_CATEGORY_SHARDING:
		return self.ReplicationSetNum
	case api.MONGO_DB_CATEGORY_REPLICATE:
		return 3
	}
	return self.ReplicationSetNum
}

func (self *SMongoDB) GetDiskSizeMb() int {
	return self.Volume
}

func (self *SMongoDB) GetZoneId() string {
	return self.Zone
}

func (self *SMongoDB) GetBillingType() string {
	// 计费模式：0-按量计费，1-包年包月
	if self.PayMode == 1 {
		return billing_api.BILLING_TYPE_PREPAID
	} else {
		return billing_api.BILLING_TYPE_POSTPAID
	}
}

func (self *SMongoDB) IsAutoRenew() bool {
	return self.AutoRenewFlag == 1
}

func (self *SMongoDB) GetCategory() string {
	switch self.ClusterType {
	case 0:
		return api.MONGO_DB_CATEGORY_REPLICATE
	case 1:
		return api.MONGO_DB_CATEGORY_SHARDING
	default:
		return fmt.Sprintf("%d", self.ClusterType)
	}
}

func (self *SMongoDB) GetEngine() string {
	if utils.IsInStringArray("WT", strings.Split(self.MongoVersion, "_")) {
		return api.MONGO_DB_ENGINE_WIRED_TIGER
	}
	return api.MONGO_DB_ENGINE_ROCKS
}

func (self *SMongoDB) GetEngineVersion() string {
	vers := strings.Split(self.MongoVersion, "_")
	if len(vers) > 1 {
		return strings.Join(strings.Split(vers[1], ""), ".")
	}
	return ""
}

func (self *SMongoDB) GetInstanceType() string {
	return self.MachineType
}

func (self *SMongoDB) GetMaintainTime() string {
	return fmt.Sprintf("%s-%s", self.MaintenanceStart, self.MaintenanceEnd)
}

func (self *SMongoDB) GetPort() int {
	return self.Vport
}

func (self *SMongoDB) Delete() error {
	return self.region.DeleteMongoDB(self.InstanceId)
}

func (self *SRegion) DeleteMongoDB(id string) error {
	err := self.IsolateMongoDB(id)
	if err != nil {
		return errors.Wrapf(err, "IsolateDBInstance")
	}
	return cloudprovider.Wait(time.Second*10, time.Minute*3, func() (bool, error) {
		err = self.OfflineIsolatedMongoDB(id)
		if err == nil {
			return true, nil
		}
		if e, ok := errors.Cause(err).(*sdkerrors.TencentCloudSDKError); ok && e.Code == "InvalidParameterValue.LockFailed" {
			return false, nil
		}
		return true, err
	})
}

func (self *SMongoDB) SetTags(tags map[string]string, replace bool) error {
	return self.region.SetResourceTags("mongodb", "instance", []string{self.InstanceId}, tags, replace)
}

func (self *SMongoDB) GetIBackups() ([]cloudprovider.SMongoDBBackup, error) {
	return self.region.GetMongoDBBackups(self.InstanceId)
}

func (self *SRegion) IsolateMongoDB(id string) error {
	params := map[string]string{
		"InstanceId": id,
	}
	_, err := self.mongodbRequest("IsolateDBInstance", params)
	return errors.Wrapf(err, "IsolateDBInstance")
}

func (self *SRegion) OfflineIsolatedMongoDB(id string) error {
	params := map[string]string{
		"InstanceId": id,
	}
	_, err := self.mongodbRequest("OfflineIsolatedDBInstance", params)
	return errors.Wrapf(err, "OfflineIsolatedDBInstance")
}

func (self *SRegion) GetMongoDBs(ids []string, limit, offset int) ([]SMongoDB, int, error) {
	if limit < 1 || limit > 100 {
		limit = 100
	}
	params := map[string]string{
		"Limit":  fmt.Sprintf("%d", limit),
		"Offset": fmt.Sprintf("%d", offset),
	}
	for i, id := range ids {
		params[fmt.Sprintf("InstanceIds.%d", i)] = id
	}
	resp, err := self.mongodbRequest("DescribeDBInstances", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeDBInstances")
	}
	dbs := []SMongoDB{}
	err = resp.Unmarshal(&dbs, "InstanceDetails")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	totalCount, _ := resp.Float("TotalCount")
	return dbs, int(totalCount), nil
}

func (self *SMongoDB) GetIops() int {
	return self.IOPS
}

func (self *SMongoDB) GetNetworkAddress() string {
	return self.NetworkAddress
}

func (self *SRegion) GetICloudMongoDBs() ([]cloudprovider.ICloudMongoDB, error) {
	dbs := []SMongoDB{}
	for {
		part, total, err := self.GetMongoDBs(nil, 100, len(dbs))
		if err != nil {
			return nil, errors.Wrapf(err, "GetMongoDBs")
		}
		dbs = append(dbs, part...)
		if len(dbs) >= total {
			break
		}
	}
	ret := []cloudprovider.ICloudMongoDB{}
	for i := range dbs {
		dbs[i].region = self
		ret = append(ret, &dbs[i])
	}
	return ret, nil
}

func (self *SRegion) GetMongoDBBackups(id string) ([]cloudprovider.SMongoDBBackup, error) {
	params := map[string]string{
		"BackupMethod": "2",
		"InstanceId":   id,
	}
	resp, err := self.mongodbRequest("DescribeDBBackups", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeDBBackups")
	}
	backups := []struct {
		InstanceId   string
		BackupType   int
		BackupName   string
		BackupDesc   string
		BackupSize   int
		StartTime    time.Time
		EndTime      time.Time
		Status       int
		BackupMethod int
	}{}
	err = resp.Unmarshal(&backups, "BackupList")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	ret := []cloudprovider.SMongoDBBackup{}
	for _, backup := range backups {
		b := cloudprovider.SMongoDBBackup{
			Name:         backup.BackupName,
			Description:  backup.BackupDesc,
			BackupSizeKb: backup.BackupSize,
		}
		b.StartTime = backup.StartTime.Add(time.Hour * -8)
		b.EndTime = backup.EndTime.Add(time.Hour * -8)
		switch backup.Status {
		case 1:
			b.Status = cloudprovider.MongoDBBackupStatusCreating
		case 2:
			b.Status = cloudprovider.MongoDBBackupStatusAvailable
		default:
			b.Status = cloudprovider.MongoDBBackupStatusUnknown
		}
		b.BackupMethod = cloudprovider.MongoDBBackupMethodLogical
		if backup.BackupMethod == 0 {
			b.BackupMethod = cloudprovider.MongoDBBackupMethodPhysical
		}
		b.BackupType = cloudprovider.MongoDBBackupTypeAuto
		if backup.BackupType == 1 {
			b.BackupType = cloudprovider.MongoDBBackupTypeManual
		}
		ret = append(ret, b)
	}
	return ret, nil
}

func (self *SRegion) GetMongoDB(id string) (*SMongoDB, error) {
	dbs, _, err := self.GetMongoDBs([]string{id}, 1, 0)
	if err != nil {
		return nil, errors.Wrapf(err, "GetMongoDB(%s)", id)
	}
	for i := range dbs {
		dbs[i].region = self
		return &dbs[i], nil
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) GetICloudMongoDBById(id string) (cloudprovider.ICloudMongoDB, error) {
	db, err := self.GetMongoDB(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetMongoDB")
	}
	return db, nil
}
