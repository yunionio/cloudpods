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

package google

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
)

type SkuBillingCatetory struct {
	ServiceDisplayName string
	ResourceFamily     string
	ResourceGroup      string
	UsageType          string
}

type SkuPricingInfo struct {
	Summary                string
	PricingExpression      SPricingExpression
	currencyConversionRate int
	EffectiveTime          time.Time
}

type SPricingExpression struct {
	UsageUnit                string
	UsageUnitDescription     string
	BaseUnit                 string
	BaseUnitDescription      string
	BaseUnitConversionFactor string
	DisplayQuantity          int
	TieredRates              []STieredRate
}

type STieredRate struct {
	StartUsageAmount int
	UnitPrice        SUnitPrice
}

type SUnitPrice struct {
	CurrencyCode string
	Units        string
	Nanos        int
}

type SSkuBilling struct {
	Name                string
	SkuId               string
	Description         string
	Category            SkuBillingCatetory
	ServiceRegions      []string
	PricingInfo         []SkuPricingInfo
	ServiceProviderName string
}

func (region *SRegion) ListSkuBilling(pageSize int, pageToken string) ([]SSkuBilling, error) {
	skus := []SSkuBilling{}
	params := map[string]string{}
	err := region.BillingList("services/6F81-5844-456A/skus", params, pageSize, pageToken, &skus)
	if err != nil {
		return nil, err
	}
	return skus, nil
}

type SRateInfo struct {
	// region: europe-north1
	// family: Compute, Storage, Network
	// resource: CPU, Ram, Gpu, N1Standard
	// category: custome, predefine, optimized
	// startUsageAmount:
	// map[region]map[family]map[resource]map[category]map[startUsageAmount][money]

	Info map[string]map[string]map[string]map[string]map[string]float64
}

func (region *SRegion) GetSkuRateInfo(skus []SSkuBilling) SRateInfo {
	result := SRateInfo{
		Info: map[string]map[string]map[string]map[string]map[string]float64{},
	}
	for _, sku := range skus {
		if sku.ServiceProviderName == "Google" &&
			sku.Category.ServiceDisplayName == "Compute Engine" &&
			utils.IsInStringArray(sku.Category.ResourceFamily, []string{"Compute", "Storage"}) &&
			sku.Category.UsageType == "OnDemand" {
			for _, region := range sku.ServiceRegions {
				if _, ok := result.Info[region]; !ok {
					result.Info[region] = map[string]map[string]map[string]map[string]float64{}
				}
				if _, ok := result.Info[region][sku.Category.ResourceFamily]; !ok {
					result.Info[region][sku.Category.ResourceFamily] = map[string]map[string]map[string]float64{}
				}
				if sku.Category.ResourceGroup == "N1Standard" {
					if strings.Index(sku.Description, "Core") > 0 {
						sku.Category.ResourceGroup = "CPU"
					} else if strings.Index(sku.Description, "Ram") > 0 {
						sku.Category.ResourceGroup = "RAM"
					}
				}
				if !utils.IsInStringArray(sku.Category.ResourceGroup, []string{"F1Micro", "G1Small", "CPU", "RAM", "PDStandard", "SSD", "LocalSSD", "f1-micro", "g1-small"}) {
					continue
				}
				if utils.IsInStringArray(sku.Category.ResourceGroup, []string{"PDStandard", "SSD"}) && strings.Index(sku.Description, "Regional") >= 0 {
					continue
				}
				convers := map[string]string{
					"PDStandard": api.STORAGE_GOOGLE_PD_STANDARD,
					"SSD":        api.STORAGE_GOOGLE_PD_SSD,
					"LocalSSD":   api.STORAGE_GOOGLE_LOCAL_SSD,
					"F1Micro":    "f1-micro",
					"G1Small":    "g1-small",
				}
				if group, ok := convers[sku.Category.ResourceGroup]; ok {
					sku.Category.ResourceGroup = group
				}

				if _, ok := result.Info[region][sku.Category.ResourceFamily][sku.Category.ResourceGroup]; !ok {
					result.Info[region][sku.Category.ResourceFamily][sku.Category.ResourceGroup] = map[string]map[string]float64{}
				}
				description := strings.ToLower(sku.Description)
				if strings.Contains("sole", description) { //单租户
					continue
				}
				category := ""
				keys := []string{"memory optimized", "memory-optimized", "compute optimized", "n1 predefined", "n2 instance", "n2 custom extended", "custom extended", "n2 custom", "custom instance"}
				categories := map[string]string{
					"memory optimized":  "ultramem",
					"memory-optimized":  "memory-optimized",
					"compute optimized": "compute-optimized",

					"n1 predefined": "n1-predefined",
					"n2 instance":   "n2-instance",

					"n2 custom extended": "n2-custom-extended",
					"custom extended":    "custom-extended",

					"n2 custom":       "n2-custom",       //cpu ram
					"custom instance": "custom-instance", //cpu
				}
				for _, key := range keys {
					_category := categories[key]
					if strings.Contains(description, key) {
						category = _category
						break
					}
				}
				if utils.IsInStringArray(sku.Category.ResourceGroup, []string{api.STORAGE_GOOGLE_PD_STANDARD, api.STORAGE_GOOGLE_PD_SSD, api.STORAGE_GOOGLE_LOCAL_SSD, "f1-micro", "g1-small"}) {
					category = sku.Category.ResourceGroup
				}
				if len(category) == 0 {
					continue
				}
				if _, ok := result.Info[region][sku.Category.ResourceFamily][sku.Category.ResourceGroup][category]; !ok {
					result.Info[region][sku.Category.ResourceFamily][sku.Category.ResourceGroup][category] = map[string]float64{}
				}
				for _, priceInfo := range sku.PricingInfo {
					for _, price := range priceInfo.PricingExpression.TieredRates {
						result.Info[region][sku.Category.ResourceFamily][sku.Category.ResourceGroup][category][fmt.Sprintf("%d", price.StartUsageAmount)] = float64(price.UnitPrice.Nanos) / 1000000000
					}
				}
			}
		}
	}
	return result
}

