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
}

type IHost interface {
	GetHostId() string
	GetSession() *mcclient.ClientSession
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
	SetDetectedOnHost(isDetected bool)
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

	GetHotPlugOptions(isolatedDev *desc.SGuestIsolatedDevice) ([]*HotPlugOption, error)
	GetHotUnplugOptions(isolatedDev *desc.SGuestIsolatedDevice) ([]*HotUnplugOption, error)
}

type IsolatedDeviceManager interface {
	GetDevices() []IDevice
	GetDeviceByIdent(vendorDevId string, addr string) IDevice
	GetDeviceByAddr(addr string) IDevice
	ProbePCIDevices(skipGPUs, skipUSBs bool, sriovNics, ovsOffloadNics []HostNic) error
	StartDetachTask()
	BatchCustomProbe() error
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

type HostNic struct {
	Bridge    string
	Interface string
	Wire      string
}

func (man *isolatedDeviceManager) ProbePCIDevices(skipGPUs, skipUSBs bool, sriovNics, ovsOffloadNics []HostNic) error {
	man.devices = make([]IDevice, 0)
	if !skipGPUs {
		gpus, err := getPassthroughGPUS()
		if err != nil {
			// ignore getPassthroughGPUS error on old machines without VGA devices
			log.Errorf("getPassthroughGPUS: %v", err)
			return nil
		}
		for idx, gpu := range gpus {
			man.devices = append(man.devices, NewGPUHPCDevice(gpu))
			log.Infof("Add GPU device: %d => %#v", idx, gpu)
		}
	}

	if !skipUSBs {
		usbs, err := getPassthroughUSBs()
		if err != nil {
			log.Errorf("getPassthroughUSBs: %v", err)
			return nil
		}
		for idx, usb := range usbs {
			man.devices = append(man.devices, usb)
			log.Infof("Add USB device: %d => %#v", idx, usb)
		}
	}

	if len(sriovNics) > 0 {
		nics, err := getSRIOVNics(sriovNics)
		if err != nil {
			log.Errorf("getSRIOVNics: %v", err)
			return nil
		}
		for idx, nic := range nics {
			man.devices = append(man.devices, nic)
			log.Infof("Add sriov nic: %d => %#v", idx, nic)
		}
	}
	if len(ovsOffloadNics) > 0 {
		nics, err := getOvsOffloadNics(ovsOffloadNics)
		if err != nil {
			log.Errorf("getOvsOffloadNics: %v", err)
			return nil
		}
		for idx, nic := range nics {
			man.devices = append(man.devices, nic)
			log.Infof("Add sriov nic: %d => %#v", idx, nic)
		}
	}

	return nil
}

func (man *isolatedDeviceManager) getSession() *mcclient.ClientSession {
	return man.host.GetSession()
}

func (man *isolatedDeviceManager) GetDeviceByIdent(vendorDevId string, addr string) IDevice {
	for _, dev := range man.devices {
		if dev.GetVendorDeviceId() == vendorDevId && dev.GetAddr() == addr {
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

func (man *isolatedDeviceManager) BatchCustomProbe() error {
	for i, dev := range man.devices {
		if err := dev.CustomProbe(i); err != nil {
			return err
		}
	}
	return nil
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

func (dev *sBaseDevice) SetDetectedOnHost(probe bool) {
	dev.detectedOnHost = probe
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
