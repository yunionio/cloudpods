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
	"io/ioutil"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
)

// path: stats, db_stats, worker_stats
func GetStats(s *mcclient.ClientSession, path string, serviceType string) (jsonutils.JSONObject, error) {
	man := NewBaseManager(serviceType, "", "", nil, nil)
	resp, err := man.rawBaseUrlRequest(s, "GET", "/"+path, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "man.rawBaseUrlRequest")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "ioutil.ReadAll")
	}
	return jsonutils.Parse(body)
}
