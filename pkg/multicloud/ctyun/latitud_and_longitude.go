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
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"cn-beijing1":   {Latitude: 39.997743, Longitude: 116.304542, City: api.CITY_BEI_JING, CountryCode: api.COUNTRY_CODE_CN},
	"cn-hz1":        {Latitude: 30.274084, Longitude: 120.155067, City: api.CITY_HANG_ZHOU, CountryCode: api.COUNTRY_CODE_CN},
	"cn-gslz1":      {Latitude: 36.0613769373, Longitude: 103.8341600069, City: api.CITY_LAN_ZHOU, CountryCode: api.COUNTRY_CODE_CN},
	"cn-sxty1":      {Latitude: 37.8705857132, Longitude: 112.5506634865, City: api.CITY_TAI_YUAN, CountryCode: api.COUNTRY_CODE_CN},
	"cn-sh1":        {Latitude: 31.210344, Longitude: 121.455364, City: api.CITY_SHANG_HAI, CountryCode: api.COUNTRY_CODE_CN},
	"cn-gz1":        {Latitude: 26.6470035286, Longitude: 106.6302113880, City: api.CITY_GUI_YANG, CountryCode: api.COUNTRY_CODE_CN},
	"cn-sdqd1":      {Latitude: 36.067108, Longitude: 120.382607, City: api.CITY_QING_DAO, CountryCode: api.COUNTRY_CODE_CN},
	"cn-tj1":        {Latitude: 39.0850853357, Longitude: 117.1993482089, City: api.CITY_TIAN_JIN, CountryCode: api.COUNTRY_CODE_CN},
	"cn-xjcj1":      {Latitude: 43.8266013700, Longitude: 87.6168405804, City: api.CITY_WU_LU_MU_QI, CountryCode: api.COUNTRY_CODE_CN},
	"cn-cq1":        {Latitude: 29.431585, Longitude: 106.912254, City: api.CITY_CHONG_QING, CountryCode: api.COUNTRY_CODE_CN},
	"cn-gxnn1":      {Latitude: 22.8167372565, Longitude: 108.3669005333, City: api.CITY_NAN_NING, CountryCode: api.COUNTRY_CODE_CN},
	"cn-hazz1":      {Latitude: 34.7533581487, Longitude: 113.6313915479, City: api.CITY_ZHENG_ZHOU, CountryCode: api.COUNTRY_CODE_CN},
	"cn-ynkm1":      {Latitude: 24.8796595146, Longitude: 102.8332118852, City: api.CITY_KUN_MING, CountryCode: api.COUNTRY_CODE_CN},
	"cn-xian1":      {Latitude: 34.3412614674, Longitude: 108.9398165260, City: api.CITY_XI_AN, CountryCode: api.COUNTRY_CODE_CN},
	"cn-hihk1":      {Latitude: 20.0442268036, Longitude: 110.1998910288, City: api.CITY_HAI_KOU, CountryCode: api.COUNTRY_CODE_CN},
	"cn-ahwh1":      {Latitude: 31.3524675159, Longitude: 118.4331307290, City: api.CITY_WU_HU, CountryCode: api.COUNTRY_CODE_CN},
	"cn-fz1":        {Latitude: 26.0741979397, Longitude: 119.2964466153, City: api.CITY_FU_ZHOU, CountryCode: api.COUNTRY_CODE_CN},
	"cn-nmhh1":      {Latitude: 40.842358, Longitude: 111.749992, City: api.CITY_HU_HE_HAO_TE, CountryCode: api.COUNTRY_CODE_CN},
	"cn-shanghai2":  {Latitude: 31.210344, Longitude: 121.455364, City: api.CITY_SHANG_HAI, CountryCode: api.COUNTRY_CODE_CN},
	"cn-snxy1":      {Latitude: 34.3412614674, Longitude: 108.9398165260, City: api.CITY_XI_AN, CountryCode: api.COUNTRY_CODE_CN},
	"cn-hbwh1":      {Latitude: 30.5927599029, Longitude: 114.3052387810, City: api.CITY_WU_HAN, CountryCode: api.COUNTRY_CODE_CN},
	"cn-hncs1":      {Latitude: 28.2277765095, Longitude: 112.9388453666, City: api.CITY_CHANG_SHA, CountryCode: api.COUNTRY_CODE_CN},
	"cn-guangzhou2": {Latitude: 23.12911, Longitude: 113.264385, City: api.CITY_GUANG_ZHOU, CountryCode: api.COUNTRY_CODE_CN},
	"cn-guizhou2":   {Latitude: 26.6470035286, Longitude: 106.6302113880, City: api.CITY_GUI_YANG, CountryCode: api.COUNTRY_CODE_CN},
	"cn-jssz1":      {Latitude: 31.2983479333, Longitude: 120.5831894861, City: api.CITY_SHU_ZHOU, CountryCode: api.COUNTRY_CODE_CN},
	"cn-sccd1":      {Latitude: 30.572815, Longitude: 104.066803, City: api.CITY_CHENG_DU, CountryCode: api.COUNTRY_CODE_CN},
	"cn-guangzhou3": {Latitude: 23.12911, Longitude: 113.264385, City: api.CITY_GUANG_ZHOU, CountryCode: api.COUNTRY_CODE_CN},
	"cn-shanghai3":  {Latitude: 31.210344, Longitude: 121.455364, City: api.CITY_SHANG_HAI, CountryCode: api.COUNTRY_CODE_CN},
	"cn-neimeng4":   {Latitude: 40.842358, Longitude: 111.749992, City: api.CITY_HU_HE_HAO_TE, CountryCode: api.COUNTRY_CODE_CN},
	"cn-beijing3":   {Latitude: 39.904202, Longitude: 116.407394, City: api.CITY_BEI_JING, CountryCode: api.COUNTRY_CODE_CN},
	"cn-baoding1":   {Latitude: 38.8739745619, Longitude: 115.4646082830, City: api.CITY_BAO_DING, CountryCode: api.COUNTRY_CODE_CN},
	"cn-nj2":        {Latitude: 32.0584065670, Longitude: 118.7964897811, City: api.CITY_NAN_JING, CountryCode: api.COUNTRY_CODE_CN},
	"cn-gdgz1":      {Latitude: 23.12911, Longitude: 113.264385, City: api.CITY_GUANG_ZHOU, CountryCode: api.COUNTRY_CODE_CN},
	"cn-bj4":        {Latitude: 39.904202, Longitude: 116.407394, City: api.CITY_BEI_JING, CountryCode: api.COUNTRY_CODE_CN},
	"cn-neimeng5":   {Latitude: 40.842358, Longitude: 111.749992, City: api.CITY_HU_HE_HAO_TE, CountryCode: api.COUNTRY_CODE_CN},
	"cn-shanghai5":  {Latitude: 31.210344, Longitude: 121.455364, City: api.CITY_SHANG_HAI, CountryCode: api.COUNTRY_CODE_CN},
	"cn-sh6":        {Latitude: 31.210344, Longitude: 121.455364, City: api.CITY_SHANG_HAI, CountryCode: api.COUNTRY_CODE_CN},
	"cn-gdfs2":      {Latitude: 23.0218629843, Longitude: 113.1219225896, City: api.CITY_FO_SHAN, CountryCode: api.COUNTRY_CODE_CN},
}
