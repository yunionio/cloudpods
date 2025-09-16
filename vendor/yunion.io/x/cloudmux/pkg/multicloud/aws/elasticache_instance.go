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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/utils"

	billingapi "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type Endpoint struct {
	Address string `xml:"Address"`
	Port    int    `xml:"Port"`
}

type CacheNode struct {
	CacheNodeCreateTime      time.Time `xml:"CacheNodeCreateTime"`
	CacheNodeId              string    `xml:"CacheNodeId"`
	CacheNodeStatus          string    `xml:"CacheNodeStatus"`
	CustomerAvailabilityZone string    `xml:"CustomerAvailabilityZone"`
	CustomerOutpostArn       string    `xml:"CustomerOutpostArn"`
	Endpoint                 Endpoint  `xml:"Endpoint"`
	ParameterGroupStatus     string    `xml:"ParameterGroupStatus"`
	SourceCacheNodeId        string    `xml:"SourceCacheNodeId"`
}

type CacheParameterGroupStatus struct {
	CacheNodeIdsToReboot    []string `xml:"CacheNodeIdsToReboot"`
	CacheParameterGroupName string   `xml:"CacheParameterGroupName"`
	ParameterApplyStatus    string   `xml:"ParameterApplyStatus"`
}

type CacheSecurityGroupMembership struct {
	CacheSecurityGroupName string `xml:"CacheSecurityGroupName"`
	Status                 string `xml:"Status"`
}

type LogDeliveryConfiguration struct {
	DestinationType string `xml:"DestinationType"`
	LogFormat       string `xml:"LogFormat"`
	LogType         string `xml:"LogType"`
	Message         string `xml:"Message"`
	Status          string `xml:"Status"`
}

type NotificationConfiguration struct {
	TopicArn    string `xml:"TopicArn"`
	TopicStatus string `xml:"TopicStatus"`
}

type PendingModifiedValues struct {
	AuthTokenStatus      string   `xml:"AuthTokenStatus"`
	CacheNodeIdsToRemove []string `xml:"CacheNodeIdsToRemove"`
	CacheNodeType        string   `xml:"CacheNodeType"`
	EngineVersion        string   `xml:"EngineVersion"`
	NumCacheNodes        int64    `xml:"NumCacheNodes"`
}

type SecurityGroupMembership struct {
	SecurityGroupId string `xml:"SecurityGroupId"`
	Status          string `xml:"Status"`
}

type SReplicationGroup struct {
	multicloud.SElasticcacheBase
	AwsTags
	region *SRegion

	cluster *SElasticache

	ARN                        string                     `xml:"ARN"`
	AtRestEncryptionEnabled    bool                       `xml:"AtRestEncryptionEnabled"`
	AuthTokenEnabled           bool                       `xml:"AuthTokenEnabled"`
	AuthTokenLastModifiedDate  time.Time                  `xml:"AuthTokenLastModifiedDate"`
	AutomaticFailover          string                     `xml:"AutomaticFailover"`
	CacheNodeType              string                     `xml:"CacheNodeType"`
	ClusterEnabled             bool                       `xml:"ClusterEnabled"`
	ConfigurationEndpoint      Endpoint                   `xml:"ConfigurationEndpoint"`
	Description                string                     `xml:"Description"`
	GlobalReplicationGroupInfo GlobalReplicationGroupInfo `xml:"GlobalReplicationGroupInfo"`
	KmsKeyId                   string                     `xml:"KmsKeyId"`
	LogDeliveryConfigurations  []LogDeliveryConfiguration `xml:"LogDeliveryConfigurations>LogDeliveryConfiguration"`
	MemberClusters             []string                   `xml:"MemberClusters>ClusterId"`
	MemberClustersOutpostArns  []string                   `xml:"MemberClustersOutpostArns>member"`
	MultiAZ                    string                     `xml:"MultiAZ"`
	NodeGroups                 []NodeGroup                `xml:"NodeGroups>NodeGroup"`
	ReplicationGroupId         string                     `xml:"ReplicationGroupId"`
	SnapshotRetentionLimit     int64                      `xml:"SnapshotRetentionLimit"`
	SnapshotWindow             string                     `xml:"SnapshotWindow"`
	SnapshottingClusterId      string                     `xml:"SnapshottingClusterId"`
	Status                     string                     `xml:"Status"`
	TransitEncryptionEnabled   bool                       `xml:"TransitEncryptionEnabled"`
	UserGroupIds               []string                   `xml:"UserGroupIds>member"`
}

