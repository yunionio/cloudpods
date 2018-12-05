package sysutils

import (
	"reflect"
	"testing"

	"yunion.io/x/onecloud/pkg/baremetal/types"
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
		want    *types.DMIInfo
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
			want: &types.DMIInfo{
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
			want: &types.DMIInfo{
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
		want    *types.CPUInfo
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
			want: &types.CPUInfo{
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
		want *types.DMICPUInfo
	}{
		{
			name: "NormalInput",
			args: args{
				lines: []string{"Processor Information"},
			},
			want: &types.DMICPUInfo{Nodes: 1},
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
		want *types.DMIMemInfo
	}{
		{
			name: "NormalInputMB",
			args: args{
				lines: []string{
					"        Size: 16384 MB",
					"        Size: No Module Installed"},
			},
			want: &types.DMIMemInfo{Total: 16384},
		},
		{
			name: "NormalInputGB",
			args: args{
				lines: []string{
					"        Size: 16 GB",
					"        Size: No Module Installed"},
			},
			want: &types.DMIMemInfo{Total: 16 * 1024},
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
	tests := []struct {
		name string
		args args
		want []*types.NicDevInfo
	}{
		{
			name: "NormalInput",
			args: args{
				lines: []string{
					"eth0 00:22:25:0b:ab:49 0 1 1500",
					"eth1 00:22:25:0b:ab:50 0 0 1500",
				},
			},
			want: []*types.NicDevInfo{
				&types.NicDevInfo{Dev: "eth0", Mac: "00:22:25:0b:ab:49", Speed: 0, Up: true, Mtu: 1500},
				&types.NicDevInfo{Dev: "eth1", Mac: "00:22:25:0b:ab:50", Speed: 0, Up: false, Mtu: 1500},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseNicInfo(tt.args.lines); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseNicInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}
