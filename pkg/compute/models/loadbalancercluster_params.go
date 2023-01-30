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

package models

import (
	"yunion.io/x/jsonutils"
)

type SLoadbalancerClusterParams struct {
	VirtualRouterId   int `json:",omitzero"`
	Preempt           bool
	AdvertInt         int `json:",omitzero"`
	Pass              string
	GarpMasterRefresh int `json:",omitzero"`
}

func (p *SLoadbalancerClusterParams) String() string {
	return jsonutils.Marshal(p).String()
}

func (p *SLoadbalancerClusterParams) IsZero() bool {
	if *p == (SLoadbalancerClusterParams{}) {
		return true
	}
	return false
}
