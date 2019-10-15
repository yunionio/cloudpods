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

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

var HOST_SYSTEM_PROPS = []string{"name", "parent", "summary", "config", "hardware", "vm", "datastore", "network"}

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

	parent *mo.ComputeResource

	networks []SNetwork
}

func NewHost(manager *SESXiClient, host *mo.HostSystem, dc *SDatacenter) *SHost {
	if host.Config == nil {
		log.Errorf("empty host config %s", host.Name)
		return nil
	}
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

	if moHost.Config == nil || moHost.Config.Network == nil {
		return nil
	}

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

func (self *SHost) DoCreateVM(ctx context.Context, ds *SDatastore, data *jsonutils.JSONDict) (*SVirtualMachine, error) {
	var (
		deviceChange = make([]types.BaseVirtualDeviceConfigSpec, 0, 5)
		name         string
		osName             = "Linux"
		memoryMB     int64 = 2048
		numCPUs      int32 = 2
		bios         string
	)

	if data.Contains("uuid") {
		name, _ = data.GetString("uuid")
	} else {
		name, _ = data.GetString("name")
	}
	if data.Contains("os_name") {
		osName, _ = data.GetString("os_name")
	}
	datastorePath := fmt.Sprintf("[%s] %s", ds.GetName(), name)
	if data.Contains("mem") {
		memoryMB, _ = data.Int("mem")
	}
	if data.Contains("cpu") {
		cpunum, _ := data.Int("cpu")
		numCPUs = int32(cpunum)
	}

	firmware := ""
	if data.Contains("bios") {
		bios, _ = data.GetString("bios")
	}
	if len(bios) == 0 {
		if bios == "BIOS" {
			firmware = "bios"
		} else if bios == "UEFI" {
			firmware = "efi"
		}
	}

	guestId := "rhel6_64Guest"
	if osName == "Windows" {
		guestId = "windows7Server64Guest"
	}

	version := "vmx-10"
	if self.isVersion50() {
		version = "vmx-08"
	}

	uuid, _ := data.GetString("uuid")

	spec := types.VirtualMachineConfigSpec{
		Name:     name,
		Version:  version,
		Uuid:     uuid,
		GuestId:  guestId,
		NumCPUs:  numCPUs,
		MemoryMB: memoryMB,
		Firmware: firmware,
	}
	spec.Files = &types.VirtualMachineFileInfo{
		VmPathName: datastorePath,
	}

	deviceChange = append(deviceChange, addDevSpec(NewIDEDev(200, 0)))
	deviceChange = append(deviceChange, addDevSpec(NewIDEDev(200, 1)))
	deviceChange = append(deviceChange, addDevSpec(NewSVGADev(500, 100)))
	disks, _ := data.GetArray("disks")
	driver := "scsi"
	if len(disks) > 0 {
		driver, _ = disks[0].GetString("driver")
	}
	if driver == "scsi" || driver == "pvscsi" {
		if self.isVersion50() {
			driver = "scsi"
		}
		deviceChange = append(deviceChange, addDevSpec(NewSCSIDev(1000, 100, driver)))
	}
	cdromPath := ""
	if data.Contains("cdrom") {
		tmp, _ := data.Get("cdrom")
		cdromPath, _ = tmp.GetString("path")
	}
	var err error
	if len(cdromPath) != 0 && !strings.HasPrefix(cdromPath, "[") {
		cdromPath, err = self.FileUrlPathToDsPath(cdromPath)
		if err != nil {
			return nil, errors.Wrapf(err, "SHost.FileUrlPathToDsPath for cdrom path '%s'", cdromPath)
		}
	}
	deviceChange = append(deviceChange, addDevSpec(NewCDROMDev(cdromPath, 16000, 201)))

	var (
		scsiIdx = 0
		ideIdx  = 0
		index   = 0
		ctrlKey = 0
	)
	for _, disk := range disks {
		imagePath, _ := disk.GetString("image_path")
		var size int64 = 0
		if len(imagePath) == 0 {
			size, _ := disk.Int("size")
			if size == 0 {
				size = 30 * 1024
			}
		} else {
			imagePath, err = self.FileUrlPathToDsPath(imagePath)
			if err != nil {
				return nil, errors.Wrapf(err, "SHost.FileUrlPathToDsPath for image path '%s'", imagePath)
			}
		}
		uuid, _ := disk.GetString("disk_id")
		driver := "scsi"
		if disk.Contains("driver") {
			driver, _ = disk.GetString("driver")
		}
		if driver == "scsi" || driver == "pyscsi" {
			if self.isVersion50() {
				driver = "scsi"
			}
			ctrlKey = 1000
			index = scsiIdx
			scsiIdx += 1
		} else {
			ctrlKey = 200 + ideIdx/2
			index = ideIdx % 2
			ideIdx += 1
		}
		spec := addDevSpec(NewDiskDev(size, imagePath, uuid, int32(index), 2000, int32(ctrlKey)))
		spec.FileOperation = "create"
		deviceChange = append(deviceChange, spec)
	}

	nics, _ := data.GetArray("nics")
	for _, nic := range nics {
		index, _ := nic.Int("index")
		mac, _ := nic.GetString("mac")
		driver := "e1000"
		if nic.Contains("driver") {
			driver, _ = nic.GetString("driver")
		}
		if self.isVersion50() {
			driver = "e1000"
		}
		var vlanId int64 = 1
		if nic.Contains("vlan") {
			vlanId, _ = nic.Int("vlan")
		}
		dev, err := NewVNICDev(self, mac, driver, int32(vlanId), 4000, 100, int32(index))
		if err != nil {
			return nil, errors.Wrap(err, "NewVNICDev")
		}
		deviceChange = append(deviceChange, addDevSpec(dev))
	}

	spec.DeviceChange = deviceChange
	dc, err := self.GetDatacenter()
	if err != nil {
		return nil, errors.Wrapf(err, "SHost.GetDatacenter for host '%s'", self.GetId())
	}
	// get vmFloder
	folders, err := dc.getObjectDatacenter().Folders(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "object.DataCenter.Folders")
	}
	vmFolder := folders.VmFolder
	resourcePool, err := self.GetResourcePool()
	if err != nil {
		return nil, errors.Wrap(err, "SHost.GetResourcePool")
	}
	task, err := vmFolder.CreateVM(ctx, spec, resourcePool, self.GetoHostSystem())
	if err != nil {
		return nil, errors.Wrap(err, "VmFolder.Create")
	}

	info, err := task.WaitForResult(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Task.WaitForResult")
	}

	var moVM mo.VirtualMachine
	err = self.manager.reference2Object(info.Result.(types.ManagedObjectReference), VIRTUAL_MACHINE_PROPS, &moVM)
	if err != nil {
		return nil, errors.Wrap(err, "fail to fetch virtual machine just created")
	}

	return NewVirtualMachine(self.manager, &moVM, self.datacenter), nil
}

