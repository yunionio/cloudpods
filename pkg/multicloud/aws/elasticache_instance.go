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
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	billingapi "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/billing"
)

func (self *SRegion) GetElasticaches(id string) ([]SElasticache, error) {
	params := map[string]string{}
	if len(id) > 0 {
		params["ReplicationGroupId"] = id
	}

	ret := []SElasticache{}
	for {
		result := struct {
			Marker            string         `xml:"Marker"`
			ReplicationGroups []SElasticache `xml:"ReplicationGroups>ReplicationGroup"`
		}{}
		err := self.redisRequest("DescribeReplicationGroups", params, &result)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeReplicationGroups")
		}
		ret = append(ret, result.ReplicationGroups...)
		if len(result.Marker) == 0 || len(result.ReplicationGroups) == 0 {
			break
		}
		params["Marker"] = result.Marker
	}
	return ret, nil
}

func (self *SRegion) GetIElasticcaches() ([]cloudprovider.ICloudElasticcache, error) {
	caches, err := self.GetElasticaches("")
	if err != nil {
		return nil, errors.Wrap(err, "GetSElasticaches")
	}
	result := []cloudprovider.ICloudElasticcache{}
	for i := range caches {
		caches[i].region = self
		result = append(result, &caches[i])
	}
	return result, nil
}

