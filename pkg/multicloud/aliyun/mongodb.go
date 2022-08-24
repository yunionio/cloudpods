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
	"regexp"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SMOngoDBAttribute struct {
	// 实例最大IOPS
	MaxIops     int
	ReplicaSets struct {
		ReplicaSet []struct {
			ConnectionDomain string `json:"ConnectionDomain"`
		}
	}
	MaxConnections int
}

type SMongoDB struct {
	region *SRegion
	multicloud.AliyunTags
	multicloud.SBillingBase
	multicloud.SResourceBase

	ConnectionDomain string `json:"ConnectionDomain"`
	NetworkAddress   string
	ChargeType       TChargeType `json:"ChargeType"`
	LockMode         string      `json:"LockMode"`
	DBInstanceClass  string      `json:"DBInstanceClass"`
	ResourceGroupId  string      `json:"ResourceGroupId"`
	DBInstanceId     string      `json:"DBInstanceId"`
	ZoneId           string      `json:"ZoneId"`
	MongosList       struct {
		MongosAttribute []struct {
			NodeId    string `json:"NodeId"`
			NodeClass string `json:"NodeClass"`
		} `json:"MongosAttribute"`
	} `json:"MongosList"`
	DBInstanceDescription string    `json:"DBInstanceDescription"`
	Engine                string    `json:"Engine"`
	CreationTime          time.Time `json:"CreationTime"`
	NetworkType           string    `json:"NetworkType"`
	ExpireTime            time.Time `json:"ExpireTime"`
	DBInstanceType        string    `json:"DBInstanceType"`
	RegionId              string    `json:"RegionId"`
	ShardList             struct {
		ShardAttribute []struct {
			NodeId      string `json:"NodeId"`
			NodeClass   string `json:"NodeClass"`
			NodeStorage int    `json:"NodeStorage"`
		} `json:"ShardAttribute"`
	} `json:"ShardList"`
	EngineVersion    string `json:"EngineVersion"`
	DBInstanceStatus string `json:"DBInstanceStatus"`

	DBInstanceStorage int    `json:"DBInstanceStorage"`
	MaintainStartTime string `json:"MaintainStartTime"`
	MaintainEndTime   string `json:"MaintainEndTime"`
	StorageEngine     string `json:"StorageEngine"`
	VpcId             string `json:"VPCId"`
	VSwitchId         string `json:"VSwitchId"`
	VpcAuthMode       string `json:"VpcAuthMode"`
	ReplicationFactor string `json:"ReplicationFactor"`
}

var mongoSpec = map[string]struct {
	VcpuCount  int
	VmemSizeGb int
}{}

func (self *SMongoDB) GetName() string {
	if len(self.DBInstanceDescription) > 0 {
		return self.DBInstanceDescription
	}
	return self.DBInstanceId
}

func (self *SMongoDB) GetId() string {
	return self.DBInstanceId
}

func (self *SMongoDB) GetGlobalId() string {
	return self.DBInstanceId
}

func (self *SMongoDB) GetStatus() string {
	switch self.DBInstanceStatus {
	case "Creating":
		return api.MONGO_DB_STATUS_CREATING
	case "DBInstanceClassChanging":
		return api.MONGO_DB_STATUS_CHANGE_CONFIG
	case "DBInstanceNetTypeChanging", "EngineVersionUpgrading", "GuardSwitching", "HASwitching", "Importing", "ImportingFromOthers", "LinkSwitching", "MinorVersionUpgrading", "NET_CREATING", "NET_DELETING", "NodeCreating", "NodeDeleting", "Restoring", "SSLModifying", "TempDBInstanceCreating", "Transing", "TransingToOthers":
		return api.MONGO_DB_STATUS_DEPLOY
	case "Deleting":
		return api.MONGO_DB_STATUS_DELETING
	case "Rebooting":
		return api.MONGO_DB_STATUS_REBOOTING
	case "Running":
		return api.MONGO_DB_STATUS_RUNNING
	default:
		return strings.ToLower(self.DBInstanceStatus)
	}
}

func (self *SMongoDB) GetProjectId() string {
	return self.ResourceGroupId
}

func (self *SMongoDB) Refresh() error {
	db, err := self.region.GetMongoDB(self.DBInstanceId)
	if err != nil {
		return errors.Wrapf(err, "GetMongoDB")
	}
	return jsonutils.Update(self, db)
}

func (self *SMongoDB) GetCreatedAt() time.Time {
	return self.CreationTime
}

func (self *SMongoDB) GetExpiredAt() time.Time {
	return self.ExpireTime
}

func (self *SMongoDB) GetIpAddr() string {
	return ""
}

func (self *SMongoDB) GetEngine() string {
	if len(self.StorageEngine) == 0 {
		self.Refresh()
	}
	return self.StorageEngine
}

func (self *SMongoDB) GetEngineVersion() string {
	return self.EngineVersion
}

func (self *SMongoDB) GetVpcId() string {
	if self.NetworkType != "VPC" {
		return ""
	}
	if len(self.VpcId) == 0 {
		self.Refresh()
	}
	return self.VpcId
}

