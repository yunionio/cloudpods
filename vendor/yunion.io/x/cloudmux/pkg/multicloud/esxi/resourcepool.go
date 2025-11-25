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

package esxi

import (
	"strings"

	"github.com/vmware/govmomi/vim25/mo"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/log"
)

var RESOURCEPOOL_PROPS = []string{"resourcePool"}

type SResourcePool struct {
	multicloud.SProjectBase
	multicloud.STagBase

	SManagedObject
}

func (pool *SResourcePool) GetGlobalId() string {
	return pool.GetId()
}

func (pool *SResourcePool) GetStatus() string {
	return api.EXTERNAL_PROJECT_STATUS_AVAILABLE
}

func (pool *SResourcePool) getParentEntity(obj *mo.ManagedEntity) *mo.ManagedEntity {
	parent := obj.Parent
	if parent != nil {
		var entity mo.ManagedEntity
		err := pool.manager.reference2Object(*parent, []string{"name", "parent"}, &entity)
		if err != nil {
			log.Errorf("%s", err)
			return nil
		}
		return &entity
	}
	return nil
}

func (pool *SResourcePool) IsDefault() bool {
	return strings.EqualFold(pool.GetName(), "Resources")
}

func (pool *SResourcePool) fetchPath() []string {
	path := []string{pool.GetName()}
	obj := pool.object.Entity()
	for obj != nil {
		obj = pool.getParentEntity(obj)
		if obj == nil || (obj.Self.Type == "ResourcePool" && obj.Name == "Resources") {
			break
		}
		path = append(path, obj.Name)
	}
	reverseArray(path)
	return path
}

func (pool *SResourcePool) GetPath() []string {
	if pool.path == nil {
		pool.path = pool.fetchPath()
	}
	return pool.path
}

func NewResourcePool(manager *SESXiClient, rp *mo.ResourcePool, dc *SDatacenter) *SResourcePool {
	return &SResourcePool{SManagedObject: newManagedObject(manager, rp, dc)}
}
