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

package qcloud

import (
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

// ref: https://cloud.tencent.com/document/product/213/6091
// ref: https://cloud.tencent.com/developer/article/1930067
var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	// 华北
	"ap-beijing":           api.RegionBeijing,
	"ap-beijing-fsi":       api.RegionBeijing,
	"ap-tianjin":           api.RegionTianjin,
	"ap-shijiazhuang-ec":   api.RegionShijiazhuang,

	// 华东
	"ap-shanghai":      api.RegionShanghai,
	"ap-shanghai-fsi":  api.RegionShanghai,
	"ap-shanghai-adc":  api.RegionShanghai,
	"ap-nanjing":       api.RegionNanjing,
	"ap-jinan-ec":      api.RegionJiNan,
	"ap-hangzhou-ec":   api.RegionHangzhou,
	"ap-fuzhou-ec":     api.RegionFujian,
	"ap-hefei-ec":      api.RegionHefei,

	// 华南
	"ap-guangzhou":        api.RegionGuangzhou,
	"ap-guangzhou-open":   api.RegionGuangzhou,
	"ap-qingyuan":         api.RegionGuangzhou,
	"ap-qingyuan-xinan":   api.RegionGuangzhou,
	"ap-shenzhen":         api.RegionShenzhen,
	"ap-shenzhen-fsi":     api.RegionShenzhen,
	"ap-shenzhen-sycft":   api.RegionShenzhen,

	// 华中
	"ap-wuhan-ec":       api.RegionWuhan,
	"ap-changsha-ec":    api.RegionChangsha,
	"ap-zhengzhou-ec":   api.RegionZhengzhou,

	// 西南 / 西北
	"ap-chengdu":   api.RegionChengdu,
	"ap-chongqing": api.RegionChongqing,
	"ap-zhongwei":  api.RegionNingxia,
	"ap-xian-ec":   api.RegionXian,
	"ap-xibei-ec":  api.RegionLanzhou,

	// 东北
	"ap-shenyang-ec": api.RegionShenyang,

	// 港澳台
	"ap-hongkong": api.RegionHongkong,
	"ap-taipei":   api.RegionTaiwan,

	// 亚太
	"ap-singapore": api.RegionSingapore,
	"ap-jakarta":   api.RegionJakarta,
	"ap-seoul":     api.RegionSeoul,
	"ap-tokyo":     api.RegionTokyo,
	"ap-osaka":     api.RegionOsaka,
	"ap-bangkok":   api.RegionBangkok,
	"ap-mumbai":    api.RegionMumbai,

	// 中东
	"me-saudi-arabia": api.RegionDamman,

	// 欧洲
	"eu-frankfurt": api.RegionFrankfurt,
	"eu-moscow":    api.RegionMoscow,

	// 北美 / 南美
	"na-ashburn":       api.RegionVirginia,
	"na-siliconvalley": api.RegionSiliconValley,
	"na-toronto":       api.RegionToronto,
	"sa-saopaulo":      api.RegionSaoPaulo,
}
