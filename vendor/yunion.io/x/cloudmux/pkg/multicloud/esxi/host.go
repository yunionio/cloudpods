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
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/reflectutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

var (
	True  bool = true
	False bool = false
)

var (
	hostConfigProps   = []string{"config.network", "config.storageDevice"}
	hostSummaryProps  = []string{"summary.runtime", "summary.hardware", "summary.config.product", "summary.managementServerIp"}
	hostHardWareProps = []string{"hardware.systemInfo"}
)

var HOST_SYSTEM_PROPS []string

func init() {
	HOST_SYSTEM_PROPS = []string{"name", "parent", "vm", "datastore", "network"}
	HOST_SYSTEM_PROPS = append(HOST_SYSTEM_PROPS, hostConfigProps...)
	HOST_SYSTEM_PROPS = append(HOST_SYSTEM_PROPS, hostSummaryProps...)
	HOST_SYSTEM_PROPS = append(HOST_SYSTEM_PROPS, hostHardWareProps...)
}

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

	masterIp string

	nicInfo      []sHostNicInfo
	storageInfo  []SHostStorageInfo
	datastores   []cloudprovider.ICloudStorage
	storageCache *SDatastoreImageCache
	vms          []cloudprovider.ICloudVM
	parent       *mo.ComputeResource
	networks     []IVMNetwork
	tempalteVMs  []*SVirtualMachine
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

func (host *SHost) GetName() string {
	return formatName(host.SManagedObject.GetName())
}

func (host *SHost) GetSchedtags() ([]string, error) {
	clusters, err := host.datacenter.listClusters()
	if err != nil {
		return nil, err
	}
	cpName := host.datacenter.manager.cpcfg.Name
	reference := host.GetHostSystem().Reference()
	tags := make([]string, 0, 1)
	oDatacenter := host.datacenter.getDatacenter()
Loop:
	for i := range clusters {
		oc := clusters[i].getoCluster()
		if len(oc.Host) == 0 {
			continue
		}
		for _, h := range oc.Host {
			if h == reference {
				tags = append(tags, fmt.Sprintf("cluster:/%s/%s/%s", cpName, oDatacenter.Name, oc.Name))
				continue Loop
			}
		}
	}
	return tags, nil
}

func (host *SHost) getHostSystem() *mo.HostSystem {
	return host.object.(*mo.HostSystem)
}

func (host *SHost) GetGlobalId() string {
	return host.GetAccessIp()
}

func (host *SHost) GetStatus() string {
	/*
		HostSystemPowerStatePoweredOn  = HostSystemPowerState("poweredOn")
		HostSystemPowerStatePoweredOff = HostSystemPowerState("poweredOff")
		HostSystemPowerStateStandBy    = HostSystemPowerState("standBy")
		HostSystemPowerStateUnknown    = HostSystemPowerState("unknown")
	*/
	switch host.getHostSystem().Summary.Runtime.PowerState {
	case types.HostSystemPowerStatePoweredOn:
		return api.HOST_STATUS_RUNNING
	case types.HostSystemPowerStatePoweredOff:
		return api.HOST_STATUS_READY
	default:
		return api.HOST_STATUS_UNKNOWN
	}
}

func (host *SHost) Refresh() error {
	base := host.SManagedObject
	var moObj mo.HostSystem
	err := host.manager.reference2Object(host.object.Reference(), HOST_SYSTEM_PROPS, &moObj)
	if err != nil {
		return err
	}
	base.object = &moObj
	*host = SHost{}
	host.SManagedObject = base
	return nil
}

func (host *SHost) IsEmulated() bool {
	return false
}

func (host *SHost) fetchVMs(all bool) error {
	if host.vms != nil {
		return nil
	}

	dc, err := host.GetDatacenter()
	if err != nil {
		return errors.Wrapf(err, "GetDatacenter")
	}

	var vms []*SVirtualMachine
	for i := 1; i < 3; i++ {
		hostVms := host.getHostSystem().Vm
		if len(hostVms) == 0 {
			return nil
		}

		vms, err = dc.fetchVms(hostVms, all)
		if err != nil {
			e := errors.Cause(err)
			// 机器刚删除时, hostVms若不刷新, 会有类似错误: ServerFaultCode: The object 'vim.VirtualMachine:vm-1053' has already been deleted or has not been completely created
			// https://github.com/vmware/govmomi/pull/1916/files
			if soap.IsSoapFault(e) {
				_, ok := soap.ToSoapFault(e).VimFault().(types.ManagedObjectNotFound)
				if ok {
					time.Sleep(time.Second * 10)
					host.Refresh()
					continue
				}
			}
			return errors.Wrapf(err, "dc.fetchVMs")
		}
	}
	if err != nil {
		return errors.Wrapf(err, "dc.fetchVms")
	}

	for _, vm := range vms {
		if vm.IsTemplate() {
			host.tempalteVMs = append(host.tempalteVMs, vm)
		} else {
			host.vms = append(host.vms, vm)
		}
	}
	return nil
}

func (host *SHost) GetIVMs2() ([]cloudprovider.ICloudVM, error) {
	err := host.fetchVMs(true)
	if err != nil {
		return nil, errors.Wrapf(err, "fetchVMs")
	}
	return host.vms, nil
}

func (host *SHost) GetTemplateVMs() ([]*SVirtualMachine, error) {
	err := host.fetchVMs(false)
	if err != nil {
		return nil, errors.Wrap(err, "fetchVMs")
	}
	return host.tempalteVMs, nil
}

func (host *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	err := host.fetchVMs(false)
	if err != nil {
		return nil, err
	}
	return host.vms, nil
}