func (self *SReplicationGroup) GetId() string {
	return self.ReplicationGroupId
}

func (self *SReplicationGroup) GetName() string {
	return self.ReplicationGroupId
}

func (self *SReplicationGroup) GetGlobalId() string {
	return self.GetId()
}

func (self *SReplicationGroup) GetCluster() (*SElasticache, error) {
	if self.cluster != nil {
		return self.cluster, nil
	}
	for _, node := range self.NodeGroups {
		for _, member := range node.NodeGroupMembers {
			clusters, err := self.region.GetElasticaches(member.CacheClusterId, false)
			if err != nil {
				return nil, err
			}
			for i := range clusters {
				self.cluster = &clusters[i]
				return self.cluster, nil
			}
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty node")
}

func (self *SReplicationGroup) GetStatus() string {
	// creating, available, modifying, deleting, create-failed, snapshotting
	switch self.Status {
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
		return self.Status
	}
}

func (self *SReplicationGroup) Refresh() error {
	caches, err := self.region.GetReplicationGroups(self.ReplicationGroupId)
	if err != nil {
		return err
	}
	self.cluster = nil
	for i := range caches {
		if caches[i].GetGlobalId() == self.GetGlobalId() {
			return jsonutils.Update(&self, caches[i])
		}
	}
	return errors.Wrapf(cloudprovider.ErrNotFound, self.GetGlobalId())
}

func (self *SReplicationGroup) GetBillingType() string {
	return billingapi.BILLING_TYPE_POSTPAID
}

func (self *SReplicationGroup) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SReplicationGroup) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SReplicationGroup) GetInstanceType() string {
	return self.CacheNodeType
}

func (self *SReplicationGroup) GetCapacityMB() int {
	if sku, ok := redisInstanceType[self.CacheNodeType]; ok {
		return int(sku.MemeoryMb)
	}
	return 0
}

func (self *SReplicationGroup) GetArchType() string {
	return api.ELASTIC_CACHE_ARCH_TYPE_CLUSTER
}

