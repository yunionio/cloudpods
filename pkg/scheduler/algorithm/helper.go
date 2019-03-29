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

package algorithm

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/scheduler/cache/candidate"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

func ToHostCandidate(c core.Candidater) (*candidate.HostDesc, error) {
	d, ok := c.(*candidate.HostDesc)
	if !ok {
		return nil, fmt.Errorf("Can't convert %#v to '*candidate.HostDesc'", c)
	}
	return d, nil
}

func ToBaremetalCandidate(c core.Candidater) (*candidate.BaremetalDesc, error) {
	d, ok := c.(*candidate.BaremetalDesc)
	if !ok {
		return nil, fmt.Errorf("Can't convert %#v to '*candidate.BaremetalDesc'", c)
	}
	return d, nil
}
