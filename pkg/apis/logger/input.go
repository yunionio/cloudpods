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

package logger

import (
	"time"

	"yunion.io/x/onecloud/pkg/apis"
)

type BaremetalEventListInput struct {
	apis.ModelBaseListInput

	// since
	Since time.Time `json:"since"`
	// until
	Until time.Time `json:"until"`
	// host_id
	HostId []string `json:"host_id"`
	// id
	Id []int64 `json:"id"`
	// EventId
	EventId []string `json:"event_id"`
	// Type
	Type []string `json:"type"`
	// ipmi_ip
	IpmiIp []string `json:"ipmi_ip"`
}

type ActionLogListInput struct {
	apis.OpsLogListInput

	Service []string `json:"service"`

	Success *bool `json:"success"`

	Ip []string `json:"ip"`

	Severity []string `json:"severity"`

	Kind []string `json:"kind"`
}
