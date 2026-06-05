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

package oracle

import (
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

// ref: https://docs.oracle.com/en-us/iaas/Content/General/Concepts/regions.htm
// ref: https://github.com/oracle/oci-go-sdk/blob/v65.110.0/common/regions.go
var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	// Asia Pacific
	"ap-batam-1":        api.RegionJakarta,
	"ap-chiyoda-1":      api.RegionTokyo,
	"ap-chuncheon-1":    api.RegionSeoul,
	"ap-chuncheon-2":    api.RegionSeoul,
	"ap-delhi-1":        api.RegionDelhi,
	"ap-hyderabad-1":    api.RegionHyderabad,
	"ap-ibaraki-1":      api.RegionTokyo,
	"ap-kulai-2":        api.RegionKualaLumpur,
	"ap-melbourne-1":    api.RegionMelbourne,
	"ap-mumbai-1":       api.RegionMumbai,
	"ap-osaka-1":        api.RegionOsaka,
	"ap-seoul-1":        api.RegionSeoul,
	"ap-seoul-2":        api.RegionSeoul,
	"ap-singapore-1":    api.RegionSingapore,
	"ap-singapore-2":    api.RegionSingapore,
	"ap-suwon-1":        api.RegionSeoul,
	"ap-sydney-1":       api.RegionSydney,
	"ap-tokyo-1":        api.RegionTokyo,
	"ap-dcc-canberra-1": api.RegionYarralumla,
	"ap-dcc-gazipur-1":  api.RegionMumbai,

	// Canada
	"ca-montreal-1": api.RegionMontreal,
	"ca-toronto-1":  api.RegionToronto,

	// Europe
	"eu-amsterdam-1":  api.RegionHolland,
	"eu-budapest-1":   api.RegionWarsaw,
	"eu-crissier-1":   api.RegionGeneva,
	"eu-frankfurt-1":  api.RegionFrankfurt,
	"eu-frankfurt-2":  api.RegionFrankfurt,
	"eu-jovanovac-1":  api.RegionWarsaw,
	"eu-madrid-1":     api.RegionMadrid,
	"eu-madrid-2":     api.RegionMadrid,
	"eu-madrid-3":     api.RegionMadrid,
	"eu-marseille-1":  api.RegionParis,
	"eu-milan-1":      api.RegionMilan,
	"eu-paris-1":      api.RegionParis,
	"eu-stockholm-1":  api.RegionStockholm,
	"eu-turin-1":      api.RegionTurin,
	"eu-zurich-1":     api.RegionZurich,
	"eu-dcc-dublin-1": api.RegionDublin,
	"eu-dcc-dublin-2": api.RegionDublin,
	"eu-dcc-milan-1":  api.RegionMilan,
	"eu-dcc-milan-2":  api.RegionMilan,
	"eu-dcc-rating-1": api.RegionFrankfurt,
	"eu-dcc-rating-2": api.RegionFrankfurt,
	"eu-dcc-zurich-1": api.RegionZurich,
	"uk-cardiff-1":    api.RegionLondon,
	"uk-london-1":     api.RegionLondon,
	"uk-gov-cardiff-1": api.RegionLondon,
	"uk-gov-london-1": api.RegionLondon,

	// Middle East
	"me-abudhabi-1":   api.RegionAbuDhabi,
	"me-abudhabi-2":   api.RegionAbuDhabi,
	"me-abudhabi-3":   api.RegionAbuDhabi,
	"me-abudhabi-4":   api.RegionAbuDhabi,
	"me-alain-1":      api.RegionAbuDhabi,
	"me-dcc-doha-1":   api.RegionDoha,
	"me-dcc-muscat-1": api.RegionDubai,
	"me-dubai-1":      api.RegionDubai,
	"me-ibri-1":       api.RegionDubai,
	"me-jeddah-1":     api.RegionDamman,
	"me-riyadh-1":     api.RegionDamman,

	// Africa
	"af-casablanca-1":   api.RegionMadrid,
	"af-johannesburg-1": api.RegionJohannesburg,

	// Israel
	"il-jerusalem-1": api.RegionTelAviv,

	// North America
	"us-ashburn-1":     api.RegionNothVirginia,
	"us-ashburn-2":     api.RegionNothVirginia,
	"us-chicago-1":     api.RegionChicago,
	"us-gov-ashburn-1": api.RegionNothVirginia,
	"us-gov-chicago-1": api.RegionChicago,
	"us-gov-phoenix-1": api.RegionPhoenix,
	"us-langley-1":     api.RegionVirginia,
	"us-luke-1":        api.RegionPhoenix,
	"us-newark-1":      api.RegionVirginia,
	"us-phoenix-1":     api.RegionPhoenix,
	"us-saltlake-2":    api.RegionSaltLakeCity,
	"us-sanjose-1":     api.RegionSiliconValley,
	"us-somerset-1":    api.RegionVirginia,
	"us-thames-1":      api.RegionVirginia,
	"mx-monterrey-1":   api.RegionMexico,
	"mx-queretaro-1":   api.RegionMexico,

	// South America
	"sa-bogota-1":       api.RegionSaoPaulo,
	"sa-riodejaneiro-1": api.RegionRioDeJaneiro,
	"sa-santiago-1":     api.RegionSantiago,
	"sa-saopaulo-1":     api.RegionSaoPaulo,
	"sa-valparaiso-1":   api.RegionSantiago,
	"sa-vinhedo-1":      api.RegionSaoPaulo,
}
