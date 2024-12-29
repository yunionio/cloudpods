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

package isolated_device

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

const (
	RESOURCE = "isolated_devices"
)

type CloudDeviceInfo struct {
	Id                  string                      `json:"id"`
	GuestId             string                      `json:"guest_id"`
	HostId              string                      `json:"host_id"`
	DevType             string                      `json:"dev_type"`
	VendorDeviceId      string                      `json:"vendor_device_id"`
	Addr                string                      `json:"addr"`
	DetectedOnHost      bool                        `json:"detected_on_host"`
	MdevId              string                      `json:"mdev_id"`
	Model               string                      `json:"model"`
	WireId              string                      `json:"wire_id"`
	OvsOffloadInterface string                      `json:"ovs_offload_interface"`
	IsInfinibandNic     bool                        `json:"is_infiniband_nic"`
	NvmeSizeMB          int                         `json:"nvme_size_mb"`
	DevicePath          string                      `json:"device_path"`
	MpsMemoryLimit      int                         `json:"mps_memory_limit"`
	MpsMemoryTotal      int                         `json:"mps_memory_total"`
	MpsThreadPercentage int                         `json:"mps_thread_percentage"`
	NumaNode            int                         `json:"numa_node"`
	PcieInfo            *api.IsolatedDevicePCIEInfo `json:"pcie_info"`

	// The frame rate limiter (FRL) configuration in frames per second
	FRL string `json:"frl"`
	// The frame buffer size in Mbytes
	Framebuffer string `json:"framebuffer"`
	// The maximum resolution per display head, eg: 5120x2880
	MaxResolution string `json:"max_resolution"`
	// The maximum number of virtual display heads that the vGPU type supports
	// In computer graphics and display technology, the term "head" is commonly used to
	// describe the physical interface of a display device or display output.
	// It refers to a connection point on the monitor, such as HDMI, DisplayPort, or VGA interface.
	NumHeads string `json:"num_heads"`
	// The maximum number of vGPU instances per physical GPU
	MaxInstance string `json:"max_instance"`
}

type IHost interface {
	GetHostId() string
	GetSession() *mcclient.ClientSession
	IsContainerHost() bool

	AppendHostError(content string)
	AppendError(content, objType, id, name string)

	GetContainerDeviceConfigurationFilePath() string
}

type HotPlugOption struct {
	Device  string
	Options map[string]string
}

type HotUnplugOption struct {
	Id string
}

type IDevice interface {
	String() string
	GetCloudId() string
	GetHostId() string
	SetHostId(hId string)
	GetGuestId() string
	GetWireId() string
	IsInfinibandNic() bool
	GetOvsOffloadInterfaceName() string
	GetVendorDeviceId() string
	GetAddr() string
	GetDeviceType() string
	GetModelName() string
	CustomProbe(idx int) error
	SetDeviceInfo(info CloudDeviceInfo)
	DetectByAddr() error

	GetPassthroughOptions() map[string]string
	GetPassthroughCmd(index int) string
	GetIOMMUGroupDeviceCmd() string
	GetIOMMUGroupRestAddrs() []string
	GetVGACmd() string
	GetCPUCmd() string
	GetQemuId() string
	GetNumaNode() (int, error)

	// sriov nic
	GetPfName() string
	GetVirtfn() int

	// NVMe disk
	GetNVMESizeMB() int

	// legacy nvidia vgpu
	GetMdevId() string
	GetNVIDIAVgpuProfile() map[string]string

	GetHotPlugOptions(isolatedDev *desc.SGuestIsolatedDevice, guestDesc *desc.SGuestDesc) ([]*HotPlugOption, error)
	GetHotUnplugOptions(isolatedDev *desc.SGuestIsolatedDevice) ([]*HotUnplugOption, error)

	// Get extra PCIE information
	GetPCIEInfo() *api.IsolatedDevicePCIEInfo
	GetDevicePath() string

	// mps infos
	GetNvidiaMpsMemoryLimit() int
	GetNvidiaMpsMemoryTotal() int
	GetNvidiaMpsThreadPercentage() int
}

