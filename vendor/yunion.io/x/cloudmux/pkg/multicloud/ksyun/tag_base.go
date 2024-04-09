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

type SKsTag struct{}

func (tag SKsTag) GetName() string {
	return ""
}

func (tag SKsTag) GetDescription() string {
	return ""
}

func (tag *SKsTag) GetTags() (map[string]string, error) {
	return nil, nil
}

func (tag *SKsTag) GetSysTags() map[string]string {
	return nil
}

func (tag *SKsTag) SetTags(tags map[string]string, replace bool) error {
	return nil
}
