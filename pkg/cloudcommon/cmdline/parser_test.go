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

package cmdline

import (
	"reflect"
	"testing"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis/compute"
)

func TestParseSchedtagConfig(t *testing.T) {
	type args struct {
		desc string
	}
	tests := []struct {
		name    string
		args    args
		want    *compute.SchedtagConfig
		wantErr bool
	}{
		{
			name:    "empty input",
			args:    args{""},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "normal",
			args:    args{"ssd:require"},
			want:    &compute.SchedtagConfig{Id: "ssd", Strategy: "require"},
			wantErr: false,
		},
		{
			name:    "with resource type",
			args:    args{"ssd:require:zones"},
			want:    &compute.SchedtagConfig{Id: "ssd", Strategy: "require", ResourceType: "zones"},
			wantErr: false,
		},
		{
			name:    "invalid strategy",
			args:    args{"ssd:require2"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSchedtagConfig(tt.args.desc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSchedtagConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseSchedtagConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseRange(t *testing.T) {
	type args struct {
		rangeStr string
	}
	tests := []struct {
		name    string
		args    args
		wantRet []int64
		wantErr bool
	}{
		{
			name:    "range number and , test",
			args:    args{"11-7,, 9-9, 10, 15, 1-2,4-5,,"},
			wantRet: []int64{1, 2, 4, 5, 7, 8, 9, 10, 11, 15},
			wantErr: false,
		},
		{
			name:    "range,range",
			args:    args{"1-2,4-5"},
			wantRet: []int64{1, 2, 4, 5},
			wantErr: false,
		},
		{
			name:    "numbers",
			args:    args{"1, 1"},
			wantRet: []int64{1},
			wantErr: false,
		},
		{
			name:    "numbersErr",
			args:    args{"1-, 1"},
			wantRet: nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRet, err := ParseRange(tt.args.rangeStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotRet, tt.wantRet) {
				t.Errorf("ParseRange() = %v, want %v", gotRet, tt.wantRet)
			}
		})
	}
}

func TestFetchDiskConfigsByJSON(t *testing.T) {
	type args struct {
		obj jsonutils.JSONObject
	}

	testDisks := []*compute.DiskConfig{
		{
			SizeMb: 1024,
			Medium: compute.DISK_TYPE_SSD,
		},
		{
			SizeMb: 2048,
			Medium: compute.DISK_TYPE_ROTATE,
		},
	}

	testDisksObj := map[string][]*compute.DiskConfig{"disks": testDisks}

	tests := []struct {
		name    string
		args    args
		want    []*compute.DiskConfig
		wantErr bool
	}{
		{
			name:    "empty",
			args:    args{nil},
			want:    nil,
			wantErr: false,
		},
		{
			name: "parse strings",
			args: args{obj: jsonutils.Marshal(
				map[string]string{
					"disk.0": "10g:/data1",
					"disk.1": "20g:ext4:ssd",
				},
			)},
			want: []*compute.DiskConfig{
				{
					Index:      0,
					SizeMb:     10 * 1024,
					Mountpoint: "/data1",
				},
				{
					Index:  1,
					SizeMb: 20 * 1024,
					Fs:     "ext4",
					Medium: compute.DISK_TYPE_SSD,
				},
			},
			wantErr: false,
		},
		{
			name:    "parse struct",
			args:    args{obj: jsonutils.Marshal(testDisksObj)},
			want:    testDisks,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FetchDiskConfigsByJSON(tt.args.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchDiskConfigsByJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FetchDiskConfigsByJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseIsolatedDevice(t *testing.T) {
	tests := []struct {
		desc    string
		idx     int
		want    *compute.IsolatedDeviceConfig
		wantErr bool
	}{
		{
			desc: "device_path=/dev/nvme0",
			idx:  0,
			want: &compute.IsolatedDeviceConfig{
				DevicePath: "/dev/nvme0",
			},
		},
		{
			desc: "GPU_HPC:device_path=/dev/nvme0",
			idx:  0,
			want: &compute.IsolatedDeviceConfig{
				DevicePath: "/dev/nvme0",
				Model:      "GPU_HPC",
			},
		},
		{
			desc: "GPU_HPC:device_path=/dev/nvme0:1d3ce781-2b64-4ee7-8d23-6bbcf478c54b",
			idx:  0,
			want: &compute.IsolatedDeviceConfig{
				Id:         "1d3ce781-2b64-4ee7-8d23-6bbcf478c54b",
				DevicePath: "/dev/nvme0",
				Model:      "GPU_HPC",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := ParseIsolatedDevice(tt.desc, tt.idx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIsolatedDevice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseIsolatedDevice() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseBaremetalRootDiskMatcher(t *testing.T) {
	tests := []struct {
		args    string
		want    *compute.BaremetalRootDiskMatcher
		wantErr bool
	}{
		{
			args: "size=100G",
			want: &compute.BaremetalRootDiskMatcher{SizeMB: 102400},
		},
		{
			args: "device=/dev/sda",
			want: &compute.BaremetalRootDiskMatcher{Device: "/dev/sda"},
		},
		{
			args: "size_end=100G",
			want: &compute.BaremetalRootDiskMatcher{SizeMBRange: &compute.RootDiskMatcherSizeMBRange{End: 102400}},
		},
		{
			args: "size_end=100G,size_start=50G",
			want: &compute.BaremetalRootDiskMatcher{
				SizeMBRange: &compute.RootDiskMatcherSizeMBRange{
					Start: 51200,
					End:   102400,
				}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.args, func(t *testing.T) {
			got, err := ParseBaremetalRootDiskMatcher(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBaremetalRootDiskMatcher() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseBaremetalRootDiskMatcher() got = %v, want %v", got, tt.want)
			}
		})
	}
}
