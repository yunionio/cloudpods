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
	NetInterfaceResourceName = "netinterface"
)

type NetInterface struct {
	Mac         string `json:"mac,omitempty" gorm:"not null"`
	BaremetalId string `json:"baremetal_id,omitempty"`
	WireId      string `json:"wire_id,omitempty"`
	Rate        int64  `json:"rate,omitempty"`
	NicType     string `json:"nic_type,omitempty"`
	Index       int    `json:"index,omitempty"`
	LinkUp      int    `json:"link_up,omitempty"`
	Mtu         int64  `json:"mtu,omitempty"`
}

func (n NetInterface) TableName() string {
	return netinterfacesTable
}

func (n NetInterface) String() string {
	str, _ := JsonString(n)
	return str
}

func NewNetInterfacesResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &NetInterface{}
	}
	models := func() interface{} {
		netInterfaces := []NetInterface{}
		return &netInterfaces
	}

	return newResource(db, netinterfacesTable, model, models)
}

type BaremetalWire struct {
	BaremetalID string `json:"baremetal_id" gorm:"column:baremetal_id;not null"`
	WireID      string `json:"wire_id" gorm:"column:wire_id;not null"`
}

func (c BaremetalWire) First() string {
	return c.WireID
}

func SelectWiresWithBaremetalID(baremetalID string) ([]BaremetalWire, error) {
	baremetalWires := []BaremetalWire{}
	err := NetInterfaces.DB().Table(netinterfacesTable).
		Select("distinct wire_id").
		Where(fmt.Sprintf("baremetal_id = '%s'", baremetalID)).
		Scan(&baremetalWires).Error

	return baremetalWires, err
}

func SelectWiresAndBaremetals() ([]BaremetalWire, error) {
	baremetalWires := []BaremetalWire{}
	err := NetInterfaces.DB().Table(netinterfacesTable).
		Select("baremetal_id,wire_id").
		Scan(&baremetalWires).Error

	return baremetalWires, err
}
