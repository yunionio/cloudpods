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

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SManagedObject struct {
	manager    *SESXiClient
	datacenter *SDatacenter
	object     mo.Entity

	path []string
}

func newManagedObject(manager *SESXiClient, moobj mo.Entity, dc *SDatacenter) SManagedObject {
	return SManagedObject{manager: manager, object: moobj, datacenter: dc}
}

func (self *SManagedObject) GetName() string {
	return self.object.Entity().Name
}

func (self *SManagedObject) GetId() string {
	return moRefId(self.object.Entity().Self)
}

func (self *SManagedObject) GetType() string {
	return moRefType(self.object.Entity().Self)
}

func (self *SManagedObject) getCurrentParentEntity() *mo.ManagedEntity {
	return self.getParentEntity(self.object.Entity())
}

func (self *SManagedObject) getParentEntity(obj *mo.ManagedEntity) *mo.ManagedEntity {
	parent := obj.Parent
	if parent != nil {
		var entity mo.ManagedEntity
		err := self.manager.reference2Object(*parent, []string{"name", "parent"}, &entity)
		if err != nil {
			log.Errorf("%s", err)
			return nil
		}
		// log.Debugf("getParentEntity %s %s %s", entity.Self.Type, entity.Self.Value, entity.Name)
		return &entity
	}
	return nil
}

func (self *SManagedObject) fetchPath() []string {
	path := make([]string, 0)
	obj := self.object.Entity()
	for obj != nil {
		path = append(path, obj.Name)
		obj = self.getParentEntity(obj)
	}
	reverseArray(path)
	return path
}

func (self *SManagedObject) GetPath() []string {
	if self.path == nil {
		self.path = self.fetchPath()
	}
	return self.path
}

func (self *SManagedObject) findInParents(objType string) *mo.ManagedEntity {
	obj := self.object.Entity()

	for obj != nil && obj.Self.Type != objType {
		log.Debugf("find %s want %s", obj.Self.Type, objType)
		obj = self.getParentEntity(obj)
	}

	return obj
}

func (self *SManagedObject) fetchDatacenter() (*SDatacenter, error) {
	me := self.findInParents("Datacenter")
	if me == nil {
		return nil, cloudprovider.ErrNotFound
	}
	return self.manager.FindDatacenterByMoId(me.Self.Value)
}

func (self *SManagedObject) GetDatacenter() (*SDatacenter, error) {
	if self.datacenter == nil {
		dc, err := self.fetchDatacenter()
		if err != nil {
			return nil, err
		}
		self.datacenter = dc
	}
	return self.datacenter, nil
}

func (self *SManagedObject) GetDatacenterPath() []string {
	dc, err := self.GetDatacenter()
	if err != nil {
		log.Errorf("cannot find datacenter")
		return nil
	}
	path := dc.GetPath()
	return path[1:]
}

func (self *SManagedObject) GetDatacenterPathString() string {
	path := self.GetDatacenterPath()
	if path != nil {
		return strings.Join(path, "/")
	}
	return ""
}

func (self *SManagedObject) getManagerUri() string {
	return self.manager.getUrl()
}
