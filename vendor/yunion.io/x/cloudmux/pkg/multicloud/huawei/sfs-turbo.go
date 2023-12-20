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

package huawei

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SfsTurbo struct {
	multicloud.SNasBase
	HuaweiTags
	region *SRegion

	EnterpriseProjectId string
	Actions             []string
	AvailCapacity       float64
	AvailabilityZone    string
	AzName              string
	CreatedAt           time.Time
	CryptKeyId          string
	ExpandType          string
	ExportLocation      string
	Id                  string
	Name                string
	PayModel            string
	Region              string
	SecurityGroupId     string
	ShareProto          string
	ShareType           string
	Size                float64
	Status              string
	SubStatus           string
	SubnetId            string
	VpcId               string
	Description         string
}

func (self *SfsTurbo) GetName() string {
	return self.Name
}

func (self *SfsTurbo) GetId() string {
	return self.Id
}

func (self *SfsTurbo) GetGlobalId() string {
	return self.Id
}

func (self *SfsTurbo) GetFileSystemType() string {
	return "SFS Turbo"
}

func (self *SfsTurbo) Refresh() error {
	sf, err := self.region.GetSfsTurbo(self.Id)
	if err != nil {
		return errors.Wrapf(err, "GetSfsTurbo")
	}
	return jsonutils.Update(self, sf)
}

func (self *SfsTurbo) GetBillingType() string {
	if self.PayModel == "0" {
		return billing_api.BILLING_TYPE_POSTPAID
	}
	return billing_api.BILLING_TYPE_PREPAID
}

func (self *SfsTurbo) GetStorageType() string {
	if len(self.ExpandType) == 0 {
		return strings.ToLower(self.ShareType)
	}
	return strings.ToLower(self.ShareType) + ".enhanced"
}

func (self *SfsTurbo) GetProtocol() string {
	return self.ShareProto
}

func (self *SfsTurbo) GetStatus() string {
	switch self.Status {
	case "100":
		return api.NAS_STATUS_CREATING
	case "200":
		return api.NAS_STATUS_AVAILABLE
	case "300":
		return api.NAS_STATUS_UNKNOWN
	case "303":
		return api.NAS_STATUS_CREATE_FAILED
	case "400":
		return api.NAS_STATUS_DELETING
	case "800":
		return api.NAS_STATUS_UNAVAILABLE
	default:
		return self.Status
	}
}

func (self *SfsTurbo) GetCreatedAt() time.Time {
	return self.CreatedAt
}

func (self *SfsTurbo) GetCapacityGb() int64 {
	return int64(self.Size)
}

func (self *SfsTurbo) GetUsedCapacityGb() int64 {
	return int64(self.Size - self.AvailCapacity)
}

func (self *SfsTurbo) GetMountTargetCountLimit() int {
	return 1
}

func (self *SfsTurbo) GetZoneId() string {
	return self.AvailabilityZone
}

func (self *SfsTurbo) GetMountTargets() ([]cloudprovider.ICloudMountTarget, error) {
	mt := &sMoutTarget{sfs: self}
	return []cloudprovider.ICloudMountTarget{mt}, nil
}

func (self *SfsTurbo) CreateMountTarget(opts *cloudprovider.SMountTargetCreateOptions) (cloudprovider.ICloudMountTarget, error) {
	return nil, errors.Wrap(cloudprovider.ErrNotSupported, "CreateMountTarget")
}

func (self *SfsTurbo) Delete() error {
	return self.region.DeleteSfsTurbo(self.Id)
}

func (self *SRegion) GetICloudFileSystems() ([]cloudprovider.ICloudFileSystem, error) {
	sfs, err := self.GetSfsTurbos()
	if err != nil {
		return nil, errors.Wrapf(err, "self.GetSfsTurbos")
	}
	ret := []cloudprovider.ICloudFileSystem{}
	for i := range sfs {
		sfs[i].region = self
		ret = append(ret, &sfs[i])
	}
	return ret, nil
}

func (self *SRegion) GetICloudFileSystemById(id string) (cloudprovider.ICloudFileSystem, error) {
	sf, err := self.GetSfsTurbo(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetSfsTurbo(%s)", id)
	}
	return sf, nil
}

