package esxi

import (
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/yunionio/log"
	"github.com/yunionio/onecloud/pkg/cloudprovider"
	"reflect"
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
	return self.object.Entity().Self.Value
}

func (self *SManagedObject) GetType() string {
	return self.object.Entity().Self.Type
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
		log.Debugf("getParentEntity %s %s %s", entity.Self.Type, entity.Self.Value, entity.Name)
		return &entity
	}
	return nil
}

func reverseArray(array interface{}) {
	arrayValue := reflect.Indirect(reflect.ValueOf(array))
	if arrayValue.Kind() != reflect.Slice && arrayValue.Kind() != reflect.Array {
		log.Errorf("reverse non array or slice")
		return
	}
	tmp := reflect.Indirect(reflect.New(arrayValue.Type().Elem()))
	for i, j := 0, arrayValue.Len()-1; i < j; i, j = i+1, j-1 {
		tmpi := arrayValue.Index(i)
		tmpj := arrayValue.Index(j)
		tmp.Set(tmpi)
		tmpi.Set(tmpj)
		tmpj.Set(tmp)
	}
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
		obj = self.getParentEntity(obj)
	}

	return obj
}

func (self *SManagedObject) fetchDatacenter() (*SDatacenter, error) {
	me := self.findInParents("Datacenter")
	if me == nil {
		return nil, cloudprovider.ErrNotFound
	}
	return self.manager.FindDatacenterById(me.Self.Value)
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
