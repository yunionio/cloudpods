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

package cloudpods

import (
	"strings"

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
)

type CloudpodsTags struct {
	Metadata map[string]string
}

func (self *CloudpodsTags) GetTags() (map[string]string, error) {
	metadatas := map[string]string{}
	for k, v := range self.Metadata {
		if strings.HasPrefix(k, apis.USER_TAG_PREFIX) {
			metadatas[strings.TrimPrefix(k, apis.USER_TAG_PREFIX)] = v
		}
	}
	return metadatas, nil
}

func (self *CloudpodsTags) GetSysTags() map[string]string {
	return nil
}

func (self *CloudpodsTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}
