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

package regiondrivers

import (
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SCasRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SCasRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SCasRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_CAS
}

func (self *SCasRegionDriver) IsSupportedElasticcacheSecgroup() bool {
	return false
}

func (self *SCasRegionDriver) GetMaxElasticcacheSecurityGroupCount() int {
	return 1
}

func (self *SCasRegionDriver) GetSecurityGroupFilter(vpc *models.SVpc) (func(q *sqlchemy.SQuery) *sqlchemy.SQuery, error) {
	return func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("cloudregion_id", vpc.CloudregionId).Equals("manager_id", vpc.ManagerId)
	}, nil
}
