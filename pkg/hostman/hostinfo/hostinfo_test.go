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
