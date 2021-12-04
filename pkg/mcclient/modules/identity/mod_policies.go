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

package identity

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SPolicyManager struct {
	modulebase.ResourceManager
}

var Policies SPolicyManager

func policyReadFilter(session *mcclient.ClientSession, s jsonutils.JSONObject, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ss, ok := s.(*jsonutils.JSONDict)
	if !ok {
		return s, nil
	}
	ret := ss.CopyExcludes("blob", "type")
	blobJson, _ := ss.Get("blob")
	if blobJson != nil {
		blobStr, _ := blobJson.GetString()
		if len(blobStr) > 0 {
			blobJson, _ = jsonutils.ParseString(blobStr)
		}
		policy, err := rbacutils.DecodeRawPolicyData(blobJson)
		if err != nil {
			return nil, errors.Wrap(err, "rbacutils.DecodePolicyData")
		}
		blobJson = policy.EncodeRawData()
		var format string
		if query != nil {
			format, _ = query.GetString("format")
		}
		if format == "yaml" {
			var policy string
			if blobJson != nil {
				policy = blobJson.YAMLString()
			}
			ret.Add(jsonutils.NewString(policy), "policy")
		} else {
			ret.Add(blobJson, "policy")
		}
	}
	return ret, nil
}

func policyWriteFilter(session *mcclient.ClientSession, s jsonutils.JSONObject, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ret := s.(*jsonutils.JSONDict).CopyExcludes("policy")
	if s.Contains("policy") {
		blobJson, err := s.Get("policy")
		if err != nil {
			return nil, err
		}
		switch blob := blobJson.(type) {
		case *jsonutils.JSONString:
			blobStr, _ := blob.GetString()
			blobJson, err = jsonutils.ParseYAML(blobStr)
			if err != nil {
				return nil, err
			}
		}
		// ret.Add(jsonutils.NewString(blobJson.String()), "blob")
		ret.Add(blobJson, "blob")
	}
	return ret, nil
}

func init() {
	Policies = SPolicyManager{modules.NewIdentityV3Manager(
		"policy",
		"policies",
		[]string{"id", "name", "policy", "scope", "enabled",
			"domain_id", "domain", "project_domain", "public_scope",
			"is_public", "description", "is_system",
		},
		[]string{})}

	Policies.SetReadFilter(policyReadFilter).SetWriteFilter(policyWriteFilter) // .SetNameField("type")

	modules.Register(&Policies)
}
