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
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/pkg/mcclient"
)

func GetVersion(s *mcclient.ClientSession, serviceType string) (string, error) {
	man := NewBaseManager(serviceType, "", "", nil, nil)
	resp, err := man.rawBaseUrlRequest(s, "GET", "/version", nil, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func ListWorkers(s *mcclient.ClientSession, serviceType string) (*printutils.ListResult, error) {
	man := NewBaseManager(serviceType, "", "", nil, nil)
	resp, err := man.rawBaseUrlRequest(s, "GET", "/worker_stats", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	ret := printutils.ListResult{}
	if workers, _ := jsonutils.Parse(body); workers != nil {
		workers.Unmarshal(&ret.Data, "workers")
		ret.Total = len(ret.Data)
	}
	return &ret, nil
}
