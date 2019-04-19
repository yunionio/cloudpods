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

package compute

import (
	"yunion.io/x/onecloud/pkg/apis"
)

type HostSpec struct {
	apis.Meta

	Cpu         int            `json:"cpu"`
	Mem         int            `json:"mem"`
	NicCount    int            `json:"nic_count"`
	Manufacture string         `json:"manufacture"`
	Model       string         `json:"model"`
	Disk        DiskDriverSpec `json:"disk"`
	Driver      string         `json:"driver"`
}

type DiskDriverSpec map[string]DiskAdapterSpec

type DiskAdapterSpec map[string][]*DiskSpec

type DiskSpec struct {
	apis.Meta

	Type       string `json:"type"`
	Size       int64  `json:"size"`
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
	Count      int    `json:"count"`
}