func (host *SHost) GetTemplateVMById(id string) (*SVirtualMachine, error) {
	id = host.manager.getPrivateId(id)
	temVms, err := host.GetTemplateVMs()
	if err != nil {
		return nil, err
	}
	for i := range temVms {
		if temVms[i].GetGlobalId() == id {
			return temVms[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (host *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	id = host.manager.getPrivateId(id)

	vms, err := host.GetIVMs()
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

func (host *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return host.GetDataStores()
}

func (host *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	istorages, err := host.GetIStorages()
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

func (host *SHost) GetEnabled() bool {
	if host.getHostSystem().Summary.Runtime.InMaintenanceMode {
		return false
	}
	return true
}

func (host *SHost) GetHostStatus() string {
	/*
		HostSystemConnectionStateConnected     = HostSystemConnectionState("connected")
		HostSystemConnectionStateNotResponding = HostSystemConnectionState("notResponding")
		HostSystemConnectionStateDisconnected  = HostSystemConnectionState("disconnected")
	*/
	if host.getHostSystem().Summary.Runtime.InMaintenanceMode {
		return api.HOST_OFFLINE
	}
	switch host.getHostSystem().Summary.Runtime.ConnectionState {
	case types.HostSystemConnectionStateConnected:
		return api.HOST_ONLINE
	default:
		return api.HOST_OFFLINE
	}
}

func findHostNicByMac(nicInfoList []sHostNicInfo, mac string) *sHostNicInfo {
	for i := 0; i < len(nicInfoList); i += 1 {
		if nicInfoList[i].Mac == mac {
			return &nicInfoList[i]
		}
	}
	return nil
}

func (host *SHost) getAdminNic() *sHostNicInfo {
	nics := host.getNicInfo(false)
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

func (host *SHost) getNicInfo(debug bool) []sHostNicInfo {
	if host.nicInfo == nil {
		host.nicInfo = host.fetchNicInfo(debug)
	}
	return host.nicInfo
}

func mask2len(mask string) int8 {
	maskAddr, _ := netutils.NewIPV4Addr(mask)
	return netutils.Mask2Len(maskAddr)
}

func (host *SHost) isVnicAdmin(nic types.HostVirtualNic) bool {
	if len(host.masterIp) > 0 {
		if host.masterIp == nic.Spec.Ip.IpAddress {
			return true
		} else {
			return false
		}
	}
	exist, err := host.manager.IsHostIpExists(nic.Spec.Ip.IpAddress)
	if err != nil {
		log.Errorf("IsHostIpExists %s fail %s", nic.Spec.Ip.IpAddress, err)
		return false
	}
	if exist {
		host.masterIp = nic.Spec.Ip.IpAddress
		return true
	}
	return false
}

func (host *SHost) fetchNicInfo(debug bool) []sHostNicInfo {
	moHost := host.getHostSystem()

	if moHost.Config == nil || moHost.Config.Network == nil {
		return nil
	}

	if debug {
		log.Debugf("%s", jsonutils.Marshal(moHost.Config.Network).PrettyString())
	}

	nicInfoList := make([]sHostNicInfo, 0)

	for _, nic := range moHost.Config.Network.Pnic {
		// log.Debugf("pnic %d: %s %#v", i, jsonutils.Marshal(nic), nic)
		info := sHostNicInfo{
			host: host,
		}
		info.Dev = nic.Device
		info.Driver = nic.Driver
		info.Mac = netutils.FormatMacAddr(nic.Mac)
		info.Index = int8(len(nicInfoList))
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

	for _, nic := range vnics {
		// log.Debugf("vnic %d: %s %#v", i, jsonutils.Marshal(nic), nic)
		mac := netutils.FormatMacAddr(nic.Spec.Mac)
		pnic := findHostNicByMac(nicInfoList, mac)
		if pnic != nil {
			// findMaster = true
			pnic.IpAddr = nic.Spec.Ip.IpAddress
			pnic.IpAddrPrefixLen = mask2len(nic.Spec.Ip.SubnetMask)
			if nic.Spec.Ip.IpV6Config != nil && len(nic.Spec.Ip.IpV6Config.IpV6Address) > 0 {
				pnic.IpAddr6 = nic.Spec.Ip.IpV6Config.IpV6Address[0].IpAddress
				pnic.IpAddr6PrefixLen = int8(nic.Spec.Ip.IpV6Config.IpV6Address[0].PrefixLength)
			}
			if host.isVnicAdmin(nic) {
				pnic.NicType = api.NIC_TYPE_ADMIN
			}
			pnic.LinkUp = true
			pnic.Mtu = nic.Spec.Mtu
		} else {
			info := sHostNicInfo{}
			info.Dev = nic.Device
			info.Driver = "vmkernel"
			info.Mac = mac
			info.Index = int8(len(nicInfoList))
			info.LinkUp = true
			info.IpAddr = nic.Spec.Ip.IpAddress
			info.IpAddrPrefixLen = mask2len(nic.Spec.Ip.SubnetMask)
			if nic.Spec.Ip.IpV6Config != nil && len(nic.Spec.Ip.IpV6Config.IpV6Address) > 0 {
				info.IpAddr6 = nic.Spec.Ip.IpV6Config.IpV6Address[0].IpAddress
				info.IpAddr6PrefixLen = int8(nic.Spec.Ip.IpV6Config.IpV6Address[0].PrefixLength)
			}
			info.Mtu = nic.Spec.Mtu
			if host.isVnicAdmin(nic) {
				info.NicType = api.NIC_TYPE_ADMIN
			}
			nicInfoList = append(nicInfoList, info)
		}
	}

	visited := make(map[string]IVMNetwork)
	dc, err := host.GetDatacenter()
	if err != nil {
		log.Errorf("fetchNicInfo GetDatacenter fail %s", err)
	} else {
		// map portgroup
		for _, pg := range moHost.Config.Network.Portgroup {
			// log.Debugf("portgroup: %s %#v", jsonutils.Marshal(pg), pg)
			netName := pg.Spec.Name
			net, _ := dc.getNetworkByName(netName)
			if net != nil {
				visited[net.GetName()] = net
				info := sHostNicInfo{
					host:    host,
					network: net,
				}
				info.Dev = pg.Spec.VswitchName
				info.Driver = "portgroup"
				info.LinkUp = true
				info.Mtu = 1500
				info.Index = int8(len(nicInfoList))
				info.VlanId = int(pg.Spec.VlanId)
				nicInfoList = append(nicInfoList, info)
			}
		}
	}

	networks, err := host.GetNetworks()
	if err != nil {
		log.Errorf("GetNetworks %s", err)
	} else {
		for i := range networks {
			net := networks[i]

			if _, ok := visited[net.GetName()]; ok {
				continue
			}
			visited[net.GetName()] = net

			info := sHostNicInfo{
				host:    host,
				network: net,
			}
			info.Dev = net.GetVswitchName()
			info.Driver = net.GetType()
			info.LinkUp = true
			info.Mtu = 1500
			info.Index = int8(len(nicInfoList))
			info.VlanId = int(net.GetVlanId())
			nicInfoList = append(nicInfoList, info)
		}
	}

	return nicInfoList
}

func (host *SHost) GetAccessIp() string {
	adminNic := host.getAdminNic()
	if adminNic != nil {
		return adminNic.IpAddr
	}
	return ""
}

func (host *SHost) GetAccessMac() string {
	adminNic := host.getAdminNic()
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

func (host *SHost) GetSysInfo() jsonutils.JSONObject {
	sysinfo := SSysInfo{}
	hostsys := host.getHostSystem()
	sysinfo.Manufacture = hostsys.Summary.Hardware.Vendor
	sysinfo.Model = hostsys.Summary.Hardware.Model
	if hostsys.Hardware != nil {
		sysinfo.SerialNumber = hostsys.Hardware.SystemInfo.SerialNumber
	}
	return jsonutils.Marshal(&sysinfo)
}

func (host *SHost) GetSN() string {
	hostsys := host.getHostSystem()
	if hostsys.Hardware != nil {
		return hostsys.Hardware.SystemInfo.SerialNumber
	}
	return ""
}

func (host *SHost) GetCpuCount() int {
	return int(host.getHostSystem().Summary.Hardware.NumCpuThreads)
}

func (host *SHost) GetNodeCount() int8 {
	return int8(host.getHostSystem().Summary.Hardware.NumCpuPkgs)
}

func (host *SHost) GetCpuDesc() string {
	return host.getHostSystem().Summary.Hardware.CpuModel
}

func (host *SHost) GetCpuMhz() int {
	return int(host.getHostSystem().Summary.Hardware.CpuMhz)
}

func (host *SHost) GetMemSizeMB() int {
	return int(host.getHostSystem().Summary.Hardware.MemorySize / 1024 / 1024)
}

func (host *SHost) GetStorageInfo() []SHostStorageInfo {
	if host.storageInfo == nil {
		host.storageInfo = host.getStorageInfo()
	}
	return host.storageInfo
}

func (host *SHost) getStorageInfo() []SHostStorageInfo {
	diskSlots := make(map[int]SHostStorageInfo)
	list := host.getStorages()
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

func (host *SHost) getStorages() []*SHostStorageAdapterInfo {
	adapterList := make([]*SHostStorageAdapterInfo, 0)
	adapterTable := make(map[string]*SHostStorageAdapterInfo)
	driversTable := make(map[string]*SHostStorageDriverInfo, 0)
	enclosuresTable := make(map[string]*SHostStorageEnclosureInfo, 0)
	moHost := host.getHostSystem()

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

func (host *SHost) GetStorageSizeMB() int {
	storages, err := host.GetIStorages()
	if err != nil {
		log.Errorf("SHost.GetStorageSizeMB: SHost.GetIStorages: %s", err)
		return 0
	}
	var size int64
	for _, stor := range storages {
		size += stor.GetCapacityMB()
	}
	return int(size)
}

func (host *SHost) GetStorageType() string {
	ssd := 0
	rotate := 0
	storages := host.GetStorageInfo()
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

func (host *SHost) GetHostType() string {
	return api.HOST_TYPE_ESXI
}

func (host *SHost) GetIsMaintenance() bool {
	moHost := host.getHostSystem()
	return moHost.Summary.Runtime.InMaintenanceMode
}

func (host *SHost) GetVersion() string {
	moHost := host.getHostSystem()
	about := moHost.Summary.Config.Product
	return fmt.Sprintf("%s-%s", about.Version, about.Build)
}

func (host *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

type SCreateVMParam struct {
	Name                 string
	Uuid                 string
	OsName               string
	CpuSockets           int
	Cpu                  int
	Mem                  int
	Bios                 string
	Cdrom                SCdromInfo
	Disks                []SDiskInfo
	Nics                 []jsonutils.JSONObject
	ResourcePool         string
	InstanceSnapshotInfo SEsxiInstanceSnapshotInfo
}

type SEsxiInstanceSnapshotInfo struct {
	InstanceSnapshotId string
	InstanceId         string
}

type SCdromInfo struct {
	ImageId string
	Path    string
	Name    string
	Size    string
}

type SDiskInfo struct {
	ImagePath     string
	Size          int64
	DiskId        string
	Driver        string
	ImageInfo     SEsxiImageInfo
	StorageId     string
	Preallocation string
}

type SEsxiImageInfo struct {
	ImageType          string
	ImageExternalId    string
	StorageCacheHostIp string
}

func (host *SHost) CreateVM2(ctx context.Context, ds *SDatastore, params SCreateVMParam) (needDeploy bool, vm *SVirtualMachine, err error) {
	needDeploy = true
	var temvm *SVirtualMachine
	if len(params.InstanceSnapshotInfo.InstanceSnapshotId) > 0 {
		temvm, err = host.manager.SearchVM(params.InstanceSnapshotInfo.InstanceId)
		if err != nil {
			err = errors.Wrapf(err, "can't find vm %q, please sync status for vm or sync cloudaccount", params.InstanceSnapshotInfo.InstanceId)
		}
		var isp cloudprovider.ICloudInstanceSnapshot
		isp, err = temvm.GetInstanceSnapshot(params.InstanceSnapshotInfo.InstanceSnapshotId)
		if err != nil {
			err = errors.Wrap(err, "unable to GetInstanceSnapshot")
			return
		}
		sp := isp.(*SVirtualMachineSnapshot)
		vm, err = host.CloneVM(ctx, temvm, &sp.snapshotTree.Snapshot, ds, params)
		return
	}
	if len(params.Disks) == 0 {
		err = errors.Error("empty disk config")
		return
	}
	imageInfo := params.Disks[0].ImageInfo
	if imageInfo.ImageType == string(cloudprovider.ImageTypeSystem) {
		temvm, err = host.manager.SearchTemplateVM(imageInfo.ImageExternalId)
		if err != nil {
			err = errors.Wrapf(err, "SEsxiClient.SearchTemplateVM for image %q", imageInfo.ImageExternalId)
			return
		}
		vm, err = host.CloneVM(ctx, temvm, nil, ds, params)
		return
	}
	return host.DoCreateVM(ctx, ds, params)
}

func (host *SHost) SearchTemplateVM(id string) (*SVirtualMachine, error) {
	return host.manager.SearchTemplateVM(id)
}

func (host *SHost) needScsi(disks []SDiskInfo) bool {
	if len(disks) == 0 {
		return false
	}
	for i := range disks {
		driver := disks[i].Driver
		if driver == "" || driver == "scsi" || driver == "pvscsi" {
			return true
		}
	}
	return false
}

func (host *SHost) addDisks(ctx context.Context, dc *SDatacenter, ds *SDatastore, disks []SDiskInfo, uuid string, objectVm *object.VirtualMachine) (*SVirtualMachine, error) {
	getVM := func() (*SVirtualMachine, error) {
		var moVM mo.VirtualMachine
		err := host.manager.reference2Object(objectVm.Reference(), VIRTUAL_MACHINE_PROPS, &moVM)
		if err != nil {
			return nil, errors.Wrap(err, "fail to fetch virtual machine just created")
		}

		evm := NewVirtualMachine(host.manager, &moVM, host.datacenter)
		if evm == nil {
			return nil, errors.Error("create successfully but unable to NewVirtualMachine")
		}
		return evm, nil
	}

	if len(disks) == 0 {
		return getVM()
	}

	var (
		scsiIdx    = 0
		ideIdx     = 0
		ide1un     = 0
		ide2un     = 1
		unitNumber = 0
		ctrlKey    = 0
	)
	deviceChange := make([]types.BaseVirtualDeviceConfigSpec, 0, 1)
	// add disks
	var rootDiskSizeMb int64
	for i, disk := range disks {
		imagePath := disk.ImagePath
		var size = disk.Size
		if len(imagePath) == 0 {
			if size == 0 {
				size = 30 * 1024
			}
		} else {
			var err error
			imagePath, err = host.FileUrlPathToDsPath(imagePath)
			if err != nil {
				return nil, errors.Wrapf(err, "SHost.FileUrlPathToDsPath")
			}
			newImagePath := fmt.Sprintf("[%s] %s/%s.vmdk", ds.GetRelName(), uuid, uuid)

			err = host.copyVirtualDisk(imagePath, newImagePath, disk.Driver)
			if err != nil {
				return nil, err
			}
			imagePath = newImagePath
			rootDiskSizeMb = size
		}
		uuid, driver := disk.DiskId, "scsi"
		if len(disk.Driver) > 0 {
			driver = disk.Driver
		}
		if driver == "scsi" || driver == "pvscsi" {
			if host.isVersion50() {
				driver = "scsi"
			}
			ctrlKey = 1000
			unitNumber = scsiIdx
			scsiIdx += 1
			if scsiIdx == 7 {
				scsiIdx++
			}
		} else {
			ideno := ideIdx % 2
			if ideno == 0 {
				unitNumber = ideIdx/2 + ide1un
			} else {
				unitNumber = ideIdx/2 + ide2un
			}
			ctrlKey = 200 + ideno
			ideIdx += 1
		}
		var tds *SDatastore
		var err error
		if disk.StorageId != "" {
			tds, err = host.FindDataStoreById(disk.StorageId)
			if err != nil {
				return nil, errors.Wrapf(err, "unable to find ds %s from host %s", disk.StorageId, host.masterIp)
			}
		} else {
			tds = ds
		}
		log.Debugf("ds: %s, size: %d, image path: %s, uuid: %s, index: %d, ctrlKey: %d, driver: %s, key: %d.", tds.getDatastoreObj().String(), size, imagePath, uuid, unitNumber, ctrlKey, disk.Driver, 2000+i)
		spec := addDevSpec(NewDiskDev(size, SDiskConfig{
			SizeMb:        size,
			Uuid:          uuid,
			ControllerKey: int32(ctrlKey),
			UnitNumber:    int32(unitNumber),
			Key:           int32(2000 + i),
			ImagePath:     imagePath,
			IsRoot:        i == 0,
			Datastore:     tds,
			Preallocation: disk.Preallocation,
		}))
		if len(imagePath) == 0 {
			spec.FileOperation = "create"
		}
		deviceChange = append(deviceChange, spec)
	}
	log.Infof("deviceChange: %s", jsonutils.Marshal(deviceChange))

	configSpec := types.VirtualMachineConfigSpec{}
	configSpec.DeviceChange = deviceChange
	task, err := objectVm.Reconfigure(ctx, configSpec)
	if err != nil {
		return nil, errors.Wrap(err, "unable to reconfigure")
	}
	err = task.Wait(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "task.Wait")
	}

	evm, err := getVM()
	if err != nil {
		return nil, err
	}

	// resize root disk
	if rootDiskSizeMb > 0 && int64(evm.vdisks[0].GetDiskSizeMB()) != rootDiskSizeMb {
		err = evm.vdisks[0].Resize(ctx, rootDiskSizeMb)
		if err != nil {
			return evm, errors.Wrap(err, "resize for root disk")
		}
	}
	return evm, nil
}

func (host *SHost) copyVirtualDisk(srcPath, dstPath, diskDriver string) error {
	dm := object.NewVirtualDiskManager(host.manager.client.Client)
	spec := &types.VirtualDiskSpec{
		DiskType: "thin",
	}
	switch diskDriver {
	case "", "scsi", "pvscsi":
		spec.AdapterType = "lsiLogic"
	default:
		spec.AdapterType = "ide"
	}
	task, err := dm.CopyVirtualDisk(host.manager.context, srcPath, host.datacenter.getDcObj(), dstPath, host.datacenter.getDcObj(), spec, true)
	if err != nil {
		return errors.Wrap(err, "unable to CopyVirtualDisk")
	}
	err = task.Wait(host.manager.context)
	if err == nil {
		return nil
	}
	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "the requested operation is not implemented by the server") {
		return errors.Wrap(err, "wait CopyVirtualDiskTask")
	}
	task, err = dm.CopyVirtualDisk(host.manager.context, srcPath, host.datacenter.getDcObj(), dstPath, host.datacenter.getDcObj(), nil, true)
	if err != nil {
		return errors.Wrap(err, "unable to CopyVirtualDisk")
	}
	err = task.Wait(host.manager.context)
	if err != nil {
		return errors.Wrap(err, "wait CopyVirtualDiskTask")
	}
	return nil
}

func (host *SHost) DoCreateVM(ctx context.Context, ds *SDatastore, params SCreateVMParam) (needDeploy bool, vm *SVirtualMachine, err error) {
	needDeploy = true
	deviceChange := make([]types.BaseVirtualDeviceConfigSpec, 0, 5)

	// uuid first
	name := params.Name
	if len(params.Uuid) != 0 {
		name = params.Uuid
	}
	datastorePath := fmt.Sprintf("[%s] ", ds.GetRelName())

	firmware := ""
	if len(params.Bios) != 0 {
		if params.Bios == "BIOS" {
			firmware = "bios"
		} else if params.Bios == "UEFI" {
			firmware = "efi"
		}
	}

	guestId := "rhel6_64Guest"
	if params.OsName == "Windows" {
		guestId = "windows7Server64Guest"
	}

	version := "vmx-10"
	if host.isVersion50() {
		version = "vmx-08"
	}

	if params.CpuSockets == 0 {
		params.CpuSockets = 1
	}

	spec := types.VirtualMachineConfigSpec{
		Name:              name,
		Version:           version,
		Uuid:              params.Uuid,
		GuestId:           guestId,
		NumCPUs:           int32(params.Cpu),
		NumCoresPerSocket: int32(params.CpuSockets),
		MemoryMB:          int64(params.Mem),
		Firmware:          firmware,

		CpuHotAddEnabled:    &True,
		CpuHotRemoveEnabled: &True,
		MemoryHotAddEnabled: &True,
	}
	spec.Files = &types.VirtualMachineFileInfo{
		VmPathName: datastorePath,
	}

	deviceChange = append(deviceChange, addDevSpec(NewIDEDev(200, 0)))
	deviceChange = append(deviceChange, addDevSpec(NewIDEDev(200, 1)))
	deviceChange = append(deviceChange, addDevSpec(NewSVGADev(500, 100)))

	if host.needScsi(params.Disks) {
		driver := "pvscsi"
		if host.isVersion50() {
			driver = "scsi"
		}
		deviceChange = append(deviceChange, addDevSpec(NewSCSIDev(1000, 100, driver)))
	}
	cdromPath := params.Cdrom.Path
	if len(cdromPath) > 0 {
		needDeploy = false
		cdromPath, err = host.FileUrlPathToDsPath(cdromPath)
		if err != nil {
			err = errors.Wrapf(err, "SHost.FileUrlPathToDsPath for cdrom path")
			return
		}
	}
	deviceChange = append(deviceChange, addDevSpec(NewCDROMDev(cdromPath, 16000, 201)))

	// add usb to support mouse
	usbController := addDevSpec(NewUSBController(nil))
	deviceChange = append(deviceChange, usbController)

	nics := params.Nics
	for _, nic := range nics {
		index, _ := nic.Int("index")
		mac, _ := nic.GetString("mac")
		bridge, _ := nic.GetString("bridge")
		driver := "e1000"
		if nic.Contains("driver") {
			driver, _ = nic.GetString("driver")
		}
		if host.isVersion50() {
			driver = "e1000"
		}
		var vlanId int64 = 1
		if nic.Contains("vlan") {
			vlanId, _ = nic.Int("vlan")
		}
		dev, err := NewVNICDev(host, mac, driver, bridge, int32(vlanId), 4000, 100, int32(index))
		if err != nil {
			return needDeploy, nil, errors.Wrap(err, "NewVNICDev")
		}
		deviceChange = append(deviceChange, addDevSpec(dev))
	}

	spec.DeviceChange = deviceChange
	dc, err := host.GetDatacenter()
	if err != nil {
		err = errors.Wrapf(err, "SHost.GetDatacenter for host '%s'", host.GetId())
		return
	}
	// get vmFloder
	folders, err := dc.getObjectDatacenter().Folders(ctx)
	if err != nil {
		err = errors.Wrap(err, "object.DataCenter.Folders")
		return
	}
	vmFolder := folders.VmFolder
	resourcePool, err := host.SyncResourcePool(params.ResourcePool)
	if err != nil {
		err = errors.Wrap(err, "SyncResourcePool")
		return
	}
	task, err := vmFolder.CreateVM(ctx, spec, resourcePool, host.GetHostSystem())
	if err != nil {
		err = errors.Wrap(err, "VmFolder.Create")
		return
	}

	info, err := task.WaitForResult(ctx, nil)
	if err != nil {
		err = errors.Wrap(err, "Task.WaitForResult")
		return
	}
	vmRef := info.Result.(types.ManagedObjectReference)
	objectVM := object.NewVirtualMachine(host.manager.client.Client, vmRef)
	vm, err = host.addDisks(ctx, dc, ds, params.Disks, params.Uuid, objectVM)
	return
}

// If snapshot is not nil, params.Disks will be ignored
func (host *SHost) CloneVM(ctx context.Context, from *SVirtualMachine, snapshot *types.ManagedObjectReference, ds *SDatastore, params SCreateVMParam) (*SVirtualMachine, error) {
	ovm := from.getVmObj()

	deviceChange := make([]types.BaseVirtualDeviceConfigSpec, 0, 3)

	macAddrs := []string{}
	for _, nic := range params.Nics {
		index, _ := nic.Int("index")
		mac, _ := nic.GetString("mac")
		macAddrs = append(macAddrs, mac)
		bridge, _ := nic.GetString("bridge")
		driver := "e1000"
		if nic.Contains("driver") {
			driver, _ = nic.GetString("driver")
		}
		if host.isVersion50() {
			driver = "e1000"
		}
		var vlanId int64 = 1
		if nic.Contains("vlan") {
			vlanId, _ = nic.Int("vlan")
		}
		dev, err := NewVNICDev(host, mac, driver, bridge, int32(vlanId), 4000, 100, int32(index))
		if err != nil {
			return nil, errors.Wrap(err, "NewVNICDev")
		}
		deviceChange = append(deviceChange, &types.VirtualDeviceConfigSpec{
			Operation: types.VirtualDeviceConfigSpecOperationAdd,
			Device:    dev,
		})
	}

	if len(params.Disks) > 0 && snapshot == nil {
		driver := params.Disks[0].Driver
		if driver == "scsi" || driver == "pvscsi" {
			scsiDevs, err := from.FindController(ctx, "scsi")
			if err != nil {
				return nil, errors.Wrap(err, "SVirtualMachine.FindController")
			}
			if len(scsiDevs) == 0 {
				key := from.FindMinDiffKey(1000)
				driver := "pvscsi"
				if host.isVersion50() {
					driver = "scsi"
				}
				deviceChange = append(deviceChange, addDevSpec(NewSCSIDev(key, 100, driver)))
			}
		} else {
			ideDevs, err := from.FindController(ctx, "ide")
			if err != nil {
				return nil, errors.Wrap(err, "SVirtualMachine.FindController")
			}
			if len(ideDevs) == 0 {
				// add ide driver
				deviceChange = append(deviceChange, addDevSpec(NewIDEDev(200, 0)))
			}
		}
	}

	diskPreallocationChanged := false
	diskChanged := []types.VirtualMachineRelocateSpecDiskLocator{}
	idisks, _ := from.GetIDisks()
	for i, disk := range idisks {
		if i > len(params.Disks)-1 {
			break
		}
		if len(params.Disks[i].Preallocation) > 0 && params.Disks[i].Preallocation != disk.GetPreallocation() {
			diskPreallocationChanged = true
		}

		locator := types.VirtualMachineRelocateSpecDiskLocator{}
		tds, _ := host.FindDataStoreById(params.Disks[i].StorageId)
		if tds != nil {
			locator.Datastore = tds.object.Reference()
		}
		backing := &types.VirtualDiskFlatVer2BackingInfo{}
		switch params.Disks[i].Preallocation {
		case api.DISK_PREALLOCATION_OFF, api.DISK_PREALLOCATION_METADATA, "":
			backing.ThinProvisioned = types.NewBool(true)
		case api.DISK_PREALLOCATION_FALLOC:
			backing.ThinProvisioned = types.NewBool(false)
			backing.EagerlyScrub = types.NewBool(false)
		case api.DISK_PREALLOCATION_FULL:
			backing.ThinProvisioned = types.NewBool(false)
			backing.EagerlyScrub = types.NewBool(true)
		}
		locator.DiskId = int32(2000 + i)
		locator.DiskBackingInfo = backing
		diskChanged = append(diskChanged, locator)
	}

	dc, err := host.GetDatacenter()
	if err != nil {
		return nil, errors.Wrapf(err, "SHost.GetDatacenter for host '%s'", host.GetId())
	}
	// get vmFloder
	folders, err := dc.getObjectDatacenter().Folders(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "object.DataCenter.Folders")
	}
	resourcePool, err := host.SyncResourcePool(params.ResourcePool)
	if err != nil {
		return nil, errors.Wrap(err, "SyncResourcePool")
	}

	folderref := folders.VmFolder.Reference()
	poolref := resourcePool.Reference()
	hostref := host.GetHostSystem().Reference()
	dsref := ds.getDatastoreObj().Reference()
	relocateSpec := types.VirtualMachineRelocateSpec{
		Folder:    &folderref,
		Pool:      &poolref,
		Host:      &hostref,
		Datastore: &dsref,
	}
	if diskPreallocationChanged {
		relocateSpec.Disk = diskChanged
	}
	cloneSpec := &types.VirtualMachineCloneSpec{
		PowerOn:  false,
		Template: false,
		Location: relocateSpec,
		Snapshot: snapshot,
	}

	// uuid first
	name := params.Name
	if len(params.Uuid) != 0 {
		name = params.Uuid
	}
	if params.CpuSockets == 0 {
		params.CpuSockets = 1
	}
	spec := types.VirtualMachineConfigSpec{
		Name:              name,
		Uuid:              params.Uuid,
		NumCPUs:           int32(params.Cpu),
		NumCoresPerSocket: int32(params.CpuSockets),
		MemoryMB:          int64(params.Mem),

		CpuHotAddEnabled:    &True,
		CpuHotRemoveEnabled: &True,
		MemoryHotAddEnabled: &True,
	}
	cloneSpec.Config = &spec
	task, err := ovm.Clone(ctx, folders.VmFolder, name, *cloneSpec)
	if err != nil {
		return nil, errors.Wrap(err, "object.VirtualMachine.Clone")
	}
	info, err := task.WaitForResult(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Task.WaitForResult")
	}

	var moVM mo.VirtualMachine
	err = host.manager.reference2Object(info.Result.(types.ManagedObjectReference), VIRTUAL_MACHINE_PROPS, &moVM)
	if err != nil {
		return nil, errors.Wrap(err, "fail to fetch virtual machine just created")
	}

	vm := NewVirtualMachine(host.manager, &moVM, host.datacenter)
	if vm == nil {
		return nil, errors.Error("clone successfully but unable to NewVirtualMachine")
	}

	// remove old nics
	svm := vm.getVirtualMachine()
	for i := range svm.Config.Hardware.Device {
		dev := svm.Config.Hardware.Device[i]
		devType := reflect.Indirect(reflect.ValueOf(dev)).Type()
		etherType := reflect.TypeOf((*types.VirtualEthernetCard)(nil)).Elem()
		if reflectutils.StructContains(devType, etherType) {
			nic := NewVirtualNIC(vm, dev, i)
			if !utils.IsInStringArray(nic.GetMAC(), macAddrs) {
				deviceChange = append(deviceChange, &types.VirtualDeviceConfigSpec{
					Operation: types.VirtualDeviceConfigSpecOperationRemove,
					Device:    nic.getVirtualEthernetCard(),
				})
			}
		}
	}
	if snapshot != nil {
		return vm, nil
	}

	// adjust disk
	var i int
	if len(params.Disks) > 0 {
		// resize system disk
		sysDiskSize := params.Disks[0].Size
		if sysDiskSize == 0 {
			sysDiskSize = 30 * 1024
		}
		if int64(vm.vdisks[0].GetDiskSizeMB()) != sysDiskSize {
			vdisk := vm.vdisks[0].getVirtualDisk()
			originSize := vdisk.CapacityInKB
			vdisk.CapacityInKB = sysDiskSize * 1024
			spec := &types.VirtualDeviceConfigSpec{}
			spec.Operation = types.VirtualDeviceConfigSpecOperationEdit
			spec.Device = vdisk
			deviceChange = append(deviceChange, spec)
			log.Infof("resize system disk: %dGB => %dGB", originSize/1024/1024, sysDiskSize/1024)
		}
		// resize existed disk
		for i = 1; i < len(params.Disks); i++ {
			if i >= len(vm.vdisks) {
				break
			}
			wantDisk := params.Disks[i]
			vdisk := vm.vdisks[i]
			modisk := vdisk.getVirtualDisk()
			if wantDisk.Size <= int64(vdisk.GetDiskSizeMB()) {
				continue
			}
			originSize := modisk.CapacityInKB
			modisk.CapacityInKB = wantDisk.Size * 1024
			spec := &types.VirtualDeviceConfigSpec{}
			spec.Operation = types.VirtualDeviceConfigSpecOperationEdit
			spec.Device = vdisk.dev
			deviceChange = append(deviceChange, spec)
			log.Infof("resize No.%d data disk: %dGB => %dGB", i, originSize/1024/1024, wantDisk.Size/1024)
		}
		// remove extra disk
		for ; i < len(vm.vdisks); i++ {
			vdisk := vm.vdisks[i]
			spec := &types.VirtualDeviceConfigSpec{}
			spec.Operation = types.VirtualDeviceConfigSpecOperationRemove
			spec.Device = vdisk.dev
			spec.FileOperation = types.VirtualDeviceConfigSpecFileOperationDestroy
			deviceChange = append(deviceChange, spec)
			log.Infof("remove No.%d data disk", i)
		}
		if len(deviceChange) > 0 {
			spec = types.VirtualMachineConfigSpec{}
			spec.DeviceChange = deviceChange
			task, err = vm.getVmObj().Reconfigure(ctx, spec)
			if err != nil {
				return vm, errors.Wrap(err, "Reconfigure to resize disks")
			}
			err = task.Wait(ctx)
			if err != nil {
				return vm, errors.Wrap(err, "Wait task to resize disks")
			}
		}
	}
	// add data disk
	for ; i < len(params.Disks); i++ {
		size := params.Disks[i].Size
		if size == 0 {
			size = 30 * 1024
		}
		uuid := params.Disks[i].DiskId
		driver := params.Disks[i].Driver
		opts := &cloudprovider.GuestDiskCreateOptions{
			SizeMb:        int(size),
			UUID:          uuid,
			Driver:        driver,
			StorageId:     params.Disks[i].StorageId,
			Preallocation: params.Disks[i].Preallocation,
		}
		_, err := vm.CreateDisk(ctx, opts)
		if err != nil {
			log.Errorf("unable to add No.%d disk for vm %s", i, vm.GetId())
			return vm, nil
		}
	}
	return vm, nil
}

func (host *SHost) changeNic(device types.BaseVirtualDevice, update types.BaseVirtualDevice) {
	current := device.(types.BaseVirtualEthernetCard).GetVirtualEthernetCard()
	changed := update.(types.BaseVirtualEthernetCard).GetVirtualEthernetCard()

	current.Backing = changed.Backing
	if changed.MacAddress != "" {
		current.MacAddress = changed.MacAddress
	}
	if changed.AddressType != "" {
		current.AddressType = changed.AddressType
	}
}

func (host *SHost) isVersion50() bool {
	version := host.GetVersion()
	if strings.HasPrefix(version, "5.") {
		return true
	}
	return false
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	nics, err := host.GetIHostNicsInternal(false)
	if err != nil {
		return nil, errors.Wrap(err, "GetIHostNicsInternal")
	}
	return nics, nil
}

func (host *SHost) GetIHostNicsInternal(debug bool) ([]cloudprovider.ICloudHostNetInterface, error) {
	nics := host.getNicInfo(debug)
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
			if errors.Cause(err) != cloudprovider.ErrNotFound {
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
		if datastores[i].GetGlobalId() == id {
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
	dss := host.getHostSystem().Datastore
	var datastores []mo.Datastore
	err = host.manager.references2Objects(dss, DATASTORE_PROPS, &datastores)
	if err != nil {
		return err
	}
	host.datastores = make([]cloudprovider.ICloudStorage, 0)
	for i := 0; i < len(datastores); i += 1 {
		ds := NewDatastore(host.manager, &datastores[i], dc)
		dsId := ds.GetGlobalId()
		if len(dsId) > 0 {
			host.datastores = append(host.datastores, ds)
		}
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
		log.Debugf("rds: %s", rds.GetUrl())
		if strings.HasPrefix(path, rds.GetUrl()) {
			newPath = fmt.Sprintf("[%s] %s", rds.GetRelName(), path[len(rds.GetUrl()):])
			break
		}
	}
	if len(newPath) == 0 {
		return newPath, fmt.Errorf("path '%s' don't belong any datastore of host '%s'", path, host.GetName())
	}
	return newPath, nil
}

/*func (host *SHost) FindNetworkByVlanID(vlanID int32) (IVMNetwork, error) {
	if host.IsActiveVlanID(vlanID) {
		net, err := host.findBasicNetwork(vlanID)
		if err != nil {
			return nil, errors.Wrap(err, "findBasicNetwork error")
		}
		if net != nil {
			return net, nil
		}

		// no found in basic network
		dvpg, err := host.findVlanDVPG(vlanID)
		if err != nil {
			return nil, errors.Wrap(err, "findVlanDVPG")
		}
		return dvpg, nil
	}
	n, err := host.findBasicNetwork(vlanID)
	if err != nil {
		return nil, errors.Wrap(err, "find Basic network")
	}
	if n != nil {
		return n, err
	}
	return host.findNovlanDVPG()
}*/

// IsActiveVlanID will detect if vlanID is active that means vlanID in (1, 4095).
func (host *SHost) IsActiveVlanID(vlanID int32) bool {
	if vlanID > 1 && vlanID < 4095 {
		return true
	}
	return false
}

/*func (host *SHost) findBasicNetwork(vlanID int32) (IVMNetwork, error) {
	nets, err := host.getBasicNetworks()
	if err != nil {
		return nil, err
	}
	if len(nets) == 0 {
		return nil, nil
	}
	if !host.IsActiveVlanID(vlanID) {
		return nets[0], nil
	}
	for i := range nets {
		if nets[i].GetVlanId() == vlanID {
			return nets[i], nil
		}
	}
	return nil, nil
}*/

func (host *SHost) getBasicNetworks() ([]IVMNetwork, error) {
	nets, err := host.GetNetworks()
	if err != nil {
		return nil, errors.Wrap(err, "GetNetworks")
	}
	ret := make([]IVMNetwork, 0)
	for i := range nets {
		if net, ok := nets[i].(*SNetwork); ok {
			ret = append(ret, net)
		}
	}
	return ret, nil
}

func (host *SHost) GetNetworks() ([]IVMNetwork, error) {
	if host.networks != nil {
		return host.networks, nil
	}

	dc, err := host.GetDatacenter()
	if err != nil {
		return nil, errors.Wrap(err, "GetDatacenter")
	}

	nets, err := dc.resolveNetworks(host.getHostSystem().Network)
	if err != nil {
		return nil, errors.Wrap(err, "resolveNetworks")
	}

	host.networks = nets
	return host.networks, nil
}

func (host *SHost) getNetworkById(netId string) (IVMNetwork, error) {
	nets, err := host.GetNetworks()
	if err != nil {
		return nil, errors.Wrap(err, "host.GetNetworks")
	}
	for i := range nets {
		if nets[i].GetId() == netId {
			return nets[i], nil
		}
	}
	return nil, errors.ErrNotFound
}

/*func (host *SHost) findNovlanDVPG() (*SDistributedVirtualPortgroup, error) {
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
		if !host.IsActiveVlanID(nvlan) {
			return dvpg, nil
		}
	}
	return nil, nil
}*/

func (host *SHost) findDVPGById(id string) (*SDistributedVirtualPortgroup, error) {
	nets, err := host.datacenter.GetNetworks()
	if err != nil {
		return nil, errors.Wrap(err, "SHost.datacenter.GetNetworks")
	}
	for _, net := range nets {
		if dvpg, ok := net.(*SDistributedVirtualPortgroup); ok && dvpg.GetId() == id {
			return dvpg, nil
		}
	}
	return nil, nil
}

/*func (host *SHost) findVlanDVPG(vlanId int32) (*SDistributedVirtualPortgroup, error) {
	nets, err := host.datacenter.GetNetworks()
	if err != nil {
		return nil, errors.Wrap(err, "SHost.datacenter.GetNetworks")
	}
	for _, net := range nets {
		dvpg, ok := net.(*SDistributedVirtualPortgroup)
		if !ok || len(dvpg.GetActivePorts()) == 0 {
			continue
		}
		nvlan := dvpg.GetVlanId()
		if nvlan == vlanId {
			if dvpg.ContainHost(host) {
				return dvpg, nil
			}
			msg := "Find dvpg with correct vlan but it didn't contain this host"
			log.Debugf(msg)
			// add host to dvg
			// err := dvpg.AddHostToDVS(host)
			// if err != nil {
			//     return nil, errors.Wrapf(err, "dvpg %s add host to dvs error", dvpg.GetName())
			// }
			continue
		}
	}
	return nil, nil
}*/

func (host *SHost) GetHostSystem() *object.HostSystem {
	return object.NewHostSystem(host.manager.client.Client, host.getHostSystem().Reference())
}

func (host *SHost) GetResourcePool() (*object.ResourcePool, error) {
	var err error
	if host.parent == nil {
		host.parent, err = host.getParent()
		if err != nil {
			return nil, err
		}
	}
	return object.NewResourcePool(host.manager.client.Client, *host.parent.ResourcePool), nil
}

func (host *SHost) getParent() (*mo.ComputeResource, error) {
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

	err := host.manager.reference2Object(*moHost.Parent, []string{"name", "resourcePool"}, parent)
	if err != nil {
		return nil, errors.Wrap(err, "SESXiClient.reference2Object")
	}
	return mcr, nil
}

func (host *SHost) GetResourcePools() ([]mo.ResourcePool, error) {
	cluster, err := host.GetCluster()
	if err != nil {
		return nil, errors.Wrap(err, "GetCluster")
	}
	return cluster.ListResourcePools()
}

func (host *SHost) GetCluster() (*SCluster, error) {
	cluster, err := host.getCluster()
	if err != nil {
		return nil, errors.Wrap(err, "getCluster")
	}
	return NewCluster(host.manager, cluster, host.datacenter), nil
}

func (host *SHost) SyncResourcePool(name string) (*object.ResourcePool, error) {
	cluster, err := host.GetCluster()
	if err != nil {
		log.Errorf("failed to get host %s cluster info: %v", host.GetName(), err)
		return host.GetResourcePool()
	}
	pool, err := cluster.SyncResourcePool(name)
	if err != nil {
		log.Errorf("failed to sync resourcePool(%s) for cluster %s error: %v", name, cluster.GetName(), err)
		return host.GetResourcePool()
	}
	return object.NewResourcePool(host.manager.client.Client, pool.Reference()), nil
}

func (host *SHost) getCluster() (*mo.ClusterComputeResource, error) {
	moHost := host.getHostSystem()
	if moHost.Parent.Type != "ClusterComputeResource" {
		return nil, fmt.Errorf("host %s parent is not the cluster resource", host.GetName())
	}
	cluster := &mo.ClusterComputeResource{}
	err := host.manager.reference2Object(*moHost.Parent, []string{"name", "resourcePool"}, cluster)
	if err != nil {
		return nil, errors.Wrap(err, "SESXiClient.reference2Object")
	}
	return cluster, nil
}

func (host *SHost) GetSiblingHosts() ([]*SHost, error) {
	rp, err := host.getParent()
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
