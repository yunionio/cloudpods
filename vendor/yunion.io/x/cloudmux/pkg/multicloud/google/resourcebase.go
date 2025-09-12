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
	"fmt"
	"strings"
)

type SResourceBase struct {
	Name     string
	SelfLink string
	Id       string
}

func (r *SResourceBase) GetId() string {
	if len(r.Id) > 0 {
		return r.Id
	}
	return r.SelfLink
}

func getGlobalId(selfLink string) string {
	return strings.TrimPrefix(selfLink, fmt.Sprintf("%s/%s/", GOOGLE_COMPUTE_DOMAIN, GOOGLE_API_VERSION))
}

func (r *SResourceBase) GetGlobalId() string {
	if len(r.Id) > 0 {
		return r.Id
	}
	return getGlobalId(r.SelfLink)
}

func (r *SResourceBase) GetName() string {
	return r.Name
}

func (r *SResourceBase) GetDescription() string {
	return ""
}

func (r *SResourceBase) IsEmulated() bool {
	return false
}
