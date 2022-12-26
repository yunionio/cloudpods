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
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SMemcached struct {
	multicloud.SElasticcacheBase
	QcloudTags
	region *SRegion

	InstanceId        string    `json:"InstanceId"`
	InstanceName      string    `json:"InstanceName"`
	AppId             int       `json:"AppId"`
	ProjectId         int       `json:"ProjectId"`
	InstanceDesc      string    `json:"InstanceDesc"`
	Vip               string    `json:"Vip"`
	Vport             int       `json:"Vport"`
	Status            int       `json:"Status"`
	AutoRenewFlag     int       `json:"AutoRenewFlag"`
	VpcId             int       `json:"VpcId"`
	SubnetId          int       `json:"SubnetId"`
	PayMode           int       `json:"PayMode"`
	ZoneId            int       `json:"ZoneId"`
	Expire            int       `json:"Expire"`
	RegionId          int       `json:"RegionId"`
	AddTimeStamp      time.Time `json:"AddTimeStamp"`
	ModtimeStamp      time.Time `json:"ModTimeStamp"`
	IsolateTimesSamp  time.Time `json:"IsolateTimeStamp"`
	UniqVpcId         string    `json:"UniqVpcId"`
	UniqSubnetId      string    `json:"UniqSubnetId"`
	DeadlineTimeStamp string    `json:"DeadlineTimeStamp"`
	SetId             int       `json:"SetId"`
	CmemId            int       `json:"CmemId"`
}

func (self *SMemcached) GetId() string {
	return self.InstanceId
}

func (self *SMemcached) GetName() string {
	return self.InstanceName
}

func (self *SMemcached) GetGlobalId() string {
	return self.GetId()
}

func (self *SMemcached) GetStatus() string {
	switch self.Status {
	case 0:
		return api.ELASTIC_CACHE_STATUS_DEPLOYING
	case 1:
		return api.ELASTIC_CACHE_STATUS_RUNNING
	case 2:
		return api.ELASTIC_CACHE_STATUS_CREATE_FAILED
	case 4, 5, 6, 7:
		return api.ELASTIC_CACHE_STATUS_RELEASING
	default:
		return fmt.Sprintf("%d", self.Status)
	}
}

func (self *SMemcached) GetProjectId() string {
	return strconv.Itoa(self.ProjectId)
}

func (self *SMemcached) GetBillingType() string {
	// 计费模式：0-按量计费，1-包年包月
	if self.PayMode == 1 {
		return billing_api.BILLING_TYPE_PREPAID
	} else {
		return billing_api.BILLING_TYPE_POSTPAID
	}
}

func (self *SMemcached) GetCreatedAt() time.Time {
	return self.AddTimeStamp
}

func (self *SMemcached) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SMemcached) SetAutoRenew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (self *SMemcached) IsAutoRenew() bool {
	return self.AutoRenewFlag == 1
}

func (self *SMemcached) GetInstanceType() string {
	return ""
}

func (self *SMemcached) GetCapacityMB() int {
	return 0
}

func (self *SMemcached) GetArchType() string {
	return api.ELASTIC_CACHE_NODE_TYPE_SINGLE
}

func (self *SMemcached) GetNodeType() string {
	return api.ELASTIC_CACHE_ARCH_TYPE_SINGLE
}

func (self *SMemcached) GetEngine() string {
	return api.ELASTIC_CACHE_ENGINE_MEMCACHED
}

func (self *SMemcached) GetEngineVersion() string {
	return "latest"
}

func (self *SMemcached) GetVpcId() string {
	return self.UniqVpcId
}

func (self *SMemcached) GetZoneId() string {
	return fmt.Sprintf("%s/%s-%d", self.region.GetGlobalId(), self.region.Region, self.ZoneId%10)
}

func (self *SMemcached) GetNetworkType() string {
	if len(self.UniqSubnetId) > 0 {
		return api.LB_NETWORK_TYPE_VPC
	}
	return api.LB_NETWORK_TYPE_CLASSIC
}

func (self *SMemcached) GetNetworkId() string {
	return self.UniqSubnetId
}

func (self *SMemcached) GetPrivateDNS() string {
	return ""
}

func (self *SMemcached) GetPrivateIpAddr() string {
	return self.Vip
}

func (self *SMemcached) GetPrivateConnectPort() int {
	return self.Vport
}

