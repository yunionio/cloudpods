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
	"cn-beijing-1":      api.RegionBeijing,
	"cn-beijing-11":     api.RegionBeijing,     // 华北-北京9
	"cn-neimenggu-1":    api.RegionHuhehaote,   // 华北-呼和浩特
	"cn-hebei-1":        api.RegionShijiazhuang, // 河北-石家庄
	"cn-shanxi-1":       api.RegionJinzhong,    // 山西-太原
	"cn-shangxi-1":      api.RegionJinzhong,    // 山西-太原（历史拼写）
	"cn-tianjin-1":      api.RegionTianjin,     // 天津-天津
	"cn-liaoning-1":     api.RegionShenyang,    // 辽宁-沈阳
	"cn-jilin-1":        api.RegionChangchun,   // 吉林-长春
	"cn-heilongjiang-1": api.RegionHaerbin,     // 黑龙江-哈尔滨
	"cn-heilongjiang-3": api.RegionHaerbin,     // 东北-哈尔滨

	"cn-jiangsu-1":  api.RegionSuzhou,   // 华东-苏州
	"cn-jiangsu-20": api.RegionNanjing,  // 华东-南京
	"cn-shanghai-1": api.RegionShanghai, // 华东-上海1
	"cn-shanghai-5": api.RegionShanghai, // 华东-上海5
	"cn-zhejiang-1": api.RegionHangzhou, // 华东-杭州
	"cn-zhejiang-6": api.RegionHangzhou, // 华东-杭州4
	"cn-shandong-1": api.RegionJiNan,    // 华东-济南
	"cn-shandong-14": api.RegionQingdao, // 华东-青岛
	"cn-anhui-1":    api.RegionHuainan,  // 安徽-淮南
	"cn-fujian-1":   api.RegionFujian,   // 福建-厦门
	"cn-jiangxi-1":  api.RegionJiangxi,   // 江西-南昌

	"cn-guangdong-1": api.RegionGuangzhou, // 华南-广州3
	"cn-guangxi-1":   api.RegionNanning,   // 广西-南宁3
	"cn-guangxi-2":   api.RegionNanning,   // 广西-南宁
	"cn-hainan-1":    api.RegionHaikou,    // 海南-海口

	"cn-henan-1":  api.RegionZhengzhou, // 华中-郑州
	"cn-hubei-1":  api.RegionWuhan,     // 湖北-武汉
	"cn-hubei-2":  api.RegionXiangyang, // 湖北-襄阳
	"cn-hunan-1":  api.RegionChangsha,  // 华中-长沙2

	"cn-sichuan-1": api.RegionChengdu,   // 西南-成都
	"cn-sichuan-7": api.RegionChengdu,   // 西南-成都4
	"cn-chongqing-1": api.RegionChongqing, // 西南-重庆
	"cn-guizhou-1": api.RegionGuiyang,   // 西南-贵阳
	"cn-guzhou-1":  api.RegionGuiyang,   // 西南-贵阳（历史拼写）
	"cn-guizhou-5": api.RegionGuiyang,   // 西南-贵阳3
	"cn-yunnan-1":  api.RegionKunming,   // 云南-昆明2
	"cn-xizang-1":  api.RegionChengdu,   // 西藏-拉萨
	"cn-xizang-2":  api.RegionChengdu,   // 西藏-拉萨2

	"cn-shaanxi-1":  api.RegionXian,      // 西北-西安
	"cn-gansu-1":    api.RegionLanzhou,   // 甘肃-兰州
	"cn-qinghai-1":  api.RegionLanzhou,   // 青海-海东
	"cn-qinghai-2":  api.RegionLanzhou,   // 青海-海东2
	"cn-qinghai-3":  api.RegionLanzhou,   // 青海-西宁
	"cn-ningxia-1":  api.RegionNingxia,   // 宁夏-中卫
	"cn-ningxia-2":  api.RegionNingxia,   // 宁夏-中卫2
	"cn-ningxia-3":  api.RegionNingxia,   // 西北-中卫
	"cn-xinjiang-1": api.RegionWulumuqi,  // 新疆-昌吉
}