func (self *SRegion) GetIElasticcacheById(id string) (cloudprovider.ICloudElasticcache, error) {
	caches, err := self.GetElasticaches(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetElasticCaches(%s)", id)
	}
	for i := range caches {
		if caches[i].GetGlobalId() == id {
			caches[i].region = self
			return &caches[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
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
	CacheNodeIdsToReboot    []string `xml:"CacheNodeIdsToReboot>CacheNodeId"`
	CacheParameterGroupName string   `xml:"CacheParameterGroupName"`
	ParameterApplyStatus    string   `xml:"ParameterApplyStatus"`
}

type CacheSecurityGroupMembership struct {
	CacheSecurityGroupName string `xml:"CacheSecurityGroupName"`
	Status                 string `xml:"Status"`
}

type NotificationConfiguration struct {
	TopicArn    string `xml:"TopicArn"`
	TopicStatus string `xml:"TopicStatus"`
}

type PendingModifiedValues struct {
	AuthTokenStatus           string                            `xml:"AuthTokenStatus"`
	CacheNodeIdsToRemove      []string                          `xml:"CacheNodeIdsToRemove>CacheNodeId"`
	CacheNodeType             string                            `xml:"CacheNodeType"`
	EngineVersion             string                            `xml:"EngineVersion"`
	LogDeliveryConfigurations []PendingLogDeliveryConfiguration `xml:"LogDeliveryConfigurations>LogDeliveryConfiguration"`
	NumCacheNodes             int64                             `xml:"NumCacheNodes"`
}

type SecurityGroupMembership struct {
	SecurityGroupId string `xml:"SecurityGroupId"`
	Status          string `xml:"Status"`
}

type SElasticacheCluster struct {
	ARN                                string                         `xml:"ARN"`
	AtRestEncryptionEnabled            bool                           `xml:"AuthTokenEnabled"`
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
	SecurityGroups                     []SecurityGroupMembership      `xml:"SecurityGroups"`
	SnapshotRetentionLimit             int64                          `xml:"SnapshotRetentionLimit"`
	SnapshotWindow                     string                         `xml:"SnapshotWindow"`
	TransitEncryptionEnabled           bool                           `xml:"TransitEncryptionEnabled"`
}

func (self *SRegion) GetElasticacheClusters() ([]SElasticacheCluster, error) {
	params := map[string]string{}
	ret := []SElasticacheCluster{}
	for {
		result := struct {
			Marker   string                `xml:"Marker"`
			Clusters []SElasticacheCluster `xml:"CacheClusters>CacheCluster"`
		}{}
		err := self.redisRequest("DescribeCacheClusters", params, &result)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeCacheClusters")
		}
		ret = append(ret, result.Clusters...)
		if len(result.Marker) == 0 || len(result.Clusters) == 0 {
			break
		}
		params["Marker"] = result.Marker
	}
	return ret, nil
}

type ElasticacheSubnetGroup struct {
	VpcId string `xml:"VpcId"`
}

func (self *SRegion) GetCacheSubnetGroups(name string) ([]ElasticacheSubnetGroup, error) {
	params := map[string]string{}
	if len(name) > 0 {
		params["CacheSubnetGroupName"] = name
	}
	ret := []ElasticacheSubnetGroup{}
	for {
		result := struct {
			Marker string                   `xml:"Marker"`
			Groups []ElasticacheSubnetGroup `xml:"CacheSubnetGroups>CacheSubnetGroup"`
		}{}
		err := self.redisRequest("DescribeCacheSubnetGroups", params, &result)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeCacheSubnetGroups")
		}
		ret = append(ret, result.Groups...)
		if len(result.Marker) == 0 || len(result.Groups) == 0 {
			break
		}
		params["Marker"] = result.Marker
	}
	return ret, nil
}

type Endpoint struct {
	Address string `xml:"Address"`
	Port    int64  `xml:"Port"`
}

type GlobalReplicationGroupInfo struct {
	GlobalReplicationGroupId         string `xml:"GlobalReplicationGroupId"`
	GlobalReplicationGroupMemberRole string `xml:"GlobalReplicationGroupMemberRole"`
}

type LogDeliveryConfiguration struct {
	DestinationDetails DestinationDetails `xml:"DestinationDetails"`
	DestinationType    string             `xml:"DestinationType"`
	LogFormat          string             `xml:"LogFormat"`
	LogType            string             `xml:"LogType"`
	Message            string             `xml:"Message"`
	// | disabling | modifying | active | error
	Status *string `xml:"Status"`
}

type CloudWatchLogsDestinationDetails struct {
	LogGroup string `xml:"LogGroup"`
}

type KinesisFirehoseDestinationDetails struct {
	DeliveryStream string `xml:"DeliveryStream"`
}

type DestinationDetails struct {
	CloudWatchLogsDetails  CloudWatchLogsDestinationDetails  `xml:"CloudWatchLogsDetails"`
	KinesisFirehoseDetails KinesisFirehoseDestinationDetails `xml:"KinesisFirehoseDetails"`
}

type NodeGroup struct {
	NodeGroupId      string            `xml:"NodeGroupId"`
	NodeGroupMembers []NodeGroupMember `xml:"NodeGroupMembers>NodeGroupMember"`
	PrimaryEndpoint  Endpoint          `xml:"PrimaryEndpoint"`
	ReaderEndpoint   Endpoint          `xml:"ReaderEndpoint"`
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

type ReplicationGroupPendingModifiedValues struct {
	AuthTokenStatus           string                            `xml:"AuthTokenStatus"`
	AutomaticFailoverStatus   string                            `xml:"AutomaticFailoverStatus"`
	LogDeliveryConfigurations []PendingLogDeliveryConfiguration `xml:"LogDeliveryConfigurations>PendingLogDeliveryConfiguration"`
	PrimaryClusterId          string                            `xml:"PrimaryClusterId"`
	Resharding                ReshardingStatus                  `xml:"Resharding"`
	UserGroups                UserGroupsUpdateStatus            `xml:"UserGroups"`
}

type UserGroupsUpdateStatus struct {
	UserGroupIdsToAdd    []string `xml:"UserGroupIdsToAdd"`
	UserGroupIdsToRemove []string `xml:"UserGroupIdsToAdd"`
}

type ReshardingStatus struct {
	SlotMigration SlotMigration `xml:"SlotMigration"`
}

type SlotMigration struct {
	ProgressPercentage float64 `xml:"ProgressPercentage"`
}

type PendingLogDeliveryConfiguration struct {
	DestinationDetails DestinationDetails `xml:"DestinationDetails"`
	DestinationType    string             `xml:"DestinationType"`
	LogFormat          string             `xml:"LogFormat"`
	LogType            string             `xml:"LogType"`
}

type SElasticache struct {
	multicloud.SElasticcacheBase
	multicloud.AwsTags

	region *SRegion

	ARN                        string                                `xml:"ARN"`
	AtRestEncryptionEnabled    bool                                  `xml:"AtRestEncryptionEnabled"`
	AuthTokenEnabled           bool                                  `xml:"AuthTokenEnabled"`
	AuthTokenLastModifiedDate  time.Time                             `xml:"AuthTokenLastModifiedDate"`
	AutomaticFailover          string                                `xml:"AutomaticFailover"`
	CacheNodeType              string                                `xml:"CacheNodeType"`
	ClusterEnabled             bool                                  `xml:"ClusterEnabled"`
	ConfigurationEndpoint      Endpoint                              `xml:"ConfigurationEndpoint"`
	Description                string                                `xml:"Description"`
	GlobalReplicationGroupInfo GlobalReplicationGroupInfo            `xml:"GlobalReplicationGroupInfo"`
	KmsKeyId                   string                                `xml:"KmsKeyId"`
	LogDeliveryConfigurations  []LogDeliveryConfiguration            `xml:"LogDeliveryConfigurations>LogDeliveryConfiguration"`
	MemberClusters             []string                              `xml:"MemberClusters>locationNameList"`
	MemberClustersOutpostArns  []string                              `xml:"MemberClustersOutpostArns>ReplicationGroupOutpostArn"`
	MultiAZ                    string                                `xml:"MultiAZ"`
	NodeGroups                 []NodeGroup                           `xml:"NodeGroups>NodeGroup"`
	PendingModifiedValues      ReplicationGroupPendingModifiedValues `xml:"PendingModifiedValues"`
	ReplicationGroupId         string                                `xml:"ReplicationGroupId"`
	SnapshotRetentionLimit     int64                                 `xml:"SnapshotRetentionLimit"`
	SnapshotWindow             string                                `xml:"SnapshotWindow"`
	SnapshottingClusterId      string                                `xml:"SnapshottingClusterId"`
	Status                     string                                `xml:"Status"`
	TransitEncryptionEnabled   bool                                  `xml:"TransitEncryptionEnabled"`
	UserGroupIds               []string                              `xml:"UserGroupIds"`

	cacheClusters []SElasticacheCluster
	subnetGroup   *ElasticacheSubnetGroup
}

func (self *SElasticache) GetId() string {
	return self.ReplicationGroupId
}

func (self *SElasticache) GetName() string {
	return self.ReplicationGroupId
}

func (self *SElasticache) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticache) GetStatus() string {
	// creating, available, modifying, deleting, create-failed, snapshotting
	switch self.Status {
	case "creating":
		return api.ELASTIC_CACHE_STATUS_DEPLOYING
	case "available":
		return api.ELASTIC_CACHE_STATUS_RUNNING
	case "modifying":
		return api.ELASTIC_CACHE_STATUS_CHANGING
	case "deleting":
		return api.ELASTIC_CACHE_STATUS_DELETING
	case "create-failed":
		return api.ELASTIC_CACHE_STATUS_CREATE_FAILED
	case "snapshotting":
		return api.ELASTIC_CACHE_STATUS_SNAPSHOTTING
	default:
		return api.ELASTIC_CACHE_STATUS_UNKNOWN
	}
}

func (self *SElasticache) Refresh() error {
	caches, err := self.region.GetElasticaches(self.ReplicationGroupId)
	if err != nil {
		return errors.Wrapf(err, "GetElasticaches")
	}

	self.cacheClusters = nil
	self.subnetGroup = nil

	for i := range caches {
		if caches[i].GetGlobalId() == self.ReplicationGroupId {
			return jsonutils.Update(self, caches[i])
		}
	}
	return errors.Wrapf(cloudprovider.ErrNotFound, self.ReplicationGroupId)
}

func (self *SElasticache) GetBillingType() string {
	return billingapi.BILLING_TYPE_POSTPAID
}

func (self *SElasticache) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SElasticache) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SElasticache) fetchClusters() error {
	if self.cacheClusters != nil {
		return nil
	}
	clusters, err := self.region.GetElasticacheClusters()
	if err != nil {
		return errors.Wrap(err, "GetElasticacheClusters")
	}
	self.cacheClusters = []SElasticacheCluster{}
	for i := range clusters {
		if clusters[i].ReplicationGroupId == self.GetId() {
			self.cacheClusters = append(self.cacheClusters, clusters[i])
		}
	}
	return nil
}

