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

package modulebase

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/pkg/mcclient"
)

func (this *ResourceManager) filterSingleResult(session *mcclient.ClientSession, result jsonutils.JSONObject, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if this.enableFilter && this.readFilter != nil {
		return this.readFilter(session, result, query)
	}
	return result, nil
}

func (this *ResourceManager) filterListResults(session *mcclient.ClientSession, results *printutils.ListResult, query jsonutils.JSONObject) (*printutils.ListResult, error) {
	if this.enableFilter && this.readFilter != nil {
		for i := 0; i < len(results.Data); i += 1 {
			val, err := this.readFilter(session, results.Data[i], query)
			if err == nil {
				results.Data[i] = val
			} else {
				log.Warningf("readFilter fail for %s: %s", results.Data[i], err)
			}
		}
	}
	return results, nil
}
