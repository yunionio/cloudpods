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
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

// https://docs.ucloud.cn/api/summary/regionlist
var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"cn-bj1":       {Latitude: 39.9041999, Longitude: 116.4073963, City: api.CITY_BEI_JING, CountryCode: api.COUNTRY_CODE_CN},
	"cn-bj2":       {Latitude: 39.9041999, Longitude: 116.4073963, City: api.CITY_BEI_JING, CountryCode: api.COUNTRY_CODE_CN},
	"cn-sh":        {Latitude: 31.2303904, Longitude: 121.4737021, City: api.CITY_SHANG_HAI, CountryCode: api.COUNTRY_CODE_CN},
	"cn-sh2":       {Latitude: 31.2303904, Longitude: 121.4737021, City: api.CITY_SHANG_HAI, CountryCode: api.COUNTRY_CODE_CN},
	"cn-gd":        {Latitude: 23.12911, Longitude: 113.264385, City: api.CITY_GUANG_ZHOU, CountryCode: api.COUNTRY_CODE_CN},
	"hk":           {Latitude: 22.396428, Longitude: 114.109497, City: api.CITY_HONG_KONG, CountryCode: api.COUNTRY_CODE_CN},
	"us-ca":        {Latitude: 34.0522342, Longitude: -118.2436849, City: api.CITY_LOS_ANGELES, CountryCode: api.COUNTRY_CODE_US},
	"us-ws":        {Latitude: 38.9071923, Longitude: -77.0368707, City: api.CITY_WASHINGTON, CountryCode: api.COUNTRY_CODE_US},
	"ge-fra":       {Latitude: 50.1109221, Longitude: 8.6821267, City: api.CITY_FRANKFURT, CountryCode: api.COUNTRY_CODE_DE},
	"th-bkk":       {Latitude: 13.7563309, Longitude: 100.5017651, City: api.CITY_BANGKOK, CountryCode: api.COUNTRY_CODE_TH},
	"kr-seoul":     {Latitude: 37.566535, Longitude: 126.9779692, City: api.CITY_SEOUL, CountryCode: api.COUNTRY_CODE_KR},
	"sg":           {Latitude: 1.352083, Longitude: 103.819836, City: api.CITY_SINGAPORE, CountryCode: api.COUNTRY_CODE_SG},
	"tw-tp":        {Latitude: 25.0329694, Longitude: 121.5654177, City: api.CITY_TAIPEI, CountryCode: api.COUNTRY_CODE_CN},
	"tw-kh":        {Latitude: 22.6272784, Longitude: 120.3014353, City: api.CITY_KAOHSIUNG, CountryCode: api.COUNTRY_CODE_CN},
	"jpn-tky":      {Latitude: 35.7090259, Longitude: 139.7319925, City: api.CITY_TOKYO, CountryCode: api.COUNTRY_CODE_JP},
	"rus-mosc":     {Latitude: 55.755826, Longitude: 37.6172999, City: api.CITY_MOSCOW, CountryCode: api.COUNTRY_CODE_RU},
	"uae-dubai":    {Latitude: 25.2048493, Longitude: 55.2707828, City: api.CITY_DUBAI, CountryCode: api.COUNTRY_CODE_AE},
	"idn-jakarta":  {Latitude: -6.2087634, Longitude: 106.845599, City: api.CITY_JAKARTA, CountryCode: api.COUNTRY_CODE_ID},
	"ind-mumbai":   {Latitude: 19.0759837, Longitude: 72.8776559, City: api.CITY_MUMBAI, CountryCode: api.COUNTRY_CODE_IN},
	"bra-saopaulo": {Latitude: -23.5505199, Longitude: -46.6333094, City: api.CITY_SAO_PAULO, CountryCode: api.COUNTRY_CODE_BR},
	"uk-london":    {Latitude: 51.5073509, Longitude: -0.1277583, City: api.CITY_LONDON, CountryCode: api.COUNTRY_CODE_GB},
	"afr-nigeria":  {Latitude: 6.5243793, Longitude: 3.3792057, City: api.CITY_LAGOS, CountryCode: api.COUNTRY_CODE_NG},
	"vn-sng":       {Latitude: 10.8230989, Longitude: 106.6296638, City: api.CITY_HO_CHI_MINH, CountryCode: api.COUNTRY_CODE_VN},
	"cn-qz":        {Latitude: 24.9037185, Longitude: 118.5134676, City: api.CITY_QUAN_ZHOU, CountryCode: api.COUNTRY_CODE_CN},
}