func (host *SHost) isVersion50() bool {
	version := host.GetVersion()
	if strings.HasPrefix(version, "5.") {
		return true
	}
	return false
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

func (host *SHost) FindDataStoreById(id string) (*SDatastore, error) {
	datastores, err := host.GetDataStores()
	if err != nil {
		return nil, err
	}
	for i := range datastores {
		if datastores[i].GetId() == id {
			return datastores[i].(*SDatastore), nil
		}
	}
	return nil, fmt.Errorf("no such datastore %s", id)
}

func (host *SHost) GetDataStores() ([]cloudprovider.ICloudStorage, error) {
	err := host.fetchDatastores()
	if err != nil {
		return nil, err
	}
	return host.datastores, nil
}

func (host *SHost) fetchDatastores() error {
	if host.datastores != nil {
		return nil
	}

	dc, err := host.GetDatacenter()
	if err != nil {
		return err
	}

	MAX_TRIES := 3
	for tried := 0; tried < MAX_TRIES; tried += 1 {
		hostDss := host.getHostSystem().Datastore
		if len(hostDss) == 0 {
			// log.Errorf("host VMs are nil!!!!!")
			return nil
		}

		dss, err := dc.fetchDatastores(hostDss)
		if err != nil {
			log.Errorf("dc.fetchVms fail %s", err)
			time.Sleep(time.Second)
			host.Refresh()
			continue
		}
		host.datastores = dss
	}
	return nil
}

func (host *SHost) FileUrlPathToDsPath(path string) (string, error) {
	var newPath string
	dss, err := host.GetDataStores()
	if err != nil {
		return newPath, err
	}
	for _, ds := range dss {
		rds := ds.(*SDatastore)
		if strings.HasPrefix(path, rds.GetUrl()) {
			newPath = fmt.Sprintf("[%s] %s", ds.GetName(), path[len(rds.GetUrl()):])
		}
		break
	}
	return newPath, nil
}

func (host *SHost) FindNetworkByVlanID(vlanID int32) (IVMNetwork, error) {
	if vlanID > 1 && vlanID < 4095 {
		return host.findVlanDVPG(int32(vlanID))
	}
	n, err := host.findNovlanDVPG()
	if err != nil {
		return nil, err
	}
	if n != nil {
		return n, err
	}
	return host.findBasicNetwork()
}

func (host *SHost) findBasicNetwork() (*SNetwork, error) {
	nets, err := host.GetNetwork()
	if err != nil {
		return nil, err
	}
	if len(nets) == 0 {
		return nil, nil
	}
	return &nets[0], nil
}

func (host *SHost) GetNetwork() ([]SNetwork, error) {
	if host.networks != nil {
		return host.networks, nil
	}
	netMobs := host.getHostSystem().Network
	nets := make([]SNetwork, 0)
	err := host.manager.references2Objects(netMobs, NETWORK_PROPS, &nets)
	if err != nil {
		return nil, errors.Wrap(err, "references2Objects")
	}
	host.networks = nets
	return host.networks, nil
}

func (host *SHost) findNovlanDVPG() (*SDistributedVirtualPortgroup, error) {
	nets, err := host.datacenter.GetNetworks()
	if err != nil {
		return nil, errors.Wrap(err, "SHost.datacenter.GetNetworks")
	}
	for _, net := range nets {
		dvpg, ok := net.(*SDistributedVirtualPortgroup)
		if !ok || !dvpg.ContainHost(host) || len(dvpg.GetActivePorts()) == 0 {
			continue
		}
		nvlan := dvpg.GetVlanId()
		if nvlan <= 1 || nvlan >= 4095 {
			return dvpg, nil
		}
	}
	return nil, nil
}

func (host *SHost) findVlanDVPG(vlanId int32) (*SDistributedVirtualPortgroup, error) {
	nets, err := host.datacenter.GetNetworks()
	if err != nil {
		return nil, errors.Wrap(err, "SHost.datacenter.GetNetworks")
	}
	for _, net := range nets {
		dvpg, ok := net.(*SDistributedVirtualPortgroup)
		if !ok || !dvpg.ContainHost(host) || len(dvpg.GetActivePorts()) == 0 {
			continue
		}
		nvlan := dvpg.GetVlanId()
		if nvlan == vlanId {
			return dvpg, nil
		}
	}
	return nil, nil
}

func (host *SHost) GetoHostSystem() *object.HostSystem {
	return object.NewHostSystem(host.manager.client.Client, host.getHostSystem().Reference())
}

func (host *SHost) GetResourcePool() (*object.ResourcePool, error) {
	var err error
	if host.parent == nil {
		host.parent, err = host.getResourcePool()
		if err != nil {
			return nil, err
		}
	}
	return object.NewResourcePool(host.manager.client.Client, *host.parent.ResourcePool), nil
}

func (host *SHost) getResourcePool() (*mo.ComputeResource, error) {
	var mcr *mo.ComputeResource
	var parent interface{}

	moHost := host.getHostSystem()

	switch moHost.Parent.Type {
	case "ComputeResource":
		mcr = new(mo.ComputeResource)
		parent = mcr
	case "ClusterComputeResource":
		mcc := new(mo.ClusterComputeResource)
		mcr = &mcc.ComputeResource
		parent = mcc
	default:
		return nil, errors.Error(fmt.Sprintf("unknown host parent type: %s", moHost.Parent.Type))
	}

	err := host.manager.reference2Object(*moHost.Parent, []string{"resourcePool"}, parent)
	if err != nil {
		return nil, errors.Wrap(err, "SESXiClient.reference2Object")
	}
	return mcr, nil
}

func (host *SHost) GetCluster() (*mo.ComputeResource, error) {
	return host.getResourcePool()
}

func (host *SHost) GetSiblingHosts() ([]*SHost, error) {
	rp, err := host.GetCluster()
	if err != nil {
		return nil, err
	}
	moHosts := make([]mo.HostSystem, 0, len(rp.Host))
	err = host.manager.references2Objects(rp.Host, HOST_SYSTEM_PROPS, &moHosts)
	if err != nil {
		return nil, errors.Wrap(err, "SESXiClient.references2Objects")
	}

	ret := make([]*SHost, len(moHosts))
	for i := range moHosts {
		ret[i] = NewHost(host.manager, &moHosts[i], host.datacenter)
	}
	return ret, nil
}