func (rate *SRateInfo) GetDiscount(sku string) []float64 {
	if strings.Index(sku, "custom") < 0 && strings.HasPrefix(sku, "c") || strings.Index(sku, "n2") >= 0 {
		return []float64{0, 0.15, 0.25, 0.4}
	}
	return []float64{0, 0.2, 0.4, 0.6}
}

func (rate *SRateInfo) GetSkuType(sku string) (string, error) {
	if strings.Index(sku, "custom") >= 0 {
		cpuType := "custom-instance"
		if strings.HasPrefix(sku, "n2") {
			cpuType = "n2-custom"
		}
		return cpuType, nil
	}
	if strings.HasPrefix(sku, "n1") {
		return "n1-predefined", nil
	}
	if strings.HasPrefix(sku, "n2") {
		return "n2-instance", nil
	}
	if strings.HasPrefix(sku, "c") {
		return "compute-optimized", nil
	}
	if strings.HasPrefix(sku, "m") {
		return "memory-optimized", nil
	}
	return "", fmt.Errorf("failed to found sku %s type", sku)
}

func (rate *SRateInfo) GetCpuPrice(regionId, cpuType string) (float64, error) {
	computePrice, err := rate.GetComputePrice(regionId)
	if err != nil {
		return 0, errors.Wrap(err, "GetComputePrice")
	}

	_cpuPrice, ok := computePrice["CPU"]
	if !ok {
		return 0, fmt.Errorf("failed to found region %s compute cpu price info", regionId)
	}

	cpuPrice, ok := _cpuPrice[cpuType]
	if !ok {
		return 0, fmt.Errorf("failed to found region %s compute %s cpu price info", regionId, cpuType)
	}
	return cpuPrice["0"], nil
}

func (rate *SRateInfo) GetExtendMemoryGb(sku string, cpu int, memoryMb int) float64 {
	maxMemoryGb := 0.0
	if strings.Index(sku, "custom") >= 0 {
		maxMemoryGb = float64(cpu) * 6.5
		if strings.Index(sku, "n2") >= 0 {
			maxMemoryGb = float64(cpu) * 8
		}
	}
	if float64(memoryMb)/1024 > maxMemoryGb {
		return float64(memoryMb)/1024 - maxMemoryGb
	}
	return 0.0
}

