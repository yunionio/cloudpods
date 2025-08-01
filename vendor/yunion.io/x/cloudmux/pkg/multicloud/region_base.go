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

package multicloud

import (
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SRegion struct {
	SResourceBase
	STagBase
}

func (r *SRegion) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIDiskById")
}

func (r *SRegion) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIHostById")
}

func (r *SRegion) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIHosts")
}

func (r *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetISnapshotById")
}

func (r *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetISnapshots")
}

func (r *SRegion) GetISecurityGroups() ([]cloudprovider.ICloudSecurityGroup, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetISecurityGroups")
}

func (r *SRegion) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIStorageById")
}

func (r *SRegion) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIStoragecacheById")
}

func (r *SRegion) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIStoragecaches")
}

func (r *SRegion) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIStorages")
}

func (r *SRegion) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIVMs")
}

func (r *SRegion) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIVMById")
}

func (r *SRegion) CreateSnapshotPolicy(input *cloudprovider.SnapshotPolicyInput) (string, error) {
	return "", errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateSnapshotPolicy")
}

func (r *SRegion) GetISnapshotPolicyById(snapshotPolicyId string) (cloudprovider.ICloudSnapshotPolicy, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetISnapshotPolicyById")
}

func (self *SRegion) GetISnapshotPolicies() ([]cloudprovider.ICloudSnapshotPolicy, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetISnapshotPolicies")
}

func (self *SRegion) GetISkus() ([]cloudprovider.ICloudSku, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "GetISkus")
}

func (self *SRegion) CreateISku(opts *cloudprovider.SServerSkuCreateOption) (cloudprovider.ICloudSku, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateISku")
}

func (self *SRegion) GetINetworkInterfaces() ([]cloudprovider.ICloudNetworkInterface, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetINetworkInterfaces")
}

func (self *SRegion) GetIDBInstances() ([]cloudprovider.ICloudDBInstance, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIDBInstances")
}

func (self *SRegion) GetIDBInstanceById(instanceId string) (cloudprovider.ICloudDBInstance, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIDBInstanceById")
}

func (self *SRegion) GetIDBInstanceBackups() ([]cloudprovider.ICloudDBInstanceBackup, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIDBInstanceBackups")
}

func (self *SRegion) GetIDBInstanceBackupById(backupId string) (cloudprovider.ICloudDBInstanceBackup, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIDBInstanceBackupById")
}

func (self *SRegion) GetIElasticcaches() ([]cloudprovider.ICloudElasticcache, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIElasticcaches")
}

func (self *SRegion) GetIElasticcacheSkus() ([]cloudprovider.ICloudElasticcacheSku, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIElasticcacheSkus")
}

func (self *SRegion) CreateIDBInstance(desc *cloudprovider.SManagedDBInstanceCreateConfig) (cloudprovider.ICloudDBInstance, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateIDBInstance")
}

func (self *SRegion) CreateIElasticcaches(ec *cloudprovider.SCloudElasticCacheInput) (cloudprovider.ICloudElasticcache, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateIElasticcaches")
}

func (self *SRegion) GetIElasticcacheById(id string) (cloudprovider.ICloudElasticcache, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIElasticcacheById")
}

func (self *SRegion) GetICloudEvents(start time.Time, end time.Time, withReadEvent bool) ([]cloudprovider.ICloudEvent, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudEvents")
}

func (self *SRegion) GetICloudQuotas() ([]cloudprovider.ICloudQuota, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudQuotas")
}

func (self *SRegion) CreateInternetGateway() (cloudprovider.ICloudInternetGateway, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "CreateInternetGateway")
}

func (self *SRegion) GetICloudFileSystems() ([]cloudprovider.ICloudFileSystem, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudFileSystems")
}

func (self *SRegion) GetICloudFileSystemById(id string) (cloudprovider.ICloudFileSystem, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudFileSystemById")
}

func (self *SRegion) GetICloudAccessGroups() ([]cloudprovider.ICloudAccessGroup, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudAccessGroups")
}

