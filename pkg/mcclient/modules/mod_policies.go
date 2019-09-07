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

package modules

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SPolicyManager struct {
	modulebase.ResourceManager
}

var Policies SPolicyManager

func policyReadFilter(session *mcclient.ClientSession, s jsonutils.JSONObject, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ss := s.(*jsonutils.JSONDict)
	ret := ss.CopyIncludes("id", "type", "enabled", "domain_id", "domain", "project_domain", "can_update", "can_delete", "is_public")
	blobJson, _ := ss.Get("blob")
	if blobJson != nil {
		policy := rbacutils.SRbacPolicy{}
		blobStr, _ := blobJson.GetString()
		if len(blobStr) > 0 {
			blobJson, _ = jsonutils.ParseString(blobStr)
		}
		err := policy.Decode(blobJson)
		if err != nil {
			return nil, err
		}
		blobJson, err = policy.Encode()
		if err != nil {
			return nil, err
		}
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
	ret := jsonutils.NewDict()
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
	for _, k := range []string{
		"type", "enabled", "domain", "domain_id", "project_domain",
	} {
		if s.Contains(k) {
			val, err := s.Get(k)
			if err != nil {
				return nil, err
			}
			ret.Add(val, k)
		}
	}
	return ret, nil
}

func init() {
	Policies = SPolicyManager{NewIdentityV3Manager("policy", "policies",
		[]string{"id", "type", "policy", "enabled", "domain_id", "domain", "project_domain", "is_public"},
		[]string{})}

	Policies.SetReadFilter(policyReadFilter).SetWriteFilter(policyWriteFilter).SetNameField("type")

	register(&Policies)
}
