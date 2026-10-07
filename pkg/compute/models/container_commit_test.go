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

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
)

func TestContainerCommitRegistryAuthNilCommon(t *testing.T) {
	reg := &api.KubeServerContainerRegistryDetails{
		Type: "common",
		Config: &api.KubeServerContainerRegistryConfig{
			Type: "common",
		},
	}
	hostInput := &hostapi.ContainerCommitInput{
		Auth: new(apis.ContainerPullImageAuthConfig),
	}
	if err := applyContainerCommitRegistryAuth(hostInput, reg); err != nil {
		t.Fatalf("anonymous common registry: %v", err)
	}
	if hostInput.Auth.Username != "" || hostInput.Auth.Password != "" {
		t.Fatalf("expected empty auth, got %#v", hostInput.Auth)
	}
}

func TestContainerCommitRegistryAuth(t *testing.T) {
	cases := []struct {
		name string
		reg  *api.KubeServerContainerRegistryDetails
		user string
		pass string
	}{
		{
			name: "common",
			reg: &api.KubeServerContainerRegistryDetails{
				Type: "common",
				Config: &api.KubeServerContainerRegistryConfig{
					Common: &api.KubeServerContainerRegistryConfigCommon{Username: "u", Password: "p"},
				},
			},
			user: "u",
			pass: "p",
		},
		{
			name: "harbor",
			reg: &api.KubeServerContainerRegistryDetails{
				Type: "harbor",
				Config: &api.KubeServerContainerRegistryConfig{
					Harbor: &api.KubeServerContainerRegistryConfigHarbor{
						KubeServerContainerRegistryConfigCommon: api.KubeServerContainerRegistryConfigCommon{Username: "hu", Password: "hp"},
					},
				},
			},
			user: "hu",
			pass: "hp",
		},
		{
			name: "custom",
			reg: &api.KubeServerContainerRegistryDetails{
				Type: "custom",
				Config: &api.KubeServerContainerRegistryConfig{
					Custom: &api.KubeServerContainerRegistryConfigCustom{
						KubeServerContainerRegistryConfigCommon: api.KubeServerContainerRegistryConfigCommon{Username: "cu", Password: "cp"},
					},
				},
			},
			user: "cu",
			pass: "cp",
		},
		{
			name: "harbor without config section",
			reg: &api.KubeServerContainerRegistryDetails{
				Type:   "harbor",
				Config: &api.KubeServerContainerRegistryConfig{Type: "harbor"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hostInput := &hostapi.ContainerCommitInput{
				Auth: new(apis.ContainerPullImageAuthConfig),
			}
			if err := applyContainerCommitRegistryAuth(hostInput, tc.reg); err != nil {
				t.Fatal(err)
			}
			if hostInput.Auth.Username != tc.user || hostInput.Auth.Password != tc.pass {
				t.Fatalf("auth = %#v, want user=%q pass=%q", hostInput.Auth, tc.user, tc.pass)
			}
		})
	}
}

func TestContainerCommitRegistryAuthInvalidType(t *testing.T) {
	reg := &api.KubeServerContainerRegistryDetails{
		Type:   "unknown",
		Config: &api.KubeServerContainerRegistryConfig{},
	}
	err := applyContainerCommitRegistryAuth(&hostapi.ContainerCommitInput{}, reg)
	if err == nil {
		t.Fatal("expected error for unknown registry type")
	}
}
