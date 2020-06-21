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

package azure

import (
	"fmt"

	"yunion.io/x/pkg/utils"
)

/*
{
	"capabilities":[
		{"name":"MaxResourceVolumeMB","value":"286720"},
		{"name":"OSVhdSizeMB","value":"1047552"},
		{"name":"vCPUs","value":"20"},
		{"name":"MemoryGB","value":"140"},
		{"name":"MaxDataDiskCount","value":"64"},
		{"name":"LowPriorityCapable","value":"True"},
		{"name":"PremiumIO","value":"True"},
		{"name":"EphemeralOSDiskSupported","value":"True"}
	],
	"family":"standardDSv2Family",
	"locations":["CentralUSEUAP"],
	"name":"Standard_DS15_v2",
	"resourceType":"virtualMachines",
	"restrictions":[],
	"size":"DS15_v2",
	"tier":"Standard"
}
*/

type SResourceSkuCapability struct {
	Name  string
	Value string
}

type TResourceSkuCapacityScaleType string

const (
	ResourceSkuCapacityScaleTypeAutomatic = TResourceSkuCapacityScaleType("Automatic")
	ResourceSkuCapacityScaleTypeManual    = TResourceSkuCapacityScaleType("Manual")
	ResourceSkuCapacityScaleTypeNone      = TResourceSkuCapacityScaleType("None")
)

type SResourceSkuCapacity struct {
	Default   int
	Maximum   int
	Minimum   int
	ScaleType TResourceSkuCapacityScaleType
}

type SResourceSkuLocationInfo struct {
	Location string
	Zones    []string
}

type TResourceSkuRestrictionsType string

const (
	ResourceSkuRestrictionsTypeLocation = TResourceSkuRestrictionsType("Location")
	ResourceSkuRestrictionsTypeZone     = TResourceSkuRestrictionsType("Zone")
)

type TResourceSkuRestrictionsReasonCode string

const (
	ResourceSkuRestrictionsReasonCodeNotAvailable = TResourceSkuRestrictionsReasonCode("NotAvailableForSubscription")
	ResourceSkuRestrictionsReasonCodeQuotaId      = TResourceSkuRestrictionsReasonCode("QuotaId")
)

type SResourceSkuRestrictionInfo struct {
	Locations []string
	Zones     []string
}

type SResourceSkuRestrictions struct {
	ReasonCode      TResourceSkuRestrictionsReasonCode
	RestrictionInfo SResourceSkuRestrictionInfo
	Type            TResourceSkuRestrictionsType
	Values          []string
}

type SResourceSku struct {
	Capabilities []SResourceSkuCapability
	Capacity     *SResourceSkuCapacity
	Family       string
	Kind         string
	LocationInfo []SResourceSkuLocationInfo
	Locations    []string
	Name         string
	ResourceType string
	Restrictions []SResourceSkuRestrictions
	Size         string
	Tier         string
}

type SResourceSkusResult struct {
	NextLink string
	Value    []SResourceSku
}

func (self *SAzureClient) ListResourceSkus() ([]SResourceSku, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	if len(self.subscriptionId) == 0 {
		return nil, fmt.Errorf("need subscription id")
	}
	url := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Compute/skus?api-version=2017-09-01", self.subscriptionId)
	skus := make([]SResourceSku, 0)
	for {
		body, err := jsonRequest(cli, "GET", self.domain, url, self.subscriptionId, "", DefaultResource)
		if err != nil {
			return nil, err
		}
		result := SResourceSkusResult{}
		err = body.Unmarshal(&result)
		if err != nil {
			return nil, err
		}
		skus = append(skus, result.Value...)
		if len(result.NextLink) > 0 {
			url = result.NextLink
		} else {
			break
		}
	}
	return skus, nil
}

func (self *SRegion) GetResourceSkus(location string) ([]SResourceSku, error) {
	skus, err := self.client.ListResourceSkus()
	if err != nil {
		return nil, err
	}
	if len(location) == 0 {
		return skus, nil
	}
	ret := make([]SResourceSku, 0)
	for i := 0; i < len(skus); i += 1 {
		if utils.IsInStringArray(location, skus[i].Locations) {
			ret = append(ret, skus[i])
		}
	}
	return ret, nil
}
