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
	"yunion.io/x/onecloud/pkg/util/procutils"
)

const (
	RESOURCE = "isolated_devices"
)

type CloudDeviceInfo struct {
	Id             string `json:"id"`
	GuestId        string `json:"guest_id"`
	HostId         string `json:"host_id"`
	DevType        string `json:"dev_type"`
	VendorDeviceId string `json:"vendor_device_id"`
	Addr           string `json:"addr"`
	DetectedOnHost bool   `json:"detected_on_host"`
	MdevId         string `json:"mdev_id"`
}

type IHost interface {
	GetHostId() string
	GetSession() *mcclient.ClientSession

	AppendHostError(content string)
	AppendError(content, objType, id, name string)
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

	// sriov nic
	GetPfName() string
	GetVirtfn() int

	// NVMe disk
	GetNVMESizeMB() int

	// legacy nvidia vgpu
	GetMdevId() string
	GetNVIDIAVgpuProfile() map[string]string

	GetHotPlugOptions(isolatedDev *desc.SGuestIsolatedDevice) ([]*HotPlugOption, error)
	GetHotUnplugOptions(isolatedDev *desc.SGuestIsolatedDevice) ([]*HotUnplugOption, error)
}

type IsolatedDeviceManager interface {
	GetDevices() []IDevice
	GetDeviceByIdent(vendorDevId, addr, mdevId string) IDevice
	GetDeviceByAddr(addr string) IDevice
	ProbePCIDevices(skipGPUs, skipUSBs, skipCustomDevs bool, sriovNics, ovsOffloadNics []HostNic, nvmePciDisks, amdVgpuPFs, nvidiaVgpuPFs []string)
	StartDetachTask()
	BatchCustomProbe()
	AppendDetachedDevice(dev *CloudDeviceInfo)
	GetQemuParams(devAddrs []string) *QemuParams
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
	// Do probe laster - Qiu Jian
	return man
}

func (man *isolatedDeviceManager) GetDevices() []IDevice {
	return man.devices
}