func (self *SReplicationGroup) GetNodeType() string {
	switch len(self.NodeGroups) {
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
	return fmt.Sprintf("%d", len(self.NodeGroups))
}

func (self *SReplicationGroup) GetEngine() string {
	return "redis"
}

func (self *SReplicationGroup) GetEngineVersion() string {
	cluster, err := self.GetCluster()
	if err != nil {
		return "unknown"
	}
	return cluster.GetEngineVersion()
}

func (self *SReplicationGroup) GetVpcId() string {
	cluster, err := self.GetCluster()
	if err != nil {
		return ""
	}
	subnets, err := self.region.DescribeCacheSubnetGroups(cluster.CacheSubnetGroupName)
	if err != nil {
		return ""
	}
	for i := range subnets {
		return subnets[i].VpcId
	}
	return ""
}

func (self *SReplicationGroup) GetZoneId() string {
	return ""
}

func (self *SReplicationGroup) GetNetworkType() string {
	return api.LB_NETWORK_TYPE_VPC
}

func (self *SReplicationGroup) GetNetworkId() string {
	return ""
}

// cluster mode(shard) 只有配置endpoint,no cluster mode 有primary/readonly endpoint
func (self *SReplicationGroup) GetPrivateDNS() string {
	if len(self.ConfigurationEndpoint.Address) > 0 {
		return self.ConfigurationEndpoint.Address
	}
	for _, node := range self.NodeGroups {
		if len(node.PrimaryEndpoint.Address) > 0 {
			return node.PrimaryEndpoint.Address
		}
		if len(node.ReaderEndpoint.Address) > 0 {
			return node.PrimaryEndpoint.Address
		}
	}
	return ""
}

func (self *SReplicationGroup) GetPrivateIpAddr() string {
	return ""
}

func (self *SReplicationGroup) GetTags() (map[string]string, error) {
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

func (self *SReplicationGroup) SetTags(tags map[string]string, replace bool) error {
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

func (self *SReplicationGroup) GetPrivateConnectPort() int {
	if self.ConfigurationEndpoint.Port > 0 {
		return int(self.ConfigurationEndpoint.Port)
	}
	for _, node := range self.NodeGroups {
		if node.PrimaryEndpoint.Port > 0 {
			return node.PrimaryEndpoint.Port
		}
		if node.ReaderEndpoint.Port > 0 {
			return node.ReaderEndpoint.Port
		}
	}
	return 0
}

func (self *SReplicationGroup) GetPublicDNS() string {
	return ""
}

func (self *SReplicationGroup) GetPublicIpAddr() string {
	return ""
}

func (self *SReplicationGroup) GetPublicConnectPort() int {
	return 0
}

func (self *SReplicationGroup) GetMaintainStartTime() string {
	cluster, err := self.GetCluster()
	if err != nil {
		return ""
	}
	splited := strings.Split(cluster.PreferredMaintenanceWindow, "-")
	if len(splited) == 2 {
		return strings.Trim(splited[0], "")
	}
	return ""
}

func (self *SReplicationGroup) GetMaintainEndTime() string {
	cluster, err := self.GetCluster()
	if err != nil {
		return ""
	}

	splited := strings.Split(cluster.PreferredMaintenanceWindow, "-")
	if len(splited) == 2 {
		return strings.Trim(splited[1], "")
	}
	return ""
}

func (self *SReplicationGroup) GetICloudElasticcacheAccounts() ([]cloudprovider.ICloudElasticcacheAccount, error) {
	return []cloudprovider.ICloudElasticcacheAccount{}, nil
}

func (self *SReplicationGroup) GetICloudElasticcacheAccount(accountId string) (cloudprovider.ICloudElasticcacheAccount, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SReplicationGroup) GetICloudElasticcacheAcls() ([]cloudprovider.ICloudElasticcacheAcl, error) {
	return []cloudprovider.ICloudElasticcacheAcl{}, nil
}

func (self *SReplicationGroup) GetICloudElasticcacheAcl(aclId string) (cloudprovider.ICloudElasticcacheAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SReplicationGroup) GetICloudElasticcacheBackups() ([]cloudprovider.ICloudElasticcacheBackup, error) {
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

func (self *SReplicationGroup) GetICloudElasticcacheBackup(backupId string) (cloudprovider.ICloudElasticcacheBackup, error) {
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

func (self *SReplicationGroup) GetICloudElasticcacheParameters() ([]cloudprovider.ICloudElasticcacheParameter, error) {
	return []cloudprovider.ICloudElasticcacheParameter{}, nil
}

func (self *SReplicationGroup) GetSecurityGroupIds() ([]string, error) {
	ret := []string{}
	cluster, err := self.GetCluster()
	if err != nil {
		return nil, err
	}
	for _, sec := range cluster.SecurityGroups {
		if !utils.IsInStringArray(sec.SecurityGroupId, ret) {
			ret = append(ret, sec.SecurityGroupId)
		}
	}
	return ret, nil
}

func (self *SReplicationGroup) AllocatePublicConnection(port int) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SReplicationGroup) ChangeInstanceSpec(spec string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SReplicationGroup) CreateAccount(input cloudprovider.SCloudElasticCacheAccountInput) (cloudprovider.ICloudElasticcacheAccount, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SReplicationGroup) CreateAcl(aclName, securityIps string) (cloudprovider.ICloudElasticcacheAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SReplicationGroup) CreateBackup(desc string) (cloudprovider.ICloudElasticcacheBackup, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SReplicationGroup) Delete() error {
	return self.region.DeleteReplicationGroup(self.ReplicationGroupId)
}

func (self *SRegion) DeleteReplicationGroup(id string) error {
	params := map[string]string{
		"ReplicationGroupId": id,
	}
	return self.ecRequest("DeleteReplicationGroup", params, nil)
}

func (self *SReplicationGroup) FlushInstance(input cloudprovider.SCloudElasticCacheFlushInstanceInput) error {
	return cloudprovider.ErrNotSupported
}

func (self *SReplicationGroup) GetAuthMode() string {
	return ""
}

func (self *SReplicationGroup) ReleasePublicConnection() error {
	return cloudprovider.ErrNotSupported
}

func (self *SReplicationGroup) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (self *SReplicationGroup) Restart() error {
	return cloudprovider.ErrNotSupported
}

func (self *SReplicationGroup) SetMaintainTime(maintainStartTime, maintainEndTime string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SReplicationGroup) UpdateAuthMode(noPwdAccess bool, password string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SReplicationGroup) UpdateBackupPolicy(config cloudprovider.SCloudElasticCacheBackupPolicyUpdateInput) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SReplicationGroup) UpdateInstanceParameters(config jsonutils.JSONObject) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SReplicationGroup) UpdateSecurityGroups(secgroupIds []string) error {
	return cloudprovider.ErrNotImplemented
}

type GlobalReplicationGroupInfo struct {
	GlobalReplicationGroupId         string `xml:"GlobalReplicationGroupId"`
	GlobalReplicationGroupMemberRole string `xml:"GlobalReplicationGroupMemberRole"`
}

type NodeGroup struct {
	NodeGroupId      string            `xml:"NodeGroupId"`
	NodeGroupMembers []NodeGroupMember `xml:"NodeGroupMembers>NodeGroupMember"`
	PrimaryEndpoint  Endpoint          `xml:"PrimaryEndpoint"`
	ReaderEndpoint   Endpoint          `xml:"ReadEndpoint"`
	Slots            string            `xml:"Slots"`
	Status           string            `xml:"Status"`
}

type NodeGroupMember struct {
	CacheClusterId            string   `xml:"CacheClusterId"`
	CacheNodeId               string   `xml:"CacheNodeId"`
	CurrentRole               string   `xml:"CurrentRole"`
	PreferredAvailabilityZone string   `xml:"PreferredAvailabilityZone"`
	PreferredOutpostArn       string   `xml:"PreferredOutpostArn"`
	ReadEndpoint              Endpoint `xml:"ReadEndpoint"`
}

func (region *SRegion) GetReplicationGroups(id string) ([]SReplicationGroup, error) {
	params := map[string]string{}
	if len(id) > 0 {
		params["ReplicationGroupId"] = id
	}
	ret := []SReplicationGroup{}
	for {
		part := struct {
			ReplicationGroups []SReplicationGroup `xml:"ReplicationGroups>ReplicationGroup"`
			Marker            string              `xml:"Marker"`
		}{}
		err := region.ecRequest("DescribeReplicationGroups", params, &part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.ReplicationGroups...)
		if len(part.Marker) == 0 || len(part.ReplicationGroups) == 0 {
			break
		}
		params["Marker"] = part.Marker
	}
	return ret, nil
}

func (region *SRegion) GetElasticaches(id string, isMemcached bool) ([]SElasticache, error) {
	ret := []SElasticache{}
	params := map[string]string{}
	if len(id) > 0 {
		params["CacheClusterId"] = id
	}
	if isMemcached {
		params["ShowCacheClustersNotInReplicationGroups"] = "true"
	}
	params["ShowCacheNodeInfo"] = "true"
	for {
		part := struct {
			CacheClusters []SElasticache `xml:"CacheClusters>CacheCluster"`
			Marker        string         `xml:"Marker"`
		}{}
		err := region.ecRequest("DescribeCacheClusters", params, &part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.CacheClusters...)
		if len(part.CacheClusters) == 0 || len(part.Marker) == 0 {
			break
		}
		params["Marker"] = part.Marker
	}
	return ret, nil
}

func (region *SRegion) GetIElasticcaches() ([]cloudprovider.ICloudElasticcache, error) {
	caches, err := region.GetReplicationGroups("")
	if err != nil {
		return nil, errors.Wrap(err, "GetSElasticaches")
	}
	result := []cloudprovider.ICloudElasticcache{}
	for i := range caches {
		caches[i].region = region
		result = append(result, &caches[i])
	}
	memcaches, err := region.GetElasticaches("", true)
	if err != nil {
		return nil, err
	}
	for i := range memcaches {
		memcaches[i].region = region
		result = append(result, &memcaches[i])
	}
	return result, nil
}

func (region *SRegion) GetIElasticcacheById(id string) (cloudprovider.ICloudElasticcache, error) {
	caches, err := region.GetReplicationGroups(id)
	if err != nil && errors.Cause(err) != cloudprovider.ErrNotFound {
		return nil, err
	}
	for i := range caches {
		if caches[i].GetGlobalId() == id {
			caches[i].region = region
			return &caches[i], nil
		}
	}
	memcaches, err := region.GetElasticaches(id, true)
	if err != nil {
		return nil, err
	}
	for i := range memcaches {
		if memcaches[i].GetGlobalId() == id {
			memcaches[i].region = region
			return &memcaches[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

type SubnetOutpost struct {
	SubnetOutpostArn string `xml:"SubnetOutpostArn"`
}

type Subnet struct {
	SubnetAvailabilityZone AvailabilityZone `xml:"SubnetAvailabilityZone"`
	SubnetIdentifier       string           `xml:"SubnetIdentifier"`
	SubnetOutpost          SubnetOutpost    `xml:"SubnetOutpost"`
}

type CacheSubnetGroup struct {
	ARN                         string   `xml:"Arn"`
	CacheSubnetGroupDescription string   `xml:"CacheSubnetGroupDescription"`
	CacheSubnetGroupName        string   `xml:"CacheSubnetGroupName"`
	Subnets                     []Subnet `xml:"Subnets>Subnet"`
	VpcId                       string   `xml:"VpcId"`
}

func (region *SRegion) DescribeCacheSubnetGroups(name string) ([]CacheSubnetGroup, error) {
	params := map[string]string{}
	if len(name) > 0 {
		params["CacheSubnetGroupName"] = name
	}
	ret := []CacheSubnetGroup{}
	for {
		part := struct {
			CacheSubnetGroups []CacheSubnetGroup `xml:"CacheSubnetGroups>CacheSubnetGroup"`
			Marker            string             `xml:"Marker"`
		}{}
		err := region.ecRequest("DescribeCacheSubnetGroups", params, &part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.CacheSubnetGroups...)
		if len(part.CacheSubnetGroups) == 0 || len(part.Marker) == 0 {
			break
		}
		params["Marker"] = part.Marker
	}
	return ret, nil
}

func (self *SRegion) CreateIElasticcaches(opts *cloudprovider.SCloudElasticCacheInput) (cloudprovider.ICloudElasticcache, error) {
	switch opts.Engine {
	case "redis":
		ret, err := self.CreateRedis(opts)
		if err != nil {
			return nil, err
		}
		ret.region = self
		return ret, nil
	case "memcached":
		ret, err := self.CreateMemcached(opts)
		if err != nil {
			return nil, err
		}
		ret.region = self
		return ret, nil
	default:
		return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "engine: %s", opts.Engine)
	}
}

func (self *SRegion) CreateRedis(opts *cloudprovider.SCloudElasticCacheInput) (*SReplicationGroup, error) {
	subnetGroupName := ""
	if len(opts.NetworkId) > 0 {
		network, err := self.getNetwork(opts.NetworkId)
		if err != nil {
			return nil, err
		}
		subnetGroups, err := self.DescribeCacheSubnetGroups("")
		if err != nil {
			return nil, err
		}
		for i := range subnetGroups {
			for _, subnet := range subnetGroups[i].Subnets {
				if subnet.SubnetIdentifier == network.SubnetId {
					subnetGroupName = subnetGroups[i].CacheSubnetGroupName
					break
				}
				if len(subnetGroupName) > 0 {
					break
				}
			}
		}
		if len(subnetGroupName) == 0 {
			subnetGroup, err := self.CreateCacheSubnetGroup(opts.InstanceName, opts.NetworkId)
			if err != nil {
				return nil, err
			}
			subnetGroupName = subnetGroup.CacheSubnetGroupName
		}
	}
	params := map[string]string{
		"ReplicationGroupId": opts.InstanceName,
		"AuthToken":          opts.Password,
		"CacheNodeType":      opts.InstanceType,
		"ClusterMode":        "disabled",
		"Engine":             "Redis",
		"EngineVersion":      opts.EngineVersion,
		"NumCacheClusters":   "2",
	}
	if len(opts.Password) > 0 {
		params["TransitEncryptionEnabled"] = "true"
		params["AuthToken"] = opts.Password
	}
	if len(subnetGroupName) > 0 {
		params["CacheSubnetGroupName"] = subnetGroupName
	}
	idx := 1
	for k, v := range opts.Tags {
		params[fmt.Sprintf("Tags.Tag.%d.Key", idx)] = k
		params[fmt.Sprintf("Tags.Tag.%d.Value", idx)] = v
		idx++
	}
	ret := struct {
		ReplicationGroup SReplicationGroup `xml:"ReplicationGroup"`
	}{}
	err := self.ecRequest("CreateReplicationGroup", params, &ret)
	if err != nil {
		return nil, err
	}
	log.Errorf("ret: %s", jsonutils.Marshal(ret.ReplicationGroup))
	return &ret.ReplicationGroup, nil
}

func (self *SRegion) CreateMemcached(opts *cloudprovider.SCloudElasticCacheInput) (*SElasticache, error) {
	subnetGroupName := ""
	if len(opts.NetworkId) > 0 {
		network, err := self.getNetwork(opts.NetworkId)
		if err != nil {
			return nil, err
		}
		subnetGroups, err := self.DescribeCacheSubnetGroups("")
		if err != nil {
			return nil, err
		}
		for i := range subnetGroups {
			for _, subnet := range subnetGroups[i].Subnets {
				if subnet.SubnetIdentifier == network.SubnetId {
					subnetGroupName = subnetGroups[i].CacheSubnetGroupName
					break
				}
				if len(subnetGroupName) > 0 {
					break
				}
			}
		}
		if len(subnetGroupName) == 0 {
			subnetGroup, err := self.CreateCacheSubnetGroup(opts.InstanceName, opts.NetworkId)
			if err != nil {
				return nil, err
			}
			subnetGroupName = subnetGroup.CacheSubnetGroupName
		}
	}

	params := map[string]string{
		"CacheClusterId": opts.InstanceName,
		"CacheNodeType":  opts.InstanceType,
		"Engine":         "memcached",
		"EngineVersion":  opts.EngineVersion,
	}
	if len(opts.Password) > 0 {
		params["TransitEncryptionEnabled"] = "true"
		params["AuthToken"] = opts.Password
	}
	if len(subnetGroupName) > 0 {
		params["CacheSubnetGroupName"] = subnetGroupName
	}
	idx := 1
	for k, v := range opts.Tags {
		params[fmt.Sprintf("Tags.Tag.%d.Key", idx)] = k
		params[fmt.Sprintf("Tags.Tag.%d.Value", idx)] = v
		idx++
	}
	ret := struct {
		CacheCluster SElasticache `xml:"CacheCluster"`
	}{}
	err := self.ecRequest("CreateCacheCluster", params, &ret)
	if err != nil {
		return nil, err
	}
	return &ret.CacheCluster, nil
}

func (self *SRegion) CreateCacheSubnetGroup(clusterName, networkId string) (*CacheSubnetGroup, error) {

	params := map[string]string{
		"CacheSubnetGroupName":         fmt.Sprintf("auto-create-for-cluster-%s", clusterName),
		"SubnetIds.SubnetIdentifier.1": networkId,
	}
	ret := struct {
		CacheSubnetGroup CacheSubnetGroup `xml:"CacheSubnetGroup"`
	}{}
	err := self.ecRequest("CreateCacheSubnetGroup", params, &ret)
	if err != nil {
		return nil, err
	}
	return &ret.CacheSubnetGroup, nil
}
