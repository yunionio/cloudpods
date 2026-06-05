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

package huawei

import (
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

// China: https://developer.huaweicloud.com/endpoint
// International: https://developer-intl.huaweicloud.com/endpoint
// ref: https://countrycode.org
var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	// 华北
	"cn-north-1":   api.RegionBeijing,
	"cn-north-4":   api.RegionBeijing,
	"cn-north-9":   api.RegionNeimenggu,
	"cn-north-11":  api.RegionWulanchabu,
	"cn-north-12":  api.RegionBeijing,
	"cn-north-219": api.RegionBeijing,

	// 华东
	"cn-east-2": api.RegionShanghai,
	"cn-east-3": api.RegionShanghai,
	"cn-east-4": api.RegionWuhu,
	"cn-east-5": api.RegionQingdao,

	// 华南
	"cn-south-1": api.RegionGuangzhou,
	"cn-south-2": api.RegionGuangzhou,
	"cn-south-4": api.RegionGuangzhou,

	// 西南 / 东北 / 西北
	"cn-southwest-2": api.RegionGuiyang,
	"cn-northeast-1": api.RegionDalian,
	"cn-northwest-1": api.RegionNingxia,

	// 亚太
	"ap-southeast-1": api.RegionHongkong,
	"ap-southeast-2": api.RegionBangkok,
	"ap-southeast-3": api.RegionSingapore,
	"ap-southeast-4": api.RegionJakarta,
	"ap-southeast-5": api.RegionManila,

	// 欧洲
	"eu-west-0": api.RegionParis,
	"eu-west-1": api.RegionDublin,

	// 中东 / 土耳其
	"me-east-1": api.RegionDamman,
	"tr-west-1": api.RegionIstanbul,

	// 非洲
	"af-south-1": api.RegionJohannesburg,
	"af-north-1": api.RegionTelAviv,

	// 拉美
	"na-mexico-1":    api.RegionMexico,
	"la-north-2":     api.RegionMexico,
	"sa-brazil-1":    api.RegionSaoPaulo,
	"la-south-2":     api.RegionSantiago,
	"sa-argentina-1": api.RegionSantiago,
	"sa-peru-1":      api.RegionSantiago,
}
