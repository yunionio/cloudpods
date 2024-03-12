package container_device

import "testing"

func Test_getCphAMDGPUPCIAddr(t *testing.T) {
	tests := []struct {
		linkPartName string
		want         string
		wantErr      bool
	}{
		{
			linkPartName: "pci-0000:03:00.0-card",
			want:         "0000:03:00.0",
			wantErr:      false,
		},
		{
			linkPartName: "",
			want:         "",
			wantErr:      true,
		},
		{
			linkPartName: "pci-0000:83:00.0-render",
			want:         "0000:83:00.0",
			wantErr:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.linkPartName, func(t *testing.T) {
			got, err := getCphAMDGPUPCIAddr(tt.linkPartName)
			if (err != nil) != tt.wantErr {
				t.Errorf("getCphAMDGPUPCIAddr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getCphAMDGPUPCIAddr() got = %v, want %v", got, tt.want)
			}
		})
	}
}
