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

package apsara

import (
	"time"
)

// "CreationTime":"2017-03-19T13:37:40Z","Description":"","RegionId":"cn-hongkong","RouteTableIds":{"RouteTableId":["vtb-j6c60lectdi80rk5xz43g"]},"VRouterId":"vrt-j6c00qrol733dg36iq4qj","VRouterName":"","VpcId":"vpc-j6c86z3sh8ufhgsxwme0q"

type SRouteTableIds struct {
	RouteTableId []string
}

type SVRouter struct {
	CreationTime  time.Time
	Description   string
	RegionId      string
	RouteTableIds SRouteTableIds
	VRouterId     string
	VRouterName   string
	VpcId         string
}
