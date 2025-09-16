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

type SElasticache struct {
	multicloud.SElasticcacheBase
	AwsTags
	region *SRegion

	ARN                                string                         `xml:"ARN"`
	AtRestEncryptionEnabled            bool                           `xml:"AtRestEncryptionEnabled"`
	AuthTokenEnabled                   bool                           `xml:"AuthTokenEnabled"`
	AuthTokenLastModifiedDate          time.Time                      `xml:"AuthTokenLastModifiedDate"`
	AutoMinorVersionUpgrade            bool                           `xml:"AutoMinorVersionUpgrade"`
	CacheClusterCreateTime             time.Time                      `xml:"CacheClusterCreateTime"`
	CacheClusterId                     string                         `xml:"CacheClusterId"`
	CacheClusterStatus                 string                         `xml:"CacheClusterStatus"`
	CacheNodeType                      string                         `xml:"CacheNodeType"`
	CacheNodes                         []CacheNode                    `xml:"CacheNodes>CacheNode"`
	CacheParameterGroup                CacheParameterGroupStatus      `xml:"CacheParameterGroup"`
	CacheSecurityGroups                []CacheSecurityGroupMembership `xml:"CacheSecurityGroups>CacheSecurityGroup"`
	CacheSubnetGroupName               string                         `xml:"CacheSubnetGroupName"`
	ClientDownloadLandingPage          string                         `xml:"ClientDownloadLandingPage"`
	ConfigurationEndpoint              Endpoint                       `xml:"ConfigurationEndpoint"`
	Engine                             string                         `xml:"Engine"`
	EngineVersion                      string                         `xml:"EngineVersion"`
	LogDeliveryConfigurations          []LogDeliveryConfiguration     `xml:"LogDeliveryConfigurations>LogDeliveryConfiguration"`
	NotificationConfiguration          NotificationConfiguration      `xml:"NotificationConfiguration"`
	NumCacheNodes                      int64                          `xml:"NumCacheNodes"`
	PendingModifiedValues              PendingModifiedValues          `xml:"PendingModifiedValues"`
	PreferredAvailabilityZone          string                         `xml:"PreferredAvailabilityZone"`
	PreferredMaintenanceWindow         string                         `xml:"PreferredMaintenanceWindow"`
	PreferredOutpostArn                string                         `xml:"PreferredOutpostArn"`
	ReplicationGroupId                 string                         `xml:"ReplicationGroupId"`
	ReplicationGroupLogDeliveryEnabled bool                           `xml:"ReplicationGroupLogDeliveryEnabled"`
	SecurityGroups                     []SecurityGroupMembership      `xml:"SecurityGroups>member"`
	SnapshotRetentionLimit             int64                          `xml:"SnapshotRetentionLimit"`
	SnapshotWindow                     string                         `xml:"SnapshotWindow"`
	TransitEncryptionEnabled           bool                           `xml:"TransitEncryptionEnabled"`
}

func (self *SElasticache) GetId() string {
	return self.CacheClusterId
}

func (self *SElasticache) GetName() string {
	if len(self.ReplicationGroupId) > 0 {
		return self.ReplicationGroupId
	}
	return self.CacheClusterId
}

func (self *SElasticache) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticache) GetStatus() string {
	// creating, available, modifying, deleting, create-failed, snapshotting
	switch self.CacheClusterStatus {
	case "creating":
		return api.ELASTIC_CACHE_STATUS_DEPLOYING
	case "available", "rebooting cluster nodes":
		return api.ELASTIC_CACHE_STATUS_RUNNING
	case "modifying":
		return api.ELASTIC_CACHE_STATUS_CHANGING
	case "deleting", "deleted":
		return api.ELASTIC_CACHE_STATUS_DELETING
	case "create-failed":
		return api.ELASTIC_CACHE_STATUS_CREATE_FAILED
	case "snapshotting":
		return api.ELASTIC_CACHE_STATUS_SNAPSHOTTING
	default:
		return self.CacheClusterStatus
	}
}

func (self *SElasticache) Refresh() error {
	caches, err := self.region.GetElasticaches(self.CacheClusterId, true)
	if err != nil {
		return err
	}
	for i := range caches {
		if caches[i].GetGlobalId() == self.GetGlobalId() {
			return jsonutils.Update(&self, caches[i])
		}
	}
	return errors.Wrapf(cloudprovider.ErrNotFound, self.GetGlobalId())
}

func (self *SElasticache) GetBillingType() string {
	return billingapi.BILLING_TYPE_POSTPAID
}

