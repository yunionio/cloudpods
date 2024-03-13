package compute

import (
	"reflect"
	"testing"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
)

func Test_parsePodPortMapping(t *testing.T) {
	var port80 int = 80
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
				HostPort:      &port80,
			},
			wantErr: false,
		},
		{
			args: "80:8080",
			want: &computeapi.PodPortMapping{
				Protocol:      computeapi.PodPortMappingProtocolTCP,
				ContainerPort: 8080,
				HostPort:      &port80,
			},
			wantErr: false,
		},
		{
			args:    "80:8080:tcp",
			want:    nil,
			wantErr: true,
		},
		{
			args: "80",
			want: &computeapi.PodPortMapping{
				Protocol:      computeapi.PodPortMappingProtocolTCP,
				ContainerPort: 80,
			},
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

func Test_parsePodPortMappingDetails(t *testing.T) {
	port8080 := 8080
	tests := []struct {
		input   string
		want    *computeapi.PodPortMapping
		wantErr bool
	}{
		{
			input: "host_port=8080,port=80",
			want: &computeapi.PodPortMapping{
				Protocol:      computeapi.PodPortMappingProtocolTCP,
				ContainerPort: 80,
				HostPort:      &port8080,
			},
		},
		{
			input: "host_port=8080,port=80,protocol=udp",
			want: &computeapi.PodPortMapping{
				Protocol:      computeapi.PodPortMappingProtocolUDP,
				ContainerPort: 80,
				HostPort:      &port8080,
			},
		},
		{
			input:   "host_port=8080,protocol=udp",
			wantErr: true,
		},
		{
			input: "container_port=80,protocol=udp,host_port_range=20000-25000",
			want: &computeapi.PodPortMapping{
				Protocol:      computeapi.PodPortMappingProtocolUDP,
				ContainerPort: 80,
				HostPortRange: &computeapi.PodPortMappingPortRange{
					Start: 20000,
					End:   25000,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parsePodPortMappingDetails(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePodPortMappingDetails() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePodPortMappingDetails() got = %v, want %v", got, tt.want)
			}
		})
	}
}