func (self *SRegion) GetICloudAccessGroupById(id string) (cloudprovider.ICloudAccessGroup, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudAccessGroupById")
}

func (self *SRegion) CreateICloudAccessGroup(opts *cloudprovider.SAccessGroup) (cloudprovider.ICloudAccessGroup, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateICloudAccessGroup")
}

func (self *SRegion) CreateICloudFileSystem(opts *cloudprovider.FileSystemCraeteOptions) (cloudprovider.ICloudFileSystem, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateICloudFileSystem")
}

func (self *SRegion) GetICloudWafIPSets() ([]cloudprovider.ICloudWafIPSet, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudWafIPSets")
}

func (self *SRegion) GetICloudWafRegexSets() ([]cloudprovider.ICloudWafRegexSet, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudWafRegexSets")
}

func (self *SRegion) GetICloudWafInstances() ([]cloudprovider.ICloudWafInstance, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudWafInstances")
}

func (self *SRegion) GetICloudWafInstanceById(id string) (cloudprovider.ICloudWafInstance, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudWafInstanceById")
}

func (self *SRegion) CreateICloudWafInstance(opts *cloudprovider.WafCreateOptions) (cloudprovider.ICloudWafInstance, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateICloudWafInstance")
}

func (self *SRegion) GetICloudWafRuleGroups() ([]cloudprovider.ICloudWafRuleGroup, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudWafRuleGroups")
}

func (self *SRegion) GetICloudMongoDBs() ([]cloudprovider.ICloudMongoDB, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudMongoDBs")
}

func (self *SRegion) GetICloudMongoDBById(id string) (cloudprovider.ICloudMongoDB, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudMongoDBById")
}

func (self *SRegion) GetIElasticSearchs() ([]cloudprovider.ICloudElasticSearch, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIElasticSearchs")
}

func (self *SRegion) GetIElasticSearchById(id string) (cloudprovider.ICloudElasticSearch, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIElasticSearchById")
}

func (self *SRegion) GetICloudKafkas() ([]cloudprovider.ICloudKafka, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudKafkas")
}

func (self *SRegion) GetICloudKafkaById(id string) (cloudprovider.ICloudKafka, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudKafkaById")
}

func (self *SRegion) GetICloudApps() ([]cloudprovider.ICloudApp, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudApps")
}

func (self *SRegion) GetICloudAppById(id string) (cloudprovider.ICloudApp, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudAppById")
}

func (self *SRegion) GetIDBInstanceSkus() ([]cloudprovider.ICloudDBInstanceSku, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIDBInstanceSkus")
}

func (self *SRegion) GetICloudNatSkus() ([]cloudprovider.ICloudNatSku, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudNatSkus")
}

func (self *SRegion) GetICloudKubeClusters() ([]cloudprovider.ICloudKubeCluster, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudKubeClusters")
}

func (self *SRegion) CreateIKubeCluster(opts *cloudprovider.KubeClusterCreateOptions) (cloudprovider.ICloudKubeCluster, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateIKubeCluster")
}

func (self *SRegion) GetICloudKubeClusterById(id string) (cloudprovider.ICloudKubeCluster, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudKubeClusterById")
}

func (self *SRegion) GetICloudTablestores() ([]cloudprovider.ICloudTablestore, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudTablestores")
}

type SRegionZoneBase struct {
}

func (self *SRegionZoneBase) GetIZones() ([]cloudprovider.ICloudZone, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIZones")
}

func (self *SRegionZoneBase) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIZoneById")
}

type SRegionVpcBase struct {
}

func (self *SRegionVpcBase) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateIVpc")
}

func (self *SRegionVpcBase) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIVpcs")
}

func (self *SRegionVpcBase) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIVpcById")
}

type SRegionOssBase struct {
}

func (self *SRegionOssBase) CreateIBucket(name string, storageClassStr string, acl string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateIBucket")
}

