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

package compute

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Schedtaghosts          modulebase.JointResourceManager
	Schedtagstorages       modulebase.JointResourceManager
	Schedtagnetworks       modulebase.JointResourceManager
	Schedtagcloudproviders modulebase.JointResourceManager
	Schedtagcloudregions   modulebase.JointResourceManager
	Schedtagzones          modulebase.JointResourceManager
)

func newSchedtagJointManager(keyword, keywordPlural string, columns, adminColumns []string, slave modulebase.Manager) modulebase.JointResourceManager {
	columns = append(columns, "Schedtag_ID", "Schedtag")
	return modules.NewJointComputeManager(keyword, keywordPlural,
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

	Schedtagcloudproviders = newSchedtagJointManager("schedtagcloudprovider", "schedtagcloudproviders",
		[]string{"Cloudprovider_ID", "Cloudprovider"},
		[]string{},
		&Cloudproviders)

	Schedtagcloudregions = newSchedtagJointManager("schedtagcloudregion", "schedtagcloudregions",
		[]string{"Cloudregion_ID", "Cloudregion"},
		[]string{},
		&Cloudregions)

	Schedtagzones = newSchedtagJointManager("schedtagzone", "schedtagzones",
		[]string{"Zone_ID", "Zone"},
		[]string{},
		&Zones)

	for _, m := range []modulebase.IBaseManager{
		&Schedtaghosts,
		&Schedtagstorages,
		&Schedtagnetworks,
		&Schedtagcloudproviders,
		&Schedtagcloudregions,
		&Schedtagzones,
	} {
		modules.RegisterCompute(m)
	}
}
