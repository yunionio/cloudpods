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

type GroupGuest struct {
	GroupJointModel
	Tag     *string `json:"tag" gorm:"column:tag"`
	GuestID *string `json:"guest_id" gorm:"column:guest_id"`
}

func (g GroupGuest) TableName() string {
	return groupGuestTable
}

func (g GroupGuest) String() string {
	str, _ := JsonString(g)
	return str
}

func NewGroupGuestResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &GroupGuest{}
	}
	models := func() interface{} {
		groupGuests := []GroupGuest{}
		return &groupGuests
	}

	return newResource(db, groupGuestTable, model, models)
}
