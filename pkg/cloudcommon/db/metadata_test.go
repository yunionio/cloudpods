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

package db

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"

	"yunion.io/x/onecloud/pkg/mcclient"
)

func TestIsMetadataKeySystemAdmin(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{
			name: "__sys_key is system admin key",
			key:  "__sys_key",
			want: true,
		},
		{
			name: "__sys is not system admin key",
			key:  "__sys",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsMetadataKeySystemAdmin(tt.key); got != tt.want {
				t.Errorf("IsMetadataKeySystemAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsMetadataKeySysTag(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{
			name: "__qemu_version is sys tag key",
			key:  "__qemu_version",
			want: true,
		},
		{
			name: "_sys is not sys tag key",
			key:  "_sys",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsMetadataKeySysTag(tt.key); got != tt.want {
				t.Errorf("IsMetadataKeySysTag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsMetadataKeyVisiable(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want bool
	}{
		{
			name: "__qemu_version should not visiable",
			key:  "__qemu_version",
			want: false,
		},
		{
			name: "__sys_key should not visiable",
			key:  "__sys_key",
			want: false,
		},
		{
			name: "key1 should visiable",
			key:  "key1",
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsMetadataKeyVisiable(tt.key); got != tt.want {
				t.Errorf("IsMetadataKeyVisiable() = %v, want %v", got, tt.want)
			}
		})
	}
}

type MockMetadataModel struct {
	mock.Mock
	SStandaloneResourceBase
}

func (m *MockMetadataModel) GetAllMetadata(userCred mcclient.TokenCredential) (map[string]string, error) {
	args := m.Called(userCred)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockMetadataModel) GetMetadataHideKeys() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func TestGetVisiableMetadata(t *testing.T) {
	testObj := new(MockMetadataModel)
	testObj.On("GetAllMetadata", nil).Return(
		map[string]string{
			"__os_profile__":  "{\"disk_driver\":\"scsi\",\"fs_format\":\"ext4\",\"hypervisor\":\"kvm\",\"net_driver\":\"virtio\",\"os_type\":\"Linux\"}",
			"login_account":   "root",
			"os_arch":         "x86_64",
			"os_distribution": "CentOS",
		},
		nil,
	)
	testObj.On("GetMetadataHideKeys").Return([]string{"login_account"})

	tests := []struct {
		name    string
		model   IMetadataModel
		want    map[string]string
		wantErr bool
	}{
		{
			name:  "exclude sys tag and customize hide keys",
			model: testObj,
			want: map[string]string{
				"os_arch":         "x86_64",
				"os_distribution": "CentOS",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetVisiableMetadata(tt.model, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetVisiableMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetVisiableMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}
