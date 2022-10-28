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
	skus := []SResourceSku{}
	resource := "Microsoft.Compute/skus"
	return skus, self.list(resource, nil, &skus)
}
