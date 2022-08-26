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

package jdcloud

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	commodels "github.com/jdcloud-api/jdcloud-sdk-go/services/common/models"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/rds/apis"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/rds/client"
	"github.com/jdcloud-api/jdcloud-sdk-go/services/rds/models"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SDBInstance struct {
	multicloud.SDBInstanceBase
	multicloud.JdcloudTags
	region *SRegion

	models.DBInstance
}

func (self *SDBInstance) GetName() string {
	return self.InstanceName
}

func (self *SDBInstance) GetInstanceType() string {
	return self.InstanceClass
}

func (self *SDBInstance) GetCategory() string {
	return self.InstanceType
}

func (self *SDBInstance) GetDiskSizeGB() int {
	return self.InstanceStorageGB
}

func (self *SDBInstance) GetEngine() string {
	return self.Engine
}

func (self *SDBInstance) GetEngineVersion() string {
	return self.EngineVersion
}

func (self *SDBInstance) GetGlobalId() string {
	return self.InstanceId
}

func (self *SDBInstance) GetId() string {
	return self.InstanceId
}

func (self *SDBInstance) GetIVpcId() string {
	return self.VpcId
}

func (self *SDBInstance) Refresh() error {
	rds, err := self.region.GetDBInstance(self.InstanceId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, rds)
}

func (self *SDBInstance) GetMaintainTime() string {
	return ""
}

func (self *SDBInstance) GetMasterInstanceId() string {
	return self.SourceInstanceId
}

func (self *SDBInstance) GetPort() int {
	if len(self.InstancePort) == 0 {
		self.Refresh()
	}
	port, _ := strconv.Atoi(self.InstancePort)
	return int(port)
}

func (self *SDBInstance) GetCreatedAt() time.Time {
	return parseTime(self.CreateTime)
}

func (self *SDBInstance) GetExpiredAt() time.Time {
	return expireAt(&self.Charge)
}

func (self *SDBInstance) GetStatus() string {
	switch self.InstanceStatus {
	case "BUILDING", "DDLING", "PARAMETERGROUP_MODIFYING", "AUDIT_OPENING", "AUDIT_CLOSING", "SECURITY_OPENING", "SECURITY_CLOSING", "SSL_OPENING", "SSL_CLOSING":
		return api.DBINSTANCE_DEPLOYING
	case "RUNNING":
		return api.DBINSTANCE_RUNNING
	case "DELETING":
		return api.DBINSTANCE_DELETING
	case "FAILOVER", "AZ_MIGRATING":
		return api.DBINSTANCE_MIGRATING
	case "RESTORING", "DB_RESTORING":
		return api.DBINSTANCE_RESTORING
	case "MODIFYING":
		return api.DBINSTANCE_CHANGE_CONFIG
	case "BUILD_READONLY":
		return api.DBINSTANCE_DEPLOYING
	case "REBOOTING":
		return api.DBINSTANCE_REBOOTING
	case "MAINTENANCE":
		return api.DBINSTANCE_MAINTENANCE
	default:
		return self.InstanceStatus
	}
}

func (self *SDBInstance) GetStorageType() string {
	return self.InstanceStorageType
}

func (self *SDBInstance) GetVcpuCount() int {
	return self.InstanceCPU
}

func (self *SDBInstance) GetVmemSizeMB() int {
	return self.InstanceMemoryMB
}

func (self *SDBInstance) GetZone1Id() string {
	if len(self.AzId) > 0 {
		return self.AzId[0]
	}
	return ""
}

func (self *SDBInstance) GetZone2Id() string {
	if len(self.AzId) > 1 {
		return self.AzId[1]
	}
	return ""
}

func (self *SDBInstance) GetZone3Id() string {
	if len(self.AzId) > 2 {
		return self.AzId[2]
	}
	return ""
}

func (self *SDBInstance) GetConnectionStr() string {
	return self.PublicDomainName
}

func (self *SDBInstance) GetInternalConnectionStr() string {
	return self.InternalDomainName
}

func (self *SDBInstance) Delete() error {
	return self.region.DeleteDBInstance(self.InstanceId)
}

func (self *SRegion) GetIDBInstances() ([]cloudprovider.ICloudDBInstance, error) {
	rds := []SDBInstance{}
	n := 1
	for {
		part, total, err := self.GetDBInstances(n, 100)
		if err != nil {
			return nil, errors.Wrapf(err, "GetDBInstances")
		}
		rds = append(rds, part...)
		if len(rds) >= total {
			break
		}
		n++
	}
	ret := []cloudprovider.ICloudDBInstance{}
	for i := range rds {
		ret = append(ret, &rds[i])
	}
	return ret, nil
}

func (self *SRegion) GetIDBInstanceById(id string) (cloudprovider.ICloudDBInstance, error) {
	rds, err := self.GetDBInstance(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetDBInstance(%s)", id)
	}
	return rds, nil
}

func (self *SRegion) GetDBInstance(id string) (*SDBInstance, error) {
	req := apis.NewDescribeInstanceAttributesRequest(self.ID, id)
	client := client.NewRdsClient(self.getCredential())
	client.Logger = Logger{debug: self.client.debug}
	resp, err := client.DescribeInstanceAttributes(req)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeInstanceAttributes")
	}
	if resp.Error.Code == 404 || strings.Contains(resp.Error.Status, "NotFound") {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, jsonutils.Marshal(resp.Error).String())
	} else if resp.Error.Code != 0 {
		return nil, errors.Error(jsonutils.Marshal(resp.Error).String())
	}
	ret := SDBInstance{
		region: self,
	}
	jsonutils.Update(&ret.DBInstance, resp.Result.DbInstanceAttributes)
	return &ret, nil
}

func (self *SRegion) DeleteDBInstance(id string) error {
	req := apis.NewDeleteInstanceRequest(self.ID, id)
	client := client.NewRdsClient(self.getCredential())
	client.Logger = Logger{}
	resp, err := client.DeleteInstance(req)
	if err != nil {
		return err
	}
	if resp.Error.Code == 404 {
		return nil
	}
	return nil
}

func (self *SRegion) GetDBInstances(pageNumber int, pageSize int) ([]SDBInstance, int, error) {
	req := apis.NewDescribeInstancesRequestWithAllParams(self.ID, &pageNumber, &pageSize, []commodels.Filter{}, []commodels.TagFilter{})
	client := client.NewRdsClient(self.getCredential())
	client.Logger = Logger{debug: self.client.debug}
	resp, err := client.DescribeInstances(req)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeInstances")
	}
	if resp.Error.Code >= 400 {
		err = fmt.Errorf(resp.Error.Message)
		return nil, 0, err
	}
	total := resp.Result.TotalCount
	ret := []SDBInstance{}
	for i := range resp.Result.DbInstances {
		ret = append(ret, SDBInstance{
			region:     self,
			DBInstance: resp.Result.DbInstances[i],
		})
	}
	return ret, total, nil
}
