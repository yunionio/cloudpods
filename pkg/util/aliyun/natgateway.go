package aliyun

import (
	"fmt"

	"time"
	"yunion.io/x/log"
)

type SBandwidthPackageIds struct {
	BandwidthPackageId []string
}

type SForwardTableIds struct {
	ForwardTableId []string
}

type SSnatTableIds struct {
	SnatTableId []string
}

type SNatGetway struct {
	vpc *SVpc

	BandwidthPackageIds SBandwidthPackageIds
	BusinessStatus      string
	CreationTime        time.Time
	Description         string
	ForwardTableIds     SForwardTableIds
	SnatTableIds        SSnatTableIds
	InstanceChargeType  string
	Name                string
	NatGatewayId        string
	RegionId            string
	Spec                string
	Status              string
	VpcId               string
}

func (self *SRegion) GetNatGateways(vpcId string, natGwId string, offset, limit int) ([]SNatGetway, int, error) {
	if limit > 50 || limit <= 0 {
		limit = 50
	}
	params := make(map[string]string)
	params["RegionId"] = self.RegionId
	params["PageSize"] = fmt.Sprintf("%d", limit)
	params["PageNumber"] = fmt.Sprintf("%d", (offset/limit)+1)
	if len(vpcId) > 0 {
		params["VpcId"] = vpcId
	}
	if len(natGwId) > 0 {
		params["NatGatewayId"] = natGwId
	}

	body, err := self.vpcRequest("DescribeNatGateways", params)
	if err != nil {
		log.Errorf("GetVSwitches fail %s", err)
		return nil, 0, err
	}

	if self.client.Debug {
		log.Debugf("%s", body.PrettyString())
	}

	gateways := make([]SNatGetway, 0)
	err = body.Unmarshal(&gateways, "NatGateways", "NatGateway")
	if err != nil {
		log.Errorf("Unmarshal gateways fail %s", err)
		return nil, 0, err
	}
	total, _ := body.Int("TotalCount")
	return gateways, int(total), nil
}

type SSNATTableEntry struct {
	SnatEntryId     string
	SnatIp          string
	SnatTableId     string `json:"snat_table_id"`
	SourceCIDR      string `json:"source_cidr"`
	SourceVSwitchId string `json:"source_vswitch_id"`
	Status          string
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
		log.Debugf("%s", entries[i])
		if entries[i].SourceVSwitchId == vswitchId {
			err := nat.vpc.region.DeleteSnatEntry(entries[i].SnatTableId, entries[i].SnatEntryId)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
