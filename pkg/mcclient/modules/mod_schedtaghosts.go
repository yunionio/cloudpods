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

import "yunion.io/x/onecloud/pkg/mcclient/modulebase"

var (
	Schedtaghosts    modulebase.JointResourceManager
	Schedtagstorages modulebase.JointResourceManager
	Schedtagnetworks modulebase.JointResourceManager
)

func newSchedtagJointManager(keyword, keywordPlural string, columns, adminColumns []string, slave modulebase.Manager) modulebase.JointResourceManager {
	columns = append(columns, "Schedtag_ID", "Schedtag")
	return NewJointComputeManager(keyword, keywordPlural,
		columns, adminColumns, &Schedtags, slave)
}

func init() {
	Schedtaghosts = newSchedtagJointManager("schedtaghost", "schedtaghosts",
		[]string{"Host_ID", "Host"},
		[]string{},
		&Hosts)

	Schedtagstorages = newSchedtagJointManager("schedtagstorage", "schedtagstorages",
		[]string{"Storage_ID", "Storage"},
		[]string{},
		&Storages)

	Schedtagnetworks = newSchedtagJointManager("schedtagnetwork", "schedtagnetworks",
		[]string{"Network_ID", "Network"},
		[]string{},
		&Networks)

	registerCompute(&Schedtaghosts)
	registerCompute(&Schedtagstorages)
	registerCompute(&Schedtagnetworks)
}
