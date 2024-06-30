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

package baremetal

import (
	"strings"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
)

const (
	DefaultBaremetalProfileId = "default"
)

type BaremetalProfileListInput struct {
	apis.StandaloneAnonResourceListInput

	OemName []string `json:"oem_name"`
	Model   []string `json:"model"`
}

func (input *BaremetalProfileListInput) Normalize() {
	for i := range input.OemName {
		input.OemName[i] = strings.TrimSpace(input.OemName[i])
	}
	for i := range input.Model {
		input.Model[i] = strings.TrimSpace(input.Model[i])
	}
}

type BaremetalProfileCreateInput struct {
	apis.StandaloneAnonResourceCreateInput

	OemName    string
	Model      string
	LanChannel uint8
	RootId     int
	RootName   string
	StrongPass bool
}

type BaremetalProfileUpdateInput struct {
	apis.StandaloneAnonResourceBaseUpdateInput

	LanChannel uint8
	RootId     *int
	RootName   string
	StrongPass *bool
}

type BaremetalProfileDetails struct {
	SBaremetalProfile
}

func (detail BaremetalProfileDetails) ToSpec() BaremetalProfileSpec {
	channels := make([]uint8, 0)
	if detail.LanChannel > 0 {
		channels = append(channels, detail.LanChannel)
	}
	if detail.LanChannel2 > 0 {
		channels = append(channels, detail.LanChannel2)
	}
	if detail.LanChannel3 > 0 {
		channels = append(channels, detail.LanChannel3)
	}
	return BaremetalProfileSpec{
		OemName:     detail.OemName,
		Model:       detail.Model,
		LanChannels: channels,
		RootName:    detail.RootName,
		RootId:      detail.RootId,
		StrongPass:  detail.StrongPass,
	}
}

type BaremetalProfileSpec struct {
	OemName     string
	Model       string
	LanChannels []uint8
	RootName    string
	RootId      int
	StrongPass  bool
}

type BaremetalProfileSpecs []BaremetalProfileSpec

func (a BaremetalProfileSpecs) Len() int      { return len(a) }
func (a BaremetalProfileSpecs) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a BaremetalProfileSpecs) Less(i, j int) bool {
	if a[i].OemName != a[j].OemName {
		return a[i].OemName < a[j].OemName
	}
	if a[i].Model != a[j].Model {
		return a[i].Model < a[j].Model
	}
	return false
}

var PredefinedProfiles = []BaremetalProfileSpec{
	{
		OemName:     "",
		LanChannels: []uint8{1, 2, 8},
		RootName:    "root",
		RootId:      2,
	},
	{
		OemName:     types.OEM_NAME_INSPUR,
		LanChannels: []uint8{8, 1},
		RootName:    "admin",
		RootId:      2,
	},
	{
		OemName:     types.OEM_NAME_LENOVO,
		LanChannels: []uint8{1, 8},
		RootName:    "root",
		RootId:      2,
	},
	{
		OemName:     types.OEM_NAME_HP,
		LanChannels: []uint8{1, 2},
		RootName:    "root",
		RootId:      1,
	},
	{
		OemName:     types.OEM_NAME_HUAWEI,
		LanChannels: []uint8{1},
		RootName:    "root",
		RootId:      2,
		StrongPass:  true,
	},
	{
		OemName:     types.OEM_NAME_FOXCONN,
		LanChannels: []uint8{1},
		RootName:    "root",
		RootId:      2,
		StrongPass:  true,
	},
	{
		OemName:     types.OEM_NAME_QEMU,
		LanChannels: []uint8{8, 1},
		RootName:    "root",
		RootId:      2,
		StrongPass:  true,
	},
	{
		OemName:     types.OEM_NAME_H3C,
		LanChannels: []uint8{1},
		RootName:    "root",
		RootId:      2,
		StrongPass:  true,
	},
}
