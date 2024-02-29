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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SupportedFeatures struct {
	SupportedFeature []string
}

type MountTargets struct {
	MountTarget []SMountTarget
}

type Package struct {
}

type Packages struct {
	Package []Package
}

type SFileSystem struct {
	multicloud.SNasBase
	AliyunTags
	region *SRegion

	Status                string
	Description           string
	StorageType           string
	MountTargetCountLimit int
	Ldap                  string
	ZoneId                string
	// 2021-03-22T10:08:15CST
	CreateTime           string
	MeteredIASize        int
	SupportedFeatures    SupportedFeatures
	MountTargets         MountTargets
	AutoSnapshotPolicyId string
	MeteredSize          int64
	EncryptType          int
	Capacity             int64
	ProtocolType         string
	ChargeType           string
	Packages             Packages
	ExpiredTime          string
	FileSystemType       string
	FileSystemId         string
	RegionId             string
	ResourceGroupId      string
}

func (self *SFileSystem) GetId() string {
	return self.FileSystemId
}

func (self *SFileSystem) GetGlobalId() string {
	return self.FileSystemId
}

func (self *SFileSystem) GetName() string {
	if len(self.Description) > 0 {
		return self.Description
	}
	return self.FileSystemId
}

func (self *SFileSystem) GetFileSystemType() string {
	return self.FileSystemType
}

func (self *SFileSystem) GetMountTargetCountLimit() int {
	return self.MountTargetCountLimit
}

func (self *SFileSystem) GetStatus() string {
	switch self.Status {
	case "", "Running":
		return api.NAS_STATUS_AVAILABLE
	case "Extending":
		return api.NAS_STATUS_EXTENDING
	case "Stopping", "Stopped":
		return api.NAS_STATUS_UNAVAILABLE
	case "Pending":
		return api.NAS_STATUS_CREATING
	default:
		return api.NAS_STATUS_UNKNOWN
	}
}

func (self *SFileSystem) GetBillintType() string {
	if self.ChargeType == "PayAsYouGo" {
		return billing_api.BILLING_TYPE_POSTPAID
	}
	return billing_api.BILLING_TYPE_PREPAID
}

func (self *SFileSystem) GetStorageType() string {
	return strings.ToLower(self.StorageType)
}

func (self *SFileSystem) GetProtocol() string {
	return self.ProtocolType
}

func (self *SFileSystem) GetCapacityGb() int64 {
	return self.Capacity
}

func (self *SFileSystem) GetUsedCapacityGb() int64 {
	return self.MeteredSize
}

func (self *SFileSystem) GetZoneId() string {
	return self.ZoneId
}

func (self *SFileSystem) Delete() error {
	return self.region.DeleteFileSystem(self.FileSystemId)
}

func (self *SFileSystem) GetCreatedAt() time.Time {
	ret, _ := time.Parse("2006-01-02T15:04:05CST", self.CreateTime)
	if !ret.IsZero() {
		ret = ret.Add(time.Hour * 8)
	}
	return ret
}

func (self *SFileSystem) GetExpiredAt() time.Time {
	ret, _ := time.Parse("2006-01-02T15:04:05CST", self.ExpiredTime)
	if !ret.IsZero() {
		ret = ret.Add(time.Hour * 8)
	}
	return ret
}

func (self *SFileSystem) Refresh() error {
	fs, err := self.region.GetFileSystem(self.FileSystemId)
	if err != nil {
		return errors.Wrapf(err, "GetFileSystem(%s)", self.FileSystemId)
	}
	return jsonutils.Update(self, fs)
}

func (self *SRegion) GetFileSystems(id string, pageSize, pageNum int) ([]SFileSystem, int, error) {
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	if pageNum < 1 {
		pageNum = 1
	}
	params := map[string]string{
		"RegionId":   self.RegionId,
		"PageSize":   fmt.Sprintf("%d", pageSize),
		"PageNumber": fmt.Sprintf("%d", pageNum),
	}
	if len(id) > 0 {
		params["FileSystemId"] = id
	}
	resp, err := self.nasRequest("DescribeFileSystems", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeFileSystems")
	}
	ret := struct {
		TotalCount  int
		FileSystems struct {
			FileSystem []SFileSystem
		}
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	return ret.FileSystems.FileSystem, ret.TotalCount, nil
}

func (self *SRegion) GetICloudFileSystems() ([]cloudprovider.ICloudFileSystem, error) {
	nas := []SFileSystem{}
	num := 1
	for {
		part, total, err := self.GetFileSystems("", 100, num)
		if err != nil {
			return nil, errors.Wrapf(err, "GetFileSystems")
		}
		nas = append(nas, part...)
		if total <= len(nas) {
			break
		}
		num++
	}

	ret := []cloudprovider.ICloudFileSystem{}
	for i := range nas {
		nas[i].region = self
		ret = append(ret, &nas[i])
	}
	return ret, nil
}

func (self *SRegion) GetFileSystem(id string) (*SFileSystem, error) {
	if len(id) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty id")
	}
	nas, total, err := self.GetFileSystems(id, 1, 1)
	if err != nil {
		return nil, errors.Wrapf(err, "GetFileSystems")
	}
	if total == 1 {
		nas[0].region = self
		return &nas[0], nil
	}
	if total == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
	}
	return nil, errors.Wrapf(cloudprovider.ErrDuplicateId, id)
}

