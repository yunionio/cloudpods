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
	"yunion.io/x/pkg/gotypes"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SPolicyTokenCredential struct {
	// usage embedded interface
	mcclient.TokenCredential
}

func (self *SPolicyTokenCredential) HasSystemAdminPrivilege() bool {
	if consts.IsRbacEnabled() {
		return PolicyManager.IsAdminCapable(self.TokenCredential)
	}
	return self.TokenCredential.HasSystemAdminPrivilege()
}

func (self *SPolicyTokenCredential) IsAdminAllow(service string, resource string, action string, extra ...string) bool {
	if consts.IsRbacEnabled() {
		result := PolicyManager.Allow(true, self.TokenCredential, service, resource, action, extra...)
		return result == rbacutils.AdminAllow
	}
	return self.TokenCredential.IsAdminAllow(service, resource, action, extra...)
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
	if !consts.IsRbacEnabled() {
		return token
	}
	switch token.(type) {
	case *SPolicyTokenCredential:
		return token
	default:
		return &SPolicyTokenCredential{TokenCredential: token}
	}
}
