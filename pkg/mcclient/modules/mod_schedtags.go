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

package modules

import (
	"sync"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type SchedtagManager struct {
	modulebase.ResourceManager
}

var (
	Schedtags SchedtagManager
)

func (this *SchedtagManager) DoBatchSchedtagHostAddRemove(s *mcclient.ClientSession, params jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	var wg sync.WaitGroup
	ret := jsonutils.NewDict()
	hosts, e := params.GetArray("hosts")
	if e != nil {
		return ret, e
	}
	tags, e := params.GetArray("tags")
	if e != nil {
		return ret, e
	}
	action, e := params.GetString("action")
	if e != nil {
		return ret, e
	}

	wg.Add(len(tags) * len(hosts))

	for _, host := range hosts {
		for _, tag := range tags {
			go func(host, tag jsonutils.JSONObject) {
				defer wg.Done()
				_host, _ := host.GetString()
				_tag, _ := tag.GetString()
				if action == "remove" {
					Schedtaghosts.Detach(s, _tag, _host, nil)
				} else if action == "add" {
					Schedtaghosts.Attach(s, _tag, _host, nil)
				}
			}(host, tag)
		}
	}

	wg.Wait()
	return ret, nil
}

func init() {
	Schedtags = SchedtagManager{NewComputeManager("schedtag", "schedtags",
		[]string{"ID", "Name", "Default_strategy", "Resource_type", "Domain_id", "Project_id", "Metadata"},
		[]string{})}

	registerCompute(&Schedtags)
}
