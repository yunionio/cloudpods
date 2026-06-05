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

package ctyun

import (
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

// GetGeographicInfo 使用 RegionCode，无 RegionCode 时使用 RegionId 作为 key
var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	// 北京
	"cn-beijing-5": api.RegionBeijing,

	// 天津 / 华北
	"cn-tianjin-2":  api.RegionTianjin,
	"cn-tianjin-01": api.RegionTianjin,
	"e4874db42e9211ed88f70242ac110002": api.RegionTianjin, // 天津3

	// 河北
	"cn-he-sjz20-hybrid-ctcloud":       api.RegionShijiazhuang,
	"3ddd0446876d11eaab020242ac110002": api.RegionShijiazhuang, // 石家庄2
	"529d1808bcf511eb93260242ac110002": api.RegionShijiazhuang, // 石家庄3

	// 内蒙
	"cn-neimeng-6":      api.RegionNeimenggu,
	"cn-huhehaote-03":   api.RegionHuhehaote,
	"cn-huhehaote-6":    api.RegionHuhehaote, // 历史别名

	// 山西
	"cn-jinzhong-2":                    api.RegionJinzhong,
	"cn-taiyuan-04":                    api.RegionJinzhong,
	"05f7a93651a211ecbe170242ac110002": api.RegionJinzhong, // 太原2

	// 辽宁
	"cn-shenyang-08":                   api.RegionShenyang,
	"cn-ln-ly1-public-ctcloud":         api.RegionShenyang,
	"172f2480f64b11ea98a40242ac110002": api.RegionShenyang, // 沈阳4
	"200000001820":                     api.RegionShenyang, // 辽阳2

	// 黑龙江
	"cn-haerbin-2": api.RegionHaerbin,

	// 上海 / 华东
	"cn-shanghai-7":                    api.RegionShanghai,
	"cn-shanghai-15":                   api.RegionShanghai,
	"cn-shanghai-36":                   api.RegionShanghai,
	"cn-huadong-1":                     api.RegionShanghai,
	"db0fed10499511eb8a780242ac110002": api.RegionShanghai, // 上海8
	"01aaa9182e9311edad970242ac110002": api.RegionShanghai, // 上海9

	// 江苏
	"cn-nanjing-2":                     api.RegionNanjing,
	"cn-nanjing-3":                     api.RegionNanjing,
	"cn-nanjing-4":                     api.RegionNanjing,
	"cn-nanjing-5":                     api.RegionNanjing,
	"0e72cfe651a211ecabbe0242ac110002": api.RegionNanjing, // 扬州

	// 浙江
	"cn-hangzhou-2":  api.RegionHangzhou,
	"cn-hanzhou-07":  api.RegionHangzhou,
	"200000001856":   api.RegionHangzhou, // 浙江金华6

	// 安徽
	"cn-wuhu-1":   api.RegionWuhu,
	"cn-wuhu-04":  api.RegionWuhu,
	"cn-hefei2":   api.RegionHefei,
	"b02ba000ab1c11ec8c4a0242ac110002": api.RegionHefei, // 合肥
	"200000003615": api.RegionWuhu, // 芜湖5

	// 福建
	"cn-fujian-3":  api.RegionFujian,
	"cn-fuzhou-4":  api.RegionFujian,
	"cn-xiamen-3":  api.RegionFujian,
	"cn-fuzhou-25": api.RegionFujian,
	"4f23b922ab1d11ecaa9f0242ac110002": api.RegionFujian, // 福州5

	// 江西
	"cn-jiangxi-2":   api.RegionJiangxi,
	"cn-nanchang-05": api.RegionJiangxi,
	"200000001476":   api.RegionJiangxi, // 南昌18
	"200000004030":   api.RegionJiangxi, // 南昌4

	// 山东
	"cn-qingdao-20":                    api.RegionQingdao,
	"9395e62eb52511eab9d70242ac110002": api.RegionQingdao, // 青岛3
	"63daa10af68111ecb89f0242ac110002": api.RegionQingdao, // 青岛4

	// 河南
	"cn-zhengzhou-2":  api.RegionZhengzhou,
	"cn-zhengzhou-05": api.RegionZhengzhou,
	"200000004029":    api.RegionZhengzhou, // 郑州13

	// 湖北
	"cn-wuhan-3":  api.RegionWuhan,
	"cn-wuhan-5":  api.RegionWuhan,
	"cn-wuhan-41": api.RegionWuhan,

	// 湖南
	"cn-hunan-3":       api.RegionChangsha,
	"cn-changsha-42":   api.RegionChangsha,
	"cn-chenzhou-2":    api.RegionChengzhou,
	"cn-chenzhou-4":    api.RegionChengzhou, // 历史别名

	// 广东
	"cn-foshan-3":    api.RegionFoshan,
	"cn-guangzhou-5": api.RegionGuangzhou,
	"cn-huanan-02":   api.RegionGuangzhou,
	"f4b6680cac0411ecb2da0242ac110002": api.RegionGuangzhou, // 广州5

	// 广西
	"cn-nanning-2":  api.RegionNanning,
	"cn-nanning-23": api.RegionNanning,
	"200000001709":  api.RegionNanning, // 南宁24

	// 海南
	"cn-haikou-2": api.RegionHaikou,

	// 重庆
	"cn-chongqing-2": api.RegionChongqing,

	// 四川
	"cn-yaan-2":                        api.RegionChengdu,
	"ab8a29247cc111ec94230242ac110002": api.RegionChengdu, // 成都5
	"cn-xinan-01":                      api.RegionChengdu,
	"cn-lasa-4":                        api.RegionChengdu, // 拉萨

	// 贵州
	"cn-guiyang-1":    api.RegionGuiyang,
	"cn-xinan2-gz-01": api.RegionGuiyang,

	// 云南
	"cn-kunming-2": api.RegionKunming,

	// 陕西
	"cn-shanxi-2": api.RegionXian,
	"cn-xian-4":   api.RegionXian,
	"cn-xian-5":   api.RegionXian,
	"cn-xian-07":  api.RegionXian,

	// 甘肃
	"cn-lanzhou-3":   api.RegionLanzhou,
	"cn-qingyang-02": api.RegionQingYang,
	"05aa842ab38d11eba6640242ac110002": api.RegionLanzhou, // 兰州3
	"200000003684": api.RegionQingYang, // 庆阳3

	// 青海
	"cn-xining-2": api.RegionLanzhou,

	// 宁夏
	"cn-zhongwei-2":  api.RegionNingxia,
	"cn-zhongwei-05": api.RegionNingxia,
	"200000003576":   api.RegionNingxia, // 中卫6

	// 新疆
	"cn-wlmq-27":                       api.RegionWulumuqi,
	"cn-wulumuqi-07":                   api.RegionWulumuqi,
	"2d0cadee70d211ea8c4d0242ac110002": api.RegionWulumuqi, // 乌鲁木齐2

	// 港澳及海外
	"cn-xianggang-1":  api.RegionHongkong,
	"cn-xianggang-02": api.RegionHongkong,
	"4681b038013011ea9b210242ac110002": api.RegionHongkong, // 香港5
	"cn-aomen-01":     api.RegionHongkong,
	"cn-xinjiapo-3":   api.RegionSingapore,
	"cn-xinjiapo-04":  api.RegionSingapore,
	"cn-falankefu-1":  api.RegionFrankfurt,
	"cn-shengbaoluo-1": api.RegionSaoPaulo,
	"cn-dibai-1":      api.RegionDubai,
	"cn-ydnixiya-01":  api.RegionJakarta,
	"cn-badayan-01":   api.RegionManila,
}
