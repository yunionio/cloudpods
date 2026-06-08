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

package ucloud

import (
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

// https://docs.ucloud.cn/api/summary/regionlist
var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	// 中国
	"cn-bj2":      api.RegionBeijing,
	"cn-wlcb":     api.RegionWulanchabu,
	"cn-sh2":      api.RegionShanghai,
	"cn-gd":       api.RegionGuangzhou,
	"cn-guiyang1": api.RegionGuiyang,
	"hk":          api.RegionHongkong,
	"tw-tp":       api.RegionTaipei,
	// 亚太
	"sg":          api.RegionSingapore,
	"jpn-tky":     api.RegionTokyo,
	"kr-seoul":    api.RegionSeoul,
	"th-bkk":      api.RegionBangkok,
	"idn-jakarta": api.RegionJakarta,
	"vn-sng":      api.RegionHoChiMinh,
	"ph-mnl":      api.RegionManila,
	"ind-mumbai":  api.RegionMumbai,
	"pk-khi":      api.RegionKarachi,
	"uz-tas":      api.RegionTashkent,
	"kz-ala":      api.RegionAlmaty,
	// 美洲
	"us-den":       api.RegionDenver,
	"us-ca":        api.RegionLosAngeles,
	"us-ws":        api.RegionWashingtonDC,
	"bra-saopaulo": api.RegionSaoPaulo,
	"mx-mex":       api.RegionMexico,
	// 欧洲、中东及非洲
	"ge-fra":      api.RegionFrankfurt,
	"uk-london":   api.RegionLondon,
	"uae-dubai":   api.RegionDubai,
	"afr-nigeria": api.RegionLagos,
}
