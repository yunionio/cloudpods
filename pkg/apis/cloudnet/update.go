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

package cloudnet

import "yunion.io/x/onecloud/pkg/apis"

type RouterUpdateInput struct {
	apis.StandaloneResourceBaseUpdateInput

	User string `json:"user"`

	Host string `json:"host"`

	Port *int `json:"port"`

	PrivateKey string `json:"private_key"`

	RealizeWgIfaces *bool `json:"realize_wg_ifaces"`

	RealizeRoutes *bool `json:"realize_routes"`

	RealizeRules *bool `json:"realize_rules"`

	OldEndpoint string `json:"_old_endpoint"`
}

type RouteUpdateInput struct {
	apis.StandaloneResourceBaseUpdateInput

	Network string `json:"network"`

	Gateway string `json:"gateway"`
}

type RuleUpdateInput struct {
	apis.StandaloneResourceBaseUpdateInput

	Prio int `json:"prio"`

	MatchSrcNet string `json:"match_src_net"`

	MatchDestNet string `json:"match_dest_net"`

	MatchProto string `json:"match_proto"`

	MatchSrcPort int `json:"match_src_port"`

	MatchDestPort int `json:"match_dest_port"`

	MatchInIfname string `json:"match_in_ifname"`

	MatchOutIfname string `json:"match_out_ifname"`

	Action string `json:"action"`

	ActionOptions string `json:"action_options"`
}
