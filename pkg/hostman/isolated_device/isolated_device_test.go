package isolated_device

import (
	"reflect"
	"testing"
)

func Test_parseLspci(t *testing.T) {
	type args struct {
		line string
	}
	tests := []struct {
		name string
		args args
		want *PCIDevice
	}{
		{
			name: "3D controller",
			args: args{`02:00.0 "3D controller [0302]" "NVIDIA Corporation [10de]" "GM108M [GeForce 940MX] [134d]" -ra2 "Lenovo [17aa]" "GM108M [GeForce 940MX] [505e]`},
			want: &PCIDevice{
				Addr:          "02:00.0",
				ClassName:     "3D controller",
				ClassCode:     "0302",
				VendorName:    "NVIDIA Corporation",
				VendorId:      "10de",
				DeviceName:    "GM108M [GeForce 940MX]",
				DeviceId:      "134d",
				SubvendorName: "Lenovo",
				SubvendorId:   "17aa",
				SubdeviceName: "GM108M [GeForce 940MX]",
				SubdeviceId:   "505e",
				ModelName:     "GeForce 940MX",
			},
		},
		{
			name: "VGA",
			args: args{`05:00.0 "VGA compatible controller [0300]" "Advanced Micro Devices, Inc. [AMD/ATI] [1002]" "Oland [Radeon HD 8570 / R7 240/340 OEM] [6611]" "Dell [1028]" "Radeon R5 240 OEM [210b]"`},
			want: &PCIDevice{
				Addr:          "05:00.0",
				ClassName:     "VGA compatible controller",
				ClassCode:     "0300",
				VendorName:    "Advanced Micro Devices, Inc. [AMD/ATI]",
				VendorId:      "1002",
				DeviceName:    "Oland [Radeon HD 8570 / R7 240/340 OEM]",
				DeviceId:      "6611",
				SubvendorName: "Dell",
				SubvendorId:   "1028",
				SubdeviceName: "Radeon R5 240 OEM",
				SubdeviceId:   "210b",
				ModelName:     "Radeon HD 8570 / R7 240/340 OEM",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseLspci(tt.args.line); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseLspci() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