type IsolatedDeviceManager interface {
	GetDevices() []IDevice
	GetDeviceByIdent(vendorDevId, addr, mdevId string) IDevice
	GetDeviceByAddr(addr string) IDevice
	ProbePCIDevices(skipGPUs, skipUSBs, skipCustomDevs bool, sriovNics, ovsOffloadNics []HostNic, nvmePciDisks, amdVgpuPFs, nvidiaVgpuPFs []string, enableCudaMps, enableContainerNPU, enableWhitelist bool)
	StartDetachTask()
	BatchCustomProbe()
	AppendDetachedDevice(dev *CloudDeviceInfo)
	GetQemuParams(devAddrs []string) *QemuParams
	CheckDevIsNeedUpdate(dev IDevice, devInfo *CloudDeviceInfo) bool
}

type isolatedDeviceManager struct {
	host            IHost
	devices         []IDevice
	DetachedDevices []*CloudDeviceInfo
}

func NewManager(host IHost) IsolatedDeviceManager {
	man := &isolatedDeviceManager{
		host:            host,
		devices:         make([]IDevice, 0),
		DetachedDevices: make([]*CloudDeviceInfo, 0),
	}
	// Do probe later - Qiu Jian
	return man
}

func (man *isolatedDeviceManager) GetDevices() []IDevice {
	return man.devices
}

func (man *isolatedDeviceManager) getContainerDeviceConfiguration() (*ContainerDeviceConfiguration, error) {
	fp := man.host.GetContainerDeviceConfigurationFilePath()
	if fp == "" {
		return nil, nil
	}
	content, err := procutils.NewRemoteCommandAsFarAsPossible("cat", fp).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "Read container device configuration file %s", fp)
	}
	obj, err := jsonutils.ParseYAML(string(content))
	if err != nil {
		return nil, errors.Wrapf(err, "parse YAML content: %s", content)
	}
	cfg := new(ContainerDeviceConfiguration)
	if err := obj.Unmarshal(cfg); err != nil {
		return nil, errors.Wrapf(err, "unmarshal object to ContainerDeviceConfiguration")
	}
	return cfg, nil
}

func (man *isolatedDeviceManager) probeContainerDevices() {
	cfg, err := man.getContainerDeviceConfiguration()
	panicFatal := func(err error) {
		panic(err.Error())
	}
	if err != nil {
		panicFatal(errors.Wrap(err, "get container device configuration"))
	}
	if cfg == nil {
		return
	}
	for _, dev := range cfg.Devices {
		devMan, err := GetContainerDeviceManager(dev.Type)
		if err != nil {
			panicFatal(errors.Wrapf(err, "GetContainerDeviceManager by type %q", dev.Type))
		}
		iDevs, err := devMan.NewDevices(dev)
		if err != nil {
			panicFatal(errors.Wrapf(err, "NewDevices %#v", dev))
		}
		man.devices = append(man.devices, iDevs...)
	}
}

func (man *isolatedDeviceManager) probeContainerNvidiaGPUs(enableCudaMps bool) {
	devType := ContainerDeviceTypeNvidiaGpu
	if enableCudaMps {
		devType = ContainerDeviceTypeNvidiaMps
	}

	devman, err := GetContainerDeviceManager(devType)
	if err != nil {
		log.Errorf("no container device manager %s found", devType)
		return
	}
	devs, err := devman.ProbeDevices()
	if err != nil {
		log.Warningf("Probe container nvidia gpu devices: %v", err)
		return
	} else {
		for idx, dev := range devs {
			man.devices = append(man.devices, dev)
			log.Infof("Add Container nvidia GPU device: %d => %#v", idx, dev)
		}
	}
}

func (man *isolatedDeviceManager) probeContainerAscendNPUs(enable bool) {
	if !enable {
		return
	}

	devman, err := GetContainerDeviceManager(ContainerDeviceTypeAscendNpu)
	if err != nil {
		log.Errorf("no container device manager %s found", ContainerDeviceTypeAscendNpu)
		return
	}
	devs, err := devman.ProbeDevices()
	if err != nil {
		log.Warningf("Probe container Ascend npu devices: %v", err)
		return
	} else {
		for idx, dev := range devs {
			man.devices = append(man.devices, dev)
			log.Infof("Add Container Ascend npu device: %d => %#v", idx, dev)
		}
	}
}

