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

type Cloudprovider struct {
	StandaloneModel
	Status         string `json:"status" gorm:"column:status;not null"`
	Enabled        bool   `json:"enabled" gorm:"column:enabled;not null"`
	AccessUrl      string `json:"access_url" gorm:"column:access_url"`
	Provider       string `json:"provider" gorm:"column:provider"`
	CloudaccountId string `json:"cloudaccount_id" gorm:"column:cloudaccount_id"`
	ProjectId      string `json:"tenant_id" gorm:"column:tenant_id"`
}

func (c Cloudprovider) TableName() string {
	return cloudproviderTable
}

func (c Cloudprovider) String() string {
	s, _ := JsonString(c)
	return s
}

func NewCloudproviderResource(db *gorm.DB) (Resourcer, error) {
	return newResource(db, cloudproviderTable,
		func() interface{} {
			return &Cloudprovider{}
		},
		func() interface{} {
			cs := []Cloudprovider{}
			return &cs
		})
}

func FetchCloudproviderById(id string) (*Cloudprovider, error) {
	obj, err := FetchByID(CloudProviders, id)
	if err != nil {
		return nil, err
	}
	return obj.(*Cloudprovider), nil
}
