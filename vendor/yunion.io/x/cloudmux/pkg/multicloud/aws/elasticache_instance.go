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

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elasticache"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"

	billingapi "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

func (region *SRegion) DescribeElasticacheReplicationGroups(Id string) ([]*elasticache.ReplicationGroup, error) {
	ecClient, err := region.getAwsElasticacheClient()
	if err != nil {
		return nil, errors.Wrap(err, "client.getAwsElasticacheClient")
	}

	input := elasticache.DescribeReplicationGroupsInput{}

	replicaGroup := []*elasticache.ReplicationGroup{}

	marker := ""
	maxrecords := (int64)(100)
	input.MaxRecords = &maxrecords

	if len(Id) > 0 {
		input.ReplicationGroupId = &Id
	}

	for {
		if len(marker) >= 0 {
			input.Marker = &marker
		}
		out, err := ecClient.DescribeReplicationGroups(&input)
		if err != nil {
			if e, ok := err.(awserr.Error); ok && e.Code() == "ReplicationGroupNotFoundFault" {
				return nil, errors.Wrapf(cloudprovider.ErrNotFound, err.Error())
			}
			return nil, errors.Wrap(err, "ecClient.DescribeReplicationGroups")
		}
		replicaGroup = append(replicaGroup, out.ReplicationGroups...)

		if out.Marker != nil && len(*out.Marker) > 0 {
			marker = *out.Marker
		} else {
			break
		}
	}

	return replicaGroup, nil
}

func (region *SRegion) GetElasticaches() ([]SElasticache, error) {
	replicaGroups, err := region.DescribeElasticacheReplicationGroups("")
	if err != nil {
		return nil, errors.Wrap(err, " region.DescribeElasticacheReplicationGroups")
	}
	result := []SElasticache{}
	for i := range replicaGroups {
		result = append(result, SElasticache{region: region, replicaGroup: replicaGroups[i]})
	}
	return result, nil
}

func (region *SRegion) GetIElasticcaches() ([]cloudprovider.ICloudElasticcache, error) {
	sElasticahes, err := region.GetElasticaches()
	if err != nil {
		return nil, errors.Wrap(err, "region.GetSElasticaches()")
	}
	result := []cloudprovider.ICloudElasticcache{}
	for i := range sElasticahes {
		result = append(result, &sElasticahes[i])
	}
	return result, nil
}

func (region *SRegion) GetSElasticacheById(Id string) (*SElasticache, error) {
	replicaGroups, err := region.DescribeElasticacheReplicationGroups(Id)
	if err != nil {
		return nil, errors.Wrap(err, " region.DescribeElasticacheReplicationGroups")
	}
	if len(replicaGroups) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if len(replicaGroups) > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	return &SElasticache{region: region, replicaGroup: replicaGroups[0]}, nil
}

func (region *SRegion) GetIElasticcacheById(id string) (cloudprovider.ICloudElasticcache, error) {
	sElasticache, err := region.GetSElasticacheById(id)
	if err != nil {
		return nil, errors.Wrapf(err, "region.GetSElasticacheById(%s)", id)
	}
	return sElasticache, nil
}

func (region *SRegion) DescribeElasticacheClusters() ([]*elasticache.CacheCluster, error) {
	ecClient, err := region.getAwsElasticacheClient()
	if err != nil {
		return nil, errors.Wrap(err, "client.getAwsElasticacheClient")
	}

	input := elasticache.DescribeCacheClustersInput{}

	clusters := []*elasticache.CacheCluster{}
	marker := ""
	maxrecords := (int64)(100)
	input.MaxRecords = &maxrecords

	for {
		if len(marker) >= 0 {
			input.Marker = &marker
		}
		out, err := ecClient.DescribeCacheClusters(&input)
		if err != nil {
			return nil, errors.Wrap(err, "ecClient.DescribeCacheClusters")
		}
		clusters = append(clusters, out.CacheClusters...)

		if out.Marker != nil && len(*out.Marker) > 0 {
			marker = *out.Marker
		} else {
			break
		}
	}

	return clusters, nil
}

func (region *SRegion) DescribeCacheSubnetGroups(Id string) ([]*elasticache.CacheSubnetGroup, error) {
	ecClient, err := region.getAwsElasticacheClient()
	if err != nil {
		return nil, errors.Wrap(err, "client.getAwsElasticacheClient")
	}
	input := elasticache.DescribeCacheSubnetGroupsInput{}
	subnetGroups := []*elasticache.CacheSubnetGroup{}

	marker := ""
	maxrecords := (int64)(100)
	input.MaxRecords = &maxrecords
	if len(Id) > 0 {
		input.CacheSubnetGroupName = &Id
	}
	for {
		if len(marker) >= 0 {
			input.Marker = &marker
		}
		out, err := ecClient.DescribeCacheSubnetGroups(&input)
		if err != nil {
			return nil, errors.Wrap(err, "ecClient.DescribeCacheSubnetGroups")
		}
		subnetGroups = append(subnetGroups, out.CacheSubnetGroups...)

		if out.Marker != nil && len(*out.Marker) > 0 {
			marker = *out.Marker
		} else {
			break
		}
	}

	return subnetGroups, nil
}

