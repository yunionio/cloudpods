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
	"strings"
	"time"

	"github.com/jinzhu/gorm"
)

type Metadata struct {
	ID        string    `json:"id" gorm:"primary_key;column:id;type:varchar(128) CHARACTER SET ascii"`
	Key       string    `json:"key" gorm:"primary_key;column:key;type:varchar(64) CHARACTER SET ascii"`
	Value     string    `json:"value" gorm:"type:text CHARACTER SET utf8"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;type:datetime" sql:"DEFAULT:NULL"`
}

func (m Metadata) TableName() string {
	return metadataTable
}

func (m Metadata) String() string {
	str, _ := JsonString(m)
	return str
}

func NewMetadataResource(db *gorm.DB) (Resourcer, error) {
	return newResource(db, metadataTable,
		func() interface{} { return &Metadata{} },
		func() interface{} { return &([]Metadata{}) })
}

func FetchMetadatas(resourceName string, ids, keys []string) ([]interface{}, error) {
	idsWithRes := make([]string, len(ids))
	for i, id := range ids {
		idsWithRes[i] = fmt.Sprintf("%s::%s", resourceName, id)
	}
	rows, err := Metadatas.DB().Table(metadataTable).
		Where(fmt.Sprintf("id in ('%s') AND `key` in ('%s')", strings.Join(idsWithRes, "','"), strings.Join(keys, "','"))).Rows()
	if err != nil {
		return nil, err
	}
	return rowsToArray(Metadatas, rows)
}
