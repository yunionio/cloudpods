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

package zstack

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SVersion struct {
	Version string
}

func (self *SZStackClient) GetVersion() (*SVersion, error) {
	params := map[string]interface{}{
		"getVersion": map[string]string{},
		"systemTags": []string{},
		"userTags":   []string{},
	}
	resp, err := self.put("management-nodes/actions", "", jsonutils.Marshal(params))
	if err != nil {
		return nil, errors.Wrapf(err, "GetVersion")
	}
	v := &SVersion{}
	return v, resp.Unmarshal(v)
}