func (man *isolatedDeviceManager) probeGPUS(skipGPUs bool, amdVgpuPFs, nvidiaVgpuPFs []string) {
	if skipGPUs {
		return
	}
	filteredAddrs := []string{}
	filteredAddrs = append(filteredAddrs, amdVgpuPFs...)
	filteredAddrs = append(filteredAddrs, nvidiaVgpuPFs...)
	for i := 0; i < len(man.devices); i++ {
		filteredAddrs = append(filteredAddrs, man.devices[i].GetAddr())
	}

	gpus, err, warns := getPassthroughGPUS(filteredAddrs)
	if err != nil {
		// ignore getPassthroughGPUS error on old machines without VGA devices
		log.Errorf("getPassthroughGPUS: %v", err)
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

func (man *isolatedDeviceManager) probeCustomPCIDevs(skipCustomDevs bool) {
	if skipCustomDevs {
		return
	}
	devModels, err := man.getCustomIsolatedDeviceModels()
	if err != nil {
		log.Errorf("get custom isolated device models %s", err.Error())
		man.host.AppendError(fmt.Sprintf("get custom isolated device models %s", err.Error()), "isolated_devices", "", "")
	} else {
		for _, devModel := range devModels {
			devs, err := getPassthroughPCIDevs(devModel)
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

func (man *isolatedDeviceManager) ProbePCIDevices(
	skipGPUs, skipUSBs, skipCustomDevs bool,
	sriovNics, ovsOffloadNics []HostNic,
	nvmePciDisks, amdVgpuPFs, nvidiaVgpuPFs []string,
) {
	man.devices = make([]IDevice, 0)
	man.probeUSBs(skipUSBs)
	man.probeCustomPCIDevs(skipCustomDevs)
	man.probeSRIOVNics(sriovNics)
	man.probeOffloadNICS(ovsOffloadNics)
	man.probeAMDVgpus(amdVgpuPFs)
	man.probeNVIDIAVgpus(nvidiaVgpuPFs)
	man.probeGPUS(skipGPUs, amdVgpuPFs, nvidiaVgpuPFs)
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
	res, err := modules.IsolatedDeviceModels.List(man.getSession(), jsonutils.NewDict())
	if err != nil {
		return nil, err
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

type sBaseDevice struct {
	dev            *PCIDevice
	cloudId        string
	hostId         string
	guestId        string
	devType        string
	detectedOnHost bool
}

func newBaseDevice(dev *PCIDevice, devType string) *sBaseDevice {
	return &sBaseDevice{
		dev:     dev,
		devType: devType,
	}
}

func (dev *sBaseDevice) GetHostId() string {
	return dev.hostId
}

func (dev *sBaseDevice) SetHostId(hId string) {
	dev.hostId = hId
}

func (dev *sBaseDevice) String() string {
	return dev.dev.String()
}

func (dev *sBaseDevice) GetWireId() string {
	return ""
}

func (dev *sBaseDevice) SetDeviceInfo(info CloudDeviceInfo) {
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

func SyncDeviceInfo(session *mcclient.ClientSession, hostId string, dev IDevice) (jsonutils.JSONObject, error) {
	if len(dev.GetHostId()) == 0 {
		dev.SetHostId(hostId)
	}
	data := GetApiResourceData(dev)
	if len(dev.GetCloudId()) != 0 {
		log.Infof("Update %s isolated_device: %s", dev.GetCloudId(), data.String())
		return modules.IsolatedDevices.Update(session, dev.GetCloudId(), data)
	}
	log.Infof("Create new isolated_device: %s", data.String())
	return modules.IsolatedDevices.Create(session, data)
}

func (dev *sBaseDevice) GetCloudId() string {
	return dev.cloudId
}

func (dev *sBaseDevice) GetVendorDeviceId() string {
	return dev.dev.GetVendorDeviceId()
}

func (dev *sBaseDevice) GetAddr() string {
	return dev.dev.Addr
}

func (dev *sBaseDevice) GetDeviceType() string {
	return dev.devType
}

func (dev *sBaseDevice) GetPfName() string {
	return ""
}

func (dev *sBaseDevice) GetVirtfn() int {
	return -1
}

func (dev *sBaseDevice) GetOvsOffloadInterfaceName() string {
	return ""
}

func (dev *sBaseDevice) GetNVMESizeMB() int {
	return -1
}

func (dev *sBaseDevice) GetNVIDIAVgpuProfile() map[string]string {
	return nil
}

func (dev *sBaseDevice) GetMdevId() string {
	return ""
}

func (dev *sBaseDevice) GetModelName() string {
	if dev.dev.ModelName != "" {
		return dev.dev.ModelName
	} else {
		return dev.dev.DeviceName
	}
}

func (dev *sBaseDevice) GetGuestId() string {
	return dev.guestId
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
	if len(dev.GetOvsOffloadInterfaceName()) != 0 {
		data["ovs_offload_interface"] = dev.GetOvsOffloadInterfaceName()
	}
	if dev.GetNVMESizeMB() > 0 {
		data["nvme_size_mb"] = dev.GetNVMESizeMB()
	}

	if dev.GetMdevId() != "" {
		data["mdev_id"] = dev.GetMdevId()
	}
	if profile := dev.GetNVIDIAVgpuProfile(); profile != nil {
		for k, v := range profile {
			data[k] = v
		}
	}
	return jsonutils.Marshal(data).(*jsonutils.JSONDict)
}

func (dev *sBaseDevice) GetKernelDriver() (string, error) {
	return dev.dev.getKernelDriver()
}

func (dev *sBaseDevice) getVFIODeviceCmd(addr string) string {
	return fmt.Sprintf(" -device vfio-pci,host=%s", addr)
}

func (dev *sBaseDevice) GetPassthroughOptions() map[string]string {
	return nil
}

func (dev *sBaseDevice) GetPassthroughCmd(_ int) string {
	return dev.getVFIODeviceCmd(dev.GetAddr())
}

func (dev *sBaseDevice) GetIOMMUGroupRestAddrs() []string {
	addrs := []string{}
	for _, d := range dev.dev.RestIOMMUGroupDevs {
		addrs = append(addrs, d.Addr)
	}
	return addrs
}

func (dev *sBaseDevice) GetIOMMUGroupDeviceCmd() string {
	restAddrs := dev.GetIOMMUGroupRestAddrs()
	cmds := []string{}
	for _, addr := range restAddrs {
		cmds = append(cmds, dev.getVFIODeviceCmd(addr))
	}
	return strings.Join(cmds, "")
}

func (dev *sBaseDevice) DetectByAddr() error {
	return nil
}

func (dev *sBaseDevice) CustomProbe(idx int) error {
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