func (man *isolatedDeviceManager) probeGPUS(skipGPUs bool, amdVgpuPFs, nvidiaVgpuPFs []string, enableWhitelist bool, whitelistModels []IsolatedDeviceModel) {
	if skipGPUs {
		return
	}
	filteredAddrs := []string{}
	filteredAddrs = append(filteredAddrs, amdVgpuPFs...)
	filteredAddrs = append(filteredAddrs, nvidiaVgpuPFs...)
	for i := 0; i < len(man.devices); i++ {
		filteredAddrs = append(filteredAddrs, man.devices[i].GetAddr())
	}

	gpus, err, warns := getPassthroughGPUs(filteredAddrs, enableWhitelist, whitelistModels)
	if err != nil {
		// ignore getPassthroughGPUS error on old machines without VGA devices
		log.Errorf("getPassthroughGPUS error: %v", err)
		man.host.AppendError(fmt.Sprintf("get passhtrough gpus %s", err.Error()), "isolated_devices", "", " ")
	} else {
		if len(warns) > 0 {
			for i := 0; i < len(warns); i++ {
				man.host.AppendError(warns[i].Error(), "isolated_devices", "", " ")
			}
		}
		for idx, gpu := range gpus {
			man.devices = append(man.devices, NewGPUHPCDevice(gpu))
			log.Infof("Add GPU device: %d => %#v", idx, gpu)
		}
	}
}

func (man *isolatedDeviceManager) probeCustomPCIDevs(skipCustomDevs bool, devModels []IsolatedDeviceModel, filterClassCodes []string) {
	if skipCustomDevs {
		return
	}
	for _, devModel := range devModels {
		devs, err := getPassthroughPCIDevs(devModel, filterClassCodes)
		if err != nil {
			log.Errorf("getPassthroughPCIDevs %v: %s", devModel, err)
			man.host.AppendError(fmt.Sprintf("get custom passthrough pci devices %s", err.Error()), "isolated_devices", "", "")
			continue
		}
		for i, dev := range devs {
			man.devices = append(man.devices, dev)
			log.Infof("Add general pci device: %d => %#v", i, dev)
		}
	}
}

func (man *isolatedDeviceManager) probeUSBs(skipUSBs bool) {
	if skipUSBs {
		return
	}

	usbs, err := getPassthroughUSBs()
	if err != nil {
		log.Errorf("getPassthroughUSBs: %v", err)
		man.host.AppendError(fmt.Sprintf("get passthrough usb devices %s", err.Error()), "isolated_devices", "", "")
	} else {
		for idx, usb := range usbs {
			man.devices = append(man.devices, usb)
			log.Infof("Add USB device: %d => %#v", idx, usb)
		}
	}
}

type HostNic struct {
	Bridge    string
	Interface string
	Wire      string
}

func (man *isolatedDeviceManager) probeSRIOVNics(sriovNics []HostNic) {
	if len(sriovNics) > 0 {
		nics, err := getSRIOVNics(sriovNics)
		if err != nil {
			log.Errorf("getSRIOVNics: %v", err)
			man.host.AppendError(fmt.Sprintf("get sriov nic devices %s", err.Error()), "isolated_devices", "", "")
		} else {
			for idx, nic := range nics {
				man.devices = append(man.devices, nic)
				log.Infof("Add sriov nic: %d => %#v", idx, nic)
			}
		}
	}
}

func (man *isolatedDeviceManager) probeOffloadNICS(ovsOffloadNics []HostNic) {
	if len(ovsOffloadNics) > 0 {
		nics, err := getOvsOffloadNics(ovsOffloadNics)
		if err != nil {
			log.Errorf("getOvsOffloadNics: %v", err)
			man.host.AppendError(fmt.Sprintf("get ovs offload nic devices %s", err.Error()), "isolated_devices", "", "")
		} else {
			for idx, nic := range nics {
				man.devices = append(man.devices, nic)
				log.Infof("Add sriov nic: %d => %#v", idx, nic)
			}
		}
	}
}

func (man *isolatedDeviceManager) probeNVMEDisks(nvmePciDisks []string) {
	if len(nvmePciDisks) > 0 {
		nvmeDisks, err := getPassthroughNVMEDisks(nvmePciDisks)
		if err != nil {
			log.Errorf("getPassthroughNVMEDisks: %v", err)
			man.host.AppendError(fmt.Sprintf("get nvme passthrough disks %s", err.Error()), "isolated_devices", "", "")
		} else {
			for i := range nvmeDisks {
				man.devices = append(man.devices, nvmeDisks[i])
			}
		}
	}
}

