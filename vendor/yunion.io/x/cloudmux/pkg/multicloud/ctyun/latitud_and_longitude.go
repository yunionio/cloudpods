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

var CtyunRegionIdMap = map[string]string{
	"07323cf87fa811ea977e0242ac110002": "cn-lasa-4",
	"200000001627":                     "cn-fuzhou-5",
	"200000001681":                     "cn-ln-liaoyang-1",
	"200000001703":                     "cn-sd-qd20-public-ctcloud",
	"200000001704":                     "cn-gx-nn23-public-ctcloud",
	"200000001781":                     "cn-hb-wh41-public-ctcloud",
	"200000001788":                     "cn-wulumuqi-27",
	"200000001790":                     "cn-sh36-public-ctcloud",
	"200000001852":                     "cn-huabei2-public-ctcloud",
	"200000001858":                     "cfbr-fsao1-public-ctcloud",
	"200000001859":                     "cfae-fdxb1-public-ctcloud",
	"200000001860":                     "cfde-ffra1-public-ctcloud",
	"200000001861":                     "cfsg-fsin3-public-ctcloud",
	"200000002368":                     "cn-xinan1-public-ctcloud",
	"200000002401":                     "cn-hn-cs42-public-ctcloud",
	"200000002527":                     "cn-jx-khn5-public-ctcloud",
	"21c52b2a876e11ea9f6a0242ac110002": "cn-hangzhou-2",
	"22c0f506ef1d11ea80620242ac110002": "cn-nanjing-4",
	"276826f4313311eaaae30242ac110002": "cn-wuhan-3",
	"2cdd393e876f11ea98880242ac110002": "cn-yaan-2",
	"4009c41a876e11eabdc50242ac110002": "cn-wuhu-1",
	"415089caaea711eab0790242ac110002": "cn-kunming-2",
	"45d9efdad66f11ec9aab0242ac110002": "cn-hefei2",
	"461f819e6e3e11ea9ad30242ac110002": "cn-fujian-3",
	"49829300a71211ea95240242ac110002": "cn-jinzhong-2",
	"52c69bbc042411ec8dac0242ac110002": "cn-nanjing-5",
	"6019b5007a0b11eab5db0242ac110002": "cn-huhehaote-6",
	"60a39fca876e11ea91cf0242ac110002": "cn-nanjing-3",
	"705213b6876e11eaa5740242ac110002": "cn-haikou-2",
	"7dcbf0ba919c11ea83d60242ac110002": "cn-tianjin-2",
	"8062c840876e11ea9d060242ac110002": "cn-wuhan-5",
	"8d11979c4d5d11eab0520242ac110002": "cn-foshan-3",
	"8ef3dba6876e11ea8c2a0242ac110002": "cn-nanning-2",
	"9833d24065a211eaa6070242ac110002": "cn-chenzhou-4",
	"9859b8964d5d11eaba270242ac110002": "cn-jiangxi-2",
	"990ba31c22ec11eaaebd0242ac110002": "cn-hunan-3",
	"995b39bae63811ec8c4b0242ac110002": "cn-guangzhou-5",
	"a10d954c70f411eab3650242ac110002": "cn-chongqing-2",
	"a17034a4794111eaaa590242ac110002": "cn-shanghai-7",
	"a2ed23940b3911ea98040242ac110002": "cn-shanxi-2",
	"aaf589124d5d11eaa04d0242ac110002": "cn-guiyang-1",
	"ad51908ca3db11ea96c20242ac110002": "cn-haerbin-2",
	"aefabf04a3df11eaa3650242ac110002": "cn-zhengzhou-2",
	"b6bb383e876c11ea8a5e0242ac110002": "cn-beijing-5",
	"b7e069bc876e11eaa4c00242ac110002": "cn-xian-4",
	"bb9fdb42056f11eda1610242ac110002": "cn-huadong1-public-ctcloud",
	"d7d93102848711ea9ff10242ac110002": "cn-fuzhou-4",
	"dc3aceb4412211ecb8e70242ac110002": "cn-xian-5",
	"dff35c48876e11eaadc90242ac110002": "cn-lanzhou-3",
	"eeed8c16e13111e9a5b40242ac110002": "cn-nanjing-2",
}

var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"cn-beijing-5":   api.RegionBeijing,
	"cn-hangzhou-2":  api.RegionHangzhou,
	"cn-nanjing-4":   api.RegionNanjing,
	"cn-wuhan-3":     api.RegionWuhan,
	"cn-yaan-2":      api.RegionXian,
	"cn-wuhu-1":      api.RegionWuhu,
	"cn-kunming-2":   api.RegionKunming,
	"cn-hefei2":      api.RegionHefei,
	"cn-fujian-3":    api.RegionFujian,
	"cn-jinzhong-2":  api.RegionJinzhong,
	"cn-nanjing-5":   api.RegionNanning,
	"cn-huhehaote-6": api.RegionHuhehaote,
	"cn-nanjing-3":   api.RegionNanjing,
	"cn-haikou-2":    api.RegionHaikou,
	"cn-tianjin-2":   api.RegionTianjin,
	"cn-wuhan-5":     api.RegionWuhan,
	"cn-foshan-3":    api.RegionFoshan,
	"cn-nanning-2":   api.RegionNanning,
	"cn-chenzhou-4":  api.RegionChengzhou,
	"cn-jiangxi-2":   api.RegionJiangxi,
	"cn-hunan-3":     api.RegionChangsha,
	"cn-guangzhou-5": api.RegionGuangzhou,
	"cn-chongqing-2": api.RegionChongqing,
	"cn-shanghai-7":  api.RegionShanghai,
	"cn-shanxi-2":    api.RegionXian,
	"cn-guiyang-1":   api.RegionGuiyang,
	"cn-haerbin-2":   api.RegionHaerbin,
	"cn-zhengzhou-2": api.RegionZhengzhou,
	"cn-xian-4":      api.RegionXian,
	"cn-xian-5":      api.RegionXian,
	"cn-lanzhou-3":   api.RegionLanzhou,
	"cn-nanjing-2":   api.RegionNanjing,
}
