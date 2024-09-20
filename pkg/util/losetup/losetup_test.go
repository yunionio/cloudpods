package losetup

import (
	"reflect"
	"testing"
)

func Test_parseJsonOutput(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    *Devices
		wantErr bool
	}{
		{
			name: "normal input",
			content: `{
   "loopdevices": [
      {
         "name": "/dev/loop1",
         "sizelimit": 0,
         "offset": 0,
         "autoclear": true,
         "ro": false,
         "back-file": "/opt/cloud/workspace/disks/recycle_bin/20240911160650/194767a9-556f-4072-8a20-7ab1086f22ff.1726070810",
         "dio": false,
         "log-sec": 512
      },{
         "name": "/dev/loop57",
         "sizelimit": 0,
         "offset": 0,
         "autoclear": false,
         "ro": false,
         "back-file": "/opt/cloud/workspace/disks/2c10325f-7109-4cf0-8a9f-4c3724920245 (deleted)",
         "dio": false,
         "log-sec": 512
      }]}`,
			want: &Devices{
				LoopDevs: []Device{
					{
						Name:      "/dev/loop1",
						BackFile:  "/opt/cloud/workspace/disks/recycle_bin/20240911160650/194767a9-556f-4072-8a20-7ab1086f22ff.1726070810",
						SizeLimit: false,
						ReadOnly:  false,
					},
					{
						Name:      "/dev/loop57",
						BackFile:  "/opt/cloud/workspace/disks/2c10325f-7109-4cf0-8a9f-4c3724920245 (deleted)",
						SizeLimit: false,
						ReadOnly:  false,
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseJsonOutput(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseJsonOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseJsonOutput() got = %v, want %v", got, tt.want)
			}
		})
	}
}
