package losetup

import (
	"reflect"
	"testing"
)

var (
	losetupOutput string = `
NAME       BACK-FILE                                   SIZELIMIT RO
/dev/loop0 /disks/2b917686-2ace-4a57-a4af-44ece2303dd2         0  0
/dev/loop1 /disks/033d6bc0-4ce4-48c4-89d3-125077bcc28e         0  1
`
)

func Test_parseDevices(t *testing.T) {
	type args struct {
		output string
	}
	tests := []struct {
		name    string
		args    args
		want    *Devices
		wantErr bool
	}{
		{
			name: "normalOutput",
			args: args{losetupOutput},
			want: &Devices{
				[]Device{
					Device{
						Name:      "/dev/loop0",
						BackFile:  "/disks/2b917686-2ace-4a57-a4af-44ece2303dd2",
						SizeLimit: false,
						ReadOnly:  false,
					},
					Device{
						Name:      "/dev/loop1",
						BackFile:  "/disks/033d6bc0-4ce4-48c4-89d3-125077bcc28e",
						SizeLimit: false,
						ReadOnly:  true,
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDevices(tt.args.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDevices() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseDevices() = %v, want %v", got, tt.want)
			}
		})
	}
}
