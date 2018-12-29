package hostinfo

import (
	"testing"
)

func TestSHostInfo_Start(t *testing.T) {
	type fields struct {
		isRegistered     bool
		kvmModuleSupport string
		nestStatus       string
		Cpu              *SCPUInfo
		Mem              *SMemory
		sysinfo          *SSysInfo
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"HostInfo Test",
			fields{},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &SHostInfo{
				isRegistered:     tt.fields.isRegistered,
				kvmModuleSupport: tt.fields.kvmModuleSupport,
				nestStatus:       tt.fields.nestStatus,
				Cpu:              tt.fields.Cpu,
				Mem:              tt.fields.Mem,
				sysinfo:          tt.fields.sysinfo,
			}
			if err := h.Start(); (err != nil) != tt.wantErr {
				t.Errorf("SHostInfo.Start() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
