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

	api "yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SPrefixList struct {
	multicloud.SVirtualResourceBase
	AliyunTags
	region *SRegion

	PrefixListId          string
	PrefixListName        string
	PrefixListDescription string
	IpVersion             string
	CreationTime          time.Time
	CidrBlocks            []string
	MaxEntries            int
	PrefixListStatus      string
	Status                string
	RegionId              string
	ResourceGroupId       string
	ShareType             string
	PrefixListType        string
}

func (self *SPrefixList) GetId() string {
	return self.PrefixListId
}

func (self *SPrefixList) GetName() string {
	if len(self.PrefixListName) > 0 {
		return self.PrefixListName
	}
	return self.PrefixListId
}

func (self *SPrefixList) GetGlobalId() string {
	return self.PrefixListId
}

func (self *SPrefixList) GetStatus() string {
	status := self.PrefixListStatus
	if len(status) == 0 {
		status = self.Status
	}
	switch strings.ToLower(status) {
	case "created":
		return api.STATUS_AVAILABLE
	case "modifying":
		return "sync_status"
	default:
		return strings.ToLower(status)
	}
}

func (self *SPrefixList) GetCreatedAt() time.Time {
	return self.CreationTime
}

func (self *SPrefixList) GetDescription() string {
	return self.PrefixListDescription
}

func (self *SPrefixList) Refresh() error {
	pl, err := self.region.GetPrefixList(self.PrefixListId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, pl)
}

func (self *SPrefixList) GetIpSetType() string {
	if strings.Contains(strings.ToUpper(self.IpVersion), "IPV6") {
		return cloudprovider.IpSetTypeIpv6CidrList
	}
	return cloudprovider.IpSetTypeIpv4CidrList
}

func (self *SPrefixList) GetAddresses() []string {
	return self.CidrBlocks
}

func (self *SPrefixList) Update(opts *cloudprovider.IpSetUpdateOptions) error {
	return self.region.ModifyPrefixList(self.PrefixListId, opts)
}

func (self *SPrefixList) Delete() error {
	return self.region.DeletePrefixList(self.PrefixListId)
}

func ipSetTypeToAliyunIpVersion(ipSetType string) string {
	if ipSetType == cloudprovider.IpSetTypeIpv6CidrList {
		return "IPv6"
	}
	return "IPv4"
}

func (self *SRegion) listPrefixLists(id, nextToken string, maxResults int) ([]SPrefixList, string, error) {
	params := map[string]string{
		"RegionId": self.RegionId,
	}
	if len(id) > 0 {
		params["PrefixListIds.1"] = id
	}
	if len(nextToken) > 0 {
		params["NextToken"] = nextToken
	}
	if maxResults <= 0 {
		maxResults = 50
	}
	params["MaxResults"] = fmt.Sprintf("%d", maxResults)

	body, err := self.vpcRequest("ListPrefixLists", params)
	if err != nil {
		return nil, "", errors.Wrapf(err, "ListPrefixLists")
	}
	ret := struct {
		PrefixLists []SPrefixList
		NextToken   string
	}{}
	err = body.Unmarshal(&ret)
	if err != nil {
		return nil, "", errors.Wrapf(err, "Unmarshal")
	}
	return ret.PrefixLists, ret.NextToken, nil
}

func (self *SRegion) GetPrefixList(id string) (*SPrefixList, error) {
	lists, _, err := self.listPrefixLists(id, "", 1)
	if err != nil {
		return nil, err
	}
	for i := range lists {
		if lists[i].PrefixListId == id {
			lists[i].region = self
			return &lists[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "%s", id)
}

func (self *SRegion) GetPrefixLists() ([]SPrefixList, error) {
	ret := []SPrefixList{}
	nextToken := ""
	for {
		part, token, err := self.listPrefixLists("", nextToken, 100)
		if err != nil {
			return nil, err
		}
		for i := range part {
			part[i].region = self
			ret = append(ret, part[i])
		}
		if len(token) == 0 || len(part) == 0 {
			break
		}
		nextToken = token
	}
	return ret, nil
}

func (self *SRegion) GetIIpSets() ([]cloudprovider.ICloudIpSet, error) {
	lists, err := self.GetPrefixLists()
	if err != nil {
		return nil, err
	}
	ret := make([]cloudprovider.ICloudIpSet, 0, len(lists))
	for i := range lists {
		ret = append(ret, &lists[i])
	}
	return ret, nil
}

func (self *SRegion) GetIIpSetById(id string) (cloudprovider.ICloudIpSet, error) {
	return self.GetPrefixList(id)
}

func (self *SRegion) CreatePrefixList(opts *cloudprovider.IpSetCreateOptions) (*SPrefixList, error) {
	maxEntries := len(opts.Addresses)
	if maxEntries < 50 {
		maxEntries = 50
	}
	params := map[string]string{
		"RegionId":              self.RegionId,
		"PrefixListName":        opts.Name,
		"PrefixListDescription": opts.Desc,
		"IpVersion":             ipSetTypeToAliyunIpVersion(opts.IpSetType),
		"MaxEntries":            fmt.Sprintf("%d", maxEntries),
	}
	for i := range opts.Addresses {
		params[fmt.Sprintf("PrefixListEntries.%d.Cidr", i+1)] = opts.Addresses[i]
	}
	body, err := self.vpcRequest("CreateVpcPrefixList", params)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateVpcPrefixList")
	}
	id, _ := body.GetString("PrefixListId")
	if len(id) == 0 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "empty PrefixListId after create")
	}
	return self.GetPrefixList(id)
}

func (self *SRegion) CreateIIpSet(opts *cloudprovider.IpSetCreateOptions) (cloudprovider.ICloudIpSet, error) {
	return self.CreatePrefixList(opts)
}

func diffPrefixListAddresses(oldAddrs, newAddrs []string) (added, removed []string) {
	for _, addr := range newAddrs {
		if !utils.IsInStringArray(addr, oldAddrs) {
			added = append(added, addr)
		}
	}
	for _, addr := range oldAddrs {
		if !utils.IsInStringArray(addr, newAddrs) {
			removed = append(removed, addr)
		}
	}
	return added, removed
}

func (self *SRegion) ModifyPrefixList(id string, opts *cloudprovider.IpSetUpdateOptions) error {
	params := map[string]string{
		"RegionId":     self.RegionId,
		"PrefixListId": id,
	}
	if len(opts.Name) > 0 {
		params["PrefixListName"] = opts.Name
	}
	if len(opts.Addresses) > 0 {
		cur, err := self.GetPrefixList(id)
		if err != nil {
			return errors.Wrapf(err, "GetPrefixList")
		}
		added, removed := diffPrefixListAddresses(cur.GetAddresses(), opts.Addresses)
		for i := range added {
			params[fmt.Sprintf("AddPrefixListEntry.%d.Cidr", i+1)] = added[i]
		}
		for i := range removed {
			params[fmt.Sprintf("RemovePrefixListEntry.%d.Cidr", i+1)] = removed[i]
		}
		if len(opts.Addresses) > cur.MaxEntries {
			params["MaxEntries"] = fmt.Sprintf("%d", len(opts.Addresses))
		}
	}
	_, err := self.vpcRequest("ModifyVpcPrefixList", params)
	return errors.Wrapf(err, "ModifyVpcPrefixList")
}

func (self *SRegion) DeletePrefixList(id string) error {
	params := map[string]string{
		"RegionId":     self.RegionId,
		"PrefixListId": id,
	}
	_, err := self.vpcRequest("DeleteVpcPrefixList", params)
	return errors.Wrapf(err, "DeleteVpcPrefixList")
}
