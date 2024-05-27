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
	"regexp"
	"strings"
	"sync"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

var DATACENTER_PROPS = []string{"name", "parent", "datastore", "network"}

type SDatacenter struct {
	SManagedObject

	ihosts       []cloudprovider.ICloudHost
	istorages    []cloudprovider.ICloudStorage
	inetworks    map[string]IVMNetwork
	iresoucePool []cloudprovider.ICloudProject
	clusters     []*SCluster

	netLock sync.Mutex

	Name string
}

func newDatacenter(manager *SESXiClient, dc *mo.Datacenter) *SDatacenter {
	obj := SDatacenter{SManagedObject: newManagedObject(manager, dc, nil)}
	obj.datacenter = &obj
	return &obj
}

func (dc *SDatacenter) isDefaultDc() bool {
	return dc.object == &defaultDc
}

func (dc *SDatacenter) getDatacenter() *mo.Datacenter {
	return dc.object.(*mo.Datacenter)
}

func (dc *SDatacenter) getObjectDatacenter() *object.Datacenter {
	if dc.isDefaultDc() {
		return nil
	}
	return object.NewDatacenter(dc.manager.client.Client, dc.object.Reference())
}

func (dc *SDatacenter) scanResourcePool() error {
	if dc.iresoucePool == nil {
		pools, err := dc.listResourcePools()
		if err != nil {
			return errors.Wrap(err, "listResourcePools")
		}
		dc.iresoucePool = []cloudprovider.ICloudProject{}
		for i := 0; i < len(pools); i++ {
			p := NewResourcePool(dc.manager, &pools[i], dc)
			dc.iresoucePool = append(dc.iresoucePool, p)
		}
	}
	return nil
}