func (self *SElasticache) GetCreatedAt() time.Time {
	return self.CacheClusterCreateTime
}

func (self *SElasticache) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SElasticache) GetInstanceType() string {
	return self.CacheNodeType
}

func (self *SElasticache) GetCapacityMB() int {
	if sku, ok := redisInstanceType[self.CacheNodeType]; ok {
		return int(sku.MemeoryMb)
	}
	return 0
}

func (self *SElasticache) GetArchType() string {
	return api.ELASTIC_CACHE_ARCH_TYPE_SINGLE
}

func (self *SElasticache) GetNodeType() string {
	switch self.NumCacheNodes {
	case 1:
		return api.ELASTIC_CACHE_NODE_TYPE_SINGLE
	case 2:
		return api.ELASTIC_CACHE_NODE_TYPE_DOUBLE
	case 3:
		return api.ELASTIC_CACHE_NODE_TYPE_THREE
	case 4:
		return api.ELASTIC_CACHE_NODE_TYPE_FOUR
	case 5:
		return api.ELASTIC_CACHE_NODE_TYPE_FIVE
	case 6:
		return api.ELASTIC_CACHE_NODE_TYPE_SIX
	}
	return fmt.Sprintf("%d", self.NumCacheNodes)
}

func (self *SElasticache) GetEngine() string {
	return self.Engine
}

func (self *SElasticache) GetEngineVersion() string {
	return self.EngineVersion
}

func (self *SElasticache) GetVpcId() string {
	subnets, err := self.region.DescribeCacheSubnetGroups(self.CacheSubnetGroupName)
	if err != nil {
		return ""
	}
	for i := range subnets {
		return subnets[i].VpcId
	}
	return ""
}

func (self *SElasticache) GetZoneId() string {
	return ""
}

func (self *SElasticache) GetNetworkType() string {
	if len(self.CacheSubnetGroupName) > 0 {
		return api.LB_NETWORK_TYPE_VPC
	}
	return api.LB_NETWORK_TYPE_CLASSIC
}

func (self *SElasticache) GetNetworkId() string {
	return ""
}

// cluster mode(shard) 只有配置endpoint,no cluster mode 有primary/readonly endpoint
func (self *SElasticache) GetPrivateDNS() string {
	if len(self.ConfigurationEndpoint.Address) > 0 {
		return self.ConfigurationEndpoint.Address
	}
	return ""
}

func (self *SElasticache) GetPrivateIpAddr() string {
	return ""
}

func (self *SElasticache) GetTags() (map[string]string, error) {
	params := map[string]string{
		"ResourceName": self.ARN,
	}
	tags := AwsTags{}
	err := self.region.ecRequest("ListTagsForResource", params, &tags)
	if err != nil {
		return nil, errors.Wrapf(err, "ListTagsForResource")
	}
	return tags.GetTags()
}

func (self *SElasticache) SetTags(tags map[string]string, replace bool) error {
	oldTags, err := self.GetTags()
	if err != nil {
		return errors.Wrapf(err, "GetTags")
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
		params := map[string]string{
			"ResourceName": self.ARN,
		}
		i := 1
		for k := range tags {
			params[fmt.Sprintf("TagKeys.member.%d", i)] = k
			i++
		}
		return self.region.ecRequest("RemoveTagsFromResource", params, nil)
	}
	if len(added) > 0 {
		params := map[string]string{
			"ResourceName": self.ARN,
		}
		i := 1
		for k, v := range tags {
			params[fmt.Sprintf("Tags.member.%d.Key", i)] = k
			params[fmt.Sprintf("Tags.member.%d.Value", i)] = v
			i++
		}
		return self.region.ecRequest("AddTagsToResource", params, nil)
	}
	return nil
}

func (self *SElasticache) GetPrivateConnectPort() int {
	if self.ConfigurationEndpoint.Port > 0 {
		return int(self.ConfigurationEndpoint.Port)
	}
	return 0
}

func (self *SElasticache) GetPublicDNS() string {
	return ""
}

func (self *SElasticache) GetPublicIpAddr() string {
	return ""
}

func (self *SElasticache) GetPublicConnectPort() int {
	return 0
}

func (self *SElasticache) GetMaintainStartTime() string {
	splited := strings.Split(self.PreferredMaintenanceWindow, "-")
	if len(splited) == 2 {
		return strings.Trim(splited[0], "")
	}
	return ""
}

func (self *SElasticache) GetMaintainEndTime() string {
	splited := strings.Split(self.PreferredMaintenanceWindow, "-")
	if len(splited) == 2 {
		return strings.Trim(splited[1], "")
	}
	return ""
}

