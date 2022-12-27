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

package policy

import (
	"context"

	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SPolicyTokenCredential struct {
	// usage embedded interface
	mcclient.TokenCredential
}

func (self *SPolicyTokenCredential) HasSystemAdminPrivilege() bool {
	return PolicyManager.IsScopeCapable(self.TokenCredential, rbacscope.ScopeSystem)
}

func (self *SPolicyTokenCredential) IsAllow(targetScope rbacscope.TRbacScope, service string, resource string, action string, extra ...string) rbacutils.SPolicyResult {
	allowScope, result := PolicyManager.AllowScope(self.TokenCredential, service, resource, action, extra...)
	if result.Result == rbacutils.Allow && !targetScope.HigherThan(allowScope) {
		return result
	}
	return rbacutils.PolicyDeny
}

func init() {
	gotypes.RegisterSerializableTransformer(mcclient.TokenCredentialType, func(input gotypes.ISerializable) gotypes.ISerializable {
		// log.Debugf("do TokenCredential transform for %#v", input)
		switch val := input.(type) {
		case *mcclient.SSimpleToken:
			return &SPolicyTokenCredential{val}
		default:
			return val
		}
	})
}

func FilterPolicyCredential(token mcclient.TokenCredential) mcclient.TokenCredential {
	switch token.(type) {
	case *SPolicyTokenCredential:
		return token
	default:
		return &SPolicyTokenCredential{TokenCredential: token}
	}
}

func FetchUserCredential(ctx context.Context) mcclient.TokenCredential {
	token := auth.FetchUserCredential(ctx, FilterPolicyCredential)
	return token
}
