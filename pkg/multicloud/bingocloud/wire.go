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

package bingocloud

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SWire struct {
	multicloud.SResourceBase
	multicloud.STagBase

	vpc *SVpc
}

func (self *SWire) GetName() string {
	return self.vpc.GetName()
}

func (self *SWire) GetId() string {
	return self.vpc.GetId()
}

func (self *SWire) GetGlobalId() string {
	return self.vpc.GetGlobalId()
}

func (self *SWire) IsEmulated() bool {
	return true
}

func (self *SWire) GetBandwidth() int {
	return 10000
}

func (self *SWire) GetStatus() string {
	return api.WIRE_STATUS_AVAILABLE
}
