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

package megactl

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	raiddrivers "yunion.io/x/onecloud/pkg/baremetal/utils/raid"
)

func TestParseLineForStorcli(t *testing.T) {
	type testCase struct {
		name       string
		adapter    *StorcliAdaptor
		lines      []string
		assertFunc func(t *testing.T, a *StorcliAdaptor)
	}

	cases := []testCase{
		{
			name:    "Should complete",
			adapter: new(StorcliAdaptor),
			lines: []string{
				"Controller = 0",
				"Product Name = SAS3108",
				"Serial Number = 1234",
				"Bus Number = 3",
				"Device Number = 0",
				"Function Number = 0",
			},
			assertFunc: func(t *testing.T, a *StorcliAdaptor) {
				assert.Equal(t, true, a.isComplete(), a.String())
				assert.Equal(t, "SAS3108"+"1234", a.key())
			},
		},
		{
			name:    "Should complete when no space beside '='",
			adapter: new(StorcliAdaptor),
			lines: []string{
				"Controller=0",
				"Product Name = SAS3108",
				"Serial Number=1234",
				"Bus Number = 3",
				"Device Number = 0",
				"Function Number = 0",
			},
			assertFunc: func(t *testing.T, a *StorcliAdaptor) {
				assert.Equal(t, true, a.isComplete(), a.String())
				assert.Equal(t, "SAS3108"+"1234", a.key())
			},
		},
		{
			name:    "Should parse empty SN",
			adapter: new(StorcliAdaptor),
			lines: []string{
				"Controller = 0",
				"Product Name = SAS3108",
				"Serial Number =",
				"Bus Number = 3",
				"Device Number = 0",
				"Function Number = 0",
			},
			assertFunc: func(t *testing.T, a *StorcliAdaptor) {
				assert.Equal(t, true, a.isComplete(), a.String())
				assert.Equal(t, true, a.isSNEmpty, a.String())
				assert.Equal(t, "SAS3108", a.key())
			},
		},
		{
			name:    "Should parse empty SN end with space",
			adapter: new(StorcliAdaptor),
			lines: []string{
				"Controller = 0",
				"Product Name = SAS3108",
				"Serial Number =  ",
				"Bus Number = 3",
				"Device Number = 0",
				"Function Number = 0",
			},
			assertFunc: func(t *testing.T, a *StorcliAdaptor) {
				assert.Equal(t, true, a.isComplete(), a.String())
				assert.Equal(t, true, a.isSNEmpty, a.String())
				assert.Equal(t, "SAS3108", a.key())
			},
		},
		{
			name:    "Should not complete when no product name",
			adapter: new(StorcliAdaptor),
			lines: []string{
				"Controller = 0",
				"Serial Number = 1234",
				"Bus Number = 3",
				"Device Number = 0",
				"Function Number = 0",
			},
			assertFunc: func(t *testing.T, a *StorcliAdaptor) {
				assert.Equal(t, false, a.isComplete(), a.String())
			},
		},
		{
			name:    "Should not complete when no SN",
			adapter: new(StorcliAdaptor),
			lines: []string{
				"Controller = 0",
				"Product Name = SAS3108",
				"Bus Number = 3",
				"Device Number = 0",
				"Function Number = 0",
			},
			assertFunc: func(t *testing.T, a *StorcliAdaptor) {
				assert.Equal(t, false, a.isComplete(), a.String())
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, l := range c.lines {
				parseLineForStorcli(c.adapter, l)
			}
			c.assertFunc(t, c.adapter)
		})
	}
}

func Test_parseStorcliLogicalVolumes(t *testing.T) {
	type args struct {
		adapter int
		lines   []string
	}
	tests := []struct {
		name    string
		args    args
		want    []*raiddrivers.RaidLogicalVolume
		wantErr bool
	}{
		{
			name: "normal parse",
			args: args{
				adapter: 0,
				lines: []string{
					"DG/VD TYPE   State Access Consist Cache Cac sCC     Size Name",
					"0/0   RAID10 Optl  RW     Yes     RWBD  -   ON  3.270 TB sas_data_raid1",
					"1/1   RAID5  Optl  RW     Yes     RWBD  -   ON  7.275 TB sata_data_raid",
				},
			},
			want: []*raiddrivers.RaidLogicalVolume{
				{
					Index:   0,
					Adapter: 0,
				},
				{
					Index:   1,
					Adapter: 0,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseStorcliLogicalVolumes(tt.args.adapter, tt.args.lines)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseStorcliLogicalVolumes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseStorcliLogicalVolumes() = %v, want %v", got, tt.want)
			}
		})
	}
}
