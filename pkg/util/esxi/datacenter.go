package esxi

import (
	"github.com/vmware/govmomi/vim25/mo"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

var DATACENTER_PROPS = []string{"name", "parent", "datastore"}

type SDatacenter struct {
	SManagedObject

	ihosts    []cloudprovider.ICloudHost
	istorages []cloudprovider.ICloudStorage

	Name string
}

func newDatacenter(manager *SESXiClient, dc *mo.Datacenter) *SDatacenter {
	obj := SDatacenter{SManagedObject: newManagedObject(manager, dc, nil)}
	obj.datacenter = &obj
	return &obj
}

func (dc *SDatacenter) getDatacenter() *mo.Datacenter {
	return dc.object.(*mo.Datacenter)
}

func (dc *SDatacenter) scanHosts() error {
	if dc.ihosts == nil {
		var hosts []mo.HostSystem
		err := dc.manager.scanMObjects(dc.object.Entity().Self, HOST_SYSTEM_PROPS, &hosts)
		if err != nil {
			return err
		}
		dc.ihosts = make([]cloudprovider.ICloudHost, len(hosts))
		for i := 0; i < len(hosts); i += 1 {
			dc.ihosts[i] = NewHost(dc.manager, &hosts[i], dc)
		}
	}
	return nil
}

func (dc *SDatacenter) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	err := dc.scanHosts()
	if err != nil {
		return nil, err
	}
	return dc.ihosts, nil
}

func (dc *SDatacenter) scanDatastores() error {
	if dc.istorages == nil {
		stores := make([]mo.Datastore, 0)
		dsList := dc.getDatacenter().Datastore
		for i := 0; i < len(dsList); i += 1 {
			var ds mo.Datastore
			err := dc.manager.reference2Object(dsList[i], DATASTORE_PROPS, &ds)
			if err != nil {
				return err
			}
			stores = append(stores, ds)
		}
		dc.istorages = make([]cloudprovider.ICloudStorage, len(stores))
		for i := 0; i < len(stores); i += 1 {
			dc.istorages[i] = NewDatastore(dc.manager, &stores[i], dc)
		}
	}
	return nil
}

func (dc *SDatacenter) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	err := dc.scanDatastores()
	if err != nil {
		return nil, err
	}
	return dc.istorages, nil
}

func (dc *SDatacenter) GetIHostByMoId(idstr string) (cloudprovider.ICloudHost, error) {
	ihosts, err := dc.GetIHosts()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(ihosts); i += 1 {
		if ihosts[i].GetId() == idstr {
			return ihosts[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (dc *SDatacenter) GetIStorageByMoId(idstr string) (cloudprovider.ICloudStorage, error) {
	istorages, err := dc.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(istorages); i += 1 {
		if istorages[i].GetId() == idstr {
			return istorages[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}
