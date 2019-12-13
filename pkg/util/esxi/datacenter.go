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

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

var DATACENTER_PROPS = []string{"name", "parent", "datastore", "network"}

type SDatacenter struct {
	SManagedObject

	ihosts    []cloudprovider.ICloudHost
	istorages []cloudprovider.ICloudStorage
	inetworks []IVMNetwork

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
			return errors.Wrap(err, "dc.manager.scanMObjects")
		}
		dc.ihosts = make([]cloudprovider.ICloudHost, 0)
		for i := 0; i < len(hosts); i += 1 {
			h := NewHost(dc.manager, &hosts[i], dc)
			if h != nil {
				dc.ihosts = append(dc.ihosts, h)
			}
		}
	}
	return nil
}

func (dc *SDatacenter) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	err := dc.scanHosts()
	if err != nil {
		return nil, errors.Wrap(err, "dc.scanHosts")
	}
	return dc.ihosts, nil
}

func (dc *SDatacenter) scanDatastores() error {
	if dc.istorages == nil {
		var stores []mo.Datastore
		dsList := dc.getDatacenter().Datastore
		if dsList != nil {
			err := dc.manager.references2Objects(dsList, DATASTORE_PROPS, &stores)
			if err != nil {
				return errors.Wrap(err, "dc.manager.references2Objects")
			}
		}
		dc.istorages = make([]cloudprovider.ICloudStorage, 0)
		for i := 0; i < len(stores); i += 1 {
			ds := NewDatastore(dc.manager, &stores[i], dc)
			dsId := ds.GetGlobalId()
			if len(dsId) > 0 {
				dc.istorages = append(dc.istorages, ds)
			}
		}
	}
	return nil
}

func (dc *SDatacenter) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	err := dc.scanDatastores()
	if err != nil {
		return nil, errors.Wrap(err, "dc.scanDatastores")
	}
	return dc.istorages, nil
}

func (dc *SDatacenter) GetIHostByMoId(idstr string) (cloudprovider.ICloudHost, error) {
	ihosts, err := dc.GetIHosts()
	if err != nil {
		return nil, errors.Wrap(err, "dc.GetIHosts")
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
		return nil, errors.Wrap(err, "dc.GetIStorages")
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

func (dc *SDatacenter) fetchVms(vmRefs []types.ManagedObjectReference, all bool) ([]cloudprovider.ICloudVM, error) {
	var vms []mo.VirtualMachine
	if vmRefs != nil {
		err := dc.manager.references2Objects(vmRefs, VIRTUAL_MACHINE_PROPS, &vms)
		if err != nil {
			return nil, errors.Wrap(err, "dc.manager.references2Objects")
		}
	}

	retVms := make([]cloudprovider.ICloudVM, 0)
	for i := 0; i < len(vms); i += 1 {
		if all || !strings.HasPrefix(vms[i].Entity().Name, api.ESXI_IMAGE_CACHE_TMP_PREFIX) {
			vmObj := NewVirtualMachine(dc.manager, &vms[i], dc)
			if vmObj != nil {
				retVms = append(retVms, vmObj)
			}
		}
	}
	return retVms, nil
}

func (dc *SDatacenter) scanNetworks() error {
	if dc.inetworks == nil {
		dc.inetworks = make([]IVMNetwork, 0)

		netMOBs := dc.getDatacenter().Network
		for i := range netMOBs {
			dvport := mo.DistributedVirtualPortgroup{}
			err := dc.manager.reference2Object(netMOBs[i], DVPORTGROUP_PROPS, &dvport)
			if err == nil {
				net := NewDistributedVirtualPortgroup(dc.manager, &dvport, dc)
				dc.inetworks = append(dc.inetworks, net)
			} else {
				net := mo.Network{}
				err = dc.manager.reference2Object(netMOBs[i], NETWORK_PROPS, &net)
				if err == nil {
					vnet := NewNetwork(dc.manager, &net, dc)
					dc.inetworks = append(dc.inetworks, vnet)
				} else {
					return errors.Wrap(err, "dc.manager.reference2Object")
				}
			}
		}
	}
	return nil
}

func (dc *SDatacenter) GetNetworks() ([]IVMNetwork, error) {
	err := dc.scanNetworks()
	if err != nil {
		return nil, errors.Wrap(err, "dc.scanNetworks")
	}
	return dc.inetworks, nil
}
