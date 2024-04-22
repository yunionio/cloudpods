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
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// +onecloud:swagger-gen-ignore
type SImagePropertyManager struct {
	db.SResourceBaseManager
}

var ImagePropertyManager *SImagePropertyManager

func init() {
	ImagePropertyManager = &SImagePropertyManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SImageProperty{},
			"image_properties",
			"image_property",
			"image_properties",
		),
	}
	ImagePropertyManager.SetVirtualObject(ImagePropertyManager)
	ImagePropertyManager.TableSpec().AddIndex(true, "image_id", "name")
}

/*
+------------+--------------+------+-----+---------+----------------+
| Field      | Type         | Null | Key | Default | Extra          |
+------------+--------------+------+-----+---------+----------------+
| id         | int(11)      | NO   | PRI | NULL    | auto_increment |
| image_id   | varchar(36)  | NO   | MUL | NULL    |                |
| name       | varchar(255) | NO   |     | NULL    |                |
| value      | text         | YES  |     | NULL    |                |
| created_at | datetime     | NO   |     | NULL    |                |
| updated_at | datetime     | YES  |     | NULL    |                |
| deleted_at | datetime     | YES  |     | NULL    |                |
| deleted    | tinyint(1)   | NO   | MUL | NULL    |                |
+------------+--------------+------+-----+---------+----------------+
*/
// +onecloud:swagger-gen-ignore
type SImageProperty struct {
	SImagePeripheral

	Name  string `width:"255"`
	Value string `nullable:"true" create:"optional"`
}

func (manager *SImagePropertyManager) GetProperties(imageId string) (map[string]string, error) {
	properties := make([]SImageProperty, 0)
	q := manager.Query("name", "value").Equals("image_id", imageId)
	err := db.FetchModelObjects(manager, q, &properties)
	if err != nil {
		return nil, err
	}
	props := make(map[string]string)
	for i := range properties {
		props[properties[i].Name] = properties[i].Value
	}
	return props, nil
}

func (manager *SImagePropertyManager) SaveProperties(ctx context.Context, userCred mcclient.TokenCredential, imageId string, props jsonutils.JSONObject) error {
	propsJson := props.(*jsonutils.JSONDict)
	for _, k := range propsJson.SortedKeys() {
		v, _ := propsJson.GetString(k)
		_, err := manager.SaveProperty(ctx, userCred, imageId, k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (manager *SImagePropertyManager) SaveProperty(ctx context.Context, userCred mcclient.TokenCredential, imageId string, key string, value string) (*SImageProperty, error) {
	prop, _ := manager.GetProperty(imageId, key)
	if prop != nil {
		if prop.Value != value {
			return prop, prop.UpdateValue(ctx, userCred, value)
		} else {
			return prop, nil
		}
	} else {
		// create
		return manager.NewProperty(ctx, userCred, imageId, key, value)
	}
}

func (manager *SImagePropertyManager) GetProperty(imageId string, key string) (*SImageProperty, error) {
	q := manager.Query().Equals("image_id", imageId).Equals("name", key)
	prop := SImageProperty{}
	prop.SetModelManager(manager, &prop)

	err := q.First(&prop)
	if err != nil {
		return nil, err
	}

	return &prop, nil
}

func (manager *SImagePropertyManager) NewProperty(ctx context.Context, userCred mcclient.TokenCredential, imageId string, key string, value string) (*SImageProperty, error) {
	prop := SImageProperty{}
	prop.SetModelManager(manager, &prop)
	prop.ImageId = imageId
	prop.Name = key
	prop.Value = value

	err := manager.TableSpec().Insert(ctx, &prop)
	if err != nil {
		return nil, err
	}
	prop.SetModelManager(manager, &prop)
	return &prop, nil
}

func (self *SImageProperty) UpdateValue(ctx context.Context, userCred mcclient.TokenCredential, value string) error {
	_, err := db.Update(self, func() error {
		self.Value = value
		return nil
	})
	return err
}