func (self *SRegion) GetSfsTurbos() ([]SfsTurbo, error) {
	query := url.Values{}
	sfs := []SfsTurbo{}
	for {
		resp, err := self.list(SERVICE_SFS, "sfs-turbo/shares/detail", query)
		if err != nil {
			return nil, err
		}
		part := struct {
			Shares []SfsTurbo
			Count  int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		sfs = append(sfs, part.Shares...)
		if len(part.Shares) == 0 || len(sfs) >= part.Count {
			break
		}
		query.Set("offset", fmt.Sprintf("%d", len(sfs)))
	}
	return sfs, nil
}

func (self *SRegion) GetSfsTurbo(id string) (*SfsTurbo, error) {
	resp, err := self.list(SERVICE_SFS, "sfs-turbo/shares/"+id, nil)
	if err != nil {
		return nil, err
	}
	sf := &SfsTurbo{region: self}
	err = resp.Unmarshal(sf)
	if err != nil {
		return nil, err
	}
	return sf, nil
}

func (self *SRegion) DeleteSfsTurbo(id string) error {
	_, err := self.delete(SERVICE_SFS, "sfs-turbo/shares/"+id)
	if err != nil {
		return err
	}
	return nil
}

func (self *SRegion) GetSysDefaultSecgroupId() (string, error) {
	secs, err := self.GetSecurityGroups("")
	if err != nil {
		return "", errors.Wrapf(err, "GetSecurityGroups")
	}
	if len(secs) > 0 {
		return secs[0].Id, nil
	}
	return "", fmt.Errorf("not found default security group")
}

func (self *SRegion) CreateICloudFileSystem(opts *cloudprovider.FileSystemCraeteOptions) (cloudprovider.ICloudFileSystem, error) {
	fs, err := self.CreateSfsTurbo(opts)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateSfsTurbo")
	}
	return fs, nil
}

func (self *SRegion) CreateSfsTurbo(opts *cloudprovider.FileSystemCraeteOptions) (*SfsTurbo, error) {
	secId, err := self.GetSysDefaultSecgroupId()
	if err != nil {
		return nil, errors.Wrapf(err, "GetSysDefaultSecgroupId")
	}
	metadata := map[string]string{}
	if strings.HasSuffix(opts.StorageType, ".enhanced") {
		metadata["expand_type"] = "bandwidth"
	}
	params := map[string]interface{}{
		"share": map[string]interface{}{
			"name":              opts.Name,
			"share_proto":       strings.ToUpper(opts.Protocol),
			"share_type":        strings.ToUpper(strings.TrimSuffix(opts.StorageType, ".enhanced")),
			"size":              opts.Capacity,
			"availability_zone": opts.ZoneId,
			"vpc_id":            opts.VpcId,
			"subnet_id":         opts.NetworkId,
			"security_group_id": secId,
			"description":       opts.Desc,
			"metadata":          metadata,
		},
	}
	resp, err := self.post(SERVICE_SFS, "sfs-turbo/shares", params)
	if err != nil {
		return nil, errors.Wrapf(err, "Create")
	}
	id, err := resp.GetString("id")
	if err != nil {
		return nil, errors.Wrapf(err, "resp.GetString(id)")
	}
	return self.GetSfsTurbo(id)
}

func (self *SRegion) GetICloudAccessGroups() ([]cloudprovider.ICloudAccessGroup, error) {
	return []cloudprovider.ICloudAccessGroup{}, nil
}

func (self *SRegion) CreateICloudAccessGroup(opts *cloudprovider.SAccessGroup) (cloudprovider.ICloudAccessGroup, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "CreateICloudAccessGroup")
}

func (self *SRegion) GetICloudAccessGroupById(id string) (cloudprovider.ICloudAccessGroup, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "GetICloudAccessGroupById(%s)", id)
}

type sMoutTarget struct {
	sfs *SfsTurbo
}

func (self *sMoutTarget) GetName() string {
	return self.sfs.Name
}

func (self *sMoutTarget) GetGlobalId() string {
	return self.sfs.GetGlobalId()
}

func (self *sMoutTarget) GetAccessGroupId() string {
	return ""
}

func (self *sMoutTarget) GetDomainName() string {
	return self.sfs.ExportLocation
}

func (self *sMoutTarget) GetNetworkType() string {
	return api.NETWORK_TYPE_VPC
}

func (self *sMoutTarget) GetNetworkId() string {
	return self.sfs.SubnetId
}

func (self *sMoutTarget) GetVpcId() string {
	return self.sfs.VpcId
}

func (self *sMoutTarget) GetStatus() string {
	return api.MOUNT_TARGET_STATUS_AVAILABLE
}

func (self *sMoutTarget) Delete() error {
	return nil
}