func (self *SMongoDB) GetNetworkId() string {
	if self.NetworkType != "VPC" {
		return ""
	}
	if len(self.VSwitchId) == 0 {
		self.Refresh()
	}
	return self.VSwitchId
}

func (self *SMongoDB) GetZoneId() string {
	if !strings.Contains(self.ZoneId, ",") {
		return self.ZoneId
	}
	if index := strings.Index(self.ZoneId, ",") - 1; index > 0 {
		return fmt.Sprintf("%s-%s", self.region.RegionId, string(self.ZoneId[index]))
	}
	return ""
}

func (self *SMongoDB) Delete() error {
	return self.region.DeleteMongoDB(self.DBInstanceId)
}

func (self *SMongoDB) GetBillingType() string {
	return convertChargeType(self.ChargeType)
}

func (self *SMongoDB) GetCategory() string {
	return self.DBInstanceType
}

func (self *SMongoDB) GetDiskSizeMb() int {
	if self.DBInstanceStorage == 0 {
		self.Refresh()
	}
	return self.DBInstanceStorage * 1024
}

func (self *SMongoDB) GetInstanceType() string {
	return self.DBInstanceClass
}

func (self *SMongoDB) GetMaintainTime() string {
	return fmt.Sprintf("%s-%s", self.MaintainStartTime, self.MaintainEndTime)
}

func (self *SMongoDB) GetPort() int {
	return 3717
}

func (self *SMongoDB) GetReplicationNum() int {
	if len(self.ReplicationFactor) == 0 {
		self.Refresh()
	}
	num, _ := strconv.Atoi(self.ReplicationFactor)
	return int(num)
}

func (self *SMongoDB) GetVcpuCount() int {
	self.region.GetchMongoSkus()
	sku, ok := self.region.mongoSkus[self.DBInstanceClass]
	if ok {
		return sku.CpuCount
	}
	return 0
}

func (self *SMongoDB) GetVmemSizeMb() int {
	self.region.GetchMongoSkus()
	sku, ok := self.region.mongoSkus[self.DBInstanceClass]
	if ok {
		return sku.MemSizeGb * 1024
	}
	return 0
}

func (self *SMongoDB) GetIops() int {
	iops, _ := self.region.GetIops(self.DBInstanceId)
	return iops
}

func (self *SMongoDB) GetMaxConnections() int {
	maxConnection, _ := self.region.GetMaxConnections(self.DBInstanceId)
	return maxConnection
}

func (self *SMongoDB) GetNetworkAddress() string {
	addr, _ := self.region.GetNetworkAddress(self.DBInstanceId)
	return addr
}

func (self *SRegion) GetMongoDBsByType(mongoType string) ([]SMongoDB, error) {
	dbs := []SMongoDB{}
	for {
		part, total, err := self.GetMongoDBs(mongoType, 100, len(dbs)/100)
		if err != nil {
			return nil, errors.Wrapf(err, "GetMongoDB")
		}
		dbs = append(dbs, part...)
		if len(dbs) >= total {
			break
		}
	}
	return dbs, nil
}

func (self *SMongoDB) SetTags(tags map[string]string, replace bool) error {
	return self.region.SetResourceTags(ALIYUN_SERVICE_MONGO_DB, "INSTANCE", self.GetId(), tags, replace)
}

func (self *SRegion) GetICloudMongoDBById(id string) (cloudprovider.ICloudMongoDB, error) {
	db, err := self.GetMongoDB(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetMongoDB(%s)", id)
	}
	return db, nil
}

func (self *SRegion) GetICloudMongoDBs() ([]cloudprovider.ICloudMongoDB, error) {
	dbs := []SMongoDB{}
	for _, mongoType := range []string{"sharding", "replicate", "serverless"} {
		part, err := self.GetMongoDBsByType(mongoType)
		if err != nil {
			return nil, err
		}
		dbs = append(dbs, part...)
	}
	ret := []cloudprovider.ICloudMongoDB{}
	for i := range dbs {
		dbs[i].region = self
		ret = append(ret, &dbs[i])
	}
	return ret, nil
}

func (self *SRegion) GetIops(id string) (int, error) {
	ret, err := self.GetMongoDBAttribute(id)
	if err != nil {
		return 0, errors.Wrapf(err, "DescribeDBInstanceAttribute err")
	}
	if len(ret) == 0 {
		return 0, errors.Wrapf(err, "ret missing err")
	}
	return ret[0].MaxIops, nil
}

func (self *SRegion) GetMaxConnections(id string) (int, error) {
	ret, err := self.GetMongoDBAttribute(id)
	if err != nil {
		return 0, errors.Wrapf(err, "DescribeDBInstanceAttribute err")
	}
	if len(ret) == 0 {
		return 0, errors.Wrapf(err, "ret missing err")
	}
	return ret[0].MaxConnections, nil
}

func (self *SRegion) GetNetworkAddress(id string) (string, error) {
	ret, err := self.GetMongoDBAttribute(id)
	if err != nil {
		return "", errors.Wrapf(err, "DescribeDBInstanceAttribute err")
	}
	addrList := make([]string, 0)
	for _, v := range ret[0].ReplicaSets.ReplicaSet {
		addrList = append(addrList, v.ConnectionDomain)
	}
	addrs := strings.Join(addrList, ",")
	return addrs, nil
}

