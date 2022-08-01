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
	"testing"
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
				t.Logf("line result: %#v", got)

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