func (dc *SDatacenter) scanHosts() error {
	if dc.ihosts == nil {
		var hosts []mo.HostSystem
		if dc.isDefaultDc() {
			err := dc.manager.scanAllMObjects(HOST_SYSTEM_PROPS, &hosts)
			if err != nil {
				return errors.Wrap(err, "dc.manager.scanAllMObjects")
			}
		} else {
			err := dc.manager.scanMObjects(dc.object.Entity().Self, HOST_SYSTEM_PROPS, &hosts)
			if err != nil {
				return errors.Wrap(err, "dc.manager.scanMObjects")
			}
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

func (dc *SDatacenter) GetResourcePools() ([]cloudprovider.ICloudProject, error) {
	err := dc.scanResourcePool()
	if err != nil {
		return nil, errors.Wrap(err, "dc.scanResourcePool")
	}
	return dc.iresoucePool, nil
}

func (dc *SDatacenter) listResourcePools() ([]mo.ResourcePool, error) {
	var pools, result []mo.ResourcePool
	err := dc.manager.scanMObjects(dc.object.Entity().Self, RESOURCEPOOL_PROPS, &pools)
	if err != nil {
		return nil, errors.Wrap(err, "scanMObjects")
	}
	for i := range pools {
		if pools[i].Parent.Type == "ClusterComputeResource" {
			continue
		}
		result = append(result, pools[i])
	}
	return result, nil
}

func (dc *SDatacenter) ListClusters() ([]*SCluster, error) {
	return dc.listClusters()
}

func (dc *SDatacenter) GetCluster(cluster string) (*SCluster, error) {
	clusters, err := dc.ListClusters()
	if err != nil {
		return nil, errors.Wrap(err, "ListClusters")
	}
	for i := range clusters {
		if clusters[i].GetName() == cluster {
			return clusters[i], nil
		}

	}
	return nil, cloudprovider.ErrNotFound
}

func (dc *SDatacenter) listClusters() ([]*SCluster, error) {
	if dc.clusters != nil {
		return dc.clusters, nil
	}
	clusters := []mo.ClusterComputeResource{}
	err := dc.manager.scanMObjects(dc.object.Entity().Self, RESOURCEPOOL_PROPS, &clusters)
	if err != nil {
		return nil, errors.Wrap(err, "scanMObjects")
	}
	ret := []*SCluster{}
	for i := range clusters {
		c := NewCluster(dc.manager, &clusters[i], dc)
		ret = append(ret, c)
	}
	return ret, nil
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
		if dc.isDefaultDc() {
			err := dc.manager.scanAllMObjects(DATASTORE_PROPS, &stores)
			if err != nil {
				return errors.Wrap(err, "dc.manager.scanAllMObjects")
			}
		} else {
			dsList := dc.getDatacenter().Datastore
			if dsList != nil {
				err := dc.manager.references2Objects(dsList, DATASTORE_PROPS, &stores)
				if err != nil {
					return errors.Wrap(err, "dc.manager.references2Objects")
				}
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
	if dc.isDefaultDc() {
		return nil
	}
	return object.NewDatacenter(dc.manager.client.Client, dc.object.Reference())
}

func (dc *SDatacenter) fetchMoVms(filter property.Match, props []string) ([]mo.VirtualMachine, error) {
	odc := dc.getObjectDatacenter()
	root := odc.Reference()
	var movms []mo.VirtualMachine
	err := dc.manager.scanMObjectsWithFilter(root, props, &movms, filter)
	if err != nil {
		return nil, errors.Wrap(err, "scanMObjectsWithFilter")
	}
	return movms, nil
}

func (dc *SDatacenter) moVms2Vm(movms []mo.VirtualMachine, all *bool) ([]*SVirtualMachine, error) {
	// avoid applying new memory and copying
	vms := make([]*SVirtualMachine, 0, len(movms))
	for i := range movms {
		if (all != nil && *all) || !strings.HasPrefix(movms[i].Name, api.ESXI_IMAGE_CACHE_TMP_PREFIX) {
			vm := NewVirtualMachine(dc.manager, &movms[i], dc)
			// must
			if vm == nil {
				continue
			}
			vms = append(vms, vm)
		}
	}
	return vms, nil
}

func (dc *SDatacenter) fetchFakeTemplateVMs(movms []mo.VirtualMachine, regex string) ([]*SVirtualMachine, error) {
	var (
		tNameRegex *regexp.Regexp
		err        error
	)
	if len(regex) == 0 {
		tNameRegex = tempalteNameRegex
	} else {
		tNameRegex, err = regexp.Compile(regex)
		if err != nil {
			return nil, errors.Wrap(err, "generate regexp")
		}
	}
	objs := make([]types.ManagedObjectReference, 0)
	for i := range movms {
		name := movms[i].Name
		if tNameRegex != nil && tNameRegex.MatchString(name) {
			objs = append(objs, movms[i].Reference())
		}
	}
	if len(objs) == 0 {
		return nil, nil
	}
	return dc.fetchVms(objs, false)
}

func (dc *SDatacenter) fetchVmsFromCache(vmRefs []types.ManagedObjectReference) ([]*SVirtualMachine, error) {
	vmRefSet := sets.NewString()
	for i := range vmRefs {
		vmRefSet.Insert(vmRefs[i].String())
	}
	ihosts, err := dc.GetIHosts()
	if err != nil {
		return nil, err
	}
	ret := make([]*SVirtualMachine, 0, len(vmRefs))
	for i := range ihosts {
		ivms, err := ihosts[i].GetIVMs()
		if err != nil {
			return nil, errors.Wrapf(err, "GetIVMs")
		}
		for i := range ivms {
			vm := ivms[i].(*SVirtualMachine)
			s := vm.getVirtualMachine().Self.String()
			if vmRefSet.Has(s) {
				ret = append(ret, vm)
				vmRefSet.Delete(s)
			}
		}
	}
	return ret, nil
}

func (dc *SDatacenter) fetchVms(vmRefs []types.ManagedObjectReference, all bool) ([]*SVirtualMachine, error) {
	var movms []mo.VirtualMachine
	if vmRefs != nil {
		err := dc.manager.references2Objects(vmRefs, VIRTUAL_MACHINE_PROPS, &movms)
		if err != nil {
			return nil, errors.Wrap(err, "dc.manager.references2Objects")
		}
	}
	return dc.moVms2Vm(movms, &all)
}

func (dc *SDatacenter) fetchHostsInternal(vmRefs []types.ManagedObjectReference, all bool) ([]*SHost, error) {
	var movms []mo.HostSystem
	if vmRefs != nil {
		err := dc.manager.references2Objects(vmRefs, VIRTUAL_MACHINE_PROPS, &movms)
		if err != nil {
			return nil, errors.Wrap(err, "dc.manager.references2Objects")
		}
	}

	// avoid applying new memory and copying
	vms := make([]*SHost, 0, len(movms))
	for i := range movms {
		if all || !strings.HasPrefix(movms[i].Entity().Name, api.ESXI_IMAGE_CACHE_TMP_PREFIX) {
			vms = append(vms, NewHost(dc.manager, &movms[i], dc))
		}
	}
	return vms, nil
}

func (dc *SDatacenter) FetchVMs() ([]*SVirtualMachine, error) {
	return dc.fetchVMsWithFilter(property.Match{})
}

func (dc *SDatacenter) FetchNoTemplateVMs() ([]*SVirtualMachine, error) {
	filter := property.Match{}
	filter["config.template"] = false
	return dc.fetchVMsWithFilter(filter)
}

func (dc *SDatacenter) FetchFakeTempateVMs(regex string) ([]*SVirtualMachine, error) {
	filter := property.Match{}
	filter["summary.runtime.powerState"] = types.VirtualMachinePowerStatePoweredOff
	movms, err := dc.fetchMoVms(filter, []string{"name"})
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch mo.VirtualMachines")
	}
	return dc.fetchFakeTemplateVMs(movms, regex)
}

func (dc *SDatacenter) fetchVMsWithFilter(filter property.Match) ([]*SVirtualMachine, error) {
	movms, err := dc.fetchMoVms(filter, VIRTUAL_MACHINE_PROPS)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch mo.VirtualMachines")
	}
	ret, err := dc.moVms2Vm(movms, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to convert mo.VirtualMachine to VirtualMachine")
	}
	return ret, err
}

func (dc *SDatacenter) FetchNoTemplateVMEntityReferens() ([]types.ManagedObjectReference, error) {
	filter := property.Match{}
	filter["config.template"] = false
	odc := dc.getObjectDatacenter()
	root := odc.Reference()
	m := view.NewManager(dc.manager.client.Client)
	v, err := m.CreateContainerView(dc.manager.context, root, []string{"VirtualMachine"}, true)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = v.Destroy(dc.manager.context)
	}()
	objs, err := v.Find(dc.manager.context, []string{"VirtualMachine"}, filter)
	if err != nil {
		return nil, err
	}
	return objs, nil
}

func (dc *SDatacenter) FetchNoTemplateHostEntityReferens() ([]types.ManagedObjectReference, error) {
	filter := property.Match{}
	odc := dc.getObjectDatacenter()
	root := odc.Reference()
	m := view.NewManager(dc.manager.client.Client)
	v, err := m.CreateContainerView(dc.manager.context, root, []string{"HostSystem"}, true)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = v.Destroy(dc.manager.context)
	}()
	objs, err := v.Find(dc.manager.context, []string{"HostSystem"}, filter)
	if err != nil {
		return nil, err
	}
	return objs, nil
}

func (dc *SDatacenter) fetchHosts(filter property.Match) ([]*SHost, error) {
	odc := dc.getObjectDatacenter()
	root := odc.Reference()
	m := view.NewManager(dc.manager.client.Client)
	v, err := m.CreateContainerView(dc.manager.context, root, []string{"HostSystem"}, true)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = v.Destroy(dc.manager.context)
	}()
	objs, err := v.Find(dc.manager.context, []string{"HostSystem"}, filter)
	if err != nil {
		return nil, err
	}
	hosts, err := dc.fetchHostsInternal(objs, false)
	return hosts, err
}

