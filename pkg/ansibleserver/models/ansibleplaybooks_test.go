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

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func TestRequireSystemAdmin(t *testing.T) {
	systemAdmin := &mcclient.SSimpleToken{User: "admin", Project: "system", Roles: "admin"}
	if err := requireSystemAdmin(systemAdmin); err != nil {
		t.Fatalf("system admin should pass: %v", err)
	}

	tenant := &mcclient.SSimpleToken{User: "user1", Project: "proj1", Roles: "user"}
	if err := requireSystemAdmin(tenant); err == nil {
		t.Fatal("tenant should be forbidden")
	}

	projectAdmin := &mcclient.SSimpleToken{User: "owner", Project: "proj1", Roles: "admin"}
	if err := requireSystemAdmin(projectAdmin); err == nil {
		t.Fatal("project admin (non-system project) should be forbidden")
	}

	if err := requireSystemAdmin(nil); err == nil {
		t.Fatal("nil credential should be forbidden")
	}
}

func TestApplyPlaybookEnabledByCred(t *testing.T) {
	admin := &mcclient.SSimpleToken{User: "admin", Project: "system", Roles: "admin"}
	tenant := &mcclient.SSimpleToken{User: "user1", Project: "proj1", Roles: "user"}

	data := jsonutils.NewDict()
	applyPlaybookEnabledByCred(admin, data)
	enabled, _ := data.Bool("enabled")
	if !enabled {
		t.Fatal("admin create should default enabled")
	}

	data = jsonutils.NewDict()
	data.Set("enabled", jsonutils.JSONFalse)
	applyPlaybookEnabledByCred(admin, data)
	enabled, _ = data.Bool("enabled")
	if enabled {
		t.Fatal("admin should be able to create disabled")
	}

	data = jsonutils.NewDict()
	data.Set("enabled", jsonutils.JSONTrue)
	applyPlaybookEnabledByCred(tenant, data)
	enabled, _ = data.Bool("enabled")
	if enabled {
		t.Fatal("non-admin create must be disabled")
	}
}

func TestEnsurePlaybookEnabled(t *testing.T) {
	if err := ensurePlaybookEnabled(true); err != nil {
		t.Fatalf("enabled: %v", err)
	}
	if err := ensurePlaybookEnabled(false); err == nil {
		t.Fatal("disabled playbook should not run")
	}
}
