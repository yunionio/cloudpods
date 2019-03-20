package cmdline

import (
	"reflect"
	"testing"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/apis/compute"
)

func TestFetchServerConfigsByJSON(t *testing.T) {
	type args struct {
		obj jsonutils.JSONObject
	}
	tests := []struct {
		name    string
		args    args
		want    *compute.ServerConfigs
		wantErr bool
	}{
		{
			name: "all configs",
			args: args{jsonutils.Marshal(map[string]string{
				"disk.0":                  "1g:centos",
				"net.0":                   "192.168.222.3:inf0",
				"schedtag.0":              "ssd:require",
				"schedtag.1":              "container:exclude",
				"isolated_device.0":       "vendor=nvidia:p400",
				"baremetal_disk_config.0": "raid0:[1,2]",
			})},
			want: &compute.ServerConfigs{
				Disks: []*compute.DiskConfig{
					{
						Index:   0,
						SizeMb:  1024,
						ImageId: "centos",
						Medium:  "hybrid",
					},
				},
				Networks: []*compute.NetworkConfig{
					{
						Network: "inf0",
						Address: "192.168.222.3",
					},
				},
				Schedtags: []*compute.SchedtagConfig{
					{
						Id:       "ssd",
						Strategy: "require",
					},
					{
						Id:       "container",
						Strategy: "exclude",
					},
				},
				IsolatedDevices: []*compute.IsolatedDeviceConfig{
					{
						Vendor: "nvidia",
						Model:  "p400",
					},
				},
				BaremetalDiskConfigs: []*compute.BaremetalDiskConfig{
					{
						Type:  "hybrid",
						Conf:  "raid0",
						Range: []int64{1, 2},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FetchServerConfigsByJSON(tt.args.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("FetchServerConfigsByJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gotJ := jsonutils.Marshal(got).String()
			wantJ := jsonutils.Marshal(tt.want).String()
			if !reflect.DeepEqual(gotJ, wantJ) {
				t.Errorf("FetchServerConfigsByJSON() = %v, want %v", gotJ, wantJ)
			}
		})
	}
}
