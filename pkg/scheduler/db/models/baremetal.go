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
	"github.com/jinzhu/gorm"
)

type Baremetal struct {
	StandaloneModel
	Status    string `json:"status" gorm:"not null"`
	Enabled   bool   `json:"enabled" gorm:"not null"`
	CliGUID   string `json:"cli_guid,omitempty"`
	AgentID   string `json:"agent_id,omitempty"`
	CPUCount  int    `json:"cpu_count,omitempty"`
	NodeCount int    `json:"node_count,omitempty"`
	CPUDesc   string `json:"cpu_desc,omitempty"`
	CPUMHZ    int    `json:"cpu_mhz,omitempty"`
	MemSize   int    `json:"mem_size,omitempty"`

	StorageSize   int    `json:"storage_size,omitempty"`
	StorageType   string `json:"storage_type,omitempty"`
	StorageDriver string `json:"storage_driver,omitempty"`
	StorageInfo   string `json:"storage_info,omitempty"`

	IpmiInfo string `json:"ipmi_info,omitempty" gorm:"type:text"`

	Rack  string `json:"rack,omitempty"`
	Slots string `json:"slots,omitempty"`

	ServerID string `json:"server_id,omitempty"`
	UseCount int    `json:"use_count,omitempty"`

	PoolID string `json:"pool_id,omitempty"`
}

func (b Baremetal) TableName() string {
	return baremetalsTable
}

func (b Baremetal) String() string {
	str, _ := JsonString(b)
	return str
}

func NewBaremetalResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &Baremetal{}
	}
	models := func() interface{} {
		baremetals := []Baremetal{}
		return &baremetals
	}

	return newResource(db, baremetalsTable, model, models)
}
