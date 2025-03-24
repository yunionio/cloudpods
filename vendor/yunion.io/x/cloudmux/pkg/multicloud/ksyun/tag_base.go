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

package ksyun

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/pkg/errors"
)

type SKsyunTags struct {
	Tags []struct {
		TagId    string
		TagKey   string
		TagValue string
	}
}

func (tag *SKsyunTags) GetTags() (map[string]string, error) {
	ret := map[string]string{}
	for _, v := range tag.Tags {
		ret[v.TagKey] = v.TagValue
	}
	return ret, nil
}

func (tag *SKsyunTags) GetSysTags() map[string]string {
	return nil
}

func (tag *SKsyunTags) SetTags(tags map[string]string, replace bool) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "SetTags")
}
