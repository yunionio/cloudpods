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

package rbacutils

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"

	"yunion.io/x/onecloud/pkg/httperrors"
)

type TPolicyGroup map[rbacscope.TRbacScope]TPolicySet

func DecodePolicyGroup(json jsonutils.JSONObject) (TPolicyGroup, error) {
	jmap, err := json.GetMap()
	if err != nil {
		return nil, errors.Wrap(httperrors.ErrInvalidFormat, "invalid json: not a map")
	}
	group := TPolicyGroup{}
	for k := range jmap {
		scope := rbacscope.TRbacScope(k)
		group[scope], err = DecodePolicySet(jmap[k])
		if err != nil {
			return nil, errors.Wrapf(err, "decode %s", k)
		}
	}
	return group, nil
}

func (sets TPolicyGroup) HighestScope() rbacscope.TRbacScope {
	for _, s := range []rbacscope.TRbacScope{
		rbacscope.ScopeSystem,
		rbacscope.ScopeDomain,
		rbacscope.ScopeProject,
		rbacscope.ScopeUser,
	} {
		if _, ok := sets[s]; ok {
			return s
		}
	}
	return rbacscope.ScopeNone
}

func (sets TPolicyGroup) Encode() jsonutils.JSONObject {
	j := jsonutils.NewDict()
	for k := range sets {
		j.Set(string(k), sets[k].Encode())
	}
	return j
}