func (man *isolatedDeviceManager) probeAMDVgpus(amdVgpuPFs []string) {
	if len(amdVgpuPFs) > 0 {
		pattern := `^([0-9a-f]{2}):([0-9a-f]{2})\.([0-9a-f])$`
		for idx := range amdVgpuPFs {
			matched, _ := regexp.MatchString(pattern, amdVgpuPFs[idx])
			if !matched {
				err := errors.Errorf("probeAMDVgpus invaild input pci address %s", amdVgpuPFs[idx])
				log.Errorln(err)
				man.host.AppendError(err.Error(), "isolated_devices", "", "")
				continue
			}

			vgpus, err := getSRIOVGpus(amdVgpuPFs[idx])
			if err != nil {
				log.Errorf("getSRIOVGpus: %s", err)
				man.host.AppendError(fmt.Sprintf("get amd sriov vgpus %s", err.Error()), "isolated_devices", "", "")
			} else {
				for i := range vgpus {
					man.devices = append(man.devices, vgpus[i])
				}
			}
		}
	}
}

func (man *isolatedDeviceManager) probeNVIDIAVgpus(nvidiaVgpuPFs []string) {
	if len(nvidiaVgpuPFs) > 0 {
		pattern := `^([0-9a-f]{2}):([0-9a-f]{2})\.([0-9a-f])$`
		for idx := range nvidiaVgpuPFs {
			matched, _ := regexp.MatchString(pattern, nvidiaVgpuPFs[idx])
			if !matched {
				err := errors.Errorf("probeNVIDIAVgpus invaild input pci address %s", nvidiaVgpuPFs[idx])
				log.Errorln(err)
				man.host.AppendError(err.Error(), "isolated_devices", "", "")
				continue
			}
			vgpus, err := getNvidiaVGpus(nvidiaVgpuPFs[idx])
			if err != nil {
				log.Errorf("getNvidiaVGpus: %s", err)
				man.host.AppendError(fmt.Sprintf("get nvidia vgpus %s", err.Error()), "isolated_devices", "", "")
			} else {
				for i := range vgpus {
					man.devices = append(man.devices, vgpus[i])
				}
			}
		}
	}
}

func (man *isolatedDeviceManager) ProbePCIDevices(skipGPUs, skipUSBs, skipCustomDevs bool, sriovNics, ovsOffloadNics []HostNic, nvmePciDisks, amdVgpuPFs, nvidiaVgpuPFs []string, enableCudaMps, enableContainerNPU, enableWhitelist bool) {
	man.devices = make([]IDevice, 0)
	if man.host.IsContainerHost() {
		man.probeContainerNvidiaGPUs(enableCudaMps)
		man.probeContainerAscendNPUs(enableContainerNPU)
		man.probeContainerDevices()
	} else {
		devModels, err := man.getCustomIsolatedDeviceModels()
		if err != nil {
			log.Errorf("get isolated device devModels %s", err.Error())
			man.host.AppendError(fmt.Sprintf("get custom isolated device devModels %s", err.Error()), "isolated_devices", "", "")
			return
		}
		man.probeUSBs(skipUSBs)
		man.probeCustomPCIDevs(skipCustomDevs, devModels, GpuClassCodes)
		man.probeSRIOVNics(sriovNics)
		man.probeOffloadNICS(ovsOffloadNics)
		man.probeAMDVgpus(amdVgpuPFs)
		man.probeNVIDIAVgpus(nvidiaVgpuPFs)
		man.probeGPUS(skipGPUs, amdVgpuPFs, nvidiaVgpuPFs, enableWhitelist, devModels)
	}
}

type IsolatedDeviceModel struct {
	DevType  string `json:"dev_type"`
	VendorId string `json:"vendor_id"`
	DeviceId string `json:"device_id"`
	Model    string `json:"model"`
}

func (man *isolatedDeviceManager) getCustomIsolatedDeviceModels() ([]IsolatedDeviceModel, error) {
	//man.getSession().
	params := jsonutils.NewDict()
	params.Set("limit", jsonutils.NewInt(0))
	params.Set("scope", jsonutils.NewString("system"))
	params.Set("host_id", jsonutils.NewString(man.host.GetHostId()))
	res, err := modules.IsolatedDeviceModels.List(man.getSession(), jsonutils.NewDict())
	if err != nil {
		return nil, errors.Wrap(err, "list isolated_device_models from compute service")
	}
	devModels := make([]IsolatedDeviceModel, len(res.Data))
	for i, obj := range res.Data {
		if err := obj.Unmarshal(&devModels[i]); err != nil {
			return nil, errors.Wrap(err, "unmarshal isolated device model failed")
		}
	}
	return devModels, nil
}

func (man *isolatedDeviceManager) getSession() *mcclient.ClientSession {
	return man.host.GetSession()
}

