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
	"fmt"

	"github.com/jinzhu/gorm"
)

const (
	HostWireResourceName = "hostwires"
)

type HostWire struct {
	StandaloneModel
	Bridge    string `json:"bridge,omitempty" gorm:"not null"`
	Interface string `json:"interface,omitempty" gorm:"not null"`
	HostID    string `json:"host_id,omitempty" gorm:"not null"`
	WireID    string `json:"wire_id,omitempty" gorm:"not null"`
}

func (w HostWire) TableName() string {
	return hostWiresTable
}

func (w HostWire) String() string {
	str, _ := JsonString(w)
	return str
}

func NewHostWiresResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &HostWire{}
	}
	models := func() interface{} {
		hostWires := []HostWire{}
		return &hostWires
	}

	return newResource(db, hostWiresTable, model, models)
}

type Host2Wire struct {
	HostID string `json:"host_id" gorm:"column:host_id;not null"`
	WireID string `json:"wire_id" gorm:"column:wire_id;not null"`
}

func (c Host2Wire) First() string {
	return c.WireID
}

func SelectWiresWithHostID(hostID string) ([]Host2Wire, error) {
	wires := []Host2Wire{}
	err := HostWires.DB().Table(hostWiresTable).
		Select("distinct wire_id").
		Where(fmt.Sprintf("host_id = '%s' and deleted=0", hostID)).
		Scan(&wires).Error

	return wires, err
}
func SelectHostHasWires() ([]Host2Wire, error) {
	wires := []Host2Wire{}
	err := HostWires.DB().Table(hostWiresTable).
		Select("host_id,wire_id").
		Where("deleted=0").
		Scan(&wires).Error

	return wires, err
}
