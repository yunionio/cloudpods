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

package uefi

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	TestPCOutput = `BootCurrent: 0007
Timeout: 2 seconds
BootOrder: 0003,0002,0001,0000,0004,0005,0007,0008,0009,000A
Boot0000  Windows Boot Manager
Boot0001  ubuntu
Boot0002  ubuntu
Boot0003  CentOS Linux
Boot0004* Onboard NIC(IPV4)
Boot0005  Onboard NIC(IPV6)
Boot0007* PXE IPv4 Intel(R) Ethernet Server Adapter X520-2
Boot0008  PXE IPv6 Intel(R) Ethernet Server Adapter X520-2
Boot0009* PXE IPv4 Intel(R) Ethernet Server Adapter X520-2
Boot000A  PXE IPv6 Intel(R) Ethernet Server Adapter X520-2`

	TestQemuOutput = `BootCurrent: 0005
Timeout: 0 seconds
BootOrder: 0005,0009,0008,0007,0002,0003,0004,0006,0001,0000
Boot0000* UiApp
Boot0001* UEFI QEMU DVD-ROM QM00003
Boot0002* UEFI Floppy
Boot0003* UEFI Floppy 2
Boot0004* UEFI QEMU HARDDISK QM00001
Boot0005* UEFI PXEv4 (MAC:525400123456)
Boot0006* EFI Internal Shell
Boot0007* CentOS
Boot0008* CentOS Linux
Boot0009* ubuntu`
)

func Test_parseTimeout(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{
			input: "Timeout: 0 seconds",
			want:  0,
		},
		{
			input: "Timeout: 30 seconds",
			want:  30,
		},
		{
			input: "Timeout: seconds",
			want:  -1,
		},
	}

	for _, tc := range tests {
		if got := parseEFIBootMGRTimeout(tc.input); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("parseEFIBootMGRTimeout() = %v, want %v", got, tc.want)
		}
	}
}

func Test_parseEFIBootMGRBootOrder(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{
			input: "BootOrder: 0005,0009,0008,0007,0002,0003,0004,0006,0001,0000",
			want:  []string{"0005", "0009", "0008", "0007", "0002", "0003", "0004", "0006", "0001", "0000"},
		},
	}

	for _, tc := range tests {
		if got := parseEFIBootMGRBootOrder(tc.input); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("parseEFIBootMGRBootOrder() = %v, want %v", got, tc.want)
		}
	}
}

func Test_parseEFIBootMGRBootCurrent(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "BootCurrent: 0005",
			line: "BootCurrent: 0005",
			want: "0005",
		},
		{
			name: "BootCurrent: ",
			line: "BootCurrent: ",
			want: "",
		},
		{
			name: "BootCurrent:",
			line: "BootCurrent:",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseEFIBootMGRBootCurrent(tt.line); got != tt.want {
				t.Errorf("parseEFIBootMGRBootCurrent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseEFIBootMGREntry(t *testing.T) {
	tests := []struct {
		name string
		line string
		want *BootEntry
	}{
		{
			name: "Boot0009* ubuntu",
			line: "Boot0009* ubuntu",
			want: &BootEntry{
				BootNum:     "0009",
				Description: "ubuntu",
				IsActive:    true,
			},
		},
		{
			name: "Boot0005* UEFI PXEv4 (MAC:525400123456)",
			line: "Boot0005* UEFI PXEv4 (MAC:525400123456)",
			want: &BootEntry{
				BootNum:     "0005",
				Description: "UEFI PXEv4 (MAC:525400123456)",
				IsActive:    true,
			},
		},
		{
			name: "Boot0003  CentOS Linux",
			line: "Boot0003  CentOS Linux",
			want: &BootEntry{
				BootNum:     "0003",
				Description: "CentOS Linux",
				IsActive:    false,
			},
		},
		{
			name: "Timeout: 2 seconds",
			line: "Timeout: 2 seconds",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseEFIBootMGREntry(tt.line); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseEFIBootMGREntry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseEFIBootMGR(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *BootMgr
		wantErr bool
	}{
		{
			name:  "Parse PC output",
			input: TestPCOutput,
			want: &BootMgr{
				bootCurrent: "0007",
				timeout:     2,
				bootOrder:   []string{"0003", "0002", "0001", "0000", "0004", "0005", "0007", "0008", "0009", "000A"},
				entries: map[string]*BootEntry{
					"0000": {
						BootNum:     "0000",
						IsActive:    false,
						Description: "Windows Boot Manager",
					},
					"0001": {
						BootNum:     "0001",
						IsActive:    false,
						Description: "ubuntu",
					},
					"0002": {
						BootNum:     "0002",
						IsActive:    false,
						Description: "ubuntu",
					},
					"0003": {
						BootNum:     "0003",
						IsActive:    false,
						Description: "CentOS Linux",
					},
					"0004": {
						BootNum:     "0004",
						IsActive:    true,
						Description: "Onboard NIC(IPV4)",
					},
					"0005": {
						BootNum:     "0005",
						IsActive:    false,
						Description: "Onboard NIC(IPV6)",
					},
					"0007": {
						BootNum:     "0007",
						IsActive:    true,
						Description: "PXE IPv4 Intel(R) Ethernet Server Adapter X520-2",
					},
					"0008": {
						BootNum:     "0008",
						IsActive:    false,
						Description: "PXE IPv6 Intel(R) Ethernet Server Adapter X520-2",
					},
					"0009": {
						BootNum:     "0009",
						IsActive:    true,
						Description: "PXE IPv4 Intel(R) Ethernet Server Adapter X520-2",
					},
					"000A": {
						BootNum:     "000A",
						IsActive:    false,
						Description: "PXE IPv6 Intel(R) Ethernet Server Adapter X520-2",
					},
				}},
		},
	}

	assert := assert.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEFIBootMGR(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEFIBootMGR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if equal := assert.Equal(tt.want, got); !equal {
				t.Errorf("ParseEFIBootMGR() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_stringArraryMove(t *testing.T) {
	type args struct {
		items []string
		item  string
		pos   int
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "move left to right",
			args: args{
				items: []string{"1", "2", "3"},
				item:  "3",
				pos:   0,
			},
			want: []string{"3", "1", "2"},
		},
		{
			name: "move right to left",
			args: args{
				items: []string{"1", "2", "3"},
				item:  "1",
				pos:   2,
			},
			want: []string{"2", "3", "1"},
		},
		{
			name: "no move",
			args: args{
				items: []string{"1", "2", "3"},
				item:  "2",
				pos:   1,
			},
			want: []string{"1", "2", "3"},
		},
		{
			name: "add item move",
			args: args{
				items: []string{"1", "2", "3"},
				item:  "4",
				pos:   0,
			},
			want: []string{"4", "1", "2", "3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stringArraryMove(tt.args.items, tt.args.item, tt.args.pos); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stringArraryMove() = %v, want %v", got, tt.want)
			}
		})
	}
}
