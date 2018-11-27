package esxi

import (
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
	"yunion.io/x/log"
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

func (dc *SDatacenter) getObjectDatacenter() *object.Datacenter {
	return object.NewDatacenter(dc.manager.client.Client, dc.object.Reference())
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
		var stores []mo.Datastore
		dsList := dc.getDatacenter().Datastore
		err := dc.manager.references2Objects(dsList, DATASTORE_PROPS, &stores)
		if err != nil {
			return err
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

func (dc *SDatacenter) getDcObj() *object.Datacenter {
	return object.NewDatacenter(dc.manager.client.Client, dc.object.Reference())
}

func (dc *SDatacenter) fetchVms(vmRefs []types.ManagedObjectReference) ([]cloudprovider.ICloudVM, error) {
	var vms []mo.VirtualMachine
	err := dc.manager.references2Objects(vmRefs, VIRTUAL_MACHINE_PROPS, &vms)
	if err != nil {
		log.Errorf("references2Objects fail %s", err)
		return nil, err
	}

	retVms := make([]cloudprovider.ICloudVM, len(vms))
	for i := 0; i < len(vms); i += 1 {
		retVms[i] = NewVirtualMachine(dc.manager, &vms[i], dc)
	}
	return retVms, nil
}
