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

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func TestValidateHostBindPath(t *testing.T) {
	tenant := &mcclient.SSimpleToken{User: "u", Project: "p", Roles: "user"}
	admin := &mcclient.SSimpleToken{User: "admin", Project: "system", Roles: "admin"}

	got, err := ValidateHostBindPath(tenant, "/data/models/Qwen3-8B")
	if err != nil {
		t.Fatalf("tenant data path: %v", err)
	}
	if got != "/data/models/Qwen3-8B" {
		t.Fatalf("got %q", got)
	}

	if _, err := ValidateHostBindPath(tenant, "/"); err == nil {
		t.Fatal("expected error for /")
	}
	if _, err := ValidateHostBindPath(tenant, "/etc/shadow"); err == nil {
		t.Fatal("expected tenant /etc/shadow to fail")
	}
	if _, err := ValidateHostBindPath(admin, "/etc/shadow"); err != nil {
		t.Fatalf("admin /etc/shadow: %v", err)
	}
}

func TestValidateHostPathAutoCreate(t *testing.T) {
	if err := ValidateHostPathAutoCreate(true, nil); err != nil {
		t.Fatalf("auto_create: %v", err)
	}

	cfg := &apis.ContainerVolumeMountHostPathAutoCreateConfig{Permissions: "755; id"}
	if err := ValidateHostPathAutoCreate(true, cfg); err == nil {
		t.Fatal("expected invalid permissions to fail")
	}
	cfg = &apis.ContainerVolumeMountHostPathAutoCreateConfig{Permissions: "0755"}
	if err := ValidateHostPathAutoCreate(true, cfg); err != nil {
		t.Fatalf("permissions: %v", err)
	}
	if cfg.Permissions != "755" {
		t.Fatalf("canon permissions %q", cfg.Permissions)
	}
}
