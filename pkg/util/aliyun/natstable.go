package aliyun

import (
	"fmt"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/multicloud"

	"yunion.io/x/log"
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