type SElasticache struct {
	multicloud.SElasticcacheBase
	AwsTags

	region        *SRegion
	replicaGroup  *elasticache.ReplicationGroup
	cacheClusters []*elasticache.CacheCluster
	subnetGroup   *elasticache.CacheSubnetGroup
}

func (self *SElasticache) GetId() string {
	return *self.replicaGroup.ReplicationGroupId
}

func (self *SElasticache) GetName() string {
	return *self.replicaGroup.ReplicationGroupId
}

func (self *SElasticache) GetGlobalId() string {
	return self.GetId()
}

func (self *SElasticache) GetStatus() string {
	if self.replicaGroup.Status == nil {
		return api.ELASTIC_CACHE_STATUS_UNKNOWN
	}
	// creating, available, modifying, deleting, create-failed, snapshotting
	switch *self.replicaGroup.Status {
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
	if self.replicaGroup.ReplicationGroupId == nil {
		return errors.Wrap(cloudprovider.ErrNotFound, "replicationGroupId not found")
	}
	replica, err := self.region.DescribeElasticacheReplicationGroups(*self.replicaGroup.ReplicationGroupId)

	if err != nil {
		return errors.Wrapf(err, "self.region.DescribeElasticacheReplicationGroups(%s)", *self.replicaGroup.ReplicationGroupId)
	}

	if len(replica) == 0 {
		return cloudprovider.ErrNotFound
	}
	if len(replica) > 1 {
		return cloudprovider.ErrDuplicateId
	}

	self.replicaGroup = replica[0]
	self.cacheClusters = nil
	self.subnetGroup = nil

	return nil
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
	clusters, err := self.region.DescribeElasticacheClusters()
	if err != nil {
		return errors.Wrap(err, "self.region.DescribeElasticacheClusters")
	}
	self.cacheClusters = []*elasticache.CacheCluster{}
	for i := range clusters {
		if clusters[i].ReplicationGroupId != nil && *clusters[i].ReplicationGroupId == self.GetId() {
			self.cacheClusters = append(self.cacheClusters, clusters[i])
		}
	}
	return nil
}

func (self *SElasticache) GetInstanceType() string {
	if self.replicaGroup.CacheNodeType != nil {
		return *self.replicaGroup.CacheNodeType
	}
	return ""
}

func (self *SElasticache) GetCapacityMB() int {
	return 0
}

func (self *SElasticache) GetArchType() string {
	if self.replicaGroup.ClusterEnabled != nil && *self.replicaGroup.ClusterEnabled == true {
		return api.ELASTIC_CACHE_ARCH_TYPE_CLUSTER
	}

	self.fetchClusters()
	if len(self.cacheClusters) > 1 {
		return api.ELASTIC_CACHE_ARCH_TYPE_RWSPLIT
	}

	return api.ELASTIC_CACHE_ARCH_TYPE_SINGLE
}

func (self *SElasticache) GetNodeType() string {
	nodeGroupNums := 1
	for _, nodeGroup := range self.replicaGroup.NodeGroups {
		if nodeGroup != nil {
			nodeGroupNums = len(nodeGroup.NodeGroupMembers)
			break
		}
	}
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
		if cluster != nil && cluster.Engine != nil {
			return *cluster.Engine
		}
	}
	return ""
}

func (self *SElasticache) GetEngineVersion() string {
	self.fetchClusters()
	for _, cluster := range self.cacheClusters {
		if cluster != nil && cluster.EngineVersion != nil {
			return *cluster.EngineVersion
		}
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
		if cluster != nil && cluster.CacheSubnetGroupName != nil {
			subnetGroupName = *cluster.CacheSubnetGroupName
		}
	}

	if len(subnetGroupName) == 0 {
		return cloudprovider.ErrNotFound
	}

	subnetGroup, err := self.region.DescribeCacheSubnetGroups(subnetGroupName)
	if err != nil {
		return errors.Wrapf(err, "self.region.DescribeCacheSubnetGroups(%s)", subnetGroupName)
	}

	if len(subnetGroup) == 0 {
		return errors.Wrapf(cloudprovider.ErrNotFound, "subnetGroup %s not found", subnetGroupName)
	}
	if len(subnetGroup) > 1 {
		return cloudprovider.ErrDuplicateId
	}

	self.subnetGroup = subnetGroup[0]
	return nil
}

