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

package multicloud

import (
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type STagBase struct {
}

func (self STagBase) GetSysTags() map[string]string {
	return nil
}

func (self STagBase) GetTags() (map[string]string, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "GetTags")
}

func (self STagBase) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}

type STag struct {
	TagKey   string
	TagValue string

	Key   string
	Value string
}

func (self STag) IsSysTagPrefix(keys []string) bool {
	for _, prefix := range keys {
		if strings.HasPrefix(self.TagKey, prefix) {
			return true
		}
		if strings.HasPrefix(self.Key, prefix) {
			return true
		}
	}
	return false
}