func (man *isolatedDeviceManager) CheckDevIsNeedUpdate(dev IDevice, devInfo *CloudDeviceInfo) bool {
	if dev.GetDeviceType() != devInfo.DevType {
		return true
	}
	if dev.GetDevicePath() != devInfo.DevicePath {
		return true
	}
	if dev.GetModelName() != devInfo.Model {
		return true
	}
	if dev.GetWireId() != devInfo.WireId {
		return true
	}
	if dev.IsInfinibandNic() != devInfo.IsInfinibandNic {
		return true
	}
	if dev.GetOvsOffloadInterfaceName() != devInfo.OvsOffloadInterface {
		return true
	}
	if dev.GetNVMESizeMB() > 0 && devInfo.NvmeSizeMB > 0 && dev.GetNVMESizeMB() != devInfo.NvmeSizeMB {
		return true
	}
	if numaNode, _ := dev.GetNumaNode(); numaNode != devInfo.NumaNode {
		return true
	}
	if dev.GetMdevId() != devInfo.MdevId {
		return true
	}
	if info := dev.GetPCIEInfo(); info != nil && devInfo.PcieInfo == nil {
		return true
	}
	if profile := dev.GetNVIDIAVgpuProfile(); profile != nil {
		if val, _ := profile["frl"]; val != devInfo.FRL {
			return true
		}
		if val, _ := profile["framebuffer"]; val != devInfo.Framebuffer {
			return true
		}
		if val, _ := profile["max_resolution"]; val != devInfo.MaxResolution {
			return true
		}
		if val, _ := profile["num_heads"]; val != devInfo.NumHeads {
			return true
		}
		if val, _ := profile["max_instance"]; val != devInfo.MaxInstance {
			return true
		}
	}
	return false
}

func (man *isolatedDeviceManager) GetDeviceByIdent(vendorDevId, addr, mdevId string) IDevice {
	for _, dev := range man.devices {
		if dev.GetVendorDeviceId() == vendorDevId && dev.GetAddr() == addr && dev.GetMdevId() == mdevId {
			return dev
		}
	}
	return nil
}

func (man *isolatedDeviceManager) GetDeviceByVendorDevId(vendorDevId string) IDevice {
	for _, dev := range man.devices {
		if dev.GetVendorDeviceId() == vendorDevId {
			return dev
		}
	}
	return nil
}

func (man *isolatedDeviceManager) GetDeviceByAddr(addr string) IDevice {
	for _, dev := range man.devices {
		if dev.GetAddr() == addr {
			return dev
		}
	}
	return nil
}

func (man *isolatedDeviceManager) BatchCustomProbe() {
	for i, dev := range man.devices {
		if err := dev.CustomProbe(i); err != nil {
			man.host.AppendError(
				fmt.Sprintf("CustomProbe failed %s", err.Error()),
				"isolated_devices", dev.GetAddr(), dev.GetModelName())
		}
	}
}

func (man *isolatedDeviceManager) AppendDetachedDevice(dev *CloudDeviceInfo) {
	dev.DetectedOnHost = false
	man.DetachedDevices = append(man.DetachedDevices, dev)
}

func (man *isolatedDeviceManager) StartDetachTask() {
	if len(man.DetachedDevices) == 0 {
		return
	}
	go func() {
		for _, dev := range man.DetachedDevices {
			for {
				log.Infof("Start delete cloud device %s", jsonutils.Marshal(dev))
				if _, err := modules.IsolatedDevices.PerformAction(man.getSession(), dev.Id, "purge",
					jsonutils.Marshal(map[string]interface{}{
						"purge": true,
					})); err != nil {
					if errors.Cause(err) == httperrors.ErrResourceNotFound {
						break
					}
					log.Errorf("Detach device %s failed: %v, try again later", dev.Id, err)
					time.Sleep(30 * time.Second)
					continue
				}
				break
			}
		}
		man.DetachedDevices = nil
	}()
}

func (man *isolatedDeviceManager) GetQemuParams(devAddrs []string) *QemuParams {
	return getQemuParams(man, devAddrs)
}

type SBaseDevice struct {
	dev            *PCIDevice
	originAddr     string
	cloudId        string
	hostId         string
	guestId        string
	devType        string
	detectedOnHost bool
}

func NewBaseDevice(dev *PCIDevice, devType string) *SBaseDevice {
	return &SBaseDevice{
		dev:     dev,
		devType: devType,
	}
}

func (dev *SBaseDevice) GetDevicePath() string {
	return ""
}