func (self *SRegionOssBase) GetIBucketById(id string) (cloudprovider.ICloudBucket, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIBucketById")
}

func (self *SRegionOssBase) GetIBuckets() ([]cloudprovider.ICloudBucket, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIBuckets")
}

func (self *SRegionOssBase) IBucketExist(name string) (bool, error) {
	return false, cloudprovider.ErrNotImplemented
}

func (self *SRegionOssBase) GetIBucketByName(name string) (cloudprovider.ICloudBucket, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIBucketByName")
}

func (self *SRegionOssBase) DeleteIBucket(name string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "DeleteIBucket")
}

type SRegionLbBase struct {
}

func (self *SRegionLbBase) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetILoadBalancers")
}

func (self *SRegionLbBase) GetILoadBalancerAcls() ([]cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetILoadBalancerAcls")
}

func (self *SRegionLbBase) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetILoadBalancerCertificates")
}

func (self *SRegionLbBase) GetILoadBalancerById(loadbalancerId string) (cloudprovider.ICloudLoadbalancer, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetILoadBalancerById")
}

func (self *SRegionLbBase) GetILoadBalancerAclById(aclId string) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetILoadBalancerAclById")
}

func (self *SRegionLbBase) GetILoadBalancerCertificateById(certId string) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetILoadBalancerCertificateById")
}

func (self *SRegionLbBase) CreateILoadBalancer(loadbalancer *cloudprovider.SLoadbalancerCreateOptions) (cloudprovider.ICloudLoadbalancer, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateILoadBalancer")
}

func (self *SRegionLbBase) CreateILoadBalancerAcl(acl *cloudprovider.SLoadbalancerAccessControlList) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateILoadBalancerAcl")
}

func (self *SRegionLbBase) CreateILoadBalancerCertificate(cert *cloudprovider.SLoadbalancerCertificate) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateILoadBalancerCertificate")
}

type SRegionSecurityGroupBase struct {
}

func (self *SRegionSecurityGroupBase) CreateISecurityGroup(conf *cloudprovider.SecurityGroupCreateInput) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateISecurityGroup")
}

func (self *SRegionSecurityGroupBase) GetISecurityGroupById(secgroupId string) (cloudprovider.ICloudSecurityGroup, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetISecurityGroupById")
}

type SRegionEipBase struct {
}

func (self *SRegionEipBase) GetIEipById(id string) (cloudprovider.ICloudEIP, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIEipById")
}

func (self *SRegionEipBase) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIEips")
}

func (self *SRegionEipBase) CreateEIP(eip *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateEIP")
}

func (self *SRegion) GetIModelartsPools() ([]cloudprovider.ICloudModelartsPool, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIModelartsPools")
}

func (self *SRegion) GetIModelartsPoolById(id string) (cloudprovider.ICloudModelartsPool, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIModelartsPoolDetail")
}

func (self *SRegion) CreateIModelartsPool(pool *cloudprovider.ModelartsPoolCreateOption, callback func(id string)) (cloudprovider.ICloudModelartsPool, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateIModelartsPool")
}

func (self *SRegion) GetStatusMessage() string {
	return ""
}

func (self *SRegion) GetIModelartsPoolSku() ([]cloudprovider.ICloudModelartsPoolSku, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIModelartsPoolSku")
}

func (self *SRegion) GetIMiscResources() ([]cloudprovider.ICloudMiscResource, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetIMiscResources")
}

func (self *SRegion) GetISSLCertificates() ([]cloudprovider.ICloudSSLCertificate, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetISSLCertificate")
}

func (self *SRegion) GetILoadBalancerHealthChecks() ([]cloudprovider.ICloudLoadbalancerHealthCheck, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetILoadBalancerHealthChecks")
}

func (self *SRegion) CreateILoadBalancerHealthCheck(healthCheck *cloudprovider.SLoadbalancerHealthCheck) (cloudprovider.ICloudLoadbalancerHealthCheck, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateILoadBalancerHealthCheck")
}
