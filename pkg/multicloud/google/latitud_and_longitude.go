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

package google

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"asia-east1":           {Latitude: 25.0443, Longitude: 121.509, City: api.CITY_TAI_WAN, CountryCode: api.COUNTRY_CODE_CN},
	"asia-east2":           {Latitude: 22.396427, Longitude: 114.109497, City: api.CITY_HONG_KONG, CountryCode: api.COUNTRY_CODE_CN},
	"asia-northeast1":      {Latitude: 35.709026, Longitude: 139.731995, City: api.CITY_TOKYO, CountryCode: api.COUNTRY_CODE_JP},
	"asia-northeast2":      {Latitude: 34.6937378, Longitude: 135.5021651, City: api.CITY_OSAKA, CountryCode: api.COUNTRY_CODE_JP},
	"asia-south1":          {Latitude: 19.075983, Longitude: 72.877655, City: api.CITY_MUMBAI, CountryCode: api.COUNTRY_CODE_IN},
	"asia-southeast1":      {Latitude: 1.352083, Longitude: 103.819839, City: api.CITY_SINGAPORE, CountryCode: api.COUNTRY_CODE_SG},
	"australia-southeast1": {Latitude: -33.8688197, Longitude: 151.2092955, City: api.CITY_SYDNEY, CountryCode: api.COUNTRY_CODE_AU},

	"europe-north1": {Latitude: 39.904202, Longitude: 116.407394, City: api.CITY_FINLAND, CountryCode: api.COUNTRY_CODE_CN},
	"europe-west1":  {Latitude: 39.904202, Longitude: 116.407394, City: api.CITY_BELGIUM, CountryCode: api.COUNTRY_CODE_CN},
	"europe-west2":  {Latitude: 51.507351, Longitude: -0.127758, City: api.CITY_LONDON, CountryCode: api.COUNTRY_CODE_GB},
	"europe-west3":  {Latitude: 51.165691, Longitude: 10.451526, City: api.CITY_FRANKFURT, CountryCode: api.COUNTRY_CODE_DE},
	"europe-west4":  {Latitude: 52.2076831, Longitude: 4.1585786, City: api.CITY_HOLLAND, CountryCode: api.COUNTRY_CODE_NL},
	"europe-west6":  {Latitude: 47.3774497, Longitude: 8.5016958, City: api.CITY_ZURICH, CountryCode: api.COUNTRY_CODE_CH},

	"northamerica-northeast1": {Latitude: 45.5580206, Longitude: -73.8003414, City: api.CITY_MONTREAL, CountryCode: api.COUNTRY_CODE_CA},
	"southamerica-east1":      {Latitude: -23.5505199, Longitude: -46.6333094, City: api.CITY_SAO_PAULO, CountryCode: api.COUNTRY_CODE_BR},
	"us-central1":             {Latitude: 41.9328655, Longitude: -94.5106809, City: api.CITY_IOWA, CountryCode: api.COUNTRY_CODE_US},
	"us-east1":                {Latitude: 33.6194409, Longitude: -82.0475635, City: api.CITY_SOUTH_CAROLINA, CountryCode: api.COUNTRY_CODE_US},
	"us-east4":                {Latitude: 37.4315734, Longitude: -78.6568942, City: api.CITY_N_VIRGINIA, CountryCode: api.COUNTRY_CODE_US},
	"us-west1":                {Latitude: 43.8041334, Longitude: -120.5542012, City: api.CITY_OREGON, CountryCode: api.COUNTRY_CODE_US},
	"us-west2":                {Latitude: 34.0522342, Longitude: -118.2436849, City: api.CITY_LOS_ANGELES, CountryCode: api.COUNTRY_CODE_US},
}

var RegionNames = map[string]string{
	"asia-east1":           "台湾",
	"asia-east2":           "香港",
	"asia-northeast1":      "东京",
	"asia-northeast2":      "大阪",
	"asia-south1":          "孟买",
	"asia-southeast1":      "新加坡",
	"australia-southeast1": "悉尼",

	"europe-north1": "芬兰",
	"europe-west1":  "比利时",
	"europe-west2":  "伦敦",
	"europe-west3":  "法兰克福",
	"europe-west4":  "荷兰",
	"europe-west6":  "苏黎世",

	"northamerica-northeast1": "蒙特利尔",
	"southamerica-east1":      "圣保罗",
	"us-central1":             "艾奥瓦",
	"us-east1":                "南卡罗来纳州",
	"us-east4":                "北弗吉尼亚",
	"us-west1":                "俄勒冈州",
	"us-west2":                "洛杉矶",
}