func (dev *SBaseDevice) GetHostId() string {
	return dev.hostId
}

func (dev *SBaseDevice) SetHostId(hId string) {
	dev.hostId = hId
}

func (dev *SBaseDevice) String() string {
	return dev.dev.String()
}

func (dev *SBaseDevice) GetWireId() string {
	return ""
}

func (dev *SBaseDevice) SetDeviceInfo(info CloudDeviceInfo) {
	if len(info.Id) != 0 {
		dev.cloudId = info.Id
	}
	if len(info.GuestId) != 0 {
		dev.guestId = info.GuestId
	}
	if len(info.HostId) != 0 {
		dev.hostId = info.HostId
	}
	if len(info.DevType) != 0 {
		dev.devType = info.DevType
	}
}

func SyncDeviceInfo(session *mcclient.ClientSession, hostId string, dev IDevice, needUpdate bool) (jsonutils.JSONObject, error) {
	if len(dev.GetHostId()) == 0 {
		dev.SetHostId(hostId)
	}
	data := GetApiResourceData(dev)
	if len(dev.GetCloudId()) != 0 {
		if !needUpdate {
			log.Infof("Update %s isolated_device: do nothing", dev.GetCloudId())
			return nil, nil
		}

		log.Infof("Update %s isolated_device: %s", dev.GetCloudId(), data.String())
		return modules.IsolatedDevices.Update(session, dev.GetCloudId(), data)
	}
	log.Infof("Create new isolated_device: %s", data.String())
	return modules.IsolatedDevices.Create(session, data)
}

func (dev *SBaseDevice) GetCloudId() string {
	return dev.cloudId
}

func (dev *SBaseDevice) GetVendorDeviceId() string {
	return dev.dev.GetVendorDeviceId()
}

func (dev *SBaseDevice) GetAddr() string {
	return dev.dev.Addr
}

func (dev *SBaseDevice) GetOriginAddr() string {
	if dev.originAddr != "" {
		return dev.originAddr
	}
	return dev.dev.Addr
}

func (dev *SBaseDevice) SetAddr(addr, originAddr string) {
	dev.originAddr = originAddr
	dev.dev.Addr = addr
}

func (dev *SBaseDevice) GetDeviceType() string {
	return dev.devType
}

func (dev *SBaseDevice) GetPfName() string {
	return ""
}

func (dev *SBaseDevice) GetVirtfn() int {
	return -1
}

func (dev *SBaseDevice) GetNumaNode() (int, error) {
	numaNodePath := fmt.Sprintf("/sys/bus/pci/devices/0000:%s/numa_node", dev.GetAddr())
	numaNode, err := fileutils2.FileGetIntContent(numaNodePath)
	if err != nil {
		return -1, errors.Wrap(err, "get device numa node")
	}
	return numaNode, nil
}

func (dev *SBaseDevice) GetOvsOffloadInterfaceName() string {
	return ""
}

func (dev *SBaseDevice) IsInfinibandNic() bool {
	return false
}

func (dev *SBaseDevice) GetNVMESizeMB() int {
	return -1
}

func (dev *SBaseDevice) GetNVIDIAVgpuProfile() map[string]string {
	return nil
}

func (dev *SBaseDevice) GetMdevId() string {
	return ""
}

func (dev *SBaseDevice) GetModelName() string {
	if dev.dev.ModelName != "" {
		return dev.dev.ModelName
	} else {
		return dev.dev.DeviceName
	}
}

func (dev *SBaseDevice) SetModelName(modelName string) {
	if dev.dev.ModelName == "" {
		dev.dev.ModelName = modelName
	}
}

func (dev *SBaseDevice) GetGuestId() string {
	return dev.guestId
}

func (dev *SBaseDevice) GetNvidiaMpsMemoryLimit() int {
	return -1
}

func (dev *SBaseDevice) GetNvidiaMpsMemoryTotal() int {
	return -1
}

func (dev *SBaseDevice) GetNvidiaMpsThreadPercentage() int {
	return -1
}

