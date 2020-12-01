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

package qcloud

import (
	"fmt"

	"yunion.io/x/pkg/errors"
)

func (self *SQcloudClient) ListCollaborators(offset, limit int) ([]SUser, int, error) {
	if limit < 1 || limit > 50 {
		limit = 50
	}
	params := map[string]string{
		"Offset": fmt.Sprintf("%d", offset),
		"Limit":  fmt.Sprintf("%d", limit),
	}
	resp, err := self.camRequest("ListCollaborators", params)
	if err != nil {
		return nil, 0, errors.Wrapf(err, "ListCollaborators")
	}
	result := []SUser{}
	err = resp.Unmarshal(&result, "Data")
	if err != nil {
		return nil, 0, errors.Wrapf(err, "resp.Unmarshal")
	}
	total, _ := resp.Float("totalNum")
	return result, int(total), nil
}
