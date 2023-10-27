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

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SSNATEntry struct {
	multicloud.SResourceBase
	VolcEngineTags
	nat *SNatGateway

	NatGatewayId  string
	SnatEntryId   string
	SnatEntryName string
	SubnetId      string
	SourceCidr    string
	EipId         string
	EipAddress    string
	Status        string
}

func (sentry *SSNATEntry) GetName() string {
	if len(sentry.SnatEntryName) > 0 {
		return sentry.SnatEntryName
	}
	return sentry.SnatEntryId
}

func (sentry *SSNATEntry) GetId() string {
	return sentry.SnatEntryId
}

func (sentry *SSNATEntry) GetGlobalId() string {
	return sentry.SnatEntryId
}

func (sentry *SSNATEntry) GetStatus() string {
	switch sentry.Status {
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

func (sentry *SSNATEntry) GetIP() string {
	return sentry.EipAddress
}

func (sentry *SSNATEntry) GetSourceCIDR() string {
	return sentry.SourceCidr
}

func (sentry *SSNATEntry) GetNetworkId() string {
	return sentry.SubnetId
}

func (sentry *SSNATEntry) Delete() error {
	return sentry.nat.vpc.region.DeleteSnatEntry(sentry.SnatEntryId)
}

func (sentry *SSNATEntry) Refresh() error {
	new, err := sentry.nat.vpc.region.GetSnatEntry(sentry.NatGatewayId, sentry.SnatEntryId)
	if err != nil {
		return err
	}
	return jsonutils.Update(sentry, new)
}

func (nat *SNatGateway) getSnatEntries() ([]SSNATEntry, error) {
	entries := make([]SSNATEntry, 0)
	pageNumber := 1
	for {
		parts, total, err := nat.vpc.region.GetSnatEntries(nat.NatGatewayId, pageNumber, 50)
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

func (nat *SNatGateway) dissociateWithSubnet(subnetId string) error {
	entries, err := nat.getSnatEntries()
	if err != nil {
		return err
	}
	for i := range entries {
		if entries[i].SubnetId == subnetId {
			err := nat.vpc.region.DeleteSnatEntry(entries[i].SnatEntryId)
			if err != nil {
				return nil
			}
		}
	}
	return nil
}

func (region *SRegion) GetSnatEntries(natGatewayId string, pageNumber int, pageSize int) ([]SSNATEntry, int, error) {
	if pageSize > 100 || pageSize <= 0 {
		pageSize = 100
	}
	params := make(map[string]string)
	params["NatGatewayId"] = natGatewayId
	params["PageSize"] = fmt.Sprintf("%d", pageSize)
	params["PageNumber"] = fmt.Sprintf("%d", pageNumber)

	body, err := region.natRequest("DescribeSnatEntries", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "DescribeSNATEntries fail")
	}
	entries := make([]SSNATEntry, 0)
	err = body.Unmarshal(&entries, "SnatEntries")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "Unmarshal entries fail")
	}
	total, _ := body.Int("TotalCount")
	return entries, int(total), nil
}

func (region *SRegion) GetSnatEntry(natGatewayId string, snatEntryID string) (SSNATEntry, error) {
	params := make(map[string]string)
	params["NatGatewayId"] = natGatewayId
	params["SnatEntryIds.1"] = snatEntryID
	body, err := region.natRequest("DescribeSnatEntries", params)
	if err != nil {
		return SSNATEntry{}, errors.Wrapf(err, "DescribeSnatEntries fail")
	}
	entries := make([]SSNATEntry, 0)
	err = body.Unmarshal(&entries, "SnatEntries")
	if err != nil {
		return SSNATEntry{}, errors.Wrapf(err, "Unmarshal entries fail")
	}
	if len(entries) == 0 {
		return SSNATEntry{}, cloudprovider.ErrNotFound
	}
	return entries[0], nil
}

func (region *SRegion) CreateSnatEntry(rule cloudprovider.SNatSRule, natGatewayId string) (string, error) {
	params := make(map[string]string)
	params["NatGatewayId"] = natGatewayId
	params["SubnetId"] = rule.NetworkID
	eips, _, err := region.GetEips(nil, rule.ExternalIP, nil, 1, 1)
	if err != nil {
		return "", err
	}
	params["EipId"] = eips[0].AllocationId
	if len(rule.SourceCIDR) != 0 {
		params["SourceCidr"] = rule.SourceCIDR
	}
	body, err := region.natRequest("CreateSnatEntry", params)
	if err != nil {
		return "", err
	}

	entryID, _ := body.GetString("SnatEntryId")
	return entryID, nil
}

func (region *SRegion) DeleteSnatEntry(snatEntryId string) error {
	params := make(map[string]string)
	params["SnatEntryId"] = snatEntryId
	_, err := region.natRequest("DeleteSnatEntry", params)
	return errors.Wrapf(err, "DeleteSnatEntry")
}
