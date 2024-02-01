package compute

import (
	"reflect"
	"testing"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
)

func Test_parsePodPortMapping(t *testing.T) {
	tests := []struct {
		args    string
		want    *computeapi.PodPortMapping
		wantErr bool
	}{
		{
			args: "80:8080/tcp",
			want: &computeapi.PodPortMapping{
				Protocol:      computeapi.PodPortMappingProtocolTCP,
				ContainerPort: 8080,
				HostPort:      80,
			},
			wantErr: false,
		},
		{
			args: "80:8080",
			want: &computeapi.PodPortMapping{
				Protocol:      computeapi.PodPortMappingProtocolTCP,
				ContainerPort: 8080,
				HostPort:      80,
			},
			wantErr: false,
		},
		{
			args:    "80:8080:tcp",
			want:    nil,
			wantErr: true,
		},
		{
			args:    "80",
			want:    nil,
			wantErr: true,
		},
		{
			args:    "80s:ctrP",
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.args, func(t *testing.T) {
			got, err := parsePodPortMapping(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePodPortMapping() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePodPortMapping() got = %v, want %v", got, tt.want)
			}
		})
	}
}
