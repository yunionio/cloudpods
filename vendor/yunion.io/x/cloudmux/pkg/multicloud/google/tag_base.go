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

package google

import (
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/encode"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type GoogleTags struct {
	Labels map[string]string
}

func (self *GoogleTags) GetTags() (map[string]string, error) {
	ret := map[string]string{}
	for k, v := range self.Labels {
		if strings.HasPrefix(k, "goog-") {
			continue
		}
		ret[encode.DecodeGoogleLable(k)] = encode.DecodeGoogleLable(v)
	}
	return ret, nil
}

func (self *GoogleTags) GetSysTags() map[string]string {
	ret := map[string]string{}
	for k, v := range self.Labels {
		if strings.HasPrefix(k, "goog-") {
			ret[k] = v
		}
	}
	return ret
}

func (self *GoogleTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrap(cloudprovider.ErrNotImplemented, "SetTags")
}
