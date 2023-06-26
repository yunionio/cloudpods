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

package apsara

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
)

type jsonRequestFunc func(action string, params map[string]string) (jsonutils.JSONObject, error)

func unmarshalResult(resp jsonutils.JSONObject, respErr error, resultKey []string, result interface{}) error {
	if respErr != nil {
		return respErr
	}

	if result == nil {
		return nil
	}

	if resultKey != nil && len(resultKey) > 0 {
		respErr = resp.Unmarshal(result, resultKey...)
	} else {
		respErr = resp.Unmarshal(result)
	}

	if respErr != nil {
		log.Errorf("unmarshal json error %s", respErr)
	}

	return nil
}

// 执行操作
func DoAction(client jsonRequestFunc, action string, params map[string]string, resultKey []string, result interface{}) error {
	resp, err := client(action, params)
	return unmarshalResult(resp, err, resultKey, result)
}
