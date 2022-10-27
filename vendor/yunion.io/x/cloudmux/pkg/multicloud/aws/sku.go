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

package aws

import (
	"github.com/aws/aws-sdk-go/service/pricing"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type Product struct {
	ProductFamily string     `json:"productFamily"`
	Attributes    Attributes `json:"attributes"`
	Sku           string     `json:"sku"`
}

type Attributes struct {
	EnhancedNetworkingSupported string `json:"enhancedNetworkingSupported"`
	IntelTurboAvailable         string `json:"intelTurboAvailable"`
	Memory                      string `json:"memory"`
	DedicatedEbsThroughput      string `json:"dedicatedEbsThroughput"`
	Vcpu                        int    `json:"vcpu"`
	Gpu                         int    `json:"gpu"`
	Capacitystatus              string `json:"capacitystatus"`
	LocationType                string `json:"locationType"`
	Storage                     string `json:"storage"`
	InstanceFamily              string `json:"instanceFamily"`
	OperatingSystem             string `json:"operatingSystem"`
	IntelAvx2Available          string `json:"intelAvx2Available"`
	PhysicalProcessor           string `json:"physicalProcessor"`
	ClockSpeed                  string `json:"clockSpeed"`
	Ecu                         string `json:"ecu"`
	NetworkPerformance          string `json:"networkPerformance"`
	Servicename                 string `json:"servicename"`
	InstanceType                string `json:"instanceType"`
	InstanceSku                 string `json:"instancesku"`
	Tenancy                     string `json:"tenancy"`
	Usagetype                   string `json:"usagetype"`
	NormalizationSizeFactor     string `json:"normalizationSizeFactor"`
	IntelAvxAvailable           string `json:"intelAvxAvailable"`
	ProcessorFeatures           string `json:"processorFeatures"`
	Servicecode                 string `json:"servicecode"`
	LicenseModel                string `json:"licenseModel"`
	CurrentGeneration           string `json:"currentGeneration"`
	PreInstalledSw              string `json:"preInstalledSw"`
	Location                    string `json:"location"`
	ProcessorArchitecture       string `json:"processorArchitecture"`
	Operation                   string `json:"operation"`
	VolumeApiName               string `json:"volumeApiName"`
}

type Terms struct {
	OnDemand map[string]Term `json:"OnDemand"`
	Reserved map[string]Term `json:"Reserved"`
}

type Term struct {
	PriceDimensions map[string]Dimension `json:"priceDimensions"`
	Sku             string               `json:"sku"`
	EffectiveDate   string               `json:"effectiveDate"`
	OfferTermCode   string               `json:"offerTermCode"`
	TermAttributes  TermAttributes       `json:"termAttributes"`
}

type Dimension struct {
	Unit         string       `json:"unit"`
	EndRange     string       `json:"endRange"`
	Description  string       `json:"description"`
	AppliesTo    []string     `json:"appliesTo"`
	RateCode     string       `json:"rateCode"`
	BeginRange   string       `json:"beginRange"`
	PricePerUnit PricePerUnit `json:"pricePerUnit"`
}

type TermAttributes struct {
	LeaseContractLength string `json:"LeaseContractLength"`
	OfferingClass       string `json:"OfferingClass"`
	PurchaseOption      string `json:"PurchaseOption"`
}

type PricePerUnit struct {
	Usd float64 `json:"USD"`
	CNY float64 `json:"CNY"`
}

type SInstnaceType struct {
	Product         Product `json:"product"`
	ServiceCode     string  `json:"serviceCode"`
	Terms           Terms   `json:"terms"`
	Version         string  `json:"version"`
	PublicationDate string  `json:"publicationDate"`
}

func (self *SRegion) GetInstanceTypes(nextToken string) ([]SInstnaceType, string, error) {
	input := &pricing.GetProductsInput{}
	input.SetServiceCode("AmazonEC2")
	if len(nextToken) > 0 {
		input.SetNextToken(nextToken)
	}
	GetFilter := func(k, v string) pricing.Filter {
		filter := pricing.Filter{}
		filter.SetType("TERM_MATCH")
		filter.SetField(k)
		filter.SetValue(v)
		return filter
	}
	f1 := GetFilter("regionCode", self.RegionId)
	f2 := GetFilter("operatingSystem", "Linux")
	f3 := GetFilter("licenseModel", "No License required")
	f4 := GetFilter("productFamily", "Compute Instance")
	f5 := GetFilter("operation", "RunInstances")
	f6 := GetFilter("preInstalledSw", "NA")
	f7 := GetFilter("tenancy", "Shared")
	f8 := GetFilter("capacitystatus", "Used")
	input.SetFilters([]*pricing.Filter{&f1, &f2, &f3, &f4, &f5, &f6, &f7, &f8})
	s, err := self.client.getAwsSession("ap-south-1", false)
	if err != nil {
		return nil, "", err
	}

	cli := pricing.New(s)
	output, err := cli.GetProducts(input)
	if err != nil {
		return nil, "", errors.Wrap(err, "SZone.GetServerSku.GetProducts")
	}
	if output.NextToken != nil {
		nextToken = *output.NextToken
	}
	ret := []SInstnaceType{}
	return ret, nextToken, jsonutils.Update(&ret, output.PriceList)
}
