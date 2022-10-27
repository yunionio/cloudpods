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

package esxi

import (
	"net/url"
	"strings"

	"github.com/vmware/govmomi/vim25/mo"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

var RESOURCEPOOL_PROPS = []string{"name", "parent", "host"}

type SResourcePool struct {
	multicloud.SProjectBase
	multicloud.STagBase
	SManagedObject
}

func (pool *SResourcePool) GetGlobalId() string {
	return pool.GetId()
}

func (pool *SResourcePool) GetStatus() string {
	return api.EXTERNAL_PROJECT_STATUS_AVAILABLE
}

func (pool *SResourcePool) GetName() string {
	path := pool.GetPath()
	if len(path) > 5 {
		path = path[5:]
	}
	name := []string{}
	for _, _name := range path {
		p, _ := url.PathUnescape(_name)
		name = append([]string{p}, name...)
	}
	return strings.Join(name, "/")
}

func NewResourcePool(manager *SESXiClient, rp *mo.ResourcePool, dc *SDatacenter) *SResourcePool {
	return &SResourcePool{SManagedObject: newManagedObject(manager, rp, dc)}
}
