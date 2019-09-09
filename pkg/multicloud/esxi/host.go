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
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

var HOST_SYSTEM_PROPS = []string{"name", "parent", "summary", "config", "hardware", "vm", "datastore"}

type SHostStorageAdapterInfo struct {
	Device    string
	Model     string
	Driver    string
	Pci       string
	Drivers   []*SHostStorageDriverInfo
	Enclosure int
}

type SHostStorageDriverInfo struct {
	CN       string
	Name     string
	Model    string
	Vendor   string
	Revision string
	Status   string
	SSD      bool
	Dev      string
	Size     int
	Slot     int
}

type SHostStorageEnclosureInfo struct {
	CN       string
	Name     string
	Model    string
	Vendor   string
	Revision string
	Status   string
}

type SHostStorageInfo struct {
	Adapter int
	Driver  string
	Index   int
	Model   string
	Rotate  bool
	Status  string
	Size    int
}

type SHost struct {
	multicloud.SHostBase
	SManagedObject

	nicInfo     []SHostNicInfo
	storageInfo []SHostStorageInfo

	datastores []cloudprovider.ICloudStorage

	storageCache *SDatastoreImageCache

	vms []cloudprovider.ICloudVM
}

func NewHost(manager *SESXiClient, host *mo.HostSystem, dc *SDatacenter) *SHost {
	return &SHost{SManagedObject: newManagedObject(manager, host, dc)}
}

