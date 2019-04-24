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

package esxi

import (
	"yunion.io/x/pkg/tristate"
)

type SHostNicInfo struct {
	Dev     string
	Driver  string
	Mac     string
	Index   int8
	LinkUp  bool
	IpAddr  string
	Mtu     int16
	NicType string
}

func (nic *SHostNicInfo) GetDevice() string {
	return nic.Dev
}

func (nic *SHostNicInfo) GetDriver() string {
	return nic.Driver
}

func (nic *SHostNicInfo) GetMac() string {
	return nic.Mac
}

func (nic *SHostNicInfo) GetIndex() int8 {
	return nic.Index
}

func (nic *SHostNicInfo) IsLinkUp() tristate.TriState {
	if nic.LinkUp {
		return tristate.True
	}
	return tristate.False
}

func (nic *SHostNicInfo) GetIpAddr() string {
	return nic.IpAddr
}

func (nic *SHostNicInfo) GetMtu() int16 {
	return nic.Mtu
}

func (nic *SHostNicInfo) GetNicType() string {
	return nic.NicType
}
