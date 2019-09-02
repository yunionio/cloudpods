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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SSNATTableEntry struct {
	multicloud.SResourceBase
	nat *SNatGetway

	SnatEntryId     string
	SnatEntryName   string
	SnatIp          string
	SnatTableId     string `json:"snat_table_id"`
	SourceCIDR      string `json:"source_cidr"`
	SourceVSwitchId string `json:"source_vswitch_id"`
	Status          string
}

func (stable *SSNATTableEntry) GetName() string {
	if len(stable.SnatEntryName) > 0 {
		return stable.SnatEntryName
	}
	return stable.SnatEntryId
}

func (stable *SSNATTableEntry) GetId() string {
	return stable.SnatEntryId
}

func (stable *SSNATTableEntry) GetGlobalId() string {
	return stable.SnatEntryId
}

func (stable *SSNATTableEntry) GetStatus() string {
	switch stable.Status {
	case "Available":
		return api.NAT_STAUTS_AVAILABLE
	default:
		return api.NAT_STATUS_UNKNOWN
	}
}

func (stable *SSNATTableEntry) GetIP() string {
	return stable.SnatIp
}

func (stable *SSNATTableEntry) GetSourceCIDR() string {
	return stable.SourceCIDR
}

func (stable *SSNATTableEntry) GetNetworkId() string {
	return stable.SourceVSwitchId
}

func (stable *SSNATTableEntry) Delete() error {
	return stable.nat.vpc.region.DeleteSnatEntry(stable.SnatTableId, stable.GetId())
}

func (self *SRegion) GetSNATEntries(tableId string, offset, limit int) ([]SSNATTableEntry, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["PageSize"] = fmt.Sprintf("%d", limit)
	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)
	params["SnatTableId"] = tableId

	body, err := self.vpcRequest("DescribeSnatTableEntries", params)
	if err != nil {
		log.Errorf("DescribeSnatTableEntries fail %s", err)
		return nil, 0, err
	}

	if self.client.Debug {
		log.Debugf("%s", body.PrettyString())
	}

	entries := make([]SSNATTableEntry, 0)
	err = body.Unmarshal(&entries, "SnatTableEntries", "SnatTableEntry")
	if err != nil {
		log.Errorf("Unmarshal entries fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Int("TotalCount")
	return entries, int(total), nil
}

func (self *SRegion) GetSNATEntry(tableID, SNATEntryID string) (SSNATTableEntry, error) {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["SnatTableId"] = tableID
	params["SnatEntryId"] = SNATEntryID

	body, err := self.vpcRequest("DescribeSnatTableEntries", params)
	if err != nil {
		log.Errorf("DescribeSnatTableEntries fail %s", err)
		return SSNATTableEntry{}, err
	}

	if self.client.Debug {
		log.Debugf("%s", body.PrettyString())
	}

	entries := make([]SSNATTableEntry, 0)
	err = body.Unmarshal(&entries, "SnatTableEntries", "SnatTableEntry")
	if err != nil {
		log.Errorf("Unmarshal entries fail %s", err)
		return SSNATTableEntry{}, err
	}
	if len(entries) == 0 {
		return SSNATTableEntry{}, cloudprovider.ErrNotFound
	}
	return entries[0], nil

}

func (region *SRegion) CreateSNATTableEntry(rule cloudprovider.SNatSRule, tableID string) (string, error) {
	params := make(map[string]string)
	params["RegionId"] = region.RegionId
	params["SnatTableId"] = tableID
	params["SnatIp"] = rule.ExternalIP
	if len(rule.NetworkID) != 0 {
		params["SourceVSwitchId"] = rule.NetworkID
	}
	if len(rule.SourceCIDR) != 0 {
		params["SourceCIDR"] = rule.SourceCIDR
	}
	body, err := region.vpcRequest("CreateSnatEntry", params)
	if err != nil {
		return "", err
	}

	entryID, _ := body.GetString("SnatEntryId")
	return entryID, nil
}

func (region *SRegion) DeleteSnatEntry(tableId string, entryId string) error {
	params := make(map[string]string)
	params["RegionId"] = region.RegionId
	params["SnatTableId"] = tableId
	params["SnatEntryId"] = entryId
	_, err := region.vpcRequest("DeleteSnatEntry", params)
	return err
}

func (nat *SNatGetway) getSnatEntriesForTable(tblId string) ([]SSNATTableEntry, error) {
	entries := make([]SSNATTableEntry, 0)
	entryTotal := -1
	for entryTotal < 0 || len(entries) < entryTotal {
		parts, total, err := nat.vpc.region.GetSNATEntries(tblId, len(entries), 50)
		if err != nil {
			return nil, err
		}
		if len(parts) > 0 {
			entries = append(entries, parts...)
		}
		entryTotal = total
	}
	return entries, nil
}

func (nat *SNatGetway) getSnatEntries() ([]SSNATTableEntry, error) {
	entries := make([]SSNATTableEntry, 0)
	for i := range nat.SnatTableIds.SnatTableId {
		sentries, err := nat.getSnatEntriesForTable(nat.SnatTableIds.SnatTableId[i])
		if err != nil {
			return nil, err
		}
		entries = append(entries, sentries...)
	}
	return entries, nil
}

func (nat *SNatGetway) dissociateWithVswitch(vswitchId string) error {
	entries, err := nat.getSnatEntries()
	if err != nil {
		return err
	}
	for i := range entries {
		log.Debugf("%v", entries[i])
		if entries[i].SourceVSwitchId == vswitchId {
			err := nat.vpc.region.DeleteSnatEntry(entries[i].SnatTableId, entries[i].SnatEntryId)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (stable *SSNATTableEntry) Refresh() error {
	new, err := stable.nat.vpc.region.GetSNATEntry(stable.SnatTableId, stable.SnatEntryId)
	if err != nil {
		return err
	}
	return jsonutils.Update(stable, new)
}