var (
	ip4addrPattern = regexp.MustCompile(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`)
)

func formatName(name string) string {
	if ip4addrPattern.MatchString(name) {
		return strings.Replace(name, ".", "-", -1)
	} else {
		dotPos := strings.IndexByte(name, '.')
		if dotPos > 0 && !regutils.MatchIP4Addr(name) {
			name = name[:dotPos]
		}
		return name
	}
}

func (self *SHost) GetName() string {
	return formatName(self.SManagedObject.GetName())
}

func (self *SHost) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SHost) getHostSystem() *mo.HostSystem {
	return self.object.(*mo.HostSystem)
}

func (self *SHost) GetGlobalId() string {
	return self.GetAccessIp()
}

func (self *SHost) GetStatus() string {
	/*
		HostSystemPowerStatePoweredOn  = HostSystemPowerState("poweredOn")
		HostSystemPowerStatePoweredOff = HostSystemPowerState("poweredOff")
		HostSystemPowerStateStandBy    = HostSystemPowerState("standBy")
		HostSystemPowerStateUnknown    = HostSystemPowerState("unknown")
	*/
	switch self.getHostSystem().Summary.Runtime.PowerState {
	case types.HostSystemPowerStatePoweredOn:
		return api.HOST_STATUS_RUNNING
	case types.HostSystemPowerStatePoweredOff:
		return api.HOST_STATUS_READY
	default:
		return api.HOST_STATUS_UNKNOWN
	}
}

func (self *SHost) Refresh() error {
	base := self.SManagedObject
	var moObj mo.HostSystem
	err := self.manager.reference2Object(self.object.Reference(), HOST_SYSTEM_PROPS, &moObj)
	if err != nil {
		return err
	}
	base.object = &moObj
	*self = SHost{}
	self.SManagedObject = base
	return nil
}

func (self *SHost) IsEmulated() bool {
	return false
}

func (self *SHost) fetchVMs(all bool) error {
	if self.vms != nil {
		return nil
	}

	dc, err := self.GetDatacenter()
	if err != nil {
		return err
	}

	MAX_TRIES := 3
	for tried := 0; tried < MAX_TRIES; tried += 1 {
		hostVms := self.getHostSystem().Vm
		if len(hostVms) == 0 {
			// log.Errorf("host VMs are nil!!!!!")
			return nil
		}

		vms, err := dc.fetchVms(hostVms, all)
		if err != nil {
			log.Errorf("dc.fetchVms fail %s", err)
			time.Sleep(time.Second)
			self.Refresh()
			continue
		}
		self.vms = vms
	}
	return nil
}

func (self *SHost) GetIVMs2() ([]cloudprovider.ICloudVM, error) {
	err := self.fetchVMs(true)
	if err != nil {
		return nil, err
	}
	return self.vms, nil
}

func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	err := self.fetchVMs(false)
	if err != nil {
		return nil, err
	}
	return self.vms, nil
}

func (self *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	id = self.manager.getPrivateId(id)

	vms, err := self.GetIVMs()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(vms); i += 1 {
		if vms[i].GetGlobalId() == id {
			return vms[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SHost) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	moHost := self.getHostSystem()
	dc, err := self.GetDatacenter()
	if err != nil {
		return nil, err
	}
	istorages := make([]cloudprovider.ICloudStorage, len(moHost.Datastore))
	for i := 0; i < len(moHost.Datastore); i += 1 {
		storage, err := dc.GetIStorageByMoId(moRefId(moHost.Datastore[i]))
		if err != nil {
			return nil, err
		}
		istorages[i] = storage
	}
	return istorages, nil
}

func (self *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	istorages, err := self.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(istorages); i += 1 {
		if istorages[i].GetGlobalId() == id {
			return istorages[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SHost) GetEnabled() bool {
	return true
}

func (self *SHost) GetHostStatus() string {
	/*
		HostSystemConnectionStateConnected     = HostSystemConnectionState("connected")
		HostSystemConnectionStateNotResponding = HostSystemConnectionState("notResponding")
		HostSystemConnectionStateDisconnected  = HostSystemConnectionState("disconnected")
	*/
	switch self.getHostSystem().Summary.Runtime.ConnectionState {
	case types.HostSystemConnectionStateConnected:
		return api.HOST_ONLINE
	default:
		return api.HOST_OFFLINE
	}
}

func findHostNicByMac(nicInfoList []SHostNicInfo, mac string) *SHostNicInfo {
	for i := 0; i < len(nicInfoList); i += 1 {
		if nicInfoList[i].Mac == mac {
			return &nicInfoList[i]
		}
	}
	return nil
}

func (self *SHost) getAdminNic() *SHostNicInfo {
	nics := self.getNicInfo()
	for i := 0; i < len(nics); i += 1 {
		if nics[i].NicType == api.NIC_TYPE_ADMIN {
			return &nics[i]
		}
	}
	for i := 0; i < len(nics); i += 1 {
		if len(nics[i].IpAddr) > 0 {
			return &nics[i]
		}
	}
	return nil
}

func (self *SHost) getNicInfo() []SHostNicInfo {
	if self.nicInfo == nil {
		self.nicInfo = self.fetchNicInfo()
	}
	return self.nicInfo
}

func (self *SHost) fetchNicInfo() []SHostNicInfo {
	moHost := self.getHostSystem()

	nicInfoList := make([]SHostNicInfo, 0)

	for i, nic := range moHost.Config.Network.Pnic {
		info := SHostNicInfo{}
		info.Dev = nic.Device
		info.Driver = nic.Driver
		info.Mac = netutils.FormatMacAddr(nic.Mac)
		info.Index = int8(i)
		info.LinkUp = false
		nicInfoList = append(nicInfoList, info)
	}

	vnics := make([]types.HostVirtualNic, 0)
	if len(moHost.Config.Network.Vnic) > 0 {
		vnics = append(vnics, moHost.Config.Network.Vnic...)
	}
	if len(moHost.Config.Network.ConsoleVnic) > 0 {
		vnics = append(vnics, moHost.Config.Network.ConsoleVnic...)
	}
	findMaster := false
	for _, nic := range vnics {
		mac := netutils.FormatMacAddr(nic.Spec.Mac)
		pnic := findHostNicByMac(nicInfoList, mac)
		if pnic != nil {
			findMaster = true
			pnic.IpAddr = nic.Spec.Ip.IpAddress
			if nic.Spec.Portgroup == "Management Network" || nic.Spec.Portgroup == "Service Console" {
				pnic.NicType = api.NIC_TYPE_ADMIN
			}
			pnic.LinkUp = true
			pnic.Mtu = nic.Spec.Mtu
		}
	}

	if !findMaster && len(nicInfoList) > 0 {
		// no match pnic found for master nic
		// choose the first pnic
		pnic := &nicInfoList[0]
		for _, nic := range vnics {
			if nic.Spec.Portgroup == "Management Network" || nic.Spec.Portgroup == "Service Console" {
				pnic.NicType = api.NIC_TYPE_ADMIN
				pnic.IpAddr = nic.Spec.Ip.IpAddress
				pnic.LinkUp = true
				pnic.Mtu = nic.Spec.Mtu
				break
			}
		}
	}

	return nicInfoList
}

func (self *SHost) GetAccessIp() string {
	adminNic := self.getAdminNic()
	if adminNic != nil {
		return adminNic.IpAddr
	}
	return ""
}

func (self *SHost) GetAccessMac() string {
	adminNic := self.getAdminNic()
	if adminNic != nil {
		return adminNic.Mac
	}
	return ""
}

type SSysInfo struct {
	Manufacture  string
	Model        string
	SerialNumber string
}

func (self *SHost) GetSysInfo() jsonutils.JSONObject {
	sysinfo := SSysInfo{}
	sysinfo.Manufacture = self.getHostSystem().Summary.Hardware.Vendor
	sysinfo.Model = self.getHostSystem().Summary.Hardware.Model
	sysinfo.SerialNumber = self.getHostSystem().Hardware.SystemInfo.SerialNumber
	return jsonutils.Marshal(&sysinfo)
}

func (self *SHost) GetSN() string {
	return self.getHostSystem().Hardware.SystemInfo.SerialNumber
}

func (self *SHost) GetCpuCount() int {
	return int(self.getHostSystem().Summary.Hardware.NumCpuThreads)
}

func (self *SHost) GetNodeCount() int8 {
	return int8(self.getHostSystem().Summary.Hardware.NumCpuPkgs)
}

func (self *SHost) GetCpuDesc() string {
	return self.getHostSystem().Summary.Hardware.CpuModel
}

func (self *SHost) GetCpuMhz() int {
	return int(self.getHostSystem().Summary.Hardware.CpuMhz)
}

func (self *SHost) GetMemSizeMB() int {
	return int(self.getHostSystem().Summary.Hardware.MemorySize / 1024 / 1024)
}

func (self *SHost) GetStorageInfo() []SHostStorageInfo {
	if self.storageInfo == nil {
		self.storageInfo = self.getStorageInfo()
	}
	return self.storageInfo
}

func (self *SHost) getStorageInfo() []SHostStorageInfo {
	diskSlots := make(map[int]SHostStorageInfo)
	list := self.getStorages()
	for i := 0; i < len(list); i += 1 {
		for j := 0; j < len(list[i].Drivers); j += 1 {
			drv := list[i].Drivers[j]
			info := SHostStorageInfo{
				Adapter: 0,
				Driver:  "Linux",
				Index:   drv.Slot,
				Model:   strings.TrimSpace(fmt.Sprintf("%s %s", drv.Vendor, drv.Model)),
				Rotate:  !drv.SSD,
				Status:  drv.Status,
				Size:    drv.Size,
			}
			diskSlots[info.Index] = info
		}
	}
	disks := make([]SHostStorageInfo, 0)
	idx := 0
	for {
		if info, ok := diskSlots[idx]; ok {
			disks = append(disks, info)
			idx += 1
		} else {
			break
		}
	}
	return disks
}

func (self *SHost) getStorages() []*SHostStorageAdapterInfo {
	adapterList := make([]*SHostStorageAdapterInfo, 0)
	adapterTable := make(map[string]*SHostStorageAdapterInfo)
	driversTable := make(map[string]*SHostStorageDriverInfo, 0)
	enclosuresTable := make(map[string]*SHostStorageEnclosureInfo, 0)
	moHost := self.getHostSystem()

	for i := 0; i < len(moHost.Config.StorageDevice.HostBusAdapter); i += 1 {
		ad := moHost.Config.StorageDevice.HostBusAdapter[i]
		adinfo := ad.GetHostHostBusAdapter()
		if adinfo == nil {
			log.Errorf("fail to GetHostHostBusAdapter")
			continue
		}
		info := SHostStorageAdapterInfo{}
		info.Device = adinfo.Device
		info.Model = strings.TrimSpace(adinfo.Model)
		info.Driver = adinfo.Driver
		info.Pci = adinfo.Pci
		info.Drivers = make([]*SHostStorageDriverInfo, 0)
		info.Enclosure = -1

		adapterTable[adinfo.Key] = &info
		adapterList = append(adapterList, &info)
	}

	for i := 0; i < len(moHost.Config.StorageDevice.ScsiLun); i += 1 {
		drv := moHost.Config.StorageDevice.ScsiLun[i]
		lunInfo := drv.GetScsiLun()
		if lunInfo == nil {
			log.Errorf("fail to GetScsiLun")
			continue
		}

		if lunInfo.DeviceType == "disk" {
			scsiDisk := drv.(*types.HostScsiDisk)
			info := SHostStorageDriverInfo{}
			info.CN = scsiDisk.CanonicalName
			info.Name = scsiDisk.DisplayName
			info.Model = strings.TrimSpace(scsiDisk.Model)
			info.Vendor = strings.TrimSpace(scsiDisk.Vendor)
			info.Revision = scsiDisk.Revision
			info.Status = scsiDisk.OperationalState[0]
			if scsiDisk.Ssd != nil && *scsiDisk.Ssd {
				info.SSD = true
			}
			info.Dev = scsiDisk.DevicePath
			info.Size = int(int64(scsiDisk.Capacity.BlockSize) * scsiDisk.Capacity.Block / 1024 / 1024)

			driversTable[scsiDisk.Key] = &info
		} else if lunInfo.DeviceType == "enclosure" {
			enclosuresTable[lunInfo.Key] = &SHostStorageEnclosureInfo{
				CN:       lunInfo.CanonicalName,
				Name:     lunInfo.DisplayName,
				Model:    strings.TrimSpace(lunInfo.Model),
				Vendor:   strings.TrimSpace(lunInfo.Vendor),
				Revision: lunInfo.Revision,
				Status:   lunInfo.OperationalState[0],
			}
		}
	}
	for i := 0; i < len(moHost.Config.StorageDevice.ScsiTopology.Adapter); i += 1 {
		ad := moHost.Config.StorageDevice.ScsiTopology.Adapter[i]
		adapter := adapterTable[ad.Adapter]
		for j := 0; j < len(ad.Target); j += 1 {
			t := ad.Target[j]
			key := t.Lun[0].ScsiLun
			if _, ok := enclosuresTable[key]; ok {
				adapter.Enclosure = int(t.Target)
			} else if _, ok := driversTable[key]; ok {
				driver := driversTable[key]
				driver.Slot = int(t.Target)
				adapter.Drivers = append(adapter.Drivers, driver)
			}
		}
	}
	return adapterList
}

func (self *SHost) GetStorageSizeMB() int {
	size := 0
	storages := self.GetStorageInfo()
	for i := 0; i < len(storages); i += 1 {
		size += storages[i].Size
	}
	return size
}

func (self *SHost) GetStorageType() string {
	ssd := 0
	rotate := 0
	storages := self.GetStorageInfo()
	for i := 0; i < len(storages); i += 1 {
		if storages[i].Rotate {
			rotate += 1
		} else {
			ssd += 1
		}
	}
	if ssd == 0 && rotate > 0 {
		return api.DISK_TYPE_ROTATE
	} else if ssd > 0 && rotate == 0 {
		return api.DISK_TYPE_SSD
	} else {
		return api.DISK_TYPE_HYBRID
	}
}

func (self *SHost) GetHostType() string {
	return api.HOST_TYPE_ESXI
}

func (self *SHost) GetIsMaintenance() bool {
	moHost := self.getHostSystem()
	return moHost.Summary.Runtime.InMaintenanceMode
}

func (self *SHost) GetVersion() string {
	moHost := self.getHostSystem()
	about := moHost.Summary.Config.Product
	return fmt.Sprintf("%s-%s", about.Version, about.Build)
}

func (self *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	nics := host.getNicInfo()
	inics := make([]cloudprovider.ICloudHostNetInterface, len(nics))
	for i := 0; i < len(nics); i += 1 {
		inics[i] = &nics[i]
	}
	return inics, nil
}

func (host *SHost) getLocalStorageCache() (*SDatastoreImageCache, error) {
	if host.storageCache == nil {
		sc, err := host.newLocalStorageCache()
		if err != nil {
			return nil, err
		}
		host.storageCache = sc
	}
	return host.storageCache, nil
}

func (host *SHost) newLocalStorageCache() (*SDatastoreImageCache, error) {
	ctx := context.Background()

	istorages, err := host.GetIStorages()
	if err != nil {
		return nil, err
	}
	var errmsg string
	var cacheDs *SDatastore
	var maxDs *SDatastore
	var maxCapacity int64
	for i := 0; i < len(istorages); i += 1 {
		ds := istorages[i].(*SDatastore)
		if !ds.isLocalVMFS() {
			continue
		}
		_, err := ds.CheckFile(ctx, IMAGE_CACHE_DIR_NAME)
		if err != nil {
			if err != cloudprovider.ErrNotFound {
				// return nil, err
				if len(errmsg) > 0 {
					errmsg += ","
				}
				errmsg += err.Error()
			} else if maxCapacity < ds.GetCapacityMB() {
				maxCapacity = ds.GetCapacityMB()
				maxDs = ds
			}
		} else {
			cacheDs = ds
			break
		}
	}
	if cacheDs == nil {
		// if no existing image cache dir found, use the one with maximal capacilty
		cacheDs = maxDs
	}

	if cacheDs == nil {
		return nil, fmt.Errorf(errmsg)
	}

	return &SDatastoreImageCache{
		datastore: cacheDs,
		host:      host,
	}, nil
}

func (host *SHost) GetManagementServerIp() string {
	return host.getHostSystem().Summary.ManagementServerIp
}

func (host *SHost) IsManagedByVCenter() bool {
	return len(host.getHostSystem().Summary.ManagementServerIp) > 0
}
