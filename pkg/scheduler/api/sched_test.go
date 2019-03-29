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

package api

import (
	"reflect"
	"testing"
)

func TestNewSchedTagFromCmdline(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name    string
		args    args
		wantAgg Aggregate
		wantErr bool
	}{
		{
			name:    "test:avoid",
			args:    args{"test:avoid"},
			wantAgg: Aggregate{Idx: "test", Strategy: "avoid"},
			wantErr: false,
		},
		{
			name:    "test empty string",
			args:    args{""},
			wantAgg: Aggregate{},
			wantErr: true,
		},
		{
			name:    "test no Strategy string",
			args:    args{"no_strategy:"},
			wantAgg: Aggregate{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAgg, err := NewSchedTagFromCmdline(tt.args.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSchedTagFromCmdline() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotAgg, tt.wantAgg) {
				t.Errorf("NewSchedTagFromCmdline() = %v, want %v", gotAgg, tt.wantAgg)
			}
		})
	}
}

func Test_newIsolatedDeviceFromDesc(t *testing.T) {
	type args struct {
		desc string
	}
	tests := []struct {
		name    string
		args    args
		wantDev *IsolatedDevice
		wantErr bool
	}{
		{
			name:    "empty string should Invalid",
			args:    args{""},
			wantDev: nil,
			wantErr: true,
		},
		{
			name:    "parse only model",
			args:    args{"1050 Ti"},
			wantDev: &IsolatedDevice{Model: "1050 Ti"},
			wantErr: false,
		},
		{
			name: "parse uuid with model",
			args: args{"1050 Ti:f5d8c180-5a76-49a5-a296-cea73c3fe5ed"},
			wantDev: &IsolatedDevice{
				ID:    "f5d8c180-5a76-49a5-a296-cea73c3fe5ed",
				Model: "1050 Ti",
			},
			wantErr: false,
		},
		{
			name: "all info",
			args: args{"1050 Ti:f5d8c180-5a76-49a5-a296-cea73c3fe5ed:GPU-HPC"},
			wantDev: &IsolatedDevice{
				ID:    "f5d8c180-5a76-49a5-a296-cea73c3fe5ed",
				Model: "1050 Ti",
				Type:  GPU_HPC_TYPE,
			},
			wantErr: false,
		},
		{
			name: "wrong type",
			args: args{"1050 Ti:f5d8c180-5a76-49a5-a296-cea73c3fe5ed:GPU-HPC-Wrong"},
			wantDev: &IsolatedDevice{
				ID:    "f5d8c180-5a76-49a5-a296-cea73c3fe5ed",
				Model: "GPU-HPC-Wrong",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDev, err := newIsolatedDeviceFromDesc(tt.args.desc)
			if (err != nil) != tt.wantErr {
				t.Errorf("newIsolatedDeviceFromDesc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotDev, tt.wantDev) {
				t.Errorf("newIsolatedDeviceFromDesc() = %v, want %v", gotDev, tt.wantDev)
			}
		})
	}
}
