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
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/util/regutils2"
)

type sUSBDevice struct {
	*sBaseDevice
}

// TODO: rename PCIDevice
func newUSBDevice(dev *PCIDevice) *sUSBDevice {
	return &sUSBDevice{
		sBaseDevice: newBaseDevice(dev, api.USB_TYPE),
	}
}

func (dev *sUSBDevice) GetCPUCmd() string {
	return ""
}

func (dev *sUSBDevice) GetVGACmd() string {
	return ""
}

func (dev *sUSBDevice) CustomProbe(int) error {
	// do nothing
	return nil
}

func GetUSBDevId(vendorId, devId, bus, addr string) string {
	return fmt.Sprintf("dev_%s_%s-%s_%s", vendorId, devId, bus, addr)
}

func getUSBDevQemuOptions(vendorId, deviceId string, bus, addr string) (map[string]string, error) {
	// id := GetUSBDevId(vendorId, deviceId, bus, addr)
	busI, err := strconv.Atoi(bus)
	if err != nil {
		return nil, errors.Wrapf(err, "parse bus to int %q", bus)
	}
	addrI, err := strconv.Atoi(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "parse addr to int %q", bus)
	}
	return map[string]string{
		// "id": id,
		// "bus":       "usb.0",
		"vendorid":  fmt.Sprintf("0x%s", vendorId),
		"productid": fmt.Sprintf("0x%s", deviceId),
		"hostbus":   fmt.Sprintf("%d", busI),
		"hostaddr":  fmt.Sprintf("%d", addrI),
	}, nil
}

func GetUSBDevQemuOptions(vendorDevId string, addr string) (map[string]string, error) {
	parts := strings.Split(vendorDevId, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid vendor_device_id %q", vendorDevId)
	}
	vendorId := parts[0]
	productId := parts[1]

	addrParts := strings.Split(addr, ":")
	if len(addrParts) != 2 {
		return nil, fmt.Errorf("invalid addr %q", addr)
	}
	hostBus := addrParts[0]
	hostAddr := addrParts[1]

	return getUSBDevQemuOptions(vendorId, productId, hostBus, hostAddr)
}

func (dev *sUSBDevice) GetKernelDriver() (string, error) {
	return "", nil
}

func (dev *sUSBDevice) GetQemuId() string {
	addrParts := strings.Split(dev.dev.Addr, ":")
	return GetUSBDevId(dev.dev.VendorId, dev.dev.DeviceId, addrParts[0], addrParts[1])
}

func (dev *sUSBDevice) GetPassthroughOptions() map[string]string {
	opts, _ := GetUSBDevQemuOptions(dev.dev.GetVendorDeviceId(), dev.dev.Addr)
	return opts
}

func (dev *sUSBDevice) GetPassthroughCmd(index int) string {
	opts, _ := GetUSBDevQemuOptions(dev.dev.GetVendorDeviceId(), dev.dev.Addr)
	optsStr := []string{}
	for k, v := range opts {
		optsStr = append(optsStr, fmt.Sprintf("%s=%s", k, v))
	}
	opt := fmt.Sprintf(" -device usb-host,%s", strings.Join(optsStr, ","))
	return opt
}

func (dev *sUSBDevice) GetHotPlugOptions(*desc.SGuestIsolatedDevice) ([]*HotPlugOption, error) {
	opts, err := GetUSBDevQemuOptions(dev.dev.GetVendorDeviceId(), dev.dev.Addr)
	if err != nil {
		return nil, errors.Wrap(err, "GetUSBDevQemuOptions")
	}
	return []*HotPlugOption{
		{
			Device:  "usb-host",
			Options: opts,
		},
	}, nil
}

func (dev *sUSBDevice) GetHotUnplugOptions(*desc.SGuestIsolatedDevice) ([]*HotUnplugOption, error) {
	return []*HotUnplugOption{
		{Id: dev.GetQemuId()},
	}, nil
}

func getPassthroughUSBs() ([]*sUSBDevice, error) {
	ret, err := bashOutput("lsusb")
	if err != nil {
		return nil, errors.Wrap(err, "execute lsusb")
	}
	lines := []string{}
	for _, l := range ret {
		if len(l) != 0 {
			lines = append(lines, l)
		}
	}

	devs, err := parseLsusb(lines)
	if err != nil {
		return nil, errors.Wrap(err, "parseLsusb")
	}

	// fitler linux root hub
	retDev := make([]*sUSBDevice, 0)
	for _, dev := range devs {
		// REF: https://github.com/virt-manager/virt-manager/blob/0038d750c9056ddd63cb48b343e451f8db2746fa/virtinst/nodedev.py#L142
		if isUSBLinuxRootHub(dev.dev.VendorId, dev.dev.DeviceId) {
			continue
		}
		retDev = append(retDev, dev)
	}
	return retDev, nil
}

func isUSBLinuxRootHub(vendorId string, deviceId string) bool {
	if vendorId == "1d6b" && utils.IsInStringArray(deviceId, []string{"0001", "0002", "0003"}) {
		return true
	}
	return false
}

func parseLsusb(lines []string) ([]*sUSBDevice, error) {
	devs := make([]*sUSBDevice, 0)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		dev, err := parseLsusbLine(line)
		if err != nil {
			return nil, errors.Wrapf(err, "parseLsusbLine %q", line)
		}
		usbDev := newUSBDevice(dev.ToPCIDevice())
		devs = append(devs, usbDev)
	}
	return devs, nil
}

var (
	lsusbRegex = `^Bus (?P<bus_id>([0-9]{3})) Device (?P<device>([0-9]{3})): ID (?P<vendor_id>([0-9a-z]{4})):(?P<device_id>([0-9a-z]{4}))\s{0,1}(?P<name>(.*))`
)

type sLsusbLine struct {
	BusId    string `json:"bus_id"`
	Device   string `json:"device"`
	VendorId string `json:"vendor_id"`
	DeviceId string `json:"device_id"`
	Name     string `json:"name"`
}

func parseLsusbLine(line string) (*sLsusbLine, error) {
	ret := regutils2.SubGroupMatch(lsusbRegex, line)
	dev := new(sLsusbLine)
	if err := jsonutils.Marshal(ret).Unmarshal(dev); err != nil {
		return nil, err
	}
	return dev, nil
}

func (dev *sLsusbLine) ToPCIDevice() *PCIDevice {
	return &PCIDevice{
		Addr:      fmt.Sprintf("%s:%s", dev.BusId, dev.Device),
		VendorId:  dev.VendorId,
		DeviceId:  dev.DeviceId,
		ModelName: dev.Name,
	}
}