func (self *SElasticache) GetICloudElasticcacheAccounts() ([]cloudprovider.ICloudElasticcacheAccount, error) {
	return []cloudprovider.ICloudElasticcacheAccount{}, nil
}

func (self *SElasticache) GetICloudElasticcacheAccount(accountId string) (cloudprovider.ICloudElasticcacheAccount, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SElasticache) GetICloudElasticcacheAcls() ([]cloudprovider.ICloudElasticcacheAcl, error) {
	return []cloudprovider.ICloudElasticcacheAcl{}, nil
}

func (self *SElasticache) GetICloudElasticcacheAcl(aclId string) (cloudprovider.ICloudElasticcacheAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SElasticache) GetICloudElasticcacheBackups() ([]cloudprovider.ICloudElasticcacheBackup, error) {
	snapshots, err := self.region.GetCacheSnapshots(self.ReplicationGroupId, "")
	if err != nil {
		return nil, errors.Wrap(err, "self.GetSnapshots()")
	}
	result := []cloudprovider.ICloudElasticcacheBackup{}
	for i := range snapshots {
		snapshots[i].region = self.region
		result = append(result, &snapshots[i])
	}
	return result, nil
}

func (self *SElasticache) GetICloudElasticcacheBackup(backupId string) (cloudprovider.ICloudElasticcacheBackup, error) {
	snapshots, err := self.region.GetCacheSnapshots(self.ReplicationGroupId, backupId)
	if err != nil {
		return nil, errors.Wrap(err, "self.GetSnapshots()")
	}
	for i := range snapshots {
		if snapshots[i].GetId() == backupId {
			snapshots[i].region = self.region
			return &snapshots[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SElasticache) GetParameters() ([]SElasticacheParameter, error) {
	if len(self.CacheParameterGroup.CacheParameterGroupName) == 0 {
		return self.region.GetCacheParameters(self.CacheParameterGroup.CacheParameterGroupName)
	}
	return []SElasticacheParameter{}, nil
}

func (self *SElasticache) GetICloudElasticcacheParameters() ([]cloudprovider.ICloudElasticcacheParameter, error) {
	parameters, err := self.GetParameters()
	if err != nil {
		return nil, errors.Wrap(err, "self.GetParameters()")
	}
	result := []cloudprovider.ICloudElasticcacheParameter{}
	for i := range parameters {
		result = append(result, &parameters[i])
	}
	return result, nil
}

func (self *SElasticache) GetSecurityGroupIds() ([]string, error) {
	ret := []string{}
	for _, sec := range self.SecurityGroups {
		ret = append(ret, sec.SecurityGroupId)
	}
	return ret, nil
}

func (self *SElasticache) AllocatePublicConnection(port int) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SElasticache) ChangeInstanceSpec(spec string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SElasticache) CreateAccount(input cloudprovider.SCloudElasticCacheAccountInput) (cloudprovider.ICloudElasticcacheAccount, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SElasticache) CreateAcl(aclName, securityIps string) (cloudprovider.ICloudElasticcacheAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SElasticache) CreateBackup(desc string) (cloudprovider.ICloudElasticcacheBackup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SElasticache) Delete() error {
	return self.region.DeleteElastiCache(self.CacheClusterId)
}

func (self *SRegion) DeleteElastiCache(id string) error {
	params := map[string]string{
		"CacheClusterId": id,
	}
	return self.ecRequest("DeleteCacheCluster", params, nil)
}

func (self *SElasticache) FlushInstance(input cloudprovider.SCloudElasticCacheFlushInstanceInput) error {
	return cloudprovider.ErrNotSupported
}

func (self *SElasticache) GetAuthMode() string {
	return ""
}

func (self *SElasticache) ReleasePublicConnection() error {
	return cloudprovider.ErrNotSupported
}

func (self *SElasticache) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (self *SElasticache) Restart() error {
	return cloudprovider.ErrNotSupported
}

func (self *SElasticache) SetMaintainTime(maintainStartTime, maintainEndTime string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SElasticache) UpdateAuthMode(noPwdAccess bool, password string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SElasticache) UpdateBackupPolicy(config cloudprovider.SCloudElasticCacheBackupPolicyUpdateInput) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SElasticache) UpdateInstanceParameters(config jsonutils.JSONObject) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SElasticache) UpdateSecurityGroups(secgroupIds []string) error {
	return cloudprovider.ErrNotImplemented
}
