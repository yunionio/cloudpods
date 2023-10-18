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

package utils

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

func TestGenerateTelegrafConf(t *testing.T) {
	type args struct {
		serverDetails *api.ServerDetails
		influxdbUrl   string
		osType        string
		serverType    string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "test_empty",
			args:    args{new(api.ServerDetails), "", "Linux", "guest"},
			want:    "",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateTelegrafConf(tt.args.serverDetails, tt.args.influxdbUrl, tt.args.osType, tt.args.serverType)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateTelegrafConf() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			t.Logf("got %s", got)
		})
	}
}
