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

type STag struct {
	ResourceType string
	ResourceId   string
	Key          string
	Value        string
}

type TagSet []STag

func (vv TagSet) GetTags() map[string]string {
	ret := map[string]string{}
	for _, v := range vv {
		ret[v.Key] = v.Value
	}
	return ret
}

func (self *SRegion) ListTags(resType string, resId string) (*TagSet, error) {
	params := map[string]string{
		"MaxResults":       "1000",
		"Filter.1.Name":    "resource-type",
		"Filter.1.Value.1": resType,
		"Filter.2.Name":    "resource-id",
		"Filter.2.Value.1": resId,
	}
	resp, err := self.tagRequest("DescribeTags", params)
	if err != nil {
		return nil, err
	}
	ret := &TagSet{}
	err = resp.Unmarshal(&ret, "TagSet")
	if err != nil {
		return nil, err
	}
	return ret, nil
}
