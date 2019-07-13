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
	"os"
	"path/filepath"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	o "yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/regutils2"
)

const (
	CLASS_CODE_VGA = "0300"
	CLASS_CODE_3D  = "0302"
)

const (
	BUSID_REGEX = `[0-9a-fA-F]{2}:[0-9a-fA-F]{2}\.[0-9a-fA-F]`
	CODE_REGEX  = `[0-9a-fA-F]{4}`
	LABEL_REGEX = `[\w+\ \.\,\:\+\&\-\/\[\]\(\)]+`

	VFIO_PCI_KERNEL_DRIVER = "vfio-pci"
	DEFAULT_VGA_CMD        = " -vga std"
	// 在qemu/kvm下模拟Windows Hyper-V的一些半虚拟化特性，以便更好地使用Win虚拟机
	// http://blog.wikichoon.com/2014/07/enabling-hyper-v-enlightenments-with-kvm.html
	// 但实际测试不行，虚拟机不能运行nvidia驱动
	// DEFAULT_CPU_CMD = "host,kvm=off,hv_relaxed,hv_spinlocks=0x1fff,hv_vapic,hv_time"
	DEFAULT_CPU_CMD = "host,kvm=off"

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

type IDevice interface {
	String() string
	GetCloudId() string
	GetVendorDeviceId() string
	GetAddr() string
	GetDeviceType() string
	CustomProbe() error
	SetDeviceInfo(info CloudDeviceInfo)
	SetDetectedOnHost(isDetected bool)

	GetPassthroughCmd(index int) string
	GetIOMMUGroupDeviceCmd() string
	GetVGACmd() string
	GetCPUCmd() string
	SyncDeviceInfo(IHost) error
}

type IsolatedDeviceManager struct {
	host            IHost
	Devices         []IDevice
	DetachedDevices []*CloudDeviceInfo
}

func NewManager(host IHost) (*IsolatedDeviceManager, error) {
	man := &IsolatedDeviceManager{
		host:            host,
		Devices:         make([]IDevice, 0),
		DetachedDevices: make([]*CloudDeviceInfo, 0),
	}
	err := man.fillPCIDevices()
	return man, err
}

func (man *IsolatedDeviceManager) fillPCIDevices() error {
	// only support gpu by now
	gpus, err := getPassthroughGPUS()
	if err != nil {
		// ignore getPassthroughGPUS error on old machines without VGA devices
		log.Errorf("getPassthroughGPUS: %v", err)
		return nil
	}
	for idx, gpu := range gpus {
		man.Devices = append(man.Devices, newGPUHPCDevice(gpu))
		log.Infof("Add GPU device: %d => %#v", idx, gpu)
	}
	return nil
}

func (man *IsolatedDeviceManager) getSession() *mcclient.ClientSession {
	return man.host.GetSession()
}

func (man *IsolatedDeviceManager) GetDeviceByIdent(vendorDevId string, addr string) IDevice {
	for _, dev := range man.Devices {
		if dev.GetVendorDeviceId() == vendorDevId && dev.GetAddr() == addr {
			return dev
		}
	}
	return nil
}

func (man *IsolatedDeviceManager) GetDeviceByVendorDevId(vendorDevId string) IDevice {
	for _, dev := range man.Devices {
		if dev.GetVendorDeviceId() == vendorDevId {
			return dev
		}
	}
	return nil
}

func (man *IsolatedDeviceManager) GetDeviceByAddr(addr string) IDevice {
	for _, dev := range man.Devices {
		if dev.GetAddr() == addr {
			return dev
		}
	}
	return nil
}

func (man *IsolatedDeviceManager) BatchCustomProbe() error {
	for _, dev := range man.Devices {
		if err := dev.CustomProbe(); err != nil {
			return err
		}
	}
	return nil
}

func (man *IsolatedDeviceManager) AppendDetachedDevice(dev *CloudDeviceInfo) {
	dev.DetectedOnHost = false
	man.DetachedDevices = append(man.DetachedDevices, dev)
}

func (man *IsolatedDeviceManager) StartDetachTask() {
	if len(man.DetachedDevices) == 0 {
		return
	}
	go func() {
		for _, dev := range man.DetachedDevices {
			for {
				if _, err := modules.IsolatedDevices.PerformAction(man.getSession(), dev.Id, "purge", nil); err != nil {
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

func (man *IsolatedDeviceManager) GetQemuParams(devAddrs []string) *QemuParams {
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

func newBaseDevice(dev *PCIDevice) *sBaseDevice {
	return &sBaseDevice{dev: dev}
}

func (dev *sBaseDevice) String() string {
	return dev.dev.String()
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

func (dev *sBaseDevice) SyncDeviceInfo(host IHost) error {
	if len(dev.hostId) == 0 {
		dev.hostId = host.GetHostId()
	}
	data := dev.GetApiResourceData()
	if len(dev.GetCloudId()) != 0 {
		log.Infof("Update %s isolated_device: %s", dev.GetCloudId(), data.String())
		_, err := modules.IsolatedDevices.Update(host.GetSession(), dev.GetCloudId(), data)
		return err
	}
	log.Infof("Create new isolated_device: %s", data.String())
	_, err := modules.IsolatedDevices.Create(host.GetSession(), data)
	return err
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

func (dev *sBaseDevice) GetApiResourceData() jsonutils.JSONObject {
	data := map[string]interface{}{
		"dev_type":         dev.GetDeviceType(),
		"addr":             dev.GetAddr(),
		"model":            dev.dev.ModelName,
		"vendor_device_id": dev.GetVendorDeviceId(),
	}
	detected := false
	if _, err := detectPCIDevByAddr(dev.GetAddr()); err == nil {
		detected = true
	}
	data["detected_on_host"] = detected
	if len(dev.cloudId) != 0 {
		data["id"] = dev.cloudId
	}
	if len(dev.hostId) != 0 {
		data["host_id"] = dev.hostId
	}
	if len(dev.guestId) != 0 {
		data["guest_id"] = dev.guestId
	}
	return jsonutils.Marshal(data)
}

func (dev *sBaseDevice) GetKernelDriver() (string, error) {
	return dev.dev.getKernelDriver()
}

func (dev *sBaseDevice) IsPassthroughAble() bool {
	driver, _ := dev.GetKernelDriver()
	return driver == VFIO_PCI_KERNEL_DRIVER
}

func (dev *sBaseDevice) getVFIODeviceCmd(addr string) string {
	return fmt.Sprintf(" -device vfio-pci,host=%s", addr)
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

type sGPUBaseDevice struct {
	*sBaseDevice
}

func newGPUBaseDevice(dev *PCIDevice) *sGPUBaseDevice {
	return &sGPUBaseDevice{
		sBaseDevice: newBaseDevice(dev),
	}
}

func (dev *sGPUBaseDevice) GetCPUCmd() string {
	return DEFAULT_CPU_CMD
}

func (dev *sGPUBaseDevice) GetVGACmd() string {
	return DEFAULT_VGA_CMD
}

func (dev *sGPUBaseDevice) CustomProbe() error {
	// vfio kernel driver check
	for _, driver := range []string{"vfio", "vfio_iommu_type1", "vfio-pci"} {
		if err := procutils.NewCommand("modprobe", driver).Run(); err != nil {
			return fmt.Errorf("modprobe %s: %v", driver, err)
		}
	}
	// grub check
	grubCmdline, err := fileutils2.FileGetContents("/proc/cmdline")
	if err != nil {
		return err
	}
	grubCmdline = strings.TrimSpace(grubCmdline)
	params := sets.NewString(strings.Split(grubCmdline, " ")...)
	if !params.IsSuperset(sets.NewString("intel_iommu=on",
		"vfio_iommu_type1.allow_unsafe_interrupts=1")) {
		return fmt.Errorf("Some GRUB_CMDLINE iommu parameters are missing")
	}
	isNouveauBlacklisted := false
	if params.IsSuperset(sets.NewString("rdblacklist=nouveau", "nouveau.modeset=0")) ||
		params.IsSuperset(sets.NewString("rd.driver.blacklist=nouveau", "nouveau.modeset=0")) {
		isNouveauBlacklisted = true
	}
	if !isNouveauBlacklisted {
		return fmt.Errorf("Some GRUB_CMDLINE nouveau_blacklisted parameters are missing")
	}
	driver, err := dev.GetKernelDriver()
	if err != nil {
		return err
	}
	if driver != "" && driver != VFIO_PCI_KERNEL_DRIVER {
		return fmt.Errorf("GPU is occupied by another driver: %s", driver)
	}
	if driver == "" {
		//fileutils2.FilePutContents(
		//fmt.Sprintf("%s\n", strings.Replace(dev.GetVendorDeviceId(), ":", " ", -1)),
		//false)
	}
	return nil
}

type sGPUVGADevice struct {
	*sGPUBaseDevice
}

func (gpu *sGPUVGADevice) GetDeviceType() string {
	return api.GPU_VGA_TYPE
}

func (gpu *sGPUVGADevice) GetVGACmd() string {
	return " -vga none"
}

func getGuestAddr(index int) string {
	vAddr := fmt.Sprintf("0x%x", 21+index) // from 0x15 above
	return vAddr
}

func (gpu *sGPUVGADevice) GetPassthroughCmd(index int) string {
	// vAddr := getGuestAddr(index)
	return fmt.Sprintf(" -device vfio-pci,host=%s,multifunction=on,x-vga=on", gpu.GetAddr())
}

func (gpu *sGPUVGADevice) CustomProbe() error {
	_, err := bashOutput(`cat /boot/cfg-$(uname -r) | grep -E "^CONFIG_VFIO_PCI_VGA=y"`)
	if err != nil {
		return fmt.Errorf("CONFIG_VFIO_PCI_VGA=y needs to be set in kernel compiling parameters")
	}
	return nil
}

type sGPUHPCDevice struct {
	*sGPUBaseDevice
}

func newGPUHPCDevice(dev *PCIDevice) *sGPUHPCDevice {
	gpuDev := &sGPUHPCDevice{
		sGPUBaseDevice: newGPUBaseDevice(dev),
	}
	gpuDev.devType = gpuDev.GetDeviceType()
	return gpuDev
}

func (gpu *sGPUHPCDevice) GetDeviceType() string {
	return api.GPU_HPC_TYPE
}

func (gpu *sGPUHPCDevice) GetPassthroughCmd(index int) string {
	// vAddr := getGuestAddr(index)
	return fmt.Sprintf(" -device vfio-pci,host=%s,multifunction=on", gpu.GetAddr())
}

func ParseOutput(output []byte) []string {
	lines := make([]string, 0)
	for _, line := range strings.Split(string(output), "\n") {
		lines = append(lines, strings.TrimSpace(line))
	}
	return lines
}

func bashOutput(cmd string) ([]string, error) {
	args := []string{"-c", cmd}
	output, err := procutils.NewCommand("bash", args...).Output()
	if err != nil {
		return nil, err
	} else {
		return ParseOutput(output), nil
	}
}

func gpuPCIString() ([]string, error) {
	lines, err := bashOutput("lspci -nnmm | egrep '3D|VGA'")
	if err != nil {
		return nil, fmt.Errorf("Get GPU PCI: %v", err)
	}
	ret := []string{}
	for _, line := range lines {
		if len(line) != 0 {
			ret = append(ret, line)
		}
	}
	return ret, nil
}

func gpuPCIAddr() ([]string, error) {
	lines, err := gpuPCIString()
	if err != nil {
		return nil, err
	}
	addrs := []string{}
	for _, line := range lines {
		addrs = append(addrs, strings.Split(line, " ")[0])
	}
	return addrs, nil
}

type PCIDevice struct {
	Addr          string `json:"bus_id"`
	ClassName     string `json:"class_name"`
	ClassCode     string `json:"class_code"`
	VendorName    string `json:"vendor_name"`
	VendorId      string `json:"vendor_id"`
	DeviceName    string `json:"device_name"`
	DeviceId      string `json:"device_id"`
	SubvendorName string `json:"subvendor_name"`
	SubvendorId   string `json:"subvendor_id"`
	SubdeviceName string `json:"subdevice_name"`
	SubdeviceId   string `json:"subdevice_id"`
	ModelName     string `json:"model_name"`

	RestIOMMUGroupDevs []*PCIDevice `json:"-"`
}

func NewPCIDevice(line string) (*PCIDevice, error) {
	dev := parseLspci(line)
	if err := dev.checkSameIOMMUGroupDevice(); err != nil {
		return nil, err
	}
	if err := dev.forceBindVFIOPCIDriver(o.HostOptions.UseBootVga); err != nil {
		return nil, fmt.Errorf("Force bind vfio-pci driver: %v", err)
	}
	return dev, nil
}

func NewPCIDevice2(line string) *PCIDevice {
	return parseLspci(line)
}

// parseLspci parse one line output of `lspci -nnmm`
func parseLspci(line string) *PCIDevice {
	itemRegex := `(?P<bus_id>(` + BUSID_REGEX + `))` +
		`\ "(?P<class_name>` + LABEL_REGEX + `)\ \[(?P<class_code>` + CODE_REGEX + `)\]"` +
		`\ "(?P<vendor_name>` + LABEL_REGEX + `)\ \[(?P<vendor_id>` + CODE_REGEX + `)\]"` +
		`\ "(?P<device_name>` + LABEL_REGEX + `)\ \[(?P<device_id>` + CODE_REGEX + `)\]"` +
		`\ .*\"((?P<subvendor_name>` + LABEL_REGEX + `)\ \[(?P<subvendor_id>` + CODE_REGEX + `)\])*"` +
		`\ "((?P<subdevice_name>` + LABEL_REGEX + `)\ \[(?P<subdevice_id>` + CODE_REGEX + `)\])*`
	ret := regutils2.SubGroupMatch(itemRegex, line)
	dev := PCIDevice{}
	jsonutils.Marshal(ret).Unmarshal(&dev)
	deviceRegex := `(?P<code_name>` + LABEL_REGEX + `)\ \[(?P<model_name>` + LABEL_REGEX + `)\]`
	if ret := regutils2.SubGroupMatch(deviceRegex, dev.DeviceName); len(ret) != 0 {
		dev.ModelName = ret["model_name"]
	}
	return &dev
}

func (d *PCIDevice) GetVendorDeviceId() string {
	return fmt.Sprintf("%s:%s", d.VendorId, d.DeviceId)
}

// checkSameIOMMUGroupDevice check related device like Audio in same iommu group
// e.g.
// 41:00.0 VGA compatible controller [0300]: NVIDIA Corporation GP107 [GeForce GTX 1050 Ti] [10de:1c82] (rev a1)
// 41:00.1 Audio device [0403]: NVIDIA Corporation GP107GL High Definition Audio Controller [10de:0fb9] (rev a1)
func (d *PCIDevice) checkSameIOMMUGroupDevice() error {
	group, err := NewIOMMUGroup()
	if err != nil {
		return fmt.Errorf("IOMMUGroup FindSameGroupDevs: %v", err)
	}
	d.RestIOMMUGroupDevs = group.FindSameGroupDevs(d.Addr)
	return nil
}

func (d *PCIDevice) IsBootVGA() (bool, error) {
	addr := d.Addr
	output, err := procutils.NewCommand("find", "/sys/devices", "-name", "boot_vga").Output()
	if err != nil {
		return false, err
	}
	paths := ParseOutput(output)
	for _, p := range paths {
		if strings.Contains(p, addr) {
			if content, err := fileutils2.FileGetContents(p); err != nil {
				return false, err
			} else {
				if len(content) > 0 && strings.HasPrefix(content, "1") {
					log.Infof("PCI address %s is boot_vga: %s", addr, p)
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func (d *PCIDevice) forceBindVFIOPCIDriver(useBootVGA bool) error {
	if !utils.IsInStringArray(d.ClassCode, []string{CLASS_CODE_VGA, CLASS_CODE_VGA}) {
		return nil
	}
	isBootVGA, err := d.IsBootVGA()
	if err != nil {
		return err
	}
	if !useBootVGA && isBootVGA {
		log.Infof("%#v is boot vga card, skip it", d)
		return nil
	}
	if d.IsVFIOPCIDriverUsed() {
		log.Infof("%s already use vfio-pci driver", d)
		return nil
	}

	devs := []*PCIDevice{}
	devs = append(devs, d.RestIOMMUGroupDevs...)
	devs = append(devs, d)
	for _, dev := range devs {
		if err := dev.bindAddrVFIOPCI(); err != nil {
			return fmt.Errorf("bind %s vfio-pci driver: %v", dev, err)
		}
	}
	return nil
}

func (d *PCIDevice) bindAddrVFIOPCI() error {
	if err := d.unbindDriver(); err != nil {
		return fmt.Errorf("unbindDriver: %v", err)
	}
	if err := d.bindDriver(); err != nil {
		return fmt.Errorf("bindDriver: %v", err)
	}
	return nil
}

func (d *PCIDevice) unbindDriver() error {
	driver, err := d.getKernelDriver()
	if err != nil {
		return err
	}
	if len(driver) != 0 {
		if err := fileutils2.FilePutContents(
			fmt.Sprintf("/sys/bus/pci/devices/0000:%s/driver/unbind", d.Addr),
			fmt.Sprintf("0000:%s", d.Addr), false); err != nil {
			return fmt.Errorf("unbindDriver: %v", err)
		}
	}
	return nil
}

func (d *PCIDevice) bindDriver() error {
	vendorDevId := fmt.Sprintf("%s %s", d.VendorId, d.DeviceId)
	return fileutils2.FilePutContents(
		"/sys/bus/pci/drivers/vfio-pci/new_id",
		fmt.Sprintf("%s\n", vendorDevId),
		false,
	)
}

func (d *PCIDevice) String() string {
	return jsonutils.Marshal(d).String()
}

func (d *PCIDevice) IsVFIOPCIDriverUsed() bool {
	driver, _ := d.getKernelDriver()
	if driver != VFIO_PCI_KERNEL_DRIVER {
		return false
	}
	for _, dev := range d.RestIOMMUGroupDevs {
		driver, _ := dev.getKernelDriver()
		if driver != VFIO_PCI_KERNEL_DRIVER {
			return false
		}
	}
	return true
}

func (d *PCIDevice) getKernelDriver() (string, error) {
	prompt := "Kernel driver in use: "
	lines, err := bashOutput(fmt.Sprintf("lspci -k -s %s", d.Addr))
	if err != nil {
		return "", err
	}
	for _, line := range lines {
		begin := strings.Index(line, prompt)
		if begin >= 0 {
			end := begin + len(prompt)
			return line[end:], nil
		}
	}
	// no driver in use
	return "", nil
}

type IOMMUGroup struct {
	// busId: group
	group map[string]string
}

func NewIOMMUGroup() (*IOMMUGroup, error) {
	devPaths := "/sys/kernel/iommu_groups/"
	dict := make(map[string]string)
	err := filepath.Walk(devPaths, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		parts := strings.Split(path, "/")
		group := parts[4]
		busId := parts[len(parts)-1]
		dict[busId] = group
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &IOMMUGroup{group: dict}, nil
}

func (g *IOMMUGroup) ListDevices(groupNum, selfAddr string) []*PCIDevice {
	ret := []string{}
	for busId, group := range g.group {
		if groupNum == group {
			ret = append(ret, busId)
		}
	}

	devs := []*PCIDevice{}
	for _, addr := range ret {
		if addr == selfAddr {
			continue
		}
		dev, _ := detectPCIDevByAddrWithoutIOMMUGroup(addr[5:])
		if dev != nil {
			devs = append(devs, dev)
		}
	}
	return devs
}

func (g *IOMMUGroup) FindSameGroupDevs(devAddr string) []*PCIDevice {
	// devAddr: '0000:3f:0f.3' or '3f:0f.3' format
	if len(devAddr) == 7 {
		devAddr = fmt.Sprintf("0000:%s", devAddr)
	}
	group, ok := g.group[devAddr]
	if !ok {
		return nil
	}
	return g.ListDevices(group, devAddr)
}

func (g *IOMMUGroup) String() string {
	return jsonutils.Marshal(g.group).PrettyString()
}

func getGPUPCIStr() ([]string, error) {
	ret, err := bashOutput("lspci -nnmm | egrep '3D|VGA'")
	if err != nil {
		return nil, err
	}
	lines := []string{}
	for _, l := range ret {
		if len(l) != 0 {
			lines = append(lines, l)
		}
	}
	return lines, err
}

func detectPCIDevByAddr(addr string) (*PCIDevice, error) {
	ret, err := bashOutput(fmt.Sprintf("lspci -nnmm -s %s", addr))
	if err != nil {
		return nil, err
	}
	return NewPCIDevice(strings.Join(ret, ""))
}

func detectPCIDevByAddrWithoutIOMMUGroup(addr string) (*PCIDevice, error) {
	ret, err := bashOutput(fmt.Sprintf("lspci -nnmm -s %s", addr))
	if err != nil {
		return nil, err
	}
	return NewPCIDevice2(strings.Join(ret, "")), nil
}

func detectGPUS() ([]*PCIDevice, error) {
	lines, err := getGPUPCIStr()
	if err != nil {
		return nil, err
	}
	devs := []*PCIDevice{}
	for _, line := range lines {
		dev, err := NewPCIDevice(line)
		if err != nil {
			return nil, err
		}
		devs = append(devs, dev)
	}
	return devs, nil
}

func getPassthroughGPUS() ([]*PCIDevice, error) {
	gpus, err := detectGPUS()
	if err != nil {
		return nil, err
	}
	ret := []*PCIDevice{}
	for _, dev := range gpus {
		if drv, err := dev.getKernelDriver(); err != nil {
			log.Errorf("Device %#v get kernel driver error: %v", dev, err)
		} else if drv == VFIO_PCI_KERNEL_DRIVER {
			ret = append(ret, dev)
		} else {
			log.Warningf("GPU %v use kernel driver %q, skip it", dev, drv)
		}
	}
	return ret, nil
}

type QemuParams struct {
	Cpu     string
	Vga     string
	Devices []string
}

func GetDeviceCmd(dev IDevice, index int) string {
	passthroughCmd := dev.GetPassthroughCmd(index)
	groupDevCmd := dev.GetIOMMUGroupDeviceCmd()
	if len(groupDevCmd) != 0 {
		passthroughCmd = fmt.Sprintf("%s%s", passthroughCmd, groupDevCmd)
	}
	return passthroughCmd
}

func getQemuParams(man *IsolatedDeviceManager, devAddrs []string) *QemuParams {
	if len(devAddrs) == 0 {
		return nil
	}
	devCmds := []string{}
	cpuCmd := DEFAULT_CPU_CMD
	vgaCmd := DEFAULT_VGA_CMD
	for idx, addr := range devAddrs {
		dev := man.GetDeviceByAddr(addr)
		if dev == nil {
			log.Warningf("IsolatedDeviceManager not found dev %#v, ignore it!", addr)
			continue
		}
		devCmds = append(devCmds, GetDeviceCmd(dev, idx))
		if dev.GetVGACmd() != vgaCmd && dev.GetDeviceType() == api.GPU_VGA_TYPE {
			vgaCmd = dev.GetVGACmd()
		}
		if dev.GetCPUCmd() != cpuCmd {
			cpuCmd = dev.GetCPUCmd()
		}
	}
	return &QemuParams{
		Cpu:     cpuCmd,
		Vga:     vgaCmd,
		Devices: devCmds,
	}
}
