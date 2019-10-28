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

package aws

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

// https://docs.aws.amazon.com/general/latest/gr/rande.html

var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"us-east-2":      {Latitude: 40.4172871, Longitude: -82.90712300000001, City: api.CITY_OHIO, CountryCode: api.COUNTRY_CODE_US},
	"us-east-1":      {Latitude: 37.4315734, Longitude: -78.6568942, City: api.CITY_N_VIRGINIA, CountryCode: api.COUNTRY_CODE_US},
	"us-west-1":      {Latitude: 38.8375215, Longitude: -120.8958242, City: api.CITY_N_CALIFORNIA, CountryCode: api.COUNTRY_CODE_US},
	"us-west-2":      {Latitude: 43.8041334, Longitude: -120.5542012, City: api.CITY_OREGON, CountryCode: api.COUNTRY_CODE_US},
	"ap-south-1":     {Latitude: 19.0759837, Longitude: 72.8776559, City: api.CITY_MUMBAI, CountryCode: api.COUNTRY_CODE_IN},
	"ap-northeast-3": {Latitude: 34.6937378, Longitude: 135.5021651, City: api.CITY_OSAKA, CountryCode: api.COUNTRY_CODE_JP},
	"ap-northeast-2": {Latitude: 37.566535, Longitude: 126.9779692, City: api.CITY_SEOUL, CountryCode: api.COUNTRY_CODE_KR},
	"ap-southeast-1": {Latitude: 1.352083, Longitude: 103.819836, City: api.CITY_SINGAPORE, CountryCode: api.COUNTRY_CODE_SG},
	"ap-southeast-2": {Latitude: -33.8688197, Longitude: 151.2092955, City: api.CITY_SYDNEY, CountryCode: api.COUNTRY_CODE_AU},
	"ap-northeast-1": {Latitude: 35.7090259, Longitude: 139.7319925, City: api.CITY_TOKYO, CountryCode: api.COUNTRY_CODE_JP},
	"ca-central-1":   {Latitude: 56.130366, Longitude: -106.346771, City: api.CITY_CANADA_CENTRAL, CountryCode: api.COUNTRY_CODE_CA},
	"cn-north-1":     {Latitude: 39.90419989999999, Longitude: 116.4073963, City: api.CITY_BEI_JING, CountryCode: api.COUNTRY_CODE_CN},
	"cn-northwest-1": {Latitude: 37.198731, Longitude: 106.1580937, City: api.CITY_NING_XIA, CountryCode: api.COUNTRY_CODE_CN},
	"eu-central-1":   {Latitude: 50.1109221, Longitude: 8.6821267, City: api.CITY_FRANKFURT, CountryCode: api.COUNTRY_CODE_DE},
	"eu-west-1":      {Latitude: 53.41291, Longitude: -8.24389, City: api.CITY_IRELAND, CountryCode: api.COUNTRY_CODE_IE},
	"eu-west-2":      {Latitude: 51.5073509, Longitude: -0.1277583, City: api.CITY_LONDON, CountryCode: api.COUNTRY_CODE_GB},
	"eu-west-3":      {Latitude: 48.856614, Longitude: 2.3522219, City: api.CITY_PARIS, CountryCode: api.COUNTRY_CODE_FR},
	"eu-north-1":     {Latitude: 59.1946, Longitude: 18.47, City: api.CITY_STOCKHOLM, CountryCode: api.COUNTRY_CODE_SE},
	"sa-east-1":      {Latitude: -23.5505199, Longitude: -46.63330939999999, City: api.CITY_SAO_PAULO, CountryCode: api.COUNTRY_CODE_BR},
	"us-gov-west-1":  {Latitude: 37.09024, Longitude: -95.712891, City: api.CITY_US_GOV_WEST, CountryCode: api.COUNTRY_CODE_US},
}
