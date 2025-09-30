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

package ksyun

import (
	"fmt"
	"time"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type SDBInstance struct {
	region *SRegion

	multicloud.SDBInstanceBase
	SKsyunTags
	DBInstanceClass struct {
		Id    string
		Vcpus int
		Disk  int
		RAM   int
	}
	DBInstanceIdentifier             string
	DBInstanceName                   string
	DBInstanceStatus                 string
	DBInstanceType                   string
	DBParameterGroupId               string
	GroupId                          string
	Vip                              string
	Port                             int
	Engine                           string
	EngineVersion                    string
	InstanceCreateTime               string
	MasterUserName                   string
	VpcId                            string
	SubnetId                         string
	DiskUsed                         int
	PubliclyAccessible               bool
	ReadReplicaDBInstanceIdentifiers []interface{}
	BillType                         string
	OrderType                        string
	OrderSource                      string
	ProductType                      int
	MultiAvailabilityZone            bool
	MasterAvailabilityZone           string
	SlaveAvailabilityZone            string
	ProductId                        string
	OrderUse                         string
	SupportIPV6                      bool
	ProjectId                        string
	ProjectName                      string
	Region                           string
	BillTypeId                       int
	Eip                              string
	EipPort                          int
	NetworkType                      int
}

func (region *SRegion) GetIDBInstanceById(id string) (cloudprovider.ICloudDBInstance, error) {
	vm, err := region.GetDBInstance(id)
	if err != nil {
		return nil, err
	}
	return vm, nil
}

func (region *SRegion) GetIDBInstances() ([]cloudprovider.ICloudDBInstance, error) {
	vms, err := region.GetDBInstances("")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudDBInstance{}
	for i := range vms {
		vms[i].region = region
		ret = append(ret, &vms[i])
	}
	return ret, nil
}

func (region *SRegion) GetDBInstance(id string) (*SDBInstance, error) {
	vms, err := region.GetDBInstances(id)
	if err != nil {
		return nil, err
	}
	for i := range vms {
		if vms[i].GetGlobalId() == id {
			vms[i].region = region
			return &vms[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (region *SRegion) GetDBInstances(id string) ([]SDBInstance, error) {
	params := map[string]interface{}{}
	if len(id) > 0 {
		params["DBInstanceIdentifier"] = id
	}
	ret := []SDBInstance{}
	for {
		resp, err := region.rdsRequest("DescribeDBInstances", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			Instances  []SDBInstance
			TotalCount int
			Marker     string
		}{}
		err = resp.Unmarshal(&part, "Data")
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Instances...)
		if len(ret) >= part.TotalCount {
			break
		}
		params["Marker"] = part.Marker
	}
	return ret, nil
}

func (rds *SDBInstance) GetName() string {
	if len(rds.DBInstanceName) > 0 {
		return rds.DBInstanceName
	}
	return rds.DBInstanceIdentifier
}

func (rds *SDBInstance) GetId() string {
	return rds.DBInstanceIdentifier
}

func (rds *SDBInstance) GetGlobalId() string {
	return rds.GetId()
}

func (rds *SDBInstance) GetStatus() string {
	switch rds.DBInstanceStatus {
	case "ACTIVE":
		return api.DBINSTANCE_RUNNING
	default:
		log.Errorf("Unknown dbinstance status %s", rds.DBInstanceStatus)
		return api.DBINSTANCE_UNKNOWN
	}
}

func (rds *SDBInstance) GetBillingType() string {
	if rds.BillType != "HourlyInstantSettlement" {
		return billing_api.BILLING_TYPE_PREPAID
	}
	return billing_api.BILLING_TYPE_POSTPAID
}

func (rds *SDBInstance) GetExpiredAt() time.Time {
	return time.Time{}
}

func (rds *SDBInstance) GetCreatedAt() time.Time {
	t, _ := time.Parse("2006-01-02T15:04:05-0700", rds.InstanceCreateTime)
	return t
}

func (rds *SDBInstance) GetStorageType() string {
	return api.KSYUN_DBINSTANCE_STORAGE_TYPE_DEFAULT
}

func (rds *SDBInstance) GetEngine() string {
	return api.DBINSTANCE_TYPE_MYSQL
}

func (rds *SDBInstance) GetEngineVersion() string {
	return rds.EngineVersion
}

func (rds *SDBInstance) GetInstanceType() string {
	return rds.DBInstanceType
}

func (rds *SDBInstance) GetCategory() string {
	if rds.MultiAvailabilityZone {
		return api.ALIYUN_DBINSTANCE_CATEGORY_HA
	}
	return api.ALIYUN_DBINSTANCE_CATEGORY_BASIC
}

func (rds *SDBInstance) GetVcpuCount() int {
	return rds.DBInstanceClass.Vcpus
}

func (rds *SDBInstance) GetVmemSizeMB() int {
	return rds.DBInstanceClass.RAM * 1024
}

func (rds *SDBInstance) GetDiskSizeGB() int {
	return rds.DBInstanceClass.Disk * 1024
}

func (rds *SDBInstance) GetDiskSizeUsedMB() int {
	return rds.DiskUsed * 1024
}

func (rds *SDBInstance) GetPort() int {
	return rds.Port
}

func (rds *SDBInstance) GetMaintainTime() string {
	return ""
}

func (rds *SDBInstance) GetIVpcId() string {
	return rds.VpcId
}

func (rds *SDBInstance) Refresh() error {
	vm, err := rds.region.GetDBInstance(rds.DBInstanceIdentifier)
	if err != nil {
		return err
	}
	return jsonutils.Update(rds, vm)
}

func (rds *SDBInstance) GetZone1Id() string {
	return rds.MasterAvailabilityZone
}

func (rds *SDBInstance) GetZone2Id() string {
	if rds.SlaveAvailabilityZone != rds.MasterAvailabilityZone {
		return rds.SlaveAvailabilityZone
	}
	return ""
}

func (rds *SDBInstance) GetZone3Id() string {
	return ""
}

func (rds *SDBInstance) GetIOPS() int {
	return 0
}

func (rds *SDBInstance) GetNetworkAddress() string {
	return rds.Vip
}

func (rds *SDBInstance) GetDBNetworks() ([]cloudprovider.SDBInstanceNetwork, error) {
	return []cloudprovider.SDBInstanceNetwork{
		cloudprovider.SDBInstanceNetwork{
			IP:        rds.Vip,
			NetworkId: rds.SubnetId,
		},
	}, nil
}

func (rds *SDBInstance) GetInternalConnectionStr() string {
	return fmt.Sprintf("%s:%d", rds.Vip, rds.Port)
}

func (rds *SDBInstance) GetConnectionStr() string {
	if len(rds.Eip) > 0 {
		return fmt.Sprintf("%s:%d", rds.Eip, rds.EipPort)
	}
	return ""
}

func (rds *SDBInstance) GetProjectId() string {
	return rds.ProjectId
}

func (rds *SDBInstance) GetIDBInstanceDatabases() ([]cloudprovider.ICloudDBInstanceDatabase, error) {
	databases, err := rds.region.GetDBInstanceDatabases(rds.DBInstanceIdentifier)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudDBInstanceDatabase{}
	for i := 0; i < len(databases); i++ {
		databases[i].instance = rds
		ret = append(ret, &databases[i])
	}
	return ret, nil
}

func (rds *SDBInstance) Delete() error {
	return rds.region.DeleteDBInstance(rds.DBInstanceIdentifier)
}

func (region *SRegion) DeleteDBInstance(instanceId string) error {
	params := map[string]interface{}{}
	params["DBInstanceIdentifier"] = instanceId
	_, err := region.rdsRequest("DeleteDBInstance", params)
	return err
}
