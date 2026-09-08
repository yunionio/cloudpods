// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or authorized to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"testing"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

func TestValidateDeployConfigs(t *testing.T) {
	cfgs := []*api.DeployConfig{
		{Action: "create", Path: "/etc/hosts", Content: "x"},
	}
	if err := ValidateDeployConfigs(cfgs); err != nil {
		t.Fatalf("valid: %v", err)
	}
	if cfgs[0].Path != "/etc/hosts" {
		t.Fatalf("path %q", cfgs[0].Path)
	}

	rel := []*api.DeployConfig{{Action: "create", Path: "attack/etc/pwn"}}
	if err := ValidateDeployConfigs(rel); err == nil {
		t.Fatal("expected relative path to fail")
	}
	root := []*api.DeployConfig{{Action: "create", Path: "/"}}
	if err := ValidateDeployConfigs(root); err == nil {
		t.Fatal("expected / to fail")
	}
	badAct := []*api.DeployConfig{{Action: "exec", Path: "/etc/hosts"}}
	if err := ValidateDeployConfigs(badAct); err == nil {
		t.Fatal("expected invalid action to fail")
	}
	cleaned := []*api.DeployConfig{{Path: "/etc/../root/.ssh/authorized_keys"}}
	if err := ValidateDeployConfigs(cleaned); err != nil {
		t.Fatalf("cleaned path: %v", err)
	}
	if cleaned[0].Path != "/root/.ssh/authorized_keys" {
		t.Fatalf("got %q", cleaned[0].Path)
	}
}