func (self *SElasticache) GetInstanceType() string {
	return self.CacheNodeType
}

func (self *SElasticache) GetCapacityMB() int {
	return 0
}

func (self *SElasticache) GetArchType() string {
	if self.ClusterEnabled {
		return api.ELASTIC_CACHE_ARCH_TYPE_CLUSTER
	}

	self.fetchClusters()
	if len(self.cacheClusters) > 1 {
		return api.ELASTIC_CACHE_ARCH_TYPE_RWSPLIT
	}

	return api.ELASTIC_CACHE_ARCH_TYPE_SINGLE
}

func (self *SElasticache) GetNodeType() string {
	nodeGroupNums := len(self.NodeGroups)
	switch nodeGroupNums {
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

	return strconv.Itoa(nodeGroupNums)
}

func (self *SElasticache) GetEngine() string {
	self.fetchClusters()
	for _, cluster := range self.cacheClusters {
		return cluster.Engine
	}
	return ""
}

func (self *SElasticache) GetEngineVersion() string {
	self.fetchClusters()
	for _, cluster := range self.cacheClusters {
		return cluster.EngineVersion
	}
	return ""
}

func (self *SElasticache) fetchSubnetGroup() error {
	if self.subnetGroup != nil {
		return nil
	}
	err := self.fetchClusters()
	if err != nil {
		return errors.Wrap(err, "self.fetchClusters()")
	}

	subnetGroupName := ""
	for _, cluster := range self.cacheClusters {
		if len(cluster.CacheSubnetGroupName) > 0 {
			subnetGroupName = cluster.CacheSubnetGroupName
		}
	}

	if len(subnetGroupName) == 0 {
		return cloudprovider.ErrNotFound
	}

	subnetGroup, err := self.region.GetCacheSubnetGroups(subnetGroupName)
	if err != nil {
		return errors.Wrapf(err, "GetCacheSubnetGroups(%s)", subnetGroupName)
	}

	if len(subnetGroup) == 0 {
		return errors.Wrapf(cloudprovider.ErrNotFound, "subnetGroup %s not found", subnetGroupName)
	}
	if len(subnetGroup) > 1 {
		return cloudprovider.ErrDuplicateId
	}

	self.subnetGroup = &subnetGroup[0]
	return nil
}

func (self *SElasticache) GetVpcId() string {
	err := self.fetchSubnetGroup()
	if err != nil {
		log.Errorf("Error:%s,self.fetchSubnetGroup()", err)
		return ""
	}
	if self.subnetGroup != nil {
		return self.subnetGroup.VpcId
	}
	return ""
}

func (self *SElasticache) GetZoneId() string {
	return ""
}

func (self *SElasticache) GetNetworkType() string {
	if len(self.GetVpcId()) > 0 {
		return api.LB_NETWORK_TYPE_VPC
	}
	return api.LB_NETWORK_TYPE_CLASSIC
}

func (self *SElasticache) GetNetworkId() string {
	return ""
}

// cluster mode(shard) 只有配置endpoint,no cluster mode 有primary/readonly endpoint
func (self *SElasticache) GetPrivateDNS() string {
	for _, nodeGroup := range self.NodeGroups {
		if len(nodeGroup.PrimaryEndpoint.Address) > 0 {
			return nodeGroup.PrimaryEndpoint.Address
		}
	}
	return ""
}

func (self *SElasticache) GetPrivateIpAddr() string {
	return ""
}

func (self *SElasticache) GetPrivateConnectPort() int {
	for _, nodeGroup := range self.NodeGroups {
		if nodeGroup.PrimaryEndpoint.Port > 0 {
			return int(nodeGroup.PrimaryEndpoint.Port)
		}
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
	self.fetchClusters()
	window := ""
	for _, cluster := range self.cacheClusters {
		if len(cluster.PreferredMaintenanceWindow) > 0 {
			window = cluster.PreferredMaintenanceWindow
			break
		}
	}

	splited := strings.Split(window, "-")
	return splited[0]
}

func (self *SElasticache) GetMaintainEndTime() string {
	self.fetchClusters()
	window := ""
	for _, cluster := range self.cacheClusters {
		if len(cluster.PreferredMaintenanceWindow) > 0 {
			window = cluster.PreferredMaintenanceWindow
			break
		}
	}

	splited := strings.Split(window, "-")
	if len(splited) == 2 {
		return splited[1]
	}
	return ""
}

func (self *SElasticache) GetICloudElasticcacheAccounts() ([]cloudprovider.ICloudElasticcacheAccount, error) {
	users, err := self.region.GetElasticacheUsers("")
	if err != nil {
		return nil, errors.Wrap(err, "self.GetUsers()")
	}
	result := []cloudprovider.ICloudElasticcacheAccount{}
	for i := range users {
		users[i].cache = self
		result = append(result, &users[i])
	}
	return result, nil
}

func (self *SElasticache) GetUserById(id string) (*SElasticacheUser, error) {
	users, err := self.region.GetElasticacheUsers("")
	if err != nil {
		return nil, errors.Wrap(err, "self.region.DescribeUsers")
	}
	for i := range users {
		if users[i].GetGlobalId() == id {
			users[i].cache = self
			return &users[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SElasticache) GetICloudElasticcacheAccount(accountId string) (cloudprovider.ICloudElasticcacheAccount, error) {
	user, err := self.GetUserById(accountId)
	if err != nil {
		return nil, errors.Wrapf(err, "self.GetUserById(%s)", accountId)
	}
	return user, nil
}

func (self *SElasticache) GetICloudElasticcacheAcls() ([]cloudprovider.ICloudElasticcacheAcl, error) {
	return []cloudprovider.ICloudElasticcacheAcl{}, nil
}

func (self *SElasticache) GetICloudElasticcacheAcl(aclId string) (cloudprovider.ICloudElasticcacheAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SElasticache) GetICloudElasticcacheBackups() ([]cloudprovider.ICloudElasticcacheBackup, error) {
	snapshots, err := self.region.GetElasticacheSnapshots(self.GetName(), "")
	if err != nil {
		return nil, errors.Wrap(err, "self.GetSnapshots()")
	}
	result := []cloudprovider.ICloudElasticcacheBackup{}
	for i := range snapshots {
		snapshots[i].cache = self
		result = append(result, &snapshots[i])
	}
	return result, nil
}

func (self *SElasticache) GetICloudElasticcacheBackup(backupId string) (cloudprovider.ICloudElasticcacheBackup, error) {
	snapshots, err := self.region.GetElasticacheSnapshots(self.ReplicationGroupId, backupId)
	if err != nil {
		return nil, errors.Wrap(err, "self.GetSnapshots()")
	}
	for i := range snapshots {
		if snapshots[i].GetGlobalId() == backupId {
			return &snapshots[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SElasticache) GetParameterGroupName() (string, error) {
	err := self.fetchClusters()
	if err != nil {
		return "", errors.Wrap(err, "self.fetchClusters()")
	}
	for _, cluster := range self.cacheClusters {
		if len(cluster.CacheParameterGroup.CacheParameterGroupName) > 0 {
			return cluster.CacheParameterGroup.CacheParameterGroupName, nil
		}
	}

	return "", cloudprovider.ErrNotFound
}

func (self *SElasticache) GetParameters() ([]SElasticacheParameter, error) {
	groupName, err := self.GetParameterGroupName()
	if err != nil {
		return nil, errors.Wrap(err, "self.GetParameterGroupName()")
	}
	parameters, err := self.region.GetCacheParameters(groupName)
	if err != nil {
		return nil, errors.Wrapf(err, "GetCacheParameters", groupName)
	}
	for i := range parameters {
		parameters[i].parameterGroup = groupName
	}
	return parameters, nil
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
	err := self.fetchClusters()
	if err != nil {
		return nil, errors.Wrap(err, "self.fetchClusters()")
	}
	result := []string{}
	for _, cluster := range self.cacheClusters {
		for _, securityGroup := range cluster.SecurityGroups {
			if securityGroup.Status == "active" {
				if len(securityGroup.SecurityGroupId) > 0 {
					result = append(result, securityGroup.SecurityGroupId)
				}
			}
		}
	}
	return result, nil
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
	return cloudprovider.ErrNotImplemented
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