func (rate *SRateInfo) GetMemoryPrice(regionId string, memoryType string) (float64, error) {
	computePrice, err := rate.GetComputePrice(regionId)
	if err != nil {
		return 0, errors.Wrap(err, "GetComputePrice")
	}

	_memoryPrice, ok := computePrice["RAM"]
	if !ok {
		return 0, fmt.Errorf("failed to found region %s compute memory price info", regionId)
	}

	memoryPrice, ok := _memoryPrice[memoryType]
	if !ok {
		return 0, fmt.Errorf("failed to found region %s compute %s memory price info", regionId, memoryType)
	}
	return memoryPrice["0"], nil
}

func (rate *SRateInfo) GetComputePrice(regionId string) (map[string]map[string]map[string]float64, error) {
	regionPrice, ok := rate.Info[regionId]
	if !ok {
		return nil, fmt.Errorf("failed to found region %s price info", regionId)
	}
	computePrice, ok := regionPrice["Compute"]
	if !ok {
		return nil, fmt.Errorf("failed to found region %s compute price info", regionId)
	}
	return computePrice, nil
}

func (rate *SRateInfo) GetSharedSkuPrice(regionId string, sku string) (float64, error) {
	computePrice, err := rate.GetComputePrice(regionId)
	if err != nil {
		return 0.0, errors.Wrap(err, "GetComputePrice")
	}
	if _sharedSku, ok := computePrice[sku]; ok {
		if sharedSku, ok := _sharedSku[sku]; ok {
			return sharedSku["0"], nil
		}
	}
	return 0.0, fmt.Errorf("sku is not shared sku")
}

func (rate *SRateInfo) GetSkuPrice(regionId string, sku string, cpu, memoryMb int) (struct {
	Hour  float64
	Month float64
	Year  float64
}, error) {
	result := struct {
		Hour  float64
		Month float64
		Year  float64
	}{}

	discount := rate.GetDiscount(sku)

	price, err := rate.GetSharedSkuPrice(regionId, sku)
	if err != nil {
		skuType, err := rate.GetSkuType(sku)
		if err != nil {
			return result, errors.Wrap(err, "GetSkuType")
		}

		cpuPrice, err := rate.GetCpuPrice(regionId, skuType)
		if err != nil {
			return result, errors.Wrap(err, "price.GetCpuPrice")
		}

		price += float64(cpu) * cpuPrice
		log.Debugf("cpu price: %f", cpuPrice)

		extendMemoryGb := rate.GetExtendMemoryGb(sku, cpu, memoryMb)
		memoryMb = int(float64(memoryMb) - 1024*extendMemoryGb)

		memoryPrice, err := rate.GetMemoryPrice(regionId, skuType)
		if err != nil {
			return result, errors.Wrap(err, "GetMemoryPrice")
		}

		price += memoryPrice * float64(memoryMb/1024)
		log.Debugf("ramPrice: %f", memoryPrice)

		if extendMemoryGb > 0 {
			memoryType := "custom-extended"
			if strings.HasPrefix(sku, "n2") {
				memoryType = "n2-custom-extended"
			}
			extendPrice, err := rate.GetMemoryPrice(regionId, memoryType)
			if err != nil {
				return result, errors.Wrap(err, "GetMemoryPrice.Extend")
			}
			price += extendPrice * float64(extendMemoryGb)
			log.Debugf("extendPrice: %f", extendPrice)
		}
	}

	log.Debugf("totalPrice: %f", price)

	result.Month = 182.5 * price
	result.Month += 182.5 * (1 - discount[1]) * price
	result.Month += 182.5 * (1 - discount[2]) * price
	result.Month += 172.49999999999997 * (1 - discount[3]) * price
	result.Hour = result.Month / 30 / 24
	result.Year = result.Month * 12
	return result, nil
}
