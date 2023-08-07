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
	"guangzhou-2": {Latitude: 23.129110, Longitude: 113.264381, City: api.CITY_GUANG_ZHOU, CountryCode: api.COUNTRY_CODE_CN},
	"beijing-1":   {Latitude: 39.904202, Longitude: 116.407394, City: api.CITY_BEI_JING, CountryCode: api.COUNTRY_CODE_CN},
	"hunan-1":     {Latitude: 28.2277765095, Longitude: 112.9388453666, City: api.CITY_CHANG_SHA, CountryCode: api.COUNTRY_CODE_CN},
	"wuxi-1":      {Latitude: 31.2983479333, Longitude: 120.5831894861, City: api.CITY_SU_ZHOU, CountryCode: api.COUNTRY_CODE_CN},
	"dongguan-1":  api.RegionSuzhou,
	"yaan-1":      {Latitude: 30.572815, Longitude: 104.066803, City: api.CITY_CHENG_DU, CountryCode: api.COUNTRY_CODE_CN},
	"zhengzhou-1": {Latitude: 34.7533581487, Longitude: 113.6313915479, City: api.CITY_ZHENG_ZHOU, CountryCode: api.COUNTRY_CODE_CN},
	"beijing-2":   {Latitude: 39.904202, Longitude: 116.407394, City: api.CITY_BEI_JING, CountryCode: api.COUNTRY_CODE_CN},
	"zhuzhou-1":   {Latitude: 28.2277765095, Longitude: 112.9388453666, City: api.CITY_CHANG_SHA, CountryCode: api.COUNTRY_CODE_CN},
	"jinan-1":     {Latitude: 36.64889911073425, Longitude: 117.11905617575435, City: api.CITY_JI_NAM, CountryCode: api.COUNTRY_CODE_CN},
	"xian-1":      {Latitude: 34.3412614674, Longitude: 108.9398165260, City: api.CITY_XI_AN, CountryCode: api.COUNTRY_CODE_CN},
	"shanghai-1":  {Latitude: 31.210344, Longitude: 121.455364, City: api.CITY_SHANG_HAI, CountryCode: api.COUNTRY_CODE_CN},
	"chongqing-1": {Latitude: 29.431585, Longitude: 106.912254, City: api.CITY_CHONG_QING, CountryCode: api.COUNTRY_CODE_CN},
	"ningbo-1":    {Latitude: 30.274084, Longitude: 120.155067, City: api.CITY_HANG_ZHOU, CountryCode: api.COUNTRY_CODE_CN},
	"tianjin-1":   {Latitude: 39.0850853357, Longitude: 117.1993482089, City: api.CITY_TIAN_JIN, CountryCode: api.COUNTRY_CODE_CN},
	"jilin-1":     {Latitude: 43.87120919729674, Longitude: 125.3111129463539, City: api.CITY_CHANG_CHUN, CountryCode: api.COUNTRY_CODE_CN},
	"hubei-1":     {Latitude: 32.009075721852206, Longitude: 112.13485327119795, City: api.CITY_XIANG_YANG, CountryCode: api.COUNTRY_CODE_CN},
	"jiangxi-1":   {Latitude: 28.66278309381472, Longitude: 115.82816199879247, City: api.CITY_NAN_CHANG, CountryCode: api.COUNTRY_CODE_CN},
	"gansu-1":     {Latitude: 36.0613769373, Longitude: 103.8341600069, City: api.CITY_LAN_ZHOU, CountryCode: api.COUNTRY_CODE_CN},
	"shanxi-1":    {Latitude: 37.8705857132, Longitude: 112.5506634865, City: api.CITY_TAI_YUAN, CountryCode: api.COUNTRY_CODE_CN},
	"liaoning-1":  {Latitude: 41.78937667917192, Longitude: 123.43099727316815, City: api.CITY_SHEN_YANG, CountryCode: api.COUNTRY_CODE_CN},
	"yunnan-2":    {Latitude: 24.8796595146, Longitude: 102.8332118852, City: api.CITY_KUN_MING, CountryCode: api.COUNTRY_CODE_CN},
	"hebei-1":     {Latitude: 38.044044256466684, Longitude: 114.50225031469532, City: api.CITY_SHI_JIA_ZHUANG, CountryCode: api.COUNTRY_CODE_CN},
	"fujian-1":    {Latitude: 24.478556505708365, Longitude: 118.0875755539503, City: api.CITY_XIA_MEN, CountryCode: api.COUNTRY_CODE_CN},
	"guangxi-1":   {Latitude: 22.8167372565, Longitude: 108.3669005333, City: api.CITY_NAN_NING, CountryCode: api.COUNTRY_CODE_CN},
	"anhui-1":     {Latitude: 32.62657438299575, Longitude: 116.99779954519057, City: api.CITY_HUAI_NAN, CountryCode: api.COUNTRY_CODE_CN},
	"huhehaote-1": {Latitude: 40.842358, Longitude: 111.749992, City: api.CITY_HU_HE_HAO_TE, CountryCode: api.COUNTRY_CODE_CN},
	"guiyang-1":   {Latitude: 26.6470035286, Longitude: 106.6302113880, City: api.CITY_GUI_YANG, CountryCode: api.COUNTRY_CODE_CN},
}
