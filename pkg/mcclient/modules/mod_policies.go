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

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SPolicyManager struct {
	ResourceManager
}

var Policies SPolicyManager

func policyReadFilter(session *mcclient.ClientSession, s jsonutils.JSONObject, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	ss := s.(*jsonutils.JSONDict)
	ret := ss.CopyIncludes("id", "type")
	blobStr, _ := ss.GetString("blob")
	if len(blobStr) > 0 {
		policy := rbacutils.SRbacPolicy{}
		blobJson, _ := jsonutils.ParseString(blobStr)
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
		ret.Add(jsonutils.NewString(blobJson.String()), "blob")
	}
	if s.Contains("type") {
		typeStr, err := s.GetString("type")
		if err != nil {
			return nil, err
		}
		ret.Add(jsonutils.NewString(typeStr), "type")
	}
	return ret, nil
}

func init() {
	Policies = SPolicyManager{NewIdentityV3Manager("policy", "policies",
		[]string{"id", "type", "policy"},
		[]string{})}

	Policies.SetReadFilter(policyReadFilter).SetWriteFilter(policyWriteFilter).SetNameField("type")

	register(&Policies)
}
