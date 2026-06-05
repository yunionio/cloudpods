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

package rockbase

import (
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"hk":           api.RegionHongkong,
	"tw-tp":        api.RegionTaipei,
	"sg":           api.RegionSingapore,
	"jpn-tky":      api.RegionTokyo,
	"kr-seoul":     api.RegionSeoul,
	"th-bkk":       api.RegionBangkok,
	"idn-jakarta":  api.RegionJakarta,
	"vn-sng":       api.RegionHoChiMinh,
	"ph-mnl":       api.RegionManila,
	"ind-mumbai":   api.RegionMumbai,
	"us-den":       api.RegionDenver,
	"us-ca":        api.RegionLosAngeles,
	"us-ws":        api.RegionWashingtonDC,
	"bra-saopaulo": api.RegionSaoPaulo,
	"ge-fra":       api.RegionFrankfurt,
	"uk-london":    api.RegionLondon,
	"uae-dubai":    api.RegionDubai,
	"afr-nigeria":  api.RegionLagos,
}
