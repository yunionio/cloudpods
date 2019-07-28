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

package ucloud

import (
	"testing"

	"yunion.io/x/jsonutils"
)

func TestGetSignature(t *testing.T) {
	type args struct {
		params     SParams
		privateKey string
	}

	_obj, _ := jsonutils.ParseString(`{
    "Password"   :  "VUNsb3VkLmNu",
    "Region"     :  "cn-bj2",
    "Zone"       :  "cn-bj2-04",
    "ImageId"    :  "f43736e1-65a5-4bea-ad2e-8a46e18883c2", 
    "CPU"        :  2,
    "Memory"     :  2048,
    "DiskSpace"  :  10,
    "LoginMode"  :  "Password",
    "Action"     :  "CreateUHostInstance",
    "Name"       :  "Host01",
    "ChargeType" :  "Month",
    "Quantity"   :  1,
    "PublicKey"  :  "ucloudsomeone@example.com1296235120854146120"
}`)

	obj := _obj.(*jsonutils.JSONDict)

	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Ucloud api signature validate",
			args: args{
				params:     SParams{data: *obj},
				privateKey: "46f09bb9fab4f12dfc160dae12273d5332b5debe",
			},
			want: "4f9ef5df2abab2c6fccd1e9515cb7e2df8c6bb65",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetSignature(tt.args.params, tt.args.privateKey); got != tt.want {
				t.Errorf("GetSignature() = %v, want %v", got, tt.want)
			}
		})
	}
}