func GetApiResourceData(dev IDevice) *jsonutils.JSONDict {
	data := map[string]interface{}{
		"dev_type":         dev.GetDeviceType(),
		"addr":             dev.GetAddr(),
		"model":            dev.GetModelName(),
		"vendor_device_id": dev.GetVendorDeviceId(),
	}
	detected := false
	if err := dev.DetectByAddr(); err == nil {
		detected = true
	}
	data["detected_on_host"] = detected
	if len(dev.GetCloudId()) != 0 {
		data["id"] = dev.GetCloudId()
	}
	if len(dev.GetHostId()) != 0 {
		data["host_id"] = dev.GetHostId()
	}
	if len(dev.GetGuestId()) != 0 {
		data["guest_id"] = dev.GetGuestId()
	}
	if len(dev.GetWireId()) != 0 {
		data["wire_id"] = dev.GetWireId()
	}
	if dev.IsInfinibandNic() {
		data["is_infiniband_nic"] = true
	}
	if len(dev.GetOvsOffloadInterfaceName()) != 0 {
		data["ovs_offload_interface"] = dev.GetOvsOffloadInterfaceName()
	}
	if dev.GetNVMESizeMB() > 0 {
		data["nvme_size_mb"] = dev.GetNVMESizeMB()
	}
	if numaNode, err := dev.GetNumaNode(); err == nil {
		data["numa_node"] = numaNode
	} else {
		log.Warningf("failed get dev %s numa node %s", dev.GetAddr(), err)
	}

	if dev.GetMdevId() != "" {
		data["mdev_id"] = dev.GetMdevId()
	}
	if profile := dev.GetNVIDIAVgpuProfile(); profile != nil {
		for k, v := range profile {
			data[k] = v
		}
	}
	if info := dev.GetPCIEInfo(); info != nil {
		data["pcie_info"] = info
	}
	devPath := dev.GetDevicePath()
	if devPath != "" {
		data["device_path"] = devPath
	}

	if mpsMemTotal := dev.GetNvidiaMpsMemoryTotal(); mpsMemTotal > 0 {
		data["mps_memory_total"] = mpsMemTotal
	}
	if mpsMemLimit := dev.GetNvidiaMpsMemoryLimit(); mpsMemLimit > 0 {
		data["mps_memory_limit"] = mpsMemLimit
	}
	if mpsThreadPercentage := dev.GetNvidiaMpsThreadPercentage(); mpsThreadPercentage > 0 {
		data["mps_thread_percentage"] = mpsThreadPercentage
	}
	return jsonutils.Marshal(data).(*jsonutils.JSONDict)
}

func (dev *SBaseDevice) GetKernelDriver() (string, error) {
	return dev.dev.getKernelDriver()
}

func (dev *SBaseDevice) getVFIODeviceCmd(addr string) string {
	return fmt.Sprintf(" -device vfio-pci,host=%s", addr)
}

func (dev *SBaseDevice) GetPassthroughOptions() map[string]string {
	return nil
}

func (dev *SBaseDevice) GetPassthroughCmd(_ int) string {
	return dev.getVFIODeviceCmd(dev.GetAddr())
}

func (dev *SBaseDevice) GetIOMMUGroupRestAddrs() []string {
	addrs := []string{}
	for _, d := range dev.dev.RestIOMMUGroupDevs {
		addrs = append(addrs, d.Addr)
	}
	return addrs
}

func (dev *SBaseDevice) GetIOMMUGroupDeviceCmd() string {
	restAddrs := dev.GetIOMMUGroupRestAddrs()
	cmds := []string{}
	for _, addr := range restAddrs {
		cmds = append(cmds, dev.getVFIODeviceCmd(addr))
	}
	return strings.Join(cmds, "")
}

func (dev *SBaseDevice) DetectByAddr() error {
	return nil
}

func (dev *SBaseDevice) CustomProbe(idx int) error {
	// check environments on first probe
	if idx == 0 {
		for _, driver := range []string{"vfio", "vfio_iommu_type1", "vfio-pci"} {
			if err := procutils.NewRemoteCommandAsFarAsPossible("modprobe", driver).Run(); err != nil {
				return fmt.Errorf("modprobe %s: %v", driver, err)
			}
		}
	}

	driver, err := dev.GetKernelDriver()
	if err != nil {
		return fmt.Errorf("Nic %s is occupied by another driver: %s", dev.GetAddr(), driver)
	}
	if driver != VFIO_PCI_KERNEL_DRIVER {
		if driver != "" {
			if err = dev.dev.unbindDriver(); err != nil {
				return errors.Wrap(err, "unbind driver")
			}
		}
		if err = dev.dev.bindDriver(); err != nil {
			return errors.Wrap(err, "bind driver")
		}
	}
	return nil
}

