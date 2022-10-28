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

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type ClientMasterNode struct {
	EcsIp         string
	EcsId         string
	DefaultPasswd string
}

type ClientMasterNodes struct {
	ClientMasterNode []ClientMasterNode
}

type SMountTarget struct {
	fs *SFileSystem

	Status            string
	NetworkType       string
	VswId             string
	VpcId             string
	MountTargetDomain string
	AccessGroup       string
	ClientMasterNodes ClientMasterNodes
}

func (self *SMountTarget) GetGlobalId() string {
	return self.MountTargetDomain
}

func (self *SMountTarget) GetName() string {
	return self.MountTargetDomain
}

func (self *SMountTarget) GetNetworkType() string {
	return strings.ToLower(self.NetworkType)
}

func (self *SMountTarget) GetStatus() string {
	switch self.Status {
	case "Active":
		return api.MOUNT_TARGET_STATUS_AVAILABLE
	case "Inactive":
		return api.MOUNT_TARGET_STATUS_UNAVAILABLE
	case "Pending":
		return api.MOUNT_TARGET_STATUS_CREATING
	case "Deleting":
		return api.MOUNT_TARGET_STATUS_DELETING
	default:
		return strings.ToLower(self.Status)
	}
}

func (self *SMountTarget) GetVpcId() string {
	return self.VpcId
}

func (self *SMountTarget) GetNetworkId() string {
	return self.VswId
}

func (self *SMountTarget) GetDomainName() string {
	return self.MountTargetDomain
}

func (self *SMountTarget) GetAccessGroupId() string {
	return fmt.Sprintf("%s/%s", self.fs.FileSystemType, self.AccessGroup)
}

func (self *SMountTarget) Delete() error {
	return self.fs.region.DeleteMountTarget(self.fs.FileSystemId, self.MountTargetDomain)
}

func (self *SRegion) GetMountTargets(fsId, domainName string, pageSize, pageNum int) ([]SMountTarget, int, error) {
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	if pageNum < 1 {
		pageNum = 1
	}
	params := map[string]string{
		"RegionId":     self.RegionId,
		"FileSystemId": fsId,
		"PageSize":     fmt.Sprintf("%d", pageSize),
		"PageNumber":   fmt.Sprintf("%d", pageNum),
	}
	if len(domainName) > 0 {
		params["MountTargetDomain"] = domainName
	}
	resp, err := self.nasRequest("DescribeMountTargets", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeMountTargets")
	}
	ret := struct {
		TotalCount   int
		MountTargets struct {
			MountTarget []SMountTarget
		}
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	return ret.MountTargets.MountTarget, ret.TotalCount, nil
}

func (self *SFileSystem) GetMountTargets() ([]cloudprovider.ICloudMountTarget, error) {
	targets := []SMountTarget{}
	num := 1
	for {
		part, total, err := self.region.GetMountTargets(self.FileSystemId, "", 50, num)
		if err != nil {
			return nil, errors.Wrapf(err, "GetMountTargets")
		}
		targets = append(targets, part...)
		if len(part) >= total {
			break
		}
	}
	ret := []cloudprovider.ICloudMountTarget{}
	for i := range targets {
		targets[i].fs = self
		ret = append(ret, &targets[i])
	}
	return ret, nil
}

func (self *SRegion) DeleteMountTarget(fsId, id string) error {
	params := map[string]string{
		"RegionId":          self.RegionId,
		"FileSystemId":      fsId,
		"MountTargetDomain": id,
	}
	_, err := self.nasRequest("DeleteMountTarget", params)
	return errors.Wrapf(err, "DeleteMountTarget")
}
