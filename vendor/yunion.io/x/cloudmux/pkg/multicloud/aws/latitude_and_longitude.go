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
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

// https://docs.aws.amazon.com/general/latest/gr/rande.html

var LatitudeAndLongitude = map[string]cloudprovider.SGeographicInfo{
	"us-east-1":      api.RegionNothVirginia,
	"us-east-2":      api.RegionOhio,
	"us-west-1":      api.RegionNorthCalifornia,
	"us-west-2":      api.RegionOregon,
	"ap-south-1":     api.RegionMumbai,
	"ap-northeast-3": api.RegionOsaka,
	"ap-northeast-2": api.RegionSeoul,
	"ap-southeast-1": api.RegionSingapore,
	"ap-southeast-2": api.RegionSydney,
	"ap-northeast-1": api.RegionTokyo,
	"ap-east-1":      api.RegionHongkong,
	"ca-central-1":   api.RegionCanadaCentral,
	"cn-north-1":     api.RegionBeijing,
	"cn-northwest-1": api.RegionNingxia,
	"eu-central-1":   api.RegionFrankfurt,
	"eu-west-1":      api.RegionIreland,
	"eu-west-2":      api.RegionLondon,
	"eu-west-3":      api.RegionParis,
	"eu-north-1":     api.RegionStockholm,
	"sa-east-1":      api.RegionSaoPaulo,
	"us-gov-west-1":  api.RegionUSGOVWest,

	"af-south-1": api.RegionCapeTown,
	"me-south-1": api.RegionBahrain,

	"eu-south-1":     api.RegionMilan,
	"ap-southeast-4": api.RegionMelbourne,
	"ap-south-2":     api.RegionHyderabad,
	"eu-south-2":     api.RegionMadrid,
	"eu-central-2":   api.RegionAbuDhabi,
	"ap-southeast-3": api.RegionJakarta,
	"me-central-1":   api.RegionDubai,
	"il-central-1":   api.RegionTelAviv,
}
