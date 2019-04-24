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

package sysutils

import (
	"fmt"
	"net"
	"reflect"
	"testing"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
)

func Test_valueOfKeyword(t *testing.T) {
	type args struct {
		line string
		key  string
	}
	t1 := "Manufacturer: LENOVO"
	w1 := "LENOVO"
	tests := []struct {
		name string
		args args
		want *string
	}{
		{
			name: "EmptyInput",
			args: args{
				line: "",
				key:  "Family",
			},
			want: nil,
		},
		{
			name: "NormalInput",
			args: args{
				line: t1,
				key:  "manufacturer:",
			},
			want: &w1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := valueOfKeyword(tt.args.line, tt.args.key); got != nil && *got != *(tt.want) {
				t.Errorf("valueOfKeyword() = %q, want %q", *got, *(tt.want))
			} else if got == nil && tt.want != nil {
				t.Errorf("valueOfKeyword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseDMISysinfo(t *testing.T) {
	type args struct {
		lines []string
	}
	tests := []struct {
		name    string
		args    args
		want    *types.SDMISystemInfo
		wantErr bool
	}{
		{
			name:    "EmptyInput",
			args:    args{lines: nil},
			want:    nil,
			wantErr: true,
		},
		{
			name: "NormalInput",
			args: args{lines: []string{
				"Handle 0x000C, DMI type 1, 27 bytes",
				"System Information",
				"        Manufacturer: LENOVO",
				"        Product Name: 20J6CTO1WW",
				"        Version: ThinkPad T470p",
				"        Serial Number: PF112JKK",
				"        UUID: bca177cc-2bce-11b2-a85c-e98996f19d2f",
				"        SKU Number: LENOVO_MT_20J6_BU_Think_FM_ThinkPad T470p",
			}},
			want: &types.SDMISystemInfo{
				Manufacture: "LENOVO",
				Model:       "20J6CTO1WW",
				Version:     "ThinkPad T470p",
				SN:          "PF112JKK",
			},
			wantErr: false,
		},
		{
			name: "NoneVersionInput",
			args: args{lines: []string{
				"        Product Name: 20J6CTO1WW",
				"        Version: None",
				"        Serial Number: PF112JKK",
			}},
			want: &types.SDMISystemInfo{
				Model:   "20J6CTO1WW",
				Version: "",
				SN:      "PF112JKK",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDMISysinfo(tt.args.lines)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDMISysinfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseDMISysinfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseCPUInfo(t *testing.T) {
	type args struct {
		lines []string
	}
	tests := []struct {
		name    string
		args    args
		want    *types.SCPUInfo
		wantErr bool
	}{
		{
			name: "NormalInput",
			args: args{lines: []string{
				"model name      : Intel(R) Xeon(R) CPU E5-2680 v2 @ 2.80GHz",
				"cpu MHz         : 2793.238",
				"processor       : 0",
				"processor       : 1",
				"cache size      : 16384 KB",
			}},
			want: &types.SCPUInfo{
				Model: "Intel(R) Xeon(R) CPU E5-2680 v2 @ 2.80GHz",
				Count: 2,
				Freq:  2793,
				Cache: 16384,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCPUInfo(tt.args.lines)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCPUInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseCPUInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseDMICPUInfo(t *testing.T) {
	type args struct {
		lines []string
	}
	tests := []struct {
		name string
		args args
		want *types.SDMICPUInfo
	}{
		{
			name: "NormalInput",
			args: args{
				lines: []string{"Processor Information"},
			},
			want: &types.SDMICPUInfo{Nodes: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseDMICPUInfo(tt.args.lines); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseDMICPUInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseDMIMemInfo(t *testing.T) {
	type args struct {
		lines []string
	}
	tests := []struct {
		name string
		args args
		want *types.SDMIMemInfo
	}{
		{
			name: "NormalInputMB",
			args: args{
				lines: []string{
					"        Size: 16384 MB",
					"        Size: No Module Installed"},
			},
			want: &types.SDMIMemInfo{Total: 16384},
		},
		{
			name: "NormalInputGB",
			args: args{
				lines: []string{
					"        Size: 16 GB",
					"        Size: No Module Installed"},
			},
			want: &types.SDMIMemInfo{Total: 16 * 1024},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseDMIMemInfo(tt.args.lines); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseDMIMemInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseNicInfo(t *testing.T) {
	type args struct {
		lines []string
	}
	mac1Str := "00:22:25:0b:ab:49"
	mac2Str := "00:22:25:0b:ab:50"
	mac1, _ := net.ParseMAC(mac1Str)
	mac2, _ := net.ParseMAC(mac2Str)
	tests := []struct {
		name string
		args args
		want []*types.SNicDevInfo
	}{
		{
			name: "NormalInput",
			args: args{
				lines: []string{
					fmt.Sprintf("eth0 %s 0 1 1500", mac1Str),
					fmt.Sprintf("eth1 %s 0 0 1500", mac2Str),
				},
			},
			want: []*types.SNicDevInfo{
				{Dev: "eth0", Mac: mac1, Speed: 0, Up: true, Mtu: 1500},
				{Dev: "eth1", Mac: mac2, Speed: 0, Up: false, Mtu: 1500},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseNicInfo(tt.args.lines)
			gotJson := jsonutils.Marshal(got).String()
			wantJson := jsonutils.Marshal(tt.want).String()
			if gotJson != wantJson {
				t.Errorf("ParseNicInfo() = %s, want %s", gotJson, wantJson)
			}
		})
	}
}
