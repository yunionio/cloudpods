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
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	o "yunion.io/x/onecloud/pkg/hostman/options"
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
)

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

type sGPUBaseDevice struct {
	*sBaseDevice
}

func newGPUBaseDevice(dev *PCIDevice, devType string) *sGPUBaseDevice {
	return &sGPUBaseDevice{
		sBaseDevice: newBaseDevice(dev, devType),
	}
}

func (dev *sGPUBaseDevice) GetCPUCmd() string {
	return DEFAULT_CPU_CMD
}

func (dev *sGPUBaseDevice) GetVGACmd() string {
	return DEFAULT_VGA_CMD
}

func (dev *sGPUBaseDevice) DetectByAddr() error {
	_, err := detectPCIDevByAddr(dev.GetAddr())
	return err
}

func (dev *sGPUBaseDevice) CustomProbe(idx int) error {
	// check environments on first probe
	if idx == 0 {
		// vfio kernel driver check
		for _, driver := range []string{"vfio", "vfio_iommu_type1", "vfio-pci"} {
			if err := procutils.NewRemoteCommandAsFarAsPossible("modprobe", driver).Run(); err != nil {
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

func (dev *sGPUBaseDevice) GetQemuId() string {
	return fmt.Sprintf("dev_%s", strings.ReplaceAll(dev.GetAddr(), ":", "_"))
}

func (dev *sGPUBaseDevice) GetHotPlugOptions(isolatedDev *desc.SGuestIsolatedDevice) ([]*HotPlugOption, error) {
	ret := make([]*HotPlugOption, 0)

	var masterDevOpt *HotPlugOption
	for i := 0; i < len(isolatedDev.VfioDevs); i++ {
		cmd := isolatedDev.VfioDevs[i].HostAddr
		if optCmd := isolatedDev.VfioDevs[i].OptionsStr(); len(optCmd) > 0 {
			cmd += fmt.Sprintf(",%s", optCmd)
		}
		opts := map[string]string{
			"host": cmd,
			"id":   isolatedDev.VfioDevs[i].Id,
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

func (dev *sGPUBaseDevice) GetHotUnplugOptions(isolatedDev *desc.SGuestIsolatedDevice) ([]*HotUnplugOption, error) {
	if len(isolatedDev.VfioDevs) == 0 {
		return nil, errors.Errorf("device %s no pci ids", isolatedDev.Id)
	}

	return []*HotUnplugOption{
		{
			Id: isolatedDev.VfioDevs[0].Id,
		},
	}, nil
}

func getGuestAddr(index int) string {
	vAddr := fmt.Sprintf("0x%x", 21+index) // from 0x15 above
	return vAddr
}

type sGPUHPCDevice struct {
	*sGPUBaseDevice
}

func NewGPUHPCDevice(dev *PCIDevice) *sGPUHPCDevice {
	gpuDev := &sGPUHPCDevice{
		sGPUBaseDevice: newGPUBaseDevice(dev, api.GPU_HPC_TYPE),
	}
	return gpuDev
}

func (gpu *sGPUHPCDevice) GetPassthroughCmd(index int) string {
	// vAddr := getGuestAddr(index)
	return fmt.Sprintf(" -device vfio-pci,host=%s,multifunction=on", gpu.GetAddr())
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
	d.RestIOMMUGroupDevs = group.FindSameGroupDevs(d.Addr, d.VendorId)
	return nil
}

func (d *PCIDevice) IsBootVGA() (bool, error) {
	addr := d.Addr
	output, err := procutils.NewCommand("find", "/sys/devices", "-name", "boot_vga").Output()
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if code, ok := exiterr.Sys().(syscall.WaitStatus); ok &&
				code.ExitStatus() == 1 && strings.Contains(string(output), "No such file or directory") {
				log.Warningf("find boot vga %s", output)
			} else {
				return false, err
			}
		} else {
			return false, err
		}
	}
	paths := ParseOutput(output, true)
	for _, p := range paths {
		if strings.Contains(p, addr) && !strings.Contains(p, "No such file or directory") {
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

func (g *IOMMUGroup) ListDevices(groupNum, selfAddr, vendorId string) []*PCIDevice {
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
		if len(addr) < 5 {
			log.Warningf("Invalid addr %q of %q iommu_group[%s], skip it", addr, selfAddr, groupNum)
			continue
		}
		dev, _ := detectPCIDevByAddrWithoutIOMMUGroup(addr[5:])
		if dev != nil {
			if dev.VendorId == vendorId {
				devs = append(devs, dev)
			} else {
				log.Warningf("Skip append %q iommu_group[%s] device %s", selfAddr, groupNum, dev.String())
			}
		}
	}
	return devs
}

func (g *IOMMUGroup) FindSameGroupDevs(devAddr string, vendorId string) []*PCIDevice {
	// devAddr: '0000:3f:0f.3' or '3f:0f.3' format
	if len(devAddr) == 7 {
		devAddr = fmt.Sprintf("0000:%s", devAddr)
	}
	group, ok := g.group[devAddr]
	if !ok {
		return nil
	}
	return g.ListDevices(group, devAddr, vendorId)
}

func (g *IOMMUGroup) String() string {
	return jsonutils.Marshal(g.group).PrettyString()
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

func getDeviceCmd(dev IDevice, index int) string {
	passthroughCmd := dev.GetPassthroughCmd(index)
	groupDevCmd := dev.GetIOMMUGroupDeviceCmd()
	if len(groupDevCmd) != 0 {
		passthroughCmd = fmt.Sprintf("%s%s", passthroughCmd, groupDevCmd)
	}
	return passthroughCmd
}
