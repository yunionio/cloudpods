package esxi

import (
	"github.com/vmware/govmomi/vim25/mo"

	"yunion.io/x/jsonutils"

	"github.com/vmware/govmomi/vim25/types"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/pkg/util/netutils"
)

var HOST_SYSTEM_PROPS = []string{"name", "parent", "summary", "config", "hardware", "vm"}

type SHostNicInfo struct {
	Dev     string
	Driver  string
	Mac     string
	Index   int
	LinkUp  bool
	IpAddr  string
	Mtu     int
	NicType string
}

type SHostStorageAdapterInfo struct {
	Device    string
	Model     string
	Driver    string
	Pci       string
	Drivers   []SHostStorageDriverInfo
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
}

type SHostStorageEnclosureInfo struct {
	CN       string
	Name     string
	Model    string
	Vendor   string
	Revision string
	Status   string
}

type SHost struct {
	SManagedObject

	nicInfo     []SHostNicInfo
	storageInfo []SHostStorageAdapterInfo

	vms []cloudprovider.ICloudVM
}

func NewHost(manager *SESXiClient, host *mo.HostSystem, dc *SDatacenter) *SHost {
	return &SHost{SManagedObject: newManagedObject(manager, host, dc)}
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
		return models.HOST_STATUS_RUNNING
	case types.HostSystemPowerStatePoweredOff:
		return models.HOST_STATUS_READY
	default:
		return models.HOST_STATUS_UNKNOWN
	}
}

func (self *SHost) Refresh() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SHost) IsEmulated() bool {
	return false
}

func (self *SHost) fetchVMs() error {
	if self.vms != nil {
		return nil
	}
	var vms []mo.VirtualMachine
	err := self.manager.references2Objects(self.getHostSystem().Vm, VIRTUAL_MACHINE_PROPS, &vms)
	if err != nil {
		return err
	}

	dc, err := self.GetDatacenter()
	if err != nil {
		return err
	}

	self.vms = make([]cloudprovider.ICloudVM, len(vms))
	for i := 0; i < len(vms); i += 1 {
		self.vms[i] = NewVirtualMachine(self.manager, &vms[i], dc, self)
	}
	return nil
}

func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	err := self.fetchVMs()
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
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return nil, cloudprovider.ErrNotImplemented
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
		return models.HOST_ONLINE
	default:
		return models.HOST_OFFLINE
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
		if nics[i].NicType == models.NIC_TYPE_ADMIN {
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
		info.Index = i
		info.LinkUp = false
		nicInfoList = append(nicInfoList, info)
	}

	for _, nic := range moHost.Config.Network.Vnic {
		mac := netutils.FormatMacAddr(nic.Spec.Mac)
		pnic := findHostNicByMac(nicInfoList, mac)
		if pnic != nil {
			pnic.IpAddr = nic.Spec.Ip.IpAddress
			if nic.Spec.Portgroup == "Management Network" {
				pnic.NicType = models.NIC_TYPE_ADMIN
			}
			pnic.LinkUp = true
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

func (self *SHost) GetCpuCount() int8 {
	return int8(self.getHostSystem().Summary.Hardware.NumCpuThreads)
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

/*func (self *SHost) fetchStorageInfo() {
	adapterList := make([]SHostStorageAdapterInfo, 0)
	driversTable := make(map[string]SHostStorageDriverInfo, 0)
	enclosuresTable := make(map[string]SHostStorageEnclosureInfo, 0)
	moHost := self.getHostSystem()

	for i, ad := range moHost.Config.StorageDevice.HostBusAdapter {
		adinfo := ad.GetHostHostBusAdapter()
		if adinfo == nil {
			log.Errorf("Fail to GetHostHostBusAdapter")
			continue
		}
		info := SHostStorageAdapterInfo{}
		info.Device = adinfo.Device
		info.Model = adinfo.Model
		info.Driver = adinfo.Driver
		info.Pci = adinfo.Pci
		info.Drivers = make([]SHostStorageDriverInfo, 0)
		info.Enclosure = -1
		adapterList = append(adapterList, info)
	}

	for i, drv := range moHost.Config.StorageDevice.ScsiLun {
		lunInfo := drv.GetScsiLun()
		if lunInfo == nil {
			log.Errorf("fail to GetScsiLun")
			continue
		}
		if lunInfo.DeviceType == "disk" {
			info := SHostStorageDriverInfo{}
			info.CN = lunInfo.CanonicalName
			info.Name = lunInfo.DisplayName
			info.Model = lunInfo.Model
			info.Vendor = lunInfo.Vendor
			info.Revision = lunInfo.Revision
			info.Status = lunInfo.OperationalState[0]
			// info.SSD = lunInfo.
			// info.Dev =
			// info.Size = lunInfo.S
		} else if lunInfo.DeviceType == "enclosure" {

		}
	}
}
*/

func (self *SHost) GetStorageSizeMB() int {
	return 0
}

func (self *SHost) GetStorageType() string {
	return ""
}

func (self *SHost) GetHostType() string {
	return models.HOST_TYPE_ESXI
}

func (self *SHost) GetManagerId() string {
	return self.manager.providerId
}

func (self *SHost) CreateVM(name string, imgId string, sysDiskSize int, cpu int, memMB int, vswitchId string, ipAddr string, desc string,
	passwd string, storageType string, diskSizes []int, publicKey string, secGrpId string) (cloudprovider.ICloudVM, error) {
	log.Debugf("CreateVM")
	return nil, cloudprovider.ErrNotImplemented
}