func (dc *SDatacenter) FetchTemplateVMs() ([]*SVirtualMachine, error) {
	filter := property.Match{}
	filter["config.template"] = true
	return dc.fetchVMsWithFilter(filter)
}

func (dc *SDatacenter) FetchTemplateVMById(id string) (*SVirtualMachine, error) {
	filter := property.Match{}
	filter["config.template"] = true
	filter["summary.config.uuid"] = id
	vms, err := dc.fetchVMsWithFilter(filter)
	if err != nil {
		return nil, err
	}
	if len(vms) == 0 {
		return nil, errors.ErrNotFound
	}
	return vms[0], nil
}

func (dc *SDatacenter) FetchVMById(id string) (*SVirtualMachine, error) {
	filter := property.Match{}
	filter["summary.config.uuid"] = id
	vms, err := dc.fetchVMsWithFilter(filter)
	if err != nil {
		return nil, err
	}
	if len(vms) == 0 {
		return nil, errors.ErrNotFound
	}
	return vms[0], nil
}

func (dc *SDatacenter) fetchDatastores(datastoreRefs []types.ManagedObjectReference) ([]cloudprovider.ICloudStorage, error) {
	var dss []mo.Datastore
	if datastoreRefs != nil {
		err := dc.manager.references2Objects(datastoreRefs, DATASTORE_PROPS, &dss)
		if err != nil {
			return nil, errors.Wrap(err, "dc.manager.references2Objects")
		}
	}

	retDatastores := make([]cloudprovider.ICloudStorage, 0, len(dss))
	for i := range dss {
		retDatastores = append(retDatastores, NewDatastore(dc.manager, &dss[i], dc))
	}
	return retDatastores, nil
}