func (dev *SBaseDevice) GetHotPlugOptions(isolatedDev *desc.SGuestIsolatedDevice, guestDesc *desc.SGuestDesc) ([]*HotPlugOption, error) {
	ret := make([]*HotPlugOption, 0)

	var masterDevOpt *HotPlugOption
	for i := 0; i < len(isolatedDev.VfioDevs); i++ {
		opts := map[string]string{
			"host": isolatedDev.VfioDevs[i].HostAddr,
			"bus":  isolatedDev.VfioDevs[i].BusStr(),
			"addr": isolatedDev.VfioDevs[i].SlotFunc(),
			"id":   isolatedDev.VfioDevs[i].Id,
		}
		if isolatedDev.VfioDevs[i].Multi != nil {
			if *isolatedDev.VfioDevs[i].Multi {
				opts["multifunction"] = "on"
			} else {
				opts["multifunction"] = "off"
			}
		}
		if isolatedDev.VfioDevs[i].XVga {
			opts["x-vga"] = "on"
		}
		devOpt := &HotPlugOption{
			Device:  isolatedDev.VfioDevs[i].DevType,
			Options: opts,
		}
		if isolatedDev.VfioDevs[i].Function == 0 {
			masterDevOpt = devOpt
		} else {
			ret = append(ret, devOpt)
		}
	}
	// if PCI slot function 0 already assigned, qemu will reject hotplug function
	// so put function 0 at the enda
	if masterDevOpt == nil {
		return nil, errors.Errorf("GPU Device no function 0 found")
	}
	ret = append(ret, masterDevOpt)
	return ret, nil
}

func (dev *SBaseDevice) GetHotUnplugOptions(isolatedDev *desc.SGuestIsolatedDevice) ([]*HotUnplugOption, error) {
	if len(isolatedDev.VfioDevs) == 0 {
		return nil, errors.Errorf("device %s no pci ids", isolatedDev.Id)
	}

	return []*HotUnplugOption{
		{
			Id: isolatedDev.VfioDevs[0].Id,
		},
	}, nil
}

func (dev *SBaseDevice) GetPCIEInfo() *api.IsolatedDevicePCIEInfo {
	return dev.dev.PCIEInfo
}

func ParseOutput(output []byte, doTrim bool) []string {
	lines := make([]string, 0)
	for _, line := range strings.Split(string(output), "\n") {
		if doTrim {
			lines = append(lines, strings.TrimSpace(line))
		} else {
			lines = append(lines, line)
		}
	}
	return lines
}

func bashCmdOutput(cmd string, doTrim bool) ([]string, error) {
	args := []string{"-c", cmd}
	output, err := procutils.NewRemoteCommandAsFarAsPossible("bash", args...).Output()
	if err != nil {
		return nil, err
	} else {
		return ParseOutput(output, doTrim), nil
	}
}

func bashOutput(cmd string) ([]string, error) {
	return bashCmdOutput(cmd, true)
}

func bashRawOutput(cmd string) ([]string, error) {
	return bashCmdOutput(cmd, false)
}

type QemuParams struct {
	Cpu     string
	Vga     string
	Devices []string
}

func getQemuParams(man *isolatedDeviceManager, devAddrs []string) *QemuParams {
	if len(devAddrs) == 0 {
		return nil
	}
	devCmds := []string{}
	cpuCmd := DEFAULT_CPU_CMD
	vgaCmd := DEFAULT_VGA_CMD
	// group by device type firstly
	devices := make(map[string][]IDevice, 0)
	for _, addr := range devAddrs {
		dev := man.GetDeviceByAddr(addr)
		if dev == nil {
			log.Warningf("IsolatedDeviceManager not found dev %#v, ignore it!", addr)
			continue
		}
		devType := dev.GetDeviceType()
		if _, ok := devices[devType]; !ok {
			devices[devType] = []IDevice{dev}
		} else {
			devices[devType] = append(devices[devType], dev)
		}
	}

	for devType, devs := range devices {
		log.Debugf("get devices %s command", devType)
		for idx, dev := range devs {
			devCmds = append(devCmds, getDeviceCmd(dev, idx))
			if dev.GetVGACmd() != vgaCmd && dev.GetDeviceType() == api.GPU_VGA_TYPE {
				vgaCmd = dev.GetVGACmd()
			}
			if dev.GetCPUCmd() != cpuCmd {
				cpuCmd = dev.GetCPUCmd()
			}
		}
	}

	return &QemuParams{
		Cpu:     cpuCmd,
		Vga:     vgaCmd,
		Devices: devCmds,
	}
}