func (self *SRegion) GetICloudFileSystemById(id string) (cloudprovider.ICloudFileSystem, error) {
	fs, err := self.GetFileSystem(id)
	if err != nil {
		return nil, errors.Wrapf(err, "self.GetFileSystem")
	}
	return fs, nil
}

func (self *SRegion) CreateMountTarget(opts *cloudprovider.SMountTargetCreateOptions) (*SMountTarget, error) {
	params := map[string]string{
		"RegionId":        self.RegionId,
		"FileSystemId":    opts.FileSystemId,
		"AccessGroupName": strings.TrimPrefix(strings.TrimPrefix(opts.AccessGroupId, "extreme/"), "standard/"),
		"NetworkType":     utils.Capitalize(opts.NetworkType),
		"VpcId":           opts.VpcId,
		"VSwitchId":       opts.NetworkId,
	}
	resp, err := self.nasRequest("CreateMountTarget", params)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateMountTarget")
	}
	ret := struct {
		MountTargetDomain string
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrapf(err, "resp.Unmarshal")
	}
	mts, _, err := self.GetMountTargets(opts.FileSystemId, ret.MountTargetDomain, 10, 1)
	if err != nil {
		return nil, errors.Wrapf(err, "self.GetMountTargets")
	}
	for i := range mts {
		if mts[i].MountTargetDomain == ret.MountTargetDomain {
			return &mts[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "afeter create with mount domain %s", ret.MountTargetDomain)
}

func (self *SFileSystem) CreateMountTarget(opts *cloudprovider.SMountTargetCreateOptions) (cloudprovider.ICloudMountTarget, error) {
	mt, err := self.region.CreateMountTarget(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateMountTarget")
	}
	mt.fs = self
	return mt, nil
}

func (self *SRegion) DeleteFileSystem(id string) error {
	params := map[string]string{
		"RegionId":     self.RegionId,
		"FileSystemId": id,
	}
	_, err := self.nasRequest("DeleteFileSystem", params)
	return errors.Wrapf(err, "DeleteFileSystem")
}

func (self *SRegion) CreateICloudFileSystem(opts *cloudprovider.FileSystemCraeteOptions) (cloudprovider.ICloudFileSystem, error) {
	fs, err := self.CreateFileSystem(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "self.CreateFileSystem")
	}
	return fs, nil
}

func (self *SRegion) CreateFileSystem(opts *cloudprovider.FileSystemCraeteOptions) (*SFileSystem, error) {
	params := map[string]string{
		"RegionId":       self.RegionId,
		"ProtocolType":   opts.Protocol,
		"ZoneId":         opts.ZoneId,
		"EncryptType":    "0",
		"FileSystemType": opts.FileSystemType,
		"StorageType":    opts.StorageType,
		"ClientToken":    utils.GenRequestId(20),
		"Description":    opts.Name,
	}

	if self.GetCloudEnv() == ALIYUN_FINANCE_CLOUDENV {
		if opts.FileSystemType == "standard" {
			opts.ZoneId = strings.Replace(opts.ZoneId, "cn-shanghai-finance-1", "jr-cn-shanghai-", 1)
			opts.ZoneId = strings.Replace(opts.ZoneId, "cn-shenzhen-finance-1", "jr-cn-shenzhen-", 1)
			params["ZoneId"] = opts.ZoneId
		}
	}

	switch opts.FileSystemType {
	case "standard":
		params["StorageType"] = utils.Capitalize(opts.StorageType)
	case "cpfs":
		params["ProtocolType"] = "cpfs"
		switch opts.StorageType {
		case "advance_100":
			params["Bandwidth"] = "100"
		case "advance_200":
			params["Bandwidth"] = "200"
		}
	case "extreme":
		params["Capacity"] = fmt.Sprintf("%d", opts.Capacity)
	}
	if len(opts.VpcId) > 0 {
		params["VpcId"] = opts.VpcId
	}
	if len(opts.NetworkId) > 0 {
		params["VSwitchId"] = opts.NetworkId
	}
	if opts.BillingCycle != nil {
		params["ChargeType"] = "Subscription"
		params["Duration"] = fmt.Sprintf("%d", opts.BillingCycle.GetMonths())
	}
	resp, err := self.nasRequest("CreateFileSystem", params)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateFileSystem")
	}
	fsId, _ := resp.GetString("FileSystemId")
	return self.GetFileSystem(fsId)
}

func (self *SFileSystem) SetTags(tags map[string]string, replace bool) error {
	return self.region.SetResourceTags(ALIYUN_SERVICE_NAS, "filesystem", self.FileSystemId, tags, replace)
}

func (self *SFileSystem) GetProjectId() string {
	return self.ResourceGroupId
}
