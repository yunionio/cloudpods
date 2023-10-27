// Copyright 2023 Yunion
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

package volcengine

import (
	"fmt"
	"strconv"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

type SDNATEntry struct {
	multicloud.SResourceBase
	VolcEngineTags
	nat *SNatGateway

	NatGatewayId  string
	DnatEntryId   string
	DnatEntryName string
	Protocol      string
	InternalIp    string
	InternalPort  string
	ExternalIp    string
	ExternalPort  string
	Status        string
}

func (dentry *SDNATEntry) GetName() string {
	if len(dentry.DnatEntryName) > 0 {
		return dentry.DnatEntryName
	}
	return dentry.DnatEntryId
}

func (dentry *SDNATEntry) GetId() string {
	return dentry.DnatEntryId
}

func (dentry *SDNATEntry) GetGlobalId() string {
	return dentry.DnatEntryId
}

func (dentry *SDNATEntry) GetStatus() string {
	switch dentry.Status {
	case "Creating":
		return api.NAT_STATUS_ALLOCATE
	case "Available":
		return api.NAT_STAUTS_AVAILABLE
	case "Deleteting":
		return api.NAT_STATUS_DELETING
	default:
		return api.NAT_STATUS_UNKNOWN
	}
}

func (dentry *SDNATEntry) GetIpProtocol() string {
	return dentry.Protocol
}

func (dentry *SDNATEntry) GetExternalIp() string {
	return dentry.ExternalIp
}

func (dentry *SDNATEntry) GetExternalPort() int {
	port, _ := strconv.Atoi(dentry.ExternalPort)
	return port
}

func (dentry *SDNATEntry) GetInternalIp() string {
	return dentry.InternalIp
}

func (dentry *SDNATEntry) GetInternalPort() int {
	port, _ := strconv.Atoi(dentry.InternalPort)
	return port
}

func (dentry *SDNATEntry) Delete() error {
	return dentry.nat.vpc.region.DeleteDnatEntry(dentry.DnatEntryId)
}

func (nat *SNatGateway) getDnatEntries() ([]SDNATEntry, error) {
	entries := make([]SDNATEntry, 0)
	pageNumber := 1
	for {
		parts, total, err := nat.vpc.region.GetDnatEntries(nat.NatGatewayId, pageNumber, 50)
		if err != nil {
			return nil, err
		}
		entries = append(entries, parts...)
		if len(entries) >= total {
			break
		}
		pageNumber += 1
	}
	return entries, nil
}

func (region *SRegion) GetDnatEntries(natGatewayId string, pageNumber int, pageSize int) ([]SDNATEntry, int, error) {
	if pageSize > 100 || pageSize <= 0 {
		pageSize = 100
	}
	params := make(map[string]string)
	params["NatGatewayId"] = natGatewayId
	params["PageSize"] = fmt.Sprintf("%d", pageSize)
	params["PageNumber"] = fmt.Sprintf("%d", pageNumber)

	body, err := region.natRequest("DescribeDnatEntries", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeDnatEntries fail")
	}
	entries := make([]SDNATEntry, 0)
	err = body.Unmarshal(&entries, "DnatEntries")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "Unmarshal entries fail")
	}
	total, _ := body.Int("TotalCount")
	return entries, int(total), nil
}

func (region *SRegion) GetDnatEntry(natGatewayId string, dnatEntryID string) (SDNATEntry, error) {
	params := make(map[string]string)
	params["NatGatewayId"] = natGatewayId
	params["DnatEntryIds.1"] = dnatEntryID
	body, err := region.natRequest("DescribeDnatEntries", params)
	if err != nil {
		return SDNATEntry{}, errors.Wrapf(err, "DescribeDnatEntries fail")
	}
	entries := make([]SDNATEntry, 0)
	err = body.Unmarshal(&entries, "DnatEntries")
	if err != nil {
		return SDNATEntry{}, errors.Wrapf(err, "Unmarshal entries fail")
	}
	if len(entries) == 0 {
		return SDNATEntry{}, cloudprovider.ErrNotFound
	}
	return entries[0], nil
}

func (region *SRegion) CreateDnatEntry(rule cloudprovider.SNatDRule, natGatewayId string) (string, error) {
	params := make(map[string]string)
	params["NatGatewayId"] = natGatewayId
	params["ExternalIp"] = rule.ExternalIP
	params["ExternalPort"] = strconv.Itoa(rule.ExternalPort)
	params["InternalIp"] = rule.InternalIP
	params["InternalPort"] = strconv.Itoa(rule.InternalPort)
	params["Protocol"] = rule.Protocol
	body, err := region.natRequest("CreateDnatEntry", params)
	if err != nil {
		return "", err
	}
	entryID, _ := body.GetString("DnatEntryId")
	return entryID, nil
}

func (region *SRegion) DeleteDnatEntry(dnatEntryId string) error {
	params := make(map[string]string)
	params["DnatEntryId"] = dnatEntryId
	_, err := region.natRequest("DeleteDnatEntry", params)
	return err
}