func (self *SElasticache) GetVpcId() string {
	err := self.fetchSubnetGroup()
	if err != nil {
		log.Errorf("Error:%s,self.fetchSubnetGroup()", err)
		return ""
	}
	if self.subnetGroup != nil && self.subnetGroup.VpcId != nil {
		return *self.subnetGroup.VpcId
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
	for _, nodeGroup := range self.replicaGroup.NodeGroups {
		if nodeGroup != nil && nodeGroup.PrimaryEndpoint != nil && nodeGroup.PrimaryEndpoint.Address != nil {
			return *nodeGroup.PrimaryEndpoint.Address
		}
	}
	return ""
}

func (self *SElasticache) GetPrivateIpAddr() string {
	return ""
}

func (self *SElasticache) GetPrivateConnectPort() int {
	for _, nodeGroup := range self.replicaGroup.NodeGroups {
		if nodeGroup != nil && nodeGroup.PrimaryEndpoint != nil && nodeGroup.PrimaryEndpoint.Port != nil {
			return int(*nodeGroup.PrimaryEndpoint.Port)
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
		if cluster != nil && cluster.PreferredMaintenanceWindow != nil {
			window = *cluster.PreferredMaintenanceWindow
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
		if cluster != nil && cluster.PreferredMaintenanceWindow != nil {
			window = *cluster.PreferredMaintenanceWindow
			break
		}
	}

	splited := strings.Split(window, "-")
	if len(splited) == 2 {
		return splited[1]
	}
	return ""
}

func (self *SElasticache) GetUsers() ([]SElasticacheUser, error) {
	result := []SElasticacheUser{}
	users, err := self.region.DescribeUsers("")
	if err != nil {
		return nil, errors.Wrap(err, "self.region.DescribeUsers")
	}
	for i := range users {
		result = append(result, SElasticacheUser{region: self.region, user: users[i]})
	}
	return result, nil
}

func (self *SElasticache) GetICloudElasticcacheAccounts() ([]cloudprovider.ICloudElasticcacheAccount, error) {
	result := []cloudprovider.ICloudElasticcacheAccount{}
	users, err := self.GetUsers()
	if err != nil {
		return nil, errors.Wrap(err, "self.GetUsers()")
	}
	for i := range users {
		result = append(result, &users[i])
	}
	return result, nil
}

func (self *SElasticache) GetUserById(id string) (*SElasticacheUser, error) {
	users, err := self.region.DescribeUsers("")
	if err != nil {
		return nil, errors.Wrap(err, "self.region.DescribeUsers")
	}
	for i := range users {
		temp := SElasticacheUser{region: self.region, user: users[i]}
		if temp.GetId() == id {
			return &temp, nil
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

func (self *SElasticache) GetSnapshots() ([]SElasticacheSnapshop, error) {
	result := []SElasticacheSnapshop{}
	snapshots, err := self.region.DescribeSnapshots(self.GetName(), "")
	if err != nil {
		return nil, errors.Wrapf(err, " self.region.DescribeSnapshots(%s)", self.GetName())
	}

	for i := range snapshots {
		result = append(result, SElasticacheSnapshop{region: self.region, snapshot: snapshots[i]})
	}
	return result, nil
}

func (self *SElasticache) GetICloudElasticcacheBackups() ([]cloudprovider.ICloudElasticcacheBackup, error) {
	snapshots, err := self.GetSnapshots()
	if err != nil {
		return nil, errors.Wrap(err, "self.GetSnapshots()")
	}
	result := []cloudprovider.ICloudElasticcacheBackup{}
	for i := range snapshots {
		result = append(result, &snapshots[i])
	}
	return result, nil
}

func (self *SElasticache) GetICloudElasticcacheBackup(backupId string) (cloudprovider.ICloudElasticcacheBackup, error) {
	snapshots, err := self.GetSnapshots()
	if err != nil {
		return nil, errors.Wrap(err, "self.GetSnapshots()")
	}
	for i := range snapshots {
		if snapshots[i].GetId() == backupId {
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
		if cluster != nil && cluster.CacheParameterGroup != nil && cluster.CacheParameterGroup.CacheParameterGroupName != nil {
			return *cluster.CacheParameterGroup.CacheParameterGroupName, nil
		}
	}

	return "", cloudprovider.ErrNotFound
}

func (self *SElasticache) GetParameters() ([]SElasticacheParameter, error) {
	groupName, err := self.GetParameterGroupName()
	if err != nil {
		return nil, errors.Wrap(err, "self.GetParameterGroupName()")
	}
	parameters, err := self.region.DescribeCacheParameters(groupName)
	if err != nil {
		return nil, errors.Wrapf(err, "self.region.DescribeCacheParameters(%s)", groupName)
	}

	result := []SElasticacheParameter{}

	for i := range parameters {
		result = append(result, SElasticacheParameter{parameterGroup: groupName, parameter: parameters[i]})
	}
	return result, nil
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
		if cluster != nil {
			for _, securityGroup := range cluster.SecurityGroups {
				if securityGroup != nil && securityGroup.Status != nil && *securityGroup.Status == "active" {
					if securityGroup.SecurityGroupId != nil {
						result = append(result, *securityGroup.SecurityGroupId)
					}
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