func (self *SMemcached) GetPublicDNS() string {
	return ""
}

func (self *SMemcached) GetPublicIpAddr() string {
	return ""
}

func (self *SMemcached) GetPublicConnectPort() int {
	return 0
}

func (self *SMemcached) GetMaintainStartTime() string {
	return ""
}

func (self *SMemcached) GetMaintainEndTime() string {
	return ""
}

func (self *SMemcached) GetAuthMode() string {
	return "off"
}

func (self *SMemcached) GetSecurityGroupIds() ([]string, error) {
	return []string{}, nil
}

func (self *SMemcached) GetICloudElasticcacheAccounts() ([]cloudprovider.ICloudElasticcacheAccount, error) {
	return []cloudprovider.ICloudElasticcacheAccount{}, nil
}

func (self *SMemcached) GetICloudElasticcacheAcls() ([]cloudprovider.ICloudElasticcacheAcl, error) {
	return []cloudprovider.ICloudElasticcacheAcl{}, nil
}

func (self *SMemcached) GetICloudElasticcacheBackups() ([]cloudprovider.ICloudElasticcacheBackup, error) {
	return []cloudprovider.ICloudElasticcacheBackup{}, nil
}

func (self *SMemcached) GetICloudElasticcacheParameters() ([]cloudprovider.ICloudElasticcacheParameter, error) {
	return []cloudprovider.ICloudElasticcacheParameter{}, nil
}

func (self *SMemcached) GetICloudElasticcacheAccount(accountId string) (cloudprovider.ICloudElasticcacheAccount, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SMemcached) GetICloudElasticcacheAcl(aclId string) (cloudprovider.ICloudElasticcacheAcl, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SMemcached) GetICloudElasticcacheBackup(backupId string) (cloudprovider.ICloudElasticcacheBackup, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SMemcached) Restart() error {
	return errors.Wrap(cloudprovider.ErrNotSupported, "Restart")
}

func (self *SMemcached) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (self *SMemcached) CleanupInstance() error {
	return cloudprovider.ErrNotSupported
}

func (self *SMemcached) ChangeInstanceSpec(spec string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SMemcached) SetMaintainTime(maintainStartTime, maintainEndTime string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SMemcached) AllocatePublicConnection(port int) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SMemcached) ReleasePublicConnection() error {
	return errors.Wrap(cloudprovider.ErrNotSupported, "ReleasePublicConnection")
}

func (self *SMemcached) CreateAccount(account cloudprovider.SCloudElasticCacheAccountInput) (cloudprovider.ICloudElasticcacheAccount, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SMemcached) CreateAcl(aclName, securityIps string) (cloudprovider.ICloudElasticcacheAcl, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotSupported, "CreateAcl")
}

func (self *SMemcached) CreateBackup(desc string) (cloudprovider.ICloudElasticcacheBackup, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SMemcached) FlushInstance(input cloudprovider.SCloudElasticCacheFlushInstanceInput) error {
	return cloudprovider.ErrNotSupported
}

func (self *SMemcached) UpdateAuthMode(noPasswordAccess bool, password string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SMemcached) UpdateInstanceParameters(config jsonutils.JSONObject) error {
	return cloudprovider.ErrNotSupported
}

func (self *SMemcached) UpdateBackupPolicy(config cloudprovider.SCloudElasticCacheBackupPolicyUpdateInput) error {
	return cloudprovider.ErrNotSupported
}

func (self *SMemcached) UpdateSecurityGroups(secgroupIds []string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SMemcached) Renew(bc billing.SBillingCycle) error {
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) GetMemcaches(ids []string, limit, offset int) ([]SMemcached, int, error) {
	params := map[string]string{}
	if limit <= 0 {
		limit = 100
	}
	params["Limit"] = fmt.Sprintf("%d", limit)
	params["Offset"] = fmt.Sprintf("%d", offset)
	for i, id := range ids {
		params[fmt.Sprintf("InstanceIds.%d", i)] = id
	}
	resp, err := self.memcachedRequest("DescribeInstances", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeInstances")
	}
	ret := []SMemcached{}
	err = resp.Unmarshal(&ret, "InstanceList")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal InstanceList")
	}
	total, _ := resp.Float("TotalCount")
	return ret, int(total), nil
}
