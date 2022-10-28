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

package huawei

import (
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type HuaweiTags struct {
	Tags []string
}

func (self *HuaweiTags) GetTags() (map[string]string, error) {
	tags := map[string]string{}
	for _, kv := range self.Tags {
		splited := strings.Split(kv, "=")
		if len(splited) == 2 {
			tags[splited[0]] = splited[1]
		}
	}
	return tags, nil
}

func (self *HuaweiTags) GetSysTags() map[string]string {
	return nil
}

func (self *HuaweiTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type HuaweiDiskTags struct {
	Tags map[string]string
}

func (self *HuaweiDiskTags) GetTags() (map[string]string, error) {
	return self.Tags, nil
}

func (self *HuaweiDiskTags) GetSysTags() map[string]string {
	return nil
}

func (self *HuaweiDiskTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}
