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
	"net/url"

	"yunion.io/x/pkg/errors"
)

type SAppServicePlan struct {
	AzureTags
	region *SRegion

	Properties SAppServicePlanProperties
	ID         string
	Kind       string
	Location   string
	Name       string
	Type       string
	Sku        SSkuDescription
}

type SAppServicePlanProperties struct {
	FreeOfferExpirationTime   string
	GeoRegion                 string
	HostingEnvironmentProfile SHostingEnvironmentProfile
	HyperV                    bool
	IsSpot                    bool
	MaximumElasticWorkerCount int
	MaximumNumberOfWorkers    int
	NumberOfSites             int
	PerSiteScaling            bool
	// 应用服务环境的预配状态
	ProvisioningState  string
	Reserved           bool
	ResourceGroup      string
	SpotExpirationTime string
	// Creating Pending Ready
	Status             string
	Subscription       string
	TargetWorkerCount  int
	TargetWorkerSizeId int
	WorkerTierName     string
}

type SHostingEnvironmentProfile struct {
	ID   string
	Name string
	Type string
}

type SSkuDescription struct {
	Capabilities []SCapability
	Capacity     int
	Family       string
	locations    []string
	Name         string
	Size         string
	SkuCapacity  SSkuCapacity
	Tier         string
}

type SCapability struct {
	Name   string
	Reason string
	Value  string
}

type SSkuCapacity struct {
	Default   int
	Maximum   int
	Minimum   int
	ScaleType string
}

func (r *SRegion) GetAppServicePlanWithCache(id string) (*SAppServicePlan, error) {
	if r.appServicePlans == nil {
		plans, err := r.GetAppServicePlans()
		if err != nil {
			return nil, err
		}
		appServicePlans := make(map[string]*SAppServicePlan, len(plans))
		for i := range plans {
			appServicePlans[plans[i].ID] = &plans[i]
		}
		r.appServicePlans = appServicePlans
	}
	if ret, ok := r.appServicePlans[id]; ok {
		return ret, nil
	}
	return r.GetAppServicePlan(id)
}

func (r *SRegion) GetAppServicePlans() ([]SAppServicePlan, error) {
	result := []SAppServicePlan{}
	resource := "Microsoft.Web/serverfarms"
	err := r.list(resource, url.Values{"api-version": []string{"2019-08-01"}}, &result)
	if err != nil {
		return nil, err
	}
	for i := range result {
		result[i].region = r
	}
	return result, nil
}

func (r *SRegion) GetAppServicePlan(planId string) (*SAppServicePlan, error) {
	asp := &SAppServicePlan{region: r}
	params := url.Values{"api-version": []string{"2019-08-01"}}
	return asp, r.get(planId, params, asp)
}

type SServerFarmSku struct {
	ResourceType string
	Sku          SSkuSpec
	Capacity     SSkuCapacity
}

type SSkuSpec struct {
	Name string
	Tier string
}

func (asp *SAppServicePlan) GetSkus() ([]SServerFarmSku, error) {
	result := []SServerFarmSku{}
	resource := fmt.Sprintf("subscriptions/%s/resourceGroups/%s/providers/Microsoft.Web/serverfarms/%s/skus", asp.region.client._subscriptionId(), asp.Properties.ResourceGroup, asp.Name)
	err := asp.region.client.list(resource, url.Values{"api-version": []string{"2019-08-01"}}, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type SAppServicePlanVnet struct {
	Id         string
	Name       string
	Properties SASPVnetProperties
}

type SASPVnetProperties struct {
	VnetResourceId string
}

func (asp *SAppServicePlan) GetNetworks() ([]SNetwork, error) {
	vnets, err := asp.GetVnets()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get Vnets")
	}
	networks := make([]SNetwork, 0, len(vnets))
	for i := range vnets {
		network, err := asp.region.GetNetwork(vnets[i].Properties.VnetResourceId)
		if err != nil {
			return nil, errors.Wrap(err, "unable to get Network")
		}
		networks = append(networks, *network)
	}
	return networks, nil
}

func (asp *SAppServicePlan) GetVnets() ([]SAppServicePlanVnet, error) {
	result := []SAppServicePlanVnet{}
	resource := fmt.Sprintf("subscriptions/%s/resourceGroups/%s/providers/Microsoft.Web/serverfarms/%s/virtualNetworkConnections", asp.region.client._subscriptionId(), asp.Properties.ResourceGroup, asp.Name)
	err := asp.region.client.list(resource, url.Values{"api-version": []string{"2019-08-01"}}, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

type SSkuInfos struct {
	ResourceType string
	Skus         []SResourceSku
}

func (r *SRegion) GetAppServiceSkus() ([]SSkuInfos, error) {
	result := []SSkuInfos{}
	resource := "Microsoft.Web/skus"
	err := r.client.list(resource, url.Values{"api-version": []string{"2019-08-01"}}, &result)
	if err != nil {
		return nil, err
	}
	return result, err
}