func (dc *SDatacenter) scanNetworks() error {
	if dc.inetworks == nil {
		dc.inetworks = make(map[string]IVMNetwork)
		if dc.isDefaultDc() {
			return dc.scanDefaultNetworks()
		} else {
			return dc.scanDcNetworks()
		}
	}
	return nil
}

func (dc *SDatacenter) scanDefaultNetworks() error {
	err := dc.scanAllDvPortgroups()
	if err != nil {
		return errors.Wrap(err, "dc.scanAllDvPortgroups")
	}
	err = dc.scanAllNetworks()
	if err != nil {
		return errors.Wrap(err, "dc.scanAllNetworks")
	}
	return nil
}

func (dc *SDatacenter) scanAllDvPortgroups() error {
	var dvports []mo.DistributedVirtualPortgroup

	err := dc.manager.scanAllMObjects(DVPORTGROUP_PROPS, &dvports)
	if err != nil {
		return errors.Wrap(err, "dc.manager.scanAllMObjects mo.DistributedVirtualPortgroup")
	}
	dc.netLock.Lock()
	defer dc.netLock.Unlock()
	for i := range dvports {
		net := NewDistributedVirtualPortgroup(dc.manager, &dvports[i], dc)
		dc.inetworks[net.GetName()] = net
	}
	return nil
}

func (dc *SDatacenter) scanAllNetworks() error {
	var nets []mo.Network

	err := dc.manager.scanAllMObjects(NETWORK_PROPS, &nets)
	if err != nil {
		return errors.Wrap(err, "dc.manager.scanAllMObjects mo.Network")
	}
	dc.netLock.Lock()
	defer dc.netLock.Unlock()
	for i := range nets {
		net := NewNetwork(dc.manager, &nets[i], dc)
		dc.inetworks[net.GetName()] = net
	}
	return nil
}

func (dc *SDatacenter) scanDcNetworks() error {
	nets, err := dc.resolveNetworks(dc.getDatacenter().Network)
	if err != nil {
		return errors.Wrap(err, "resolveNetworks")
	}
	dc.netLock.Lock()
	defer dc.netLock.Unlock()
	for i := range nets {
		net := nets[i]
		dc.inetworks[net.GetName()] = net
	}
	return nil
}

func (dc *SDatacenter) getNetworkByName(name string) (IVMNetwork, error) {
	err := dc.scanNetworks()
	if err != nil {
		return nil, errors.Wrap(err, "dc.scanNetworks")
	}
	if net, ok := dc.inetworks[name]; !ok {
		return nil, errors.Wrap(errors.ErrNotFound, name)
	} else {
		return net, nil
	}
}

func (dc *SDatacenter) GetNetworks() ([]IVMNetwork, error) {
	err := dc.scanNetworks()
	if err != nil {
		return nil, errors.Wrap(err, "dc.scanNetworks")
	}
	nets := make([]IVMNetwork, 0, len(dc.inetworks))
	for _, net := range dc.inetworks {
		nets = append(nets, net)
	}
	return nets, nil
}

func (dc *SDatacenter) GetTemplateVMs() ([]*SVirtualMachine, error) {
	hosts, err := dc.GetIHosts()
	if err != nil {
		return nil, errors.Wrap(err, "SDatacenter.GetIHosts")
	}
	templateVms := make([]*SVirtualMachine, 5)
	for _, ihost := range hosts {
		host := ihost.(*SHost)
		tvms, err := host.GetTemplateVMs()
		if err != nil {
			return nil, errors.Wrap(err, "host.GetTemplateVMs")
		}
		templateVms = append(templateVms, tvms...)
	}
	return templateVms, nil
}
