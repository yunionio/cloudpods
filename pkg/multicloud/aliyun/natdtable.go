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
	"strconv"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SForwardTableEntry struct {
	multicloud.SResourceBase
	nat *SNatGetway

	ForwardEntryId   string
	ForwardEntryName string
	IpProtocol       string
	Status           string
	ExternalIp       string
	ForwardTableId   string
	ExternalPort     string
	InternalPort     string
	InternalIp       string
}

func (dtable *SForwardTableEntry) GetName() string {
	if len(dtable.ForwardEntryName) > 0 {
		return dtable.ForwardEntryName
	}
	return dtable.ForwardEntryId
}

func (dtable *SForwardTableEntry) GetId() string {
	return dtable.ForwardEntryId
}

func (dtable *SForwardTableEntry) GetGlobalId() string {
	return dtable.ForwardEntryId
}

func (dtable *SForwardTableEntry) GetStatus() string {
	switch dtable.Status {
	case "Available":
		return api.NAT_STAUTS_AVAILABLE
	default:
		return api.NAT_STATUS_UNKNOWN
	}
}

func (dtable *SForwardTableEntry) GetIpProtocol() string {
	return dtable.IpProtocol
}

func (dtable *SForwardTableEntry) GetExternalIp() string {
	return dtable.ExternalIp
}

func (dtable *SForwardTableEntry) GetExternalPort() int {
	port, _ := strconv.Atoi(dtable.ExternalPort)
	return port
}

func (dtable *SForwardTableEntry) GetInternalIp() string {
	return dtable.InternalIp
}

func (dtable *SForwardTableEntry) GetInternalPort() int {
	port, _ := strconv.Atoi(dtable.InternalPort)
	return port
}

func (region *SRegion) GetAllDTables(tableId string) ([]SForwardTableEntry, error) {
	dtables := []SForwardTableEntry{}
	for {
		part, total, err := region.GetForwardTableEntries(tableId, len(dtables), 50)
		if err != nil {
			return nil, err
		}
		dtables = append(dtables, part...)
		if len(dtables) >= total {
			break
		}
	}
	return dtables, nil
}

func (dtable *SForwardTableEntry) Delete() error {
	return dtable.nat.vpc.region.DeleteForwardTableEntry(dtable.ForwardTableId, dtable.GetId())
}

func (region *SRegion) GetForwardTableEntries(tableId string, offset int, limit int) ([]SForwardTableEntry, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["RegionId"] = region.RegionId
	params["PageSize"] = fmt.Sprintf("%d", limit)
	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)
	params["ForwardTableId"] = tableId

	body, err := region.vpcRequest("DescribeForwardTableEntries", params)
	if err != nil {
		return nil, 0, err
	}

	dtables := []SForwardTableEntry{}
	err = body.Unmarshal(&dtables, "ForwardTableEntries", "ForwardTableEntry")
	if err != nil {
		return nil, 0, err
	}
	total, _ := body.Int("TotalCount")
	return dtables, int(total), nil
}

func (region *SRegion) GetForwardTableEntry(tableID, forwardEntryID string) (SForwardTableEntry, error) {
	params := make(map[string]string)
	params["RegionId"] = region.RegionId
	params["ForwardTableId"] = tableID
	params["ForwardEntryId"] = forwardEntryID
	body, err := region.vpcRequest("DescribeForwardTableEntries", params)
	if err != nil {
		return SForwardTableEntry{}, err
	}

	dtables := []SForwardTableEntry{}
	err = body.Unmarshal(&dtables, "ForwardTableEntries", "ForwardTableEntry")
	if err != nil {
		return SForwardTableEntry{}, err
	}
	return dtables[0], nil
}

func (region *SRegion) DeleteForwardTableEntry(tableId string, entryId string) error {
	params := make(map[string]string)
	params["RegionId"] = region.RegionId
	params["ForwardTableId"] = tableId
	params["ForwardEntryId"] = entryId
	_, err := region.vpcRequest("DeleteForwardEntry", params)
	return err
}

func (region *SRegion) CreateForwardTableEntry(rule cloudprovider.SNatDRule, tableID string) (string, error) {
	params := make(map[string]string)
	params["RegionId"] = region.RegionId
	params["ForwardTableId"] = tableID
	params["ExternalIp"] = rule.ExternalIP
	params["ExternalPort"] = strconv.Itoa(rule.ExternalPort)
	params["InternalIp"] = rule.InternalIP
	params["InternalPort"] = strconv.Itoa(rule.InternalPort)
	params["IpProtocol"] = rule.Protocol
	body, err := region.vpcRequest("CreateForwardEntry", params)
	if err != nil {
		return "", err
	}

	entryID, _ := body.GetString("ForwardEntryId")
	return entryID, nil
}

func (dtable *SForwardTableEntry) Refresh() error {
	new, err := dtable.nat.vpc.region.GetForwardTableEntry(dtable.ForwardEntryId, dtable.ForwardTableId)
	if err != nil {
		return err
	}
	return jsonutils.Update(dtable, new)
}