func (self *SRegion) GetMongoDBAttribute(id string) ([]SMOngoDBAttribute, error) {
	params := map[string]string{
		"Action":       "DescribeDBInstanceAttribute",
		"DBInstanceId": id,
	}
	resp, err := self.mongodbRequest("DescribeDBInstanceAttribute", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeDBInstanceAttribute err")
	}
	ret := []SMOngoDBAttribute{}
	err = resp.Unmarshal(&ret, "DBInstances", "DBInstance")
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshal err")
	}
	return ret, err
}

func (self *SRegion) GetMongoDBs(mongoType string, pageSize int, pageNum int) ([]SMongoDB, int, error) {
	if pageSize < 1 || pageSize > 100 {
		pageSize = 100
	}
	if pageNum < 1 {
		pageNum = 1
	}

	params := map[string]string{
		"PageSize":   fmt.Sprintf("%d", pageSize),
		"PageNumber": fmt.Sprintf("%d", pageNum),
	}
	if len(mongoType) > 0 {
		params["DBInstanceType"] = mongoType
	}
	resp, err := self.mongodbRequest("DescribeDBInstances", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeDBInstances")
	}
	ret := []SMongoDB{}
	err = resp.Unmarshal(&ret, "DBInstances", "DBInstance")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	totalCount, _ := resp.Int("TotalCount")
	return ret, int(totalCount), nil
}

func (self *SRegion) GetMongoDB(id string) (*SMongoDB, error) {
	params := map[string]string{
		"DBInstanceId": id,
	}
	resp, err := self.mongodbRequest("DescribeDBInstanceAttribute", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeDBInstanceAttribute")
	}
	ret := []SMongoDB{}
	err = resp.Unmarshal(&ret, "DBInstances", "DBInstance")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	if len(ret) == 1 {
		ret[0].region = self
		return &ret[0], nil
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) DeleteMongoDB(id string) error {
	params := map[string]string{
		"DBInstanceId": id,
		"ClientToken":  utils.GenRequestId(20),
	}
	_, err := self.mongodbRequest("DeleteDBInstance", params)
	return errors.Wrapf(err, "DeleteDBInstance")
}

type SMongoDBAvaibaleResource struct {
	SupportedDBTypes struct {
		SupportedDBType []struct {
			DbType         string
			AvailableZones struct {
				AvailableZone []struct {
					ZoneId                  string
					RegionId                string
					SupportedEngineVersions struct {
						SupportedEngineVersion []struct {
							Version          string
							SupportedEngines struct {
								SupportedEngine []struct {
									SupportedNodeTypes struct {
										SupportedNodeType []struct {
											NetworkTypes       string
											NodeType           string
											AvailableResources struct {
												AvailableResource []struct {
													InstanceClassRemark string
													InstanceClass       string
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
}

func (self *SRegion) GetchMongoSkus() (map[string]struct {
	CpuCount  int
	MemSizeGb int
}, error) {
	if len(self.mongoSkus) > 0 {
		return self.mongoSkus, nil
	}
	self.mongoSkus = map[string]struct {
		CpuCount  int
		MemSizeGb int
	}{}
	res, err := self.GetMongoDBAvailableResource()
	if err != nil {
		return nil, err
	}
	for _, dbType := range res.SupportedDBTypes.SupportedDBType {
		for _, zone := range dbType.AvailableZones.AvailableZone {
			for _, version := range zone.SupportedEngineVersions.SupportedEngineVersion {
				for _, engine := range version.SupportedEngines.SupportedEngine {
					for _, nodeType := range engine.SupportedNodeTypes.SupportedNodeType {
						for _, sku := range nodeType.AvailableResources.AvailableResource {
							_, ok := self.mongoSkus[sku.InstanceClass]
							if !ok {
								self.mongoSkus[sku.InstanceClass] = getMongoDBSkuDetails(sku.InstanceClassRemark)
							}
						}
					}
				}
			}
		}
	}
	return self.mongoSkus, nil
}

func getMongoDBSkuDetails(remark string) struct {
	CpuCount  int
	MemSizeGb int
} {
	ret := struct {
		CpuCount  int
		MemSizeGb int
	}{}
	r, _ := regexp.Compile(`(\d{1,3})核(\d{1,3})G+`)
	result := r.FindSubmatch([]byte(remark))
	if len(result) > 2 {
		cpu, _ := strconv.Atoi(string(result[1]))
		ret.CpuCount = int(cpu)
		mem, _ := strconv.Atoi(string(result[2]))
		ret.MemSizeGb = int(mem)
	} else {
		log.Warningf("not match sku remark %s", remark)
	}
	return ret
}

func (self *SRegion) GetMongoDBAvailableResource() (*SMongoDBAvaibaleResource, error) {
	params := map[string]string{}
	resp, err := self.mongodbRequest("DescribeAvailableResource", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeAvailableResource")
	}
	ret := &SMongoDBAvaibaleResource{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	return ret, nil
}
