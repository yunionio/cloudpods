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

package azure

import (
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SEnterpriseRedisCache struct {
	multicloud.SElasticcacheBase
	AzureTags

	region   *SRegion
	ID       string `json:"id"`
	Location string `json:"location"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Sku      struct {
		Name     string `json:"name"`
		Capacity string `json:"capacity"`
	} `json:"sku"`
	Properties struct {
		Provisioningstate string      `json:"provisioningState"`
		Redisversion      string      `json:"redisVersion"`
		Accesskeys        interface{} `json:"accessKeys"`
		Hostname          string      `json:"hostName"`
	} `json:"properties"`
}

func (self *SRegion) GetEnterpriseRedisCache(id string) (*SEnterpriseRedisCache, error) {
	cache := &SEnterpriseRedisCache{region: self}
	return cache, self.get(id, url.Values{}, cache)
}

func (self *SRegion) GetEnterpriseRedisCaches() ([]SEnterpriseRedisCache, error) {
	redis := []SEnterpriseRedisCache{}
	err := self.list("Microsoft.Cache/redisEnterprise", url.Values{}, &redis)
	if err != nil {
		return nil, errors.Wrapf(err, "list")
	}
	return redis, nil
}

func (self *SEnterpriseRedisCache) GetId() string {
	return self.ID
}

func (self *SEnterpriseRedisCache) GetName() string {
	return self.Name
}

func (self *SEnterpriseRedisCache) GetProjectId() string {
	return getResourceGroup(self.ID)
}

func (self *SEnterpriseRedisCache) GetStatus() string {
	switch self.Properties.Provisioningstate {
	case "Creating":
		return api.ELASTIC_CACHE_STATUS_DEPLOYING
	case "Deleting", "Canceled":
		return api.ELASTIC_CACHE_STATUS_DELETING
	case "Disabled":
		return api.ELASTIC_CACHE_STATUS_INACTIVE
	case "Failed":
		return api.ELASTIC_CACHE_STATUS_CREATE_FAILED
	case "Succeeded", "Updating":
		return api.ELASTIC_CACHE_STATUS_RUNNING
	default:
		return strings.ToLower(self.Properties.Provisioningstate)
	}
}

func (self *SEnterpriseRedisCache) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SEnterpriseRedisCache) GetInstanceType() string {
	return self.Sku.Name
}

func (self *SEnterpriseRedisCache) GetCapacityMB() int {
	switch self.Sku.Name {
	case "EnterpriseFlash_F1500":
		return 1455 * 1024
	case "EnterpriseFlash_F300":
		return 345 * 1024
	case "EnterpriseFlash_F700":
		return 715 * 1024
	case "Enterprise_E10":
		return 12 * 1024
	case "Enterprise_E100":
		return 100 * 1024
	case "Enterprise_E20":
		return 25 * 1024
	case "Enterprise_E50":
		return 50 * 1024
	}
	return 0
}

func (self *SEnterpriseRedisCache) GetArchType() string {
	return api.ELASTIC_CACHE_ARCH_TYPE_CLUSTER
}

func (self *SEnterpriseRedisCache) GetNodeType() string {
	return api.ELASTIC_CACHE_NODE_TYPE_SINGLE
}

func (self *SEnterpriseRedisCache) GetEngine() string {
	return "Redis"
}

func (self *SEnterpriseRedisCache) GetEngineVersion() string {
	if len(self.Properties.Redisversion) == 0 {
		return "latest"
	}
	return self.Properties.Redisversion
}

func (self *SEnterpriseRedisCache) GetVpcId() string {
	return ""
}

func (self *SEnterpriseRedisCache) GetZoneId() string {
	return self.region.getZone().GetGlobalId()
}

func (self *SEnterpriseRedisCache) GetNetworkType() string {
	return api.LB_NETWORK_TYPE_CLASSIC
}

func (self *SEnterpriseRedisCache) GetNetworkId() string {
	return ""
}

func (self *SEnterpriseRedisCache) GetPrivateDNS() string {
	return ""
}

func (self *SEnterpriseRedisCache) GetPrivateIpAddr() string {
	return ""
}

func (self *SEnterpriseRedisCache) GetPrivateConnectPort() int {
	return 10000
}

func (self *SEnterpriseRedisCache) GetPublicDNS() string {
	return self.Properties.Hostname
}

func (self *SEnterpriseRedisCache) GetPublicIpAddr() string {
	return ""
}

func (self *SEnterpriseRedisCache) GetPublicConnectPort() int {
	return 10000
}

func (self *SEnterpriseRedisCache) GetMaintainStartTime() string {
	return ""
}

func (self *SEnterpriseRedisCache) GetMaintainEndTime() string {
	return ""
}

func (self *SEnterpriseRedisCache) AllocatePublicConnection(port int) (string, error) {
	return "", errors.Wrapf(cloudprovider.ErrNotImplemented, "AllocatePublicConnection")
}

func (self *SEnterpriseRedisCache) ChangeInstanceSpec(spec string) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "ChangeInstanceSpec")
}

func (self *SEnterpriseRedisCache) CreateAccount(account cloudprovider.SCloudElasticCacheAccountInput) (cloudprovider.ICloudElasticcacheAccount, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateAccount")
}

func (self *SEnterpriseRedisCache) CreateAcl(aclName, securityIps string) (cloudprovider.ICloudElasticcacheAcl, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateAcl")
}

func (self *SEnterpriseRedisCache) CreateBackup(desc string) (cloudprovider.ICloudElasticcacheBackup, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "CreateBackup")
}

func (self *SEnterpriseRedisCache) Delete() error {
	return self.region.Delete(self.ID)
}

func (self *SEnterpriseRedisCache) FlushInstance(input cloudprovider.SCloudElasticCacheFlushInstanceInput) error {
	return errors.Wrapf(cloudprovider.ErrNotSupported, "FlushInstance")
}

func (self *SEnterpriseRedisCache) GetAuthMode() string {
	return "on"
}

func (self *SEnterpriseRedisCache) GetICloudElasticcacheAccounts() ([]cloudprovider.ICloudElasticcacheAccount, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetICloudElasticcacheAccounts")
}

func (self *SEnterpriseRedisCache) GetICloudElasticcacheAcls() ([]cloudprovider.ICloudElasticcacheAcl, error) {
	return []cloudprovider.ICloudElasticcacheAcl{}, nil
}

func (self *SEnterpriseRedisCache) GetICloudElasticcacheAcl(aclId string) (cloudprovider.ICloudElasticcacheAcl, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SEnterpriseRedisCache) GetICloudElasticcacheBackups() ([]cloudprovider.ICloudElasticcacheBackup, error) {
	return []cloudprovider.ICloudElasticcacheBackup{}, nil
}

func (self *SEnterpriseRedisCache) GetICloudElasticcacheParameters() ([]cloudprovider.ICloudElasticcacheParameter, error) {
	return []cloudprovider.ICloudElasticcacheParameter{}, nil
}

func (self *SEnterpriseRedisCache) GetICloudElasticcacheAccount(accountId string) (cloudprovider.ICloudElasticcacheAccount, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SEnterpriseRedisCache) GetICloudElasticcacheBackup(backupId string) (cloudprovider.ICloudElasticcacheBackup, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SEnterpriseRedisCache) GetSecurityGroupIds() ([]string, error) {
	return []string{}, nil
}

func (self *SEnterpriseRedisCache) ReleasePublicConnection() error {
	return cloudprovider.ErrNotSupported
}

func (self *SEnterpriseRedisCache) Restart() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SEnterpriseRedisCache) SetMaintainTime(start, end string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SEnterpriseRedisCache) UpdateAuthMode(noPasswordAccess bool, password string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SEnterpriseRedisCache) UpdateBackupPolicy(config cloudprovider.SCloudElasticCacheBackupPolicyUpdateInput) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SEnterpriseRedisCache) UpdateInstanceParameters(config jsonutils.JSONObject) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SEnterpriseRedisCache) UpdateSecurityGroups(secgroupIds []string) error {
	return cloudprovider.ErrNotImplemented
}
