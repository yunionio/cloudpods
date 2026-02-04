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

package guest

import (
	"time"

	"yunion.io/x/onecloud/pkg/apis"
	compute_api "yunion.io/x/onecloud/pkg/apis/compute"
)

type GuestNetworkTrafficState string

const (
	GuestNetworkTrafficStateContinue GuestNetworkTrafficState = ""
	GuestNetworkTrafficStateStart    GuestNetworkTrafficState = "start"
)

type GuestNetworkTrafficLogListInput struct {
	apis.ProjectizedResourceListInput
	compute_api.ServerFilterListInput
	compute_api.NetworkFilterListInput

	Since time.Time `json:"since"`
	Until time.Time `json:"until"`

	IpAddr  []string `json:"ip_addr"`
	Ip6Addr []string `json:"ip6_addr"`

	State []GuestNetworkTrafficState `json:"state"`
}

type GuestNetworkTrafficLogDetails struct {
	compute_api.SGuestNetworkTrafficLog
}
