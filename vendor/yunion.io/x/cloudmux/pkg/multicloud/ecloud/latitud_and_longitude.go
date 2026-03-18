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

package ecloud

import (
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	// 对齐最新 RegionId（cn- 前缀），与 regionIdToPoolId 保持一致；尽量复用 compute.RegionXXX
	"cn-beijing-1":   api.RegionBeijing,
	"cn-jiangsu-1":   api.RegionSuzhou,    // 江苏（无锡附近）复用苏州坐标
	"cn-guangdong-1": api.RegionGuangzhou, // 广东（东莞附近）复用广州坐标
	"cn-sichuan-1":   api.RegionChengdu,   // 四川（雅安/成都附近）复用成都坐标
	"cn-henan-1":     api.RegionZhengzhou, // 河南（郑州）
	"cn-hunan-1":     api.RegionChangsha,  // 湖南（株洲/长沙）
	"cn-shandong-1":  api.RegionJiNan,     // 山东（济南）
	"cn-shaanxi-1":   api.RegionXian,      // 陕西（西安）
	"cn-shanghai-1":  api.RegionShanghai,  // 上海
	"cn-chongqing-1": api.RegionChongqing, // 重庆
	"cn-zhejiang-1":  api.RegionHangzhou,  // 浙江（宁波/杭州）
	"cn-tianjin-1":   api.RegionTianjin,   // 天津
	"cn-jilin-1":     api.RegionChangchun, // 吉林（长春）
	"cn-hubei-2":     api.RegionXiangyang, // 湖北（襄阳）
	"cn-jiangxi-1":   api.RegionJiangxi,   // 江西（南昌）
	"cn-gansu-1":     api.RegionLanzhou,   // 甘肃（兰州）
	"cn-shangxi-1":   api.RegionJinzhong,  // 山西（太原附近），复用晋中
	"cn-liaoning-1":  api.RegionShenyang,  // 辽宁（沈阳）
	"cn-yunnan-1":    api.RegionKunming,   // 云南（昆明）
	"cn-hebei-1":     api.RegionShijiazhuang, // 河北（石家庄）
	"cn-fujian-1":    api.RegionFujian,    // 福建（厦门附近）
	"cn-guangxi-1":   api.RegionNanning,   // 广西（南宁）
	"cn-anhui-1":     api.RegionHuainan,   // 安徽（淮南）
	"cn-neimenggu-1": api.RegionHuhehaote, // 内蒙古（呼和浩特）
	"cn-guzhou-1":    api.RegionGuiyang,   // 贵州（贵阳）
	"cn-hainan-1":    api.RegionHaikou,    // 海南（海口）
	"cn-xinjiang-1":  api.RegionWulumuqi,  // 新疆（乌鲁木齐）
}
