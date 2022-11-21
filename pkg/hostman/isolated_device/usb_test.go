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
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseLsusbLine(t *testing.T) {
	tests := []struct {
		name    string
		lines   []string
		want    []*sLsusbLine
		wantErr bool
	}{
		{
			name: "test",
			lines: []string{
				"Bus 004 Device 001: ID 1d6b:0003 Linux Foundation 3.0 root hub",
				"Bus 003 Device 005: ID 13d3:3563 IMC Networks Wireless_Device",
				"Bus 003 Device 003: ID 27c6:521d Shenzhen Goodix Technology Co.,Ltd. FingerPrint",
				"Bus 003 Device 006: ID 0451:82ff Texas Instruments, Inc.",
				"Bus 003 Device 008: ID feed:19c0 YANG HHKB BLE S (USB_DL1K)",
				"Bus 003 Device 007: ID 214b:7250 Huasheng Electronics USB2.0 HUB",
				"Bus 003 Device 002: ID 0451:8442 Texas Instruments, Inc.",
				"Bus 003 Device 001: ID 1d6b:0002 Linux Foundation 2.0 root hub",
				"Bus 002 Device 001: ID 1d6b:0003 Linux Foundation 3.0 root hub",
				"Bus 001 Device 004: ID 0b05:193b ASUSTek Computer, Inc. ITE Device(8910)",
				"Bus 001 Device 003: ID 0b05:19b6 ASUSTek Computer, Inc. N-KEY Device",
				"Bus 001 Device 002: ID 046d:c52f Logitech, Inc. Unifying Receiver",
				"Bus 001 Device 001: ID 1d6b:0002 Linux Foundation 2.0 root hub",
			},
			want: []*sLsusbLine{
				{BusId: "004", Device: "001", VendorId: "1d6b", DeviceId: "0003", Name: "Linux Foundation 3.0 root hub"},
				{BusId: "003", Device: "005", VendorId: "13d3", DeviceId: "3563", Name: "IMC Networks Wireless_Device"},
				{BusId: "003", Device: "003", VendorId: "27c6", DeviceId: "521d", Name: "Shenzhen Goodix Technology Co.,Ltd. FingerPrint"},
				{BusId: "003", Device: "006", VendorId: "0451", DeviceId: "82ff", Name: "Texas Instruments, Inc."},
				{BusId: "003", Device: "008", VendorId: "feed", DeviceId: "19c0", Name: "YANG HHKB BLE S (USB_DL1K)"},
				{BusId: "003", Device: "007", VendorId: "214b", DeviceId: "7250", Name: "Huasheng Electronics USB2.0 HUB"},
				{BusId: "003", Device: "002", VendorId: "0451", DeviceId: "8442", Name: "Texas Instruments, Inc."},
				{BusId: "003", Device: "001", VendorId: "1d6b", DeviceId: "0002", Name: "Linux Foundation 2.0 root hub"},
				{BusId: "002", Device: "001", VendorId: "1d6b", DeviceId: "0003", Name: "Linux Foundation 3.0 root hub"},
				{BusId: "001", Device: "004", VendorId: "0b05", DeviceId: "193b", Name: "ASUSTek Computer, Inc. ITE Device(8910)"},
				{BusId: "001", Device: "003", VendorId: "0b05", DeviceId: "19b6", Name: "ASUSTek Computer, Inc. N-KEY Device"},
				{BusId: "001", Device: "002", VendorId: "046d", DeviceId: "c52f", Name: "Logitech, Inc. Unifying Receiver"},
				{BusId: "001", Device: "001", VendorId: "1d6b", DeviceId: "0002", Name: "Linux Foundation 2.0 root hub"},
			},
			wantErr: false,
		},
		{
			name: "with_incomplete_line",
			lines: []string{
				"Bus 004 Device 002: ID 17aa:1033",
				"Bus 003 Device 002: ID 17aa:1033",
			},
			want: []*sLsusbLine{
				{BusId: "004", Device: "002", VendorId: "17aa", DeviceId: "1033", Name: ""},
				{BusId: "003", Device: "002", VendorId: "17aa", DeviceId: "1033", Name: ""},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i, line := range tt.lines {
				got, err := parseLsusbLine(line)
				if (err != nil) != tt.wantErr {
					t.Errorf("parseLsusb() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				busNum, _ := got.GetBusNumber()
				devNum, _ := got.GetDeviceNumber()
				t.Logf("line result: %#v, busNum: %d, devNum: %d", got, busNum, devNum)

				if !reflect.DeepEqual(got, tt.want[i]) {
					t.Errorf("parseLsusb() = %v, want %v", got, tt.want[i])
				}

			}
		})
	}
}

func Test_isUSBLinuxRootHub(t *testing.T) {
	type args struct {
		vendorId string
		deviceId string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "1d6b:0001",
			args: args{
				"1d6b",
				"0001",
			},
			want: true,
		},
		{
			name: "1d6b:0002",
			args: args{
				"1d6b",
				"0002",
			},
			want: true,
		},
		{
			name: "1d6b:0003",
			args: args{
				"1d6b",
				"0003",
			},
			want: true,
		},
		{
			name: "1d6b:0004",
			args: args{
				"1d6b",
				"0004",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUSBLinuxRootHub(tt.args.vendorId, tt.args.deviceId); got != tt.want {
				t.Errorf("isUSBLinuxRootHub() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getUSBDevQemuOptions(t *testing.T) {
	type args struct {
		vendorId string
		deviceId string
		bus      string
		addr     string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "",
			args: args{
				vendorId: "1d6b",
				deviceId: "0001",
				bus:      "001",
				addr:     "009",
			},
			want: map[string]string{
				"vendorid":  "0x1d6b",
				"productid": "0x0001",
				"hostbus":   "1",
				"hostaddr":  "9",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := getUSBDevQemuOptions(tt.args.vendorId, tt.args.deviceId, tt.args.bus, tt.args.addr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getUSBDevQemuOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newLsusbRootBusTreeByLine(t *testing.T) {
	assert := assert.New(t)
	tree, _ := newLsusbTreeByLine("/:  Bus 04.Port 1: Dev 1, Class=root_hub, Driver=xhci_hcd/2p, 10000M")
	assert.Equal(4, tree.Bus)
	assert.Equal(1, tree.Port)
	assert.Equal(1, tree.Dev)
	assert.Equal(true, tree.IsRootBus)
	assert.Equal("xhci_hcd/2p", tree.Driver)

	tree, _ = newLsusbTreeByLine("    |__ Port 3: Dev 2, If 0, Class=Vendor Specific Class, Driver=, 12M")
	assert.Equal(false, tree.IsRootBus)
	assert.Equal("", tree.Driver)
	assert.Equal("Vendor Specific Class", tree.Class)
	assert.Equal(3, tree.Port)
	assert.Equal(2, tree.Dev)
	assert.Equal(0, tree.If)
}

func Test_parseLsusbTrees(t *testing.T) {
	assert := assert.New(t)
	input := `
/:  Bus 04.Port 1: Dev 1, Class=root_hub, Driver=xhci_hcd/2p, 10000M
/:  Bus 03.Port 1: Dev 1, Class=root_hub, Driver=xhci_hcd/4p, 480M
    |__ Port 3: Dev 2, If 0, Class=Vendor Specific Class, Driver=, 12M
    |__ Port 4: Dev 3, If 0, Class=Wireless, Driver=btusb, 480M
    |__ Port 4: Dev 3, If 1, Class=Wireless, Driver=btusb, 480M
    |__ Port 4: Dev 3, If 2, Class=Wireless, Driver=, 480M
/:  Bus 02.Port 1: Dev 1, Class=root_hub, Driver=xhci_hcd/2p, 10000M
/:  Bus 01.Port 1: Dev 1, Class=root_hub, Driver=xhci_hcd/4p, 480M
    |__ Port 3: Dev 2, If 0, Class=Human Interface Device, Driver=usbhid, 12M
    |__ Port 3: Dev 2, If 1, Class=Human Interface Device, Driver=usbhid, 12M
    |__ Port 3: Dev 2, If 2, Class=Human Interface Device, Driver=usbhid, 12M
    |__ Port 4: Dev 3, If 0, Class=Human Interface Device, Driver=usbfs, 12M
	`
	ts, err := parseLsusbTrees(strings.Split(input, "\n"))
	assert.Equal(nil, err)
	assert.Equal(
		`/:  Bus 01.Port 1: Dev 1, Class=root_hub, Driver=xhci_hcd/4p, 480M
    |__ Port 3: Dev 2, If 0, Class=Human Interface Device, Driver=usbhid, 12M
    |__ Port 3: Dev 2, If 1, Class=Human Interface Device, Driver=usbhid, 12M
    |__ Port 3: Dev 2, If 2, Class=Human Interface Device, Driver=usbhid, 12M
    |__ Port 4: Dev 3, If 0, Class=Human Interface Device, Driver=usbfs, 12M
/:  Bus 02.Port 1: Dev 1, Class=root_hub, Driver=xhci_hcd/2p, 10000M
/:  Bus 03.Port 1: Dev 1, Class=root_hub, Driver=xhci_hcd/4p, 480M
    |__ Port 3: Dev 2, If 0, Class=Vendor Specific Class, Driver=, 12M
    |__ Port 4: Dev 3, If 0, Class=Wireless, Driver=btusb, 480M
    |__ Port 4: Dev 3, If 1, Class=Wireless, Driver=btusb, 480M
    |__ Port 4: Dev 3, If 2, Class=Wireless, Driver=, 480M
/:  Bus 04.Port 1: Dev 1, Class=root_hub, Driver=xhci_hcd/2p, 10000M`, ts.GetContent())

	bus, ok := ts.GetBus(3)
	assert.Equal(true, ok)
	dt := bus.GetDevice(1)
	assert.Equal("root_hub", dt.Class)
	dt = bus.GetDevice(2)
	assert.Equal("Vendor Specific Class", dt.Class)

	input2 := `
/:  Bus 02.Port 1: Dev 1, Class=root_hub, Driver=xhci_hcd/10p, 5000M
/:  Bus 01.Port 1: Dev 1, Class=root_hub, Driver=xhci_hcd/16p, 480M
    |__ Port 1: Dev 2, If 0, Class=Hub, Driver=hub/7p, 480M
    |__ Port 10: Dev 6, If 0, Class=Human Interface Device, Driver=usbfs, 12M
`
	ts, err = parseLsusbTrees(strings.Split(input2, "\n"))
	assert.Equal(nil, err)
	bus, ok = ts.GetBus(1)
	assert.Equal(true, ok)
	dt = bus.GetDevice(6)
	assert.Equal("Human Interface Device", dt.Class)
}
