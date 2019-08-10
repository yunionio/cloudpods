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
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func PolicyCreate(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	/*params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(args.TYPE), "type")
	params.Add(jsonutils.NewString(policy.String()), "policy")
	*/
	result, err := modules.Policies.Create(s, params)
	if err != nil {
		return nil, err
	}
	policy.PolicyManager.SyncOnce()
	return result, nil
}

func PolicyPatch(s *mcclient.ClientSession, idstr string, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	/*params := jsonutils.NewDict()
	if len(name) > 0 {
		params.Add(jsonutils.NewString(name), "type")
	}
	if policy != nil {
		params.Add(jsonutils.NewString(policy.String()), "policy")
	}*/
	result, err := modules.Policies.Patch(s, idstr, params)
	if err != nil {
		return nil, err
	}
	policy.PolicyManager.SyncOnce()
	return result, nil
}

func PolicyDelete(s *mcclient.ClientSession, idstr string) error {
	_, err := modules.Policies.Delete(s, idstr, nil)
	if err != nil {
		policy.PolicyManager.SyncOnce()
	}
	return err
}
