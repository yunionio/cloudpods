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

package models

import (
	"testing"

	"yunion.io/x/jsonutils"
)

func Test_digestQuerySignature(t *testing.T) {
	dataNoSig, _ := jsonutils.Parse([]byte(`{"name": "test"}`))
	dataWithSig, _ := jsonutils.Parse([]byte(`{"name": "test", "signature": "xxx"}`))

	tests := []struct {
		name string
		data *jsonutils.JSONDict
		want string
	}{
		{
			name: "data no signature",
			data: dataNoSig.(*jsonutils.JSONDict),
			want: "7d9fd2051fc32b32feab10946fab6bb91426ab7e39aa5439289ed892864aa91d",
		},
		{
			name: "data with signature",
			data: dataWithSig.(*jsonutils.JSONDict),
			want: "7d9fd2051fc32b32feab10946fab6bb91426ab7e39aa5439289ed892864aa91d",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := digestQuerySignature(tt.data); got != tt.want {
				t.Errorf("sumQuerySignature() = %v, want %v", got, tt.want)
			}
		})
	}
}
