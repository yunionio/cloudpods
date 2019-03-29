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

type Zone struct {
	StandaloneModel
	Status        string `json:"status" gorm:"column:status;not null"`
	Location      string `gorm:"column:location"`
	ManagerUri    string `gorm:"column:manager_uri"`
	CloudregionId string `gorm:"column:cloudregion_id"`
}

func (z Zone) TableName() string {
	return zonesTable
}

func (z Zone) String() string {
	s, _ := JsonString(z)
	return s
}

func NewZoneResource(db *gorm.DB) (Resourcer, error) {
	return newResource(db, zonesTable,
		func() interface{} {
			return &Zone{}
		},
		func() interface{} {
			zones := []Zone{}
			return &zones
		},
	)
}

func FetchZoneByID(id string) (*Zone, error) {
	zone, err := FetchByID(Zones, id)
	if err != nil {
		return nil, err
	}
	return zone.(*Zone), nil
}
