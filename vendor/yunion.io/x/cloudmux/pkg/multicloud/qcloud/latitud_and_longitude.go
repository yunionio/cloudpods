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

var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"ap-bangkok":        api.RegionBangkok,       // 腾讯云 亚太地区(曼谷)
	"ap-beijing":        api.RegionBeijing,       // 腾讯云 华北地区(北京)
	"ap-chengdu":        api.RegionChengdu,       // 腾讯云 西南地区(成都)
	"ap-chongqing":      api.RegionChongqing,     // 腾讯云 西南地区(重庆)
	"ap-guangzhou":      api.RegionGuangzhou,     // 腾讯云 华南地区(广州)
	"ap-guangzhou-open": api.RegionGuangzhou,     // 腾讯云 华南地区(广州Open)
	"ap-hongkong":       api.RegionHongkong,      // 腾讯云 东南亚地区(香港)
	"ap-mumbai":         api.RegionMumbai,        // 腾讯云 亚太地区(孟买)
	"ap-seoul":          api.RegionSeoul,         // 腾讯云 东南亚地区(首尔)
	"ap-shanghai":       api.RegionShanghai,      // 腾讯云 华东地区(上海)
	"ap-shanghai-fsi":   api.RegionShanghai,      // 腾讯云 华东地区(上海金融)
	"ap-shenzhen-fsi":   api.RegionShenzhen,      // 腾讯云 华南地区(深圳金融)
	"ap-singapore":      api.RegionSingapore,     // 腾讯云 东南亚地区(新加坡)
	"ap-tokyo":          api.RegionTokyo,         // 腾讯云 亚太地区(东京)
	"eu-frankfurt":      api.RegionFrankfurt,     // 腾讯云 欧洲地区(德国)
	"eu-moscow":         api.RegionMoscow,        // 腾讯云 欧洲地区(莫斯科)
	"na-ashburn":        api.RegionVirginia,      // 腾讯云 美国东部(弗吉尼亚)
	"na-siliconvalley":  api.RegionSiliconValley, // 腾讯云 美国西部(硅谷)
	"na-toronto":        api.RegionToronto,       // 腾讯云 北美地区(多伦多)
	"ap-nanjing":        api.RegionNanjing,
}
