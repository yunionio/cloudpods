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
	"context"
	"database/sql"

	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SZettaKitRegionDriver struct {
	SManagedVirtualizationRegionDriver
}

func init() {
	driver := SZettaKitRegionDriver{}
	models.RegisterRegionDriver(&driver)
}

func (self *SZettaKitRegionDriver) GetProvider() string {
	return api.CLOUD_PROVIDER_ZETTAKIT
}

func (self *SZettaKitRegionDriver) IsVpcCreateNeedInputCidr() bool {
	return false
}

func (self *SZettaKitRegionDriver) IsSupportedElasticcacheSecgroup() bool {
	return false
}

func (self *SZettaKitRegionDriver) GetMaxElasticcacheSecurityGroupCount() int {
	return 1
}

func (self *SZettaKitRegionDriver) ValidateCreateEipData(ctx context.Context, userCred mcclient.TokenCredential, input *api.SElasticipCreateInput) error {
	if len(input.NetworkId) == 0 {
		return httperrors.NewMissingParameterError("network_id")
	}
	_network, err := models.NetworkManager.FetchByIdOrName(ctx, userCred, input.NetworkId)
	if err != nil {
		if err == sql.ErrNoRows {
			return httperrors.NewResourceNotFoundError2("network", input.NetworkId)
		}
		return httperrors.NewGeneralError(err)
	}
	network := _network.(*models.SNetwork)
	input.NetworkId = network.Id

	vpc, _ := network.GetVpc()
	if vpc == nil {
		return httperrors.NewInputParameterError("failed to found vpc for network %s(%s)", network.Name, network.Id)
	}
	input.ManagerId = vpc.ManagerId
	region, err := vpc.GetRegion()
	if err != nil {
		return err
	}
	if region.Id != input.CloudregionId {
		return httperrors.NewUnsupportOperationError("network %s(%s) does not belong to %s", network.Name, network.Id, self.GetProvider())
	}
	return nil
}

func (self *SZettaKitRegionDriver) GetSecurityGroupFilter(vpc *models.SVpc) (func(q *sqlchemy.SQuery) *sqlchemy.SQuery, error) {
	return func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
		return q.Equals("cloudregion_id", vpc.CloudregionId).Equals("manager_id", vpc.ManagerId)
	}, nil
}
