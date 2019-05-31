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

package userdata

import "testing"

func TestEncodeDecode(t *testing.T) {
	tests := []struct {
		name     string
		userdata string
	}{
		{
			name:     "equal",
			userdata: "1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enGot, err := Encode(tt.userdata)
			if err != nil {
				t.Errorf("Encode() error = %v", err)
				return
			}

			deGot, err := Decode(enGot)
			if err != nil {
				t.Errorf("Decode() error = %v", err)
				return
			}
			if tt.userdata != deGot {
				t.Errorf("decode after encode userdata = %v want %v", deGot, tt.userdata)
			}
		})
	}
}
