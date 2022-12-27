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
	"testing"

	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type omniToken struct {
	mcclient.SSimpleToken
}

func (tk *omniToken) IsAllow(scope rbacscope.TRbacScope, service string, resource string, action string, extra ...string) rbacutils.SPolicyResult {
	return rbacutils.PolicyAllow
}

func TestListFields(t *testing.T) {
	sqlchemy.SetupMockDatabaseBackend()
	man := NewResourceBaseManager(
		&SResourceBase{},
		"tbl",
		"keyword",
		"keywords",
	)
	userCred := &omniToken{}
	inc, exc := listFields(&man, userCred)
	for _, fn := range []string{"deleted", "deleted_at"} {
		if !utils.IsInStringArray(fn, inc) {
			t.Errorf("resourcebase: field %q not included", fn)
		}
		if utils.IsInStringArray(fn, exc) {
			t.Errorf("resourcebase: field %q excluded", fn)
		}
	}
}
