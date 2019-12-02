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

package models

import (
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
)

func AttachUsageQuery(
	q *sqlchemy.SQuery,
	hosts *sqlchemy.SSubQuery,
	hostTypes []string,
	resourceTypes []string,
	providers []string, brands []string, cloudEnv string,
	rangeObjs []db.IStandaloneModel,
) *sqlchemy.SQuery {
	if len(hostTypes) > 0 {
		q = q.Filter(sqlchemy.In(hosts.Field("host_type"), hostTypes))
	}
	if len(resourceTypes) > 0 {
		if utils.IsInStringArray(api.HostResourceTypeShared, resourceTypes) {
			q = q.Filter(sqlchemy.OR(
				sqlchemy.IsNullOrEmpty(hosts.Field("resource_type")),
				sqlchemy.In(hosts.Field("resource_type"), resourceTypes),
			))
		} else {
			q = q.Filter(sqlchemy.In(hosts.Field("resource_type"), resourceTypes))
		}
	}
	q = CloudProviderFilter(q, hosts.Field("manager_id"), providers, brands, cloudEnv)
	q = rangeObjectsFilter(q, rangeObjs, nil, hosts.Field("zone_id"), hosts.Field("manager_id"))
	return q
}
