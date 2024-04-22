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
	"context"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	CloudproviderschedtagManager *SCloudproviderschedtagManager
	_                            ISchedtagJointModel = new(SCloudproviderschedtag)
)

func init() {
	db.InitManager(func() {
		CloudproviderschedtagManager = &SCloudproviderschedtagManager{
			SSchedtagJointsManager: NewSchedtagJointsManager(
				SCloudproviderschedtag{},
				"schedtag_cloudproviders_tbl",
				"schedtagcloudprovider",
				"schedtagcloudproviders",
				CloudproviderManager,
			),
		}
		CloudproviderschedtagManager.SetVirtualObject(CloudproviderschedtagManager)
	})
}

// +onecloud:swagger-gen-ignore
type SCloudproviderschedtagManager struct {
	*SSchedtagJointsManager
	SCloudproviderResourceBaseManager
}

// +onecloud:swagger-gen-ignore
type SCloudproviderschedtag struct {
	SSchedtagJointsBase
	SCloudproviderResourceBase
}

func (m *SCloudproviderschedtagManager) GetMasterFieldName() string {
	return "cloudprovider_id"
}

func (obj *SCloudproviderschedtag) GetResourceId() string {
	return obj.CloudproviderId
}

func (obj *SCloudproviderschedtag) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return obj.SSchedtagJointsBase.delete(obj, ctx, userCred)
}

func (obj *SCloudproviderschedtag) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return obj.SSchedtagJointsBase.detach(obj, ctx, userCred)
}

func (m *SCloudproviderschedtagManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CloudproviderschedtagListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = m.SSchedtagJointsManager.ListItemFilter(ctx, q, userCred, query.SchedtagJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSchedtagJointsManager.ListItemFilter")
	}
	return q, nil
}

func (obj *SCloudproviderschedtag) GetDetails(base api.SchedtagJointResourceDetails, resourceName string, isList bool) interface{} {
	out := api.CloudproviderschedtagDetails{
		SchedtagJointResourceDetails: base,
	}
	out.Cloudprovider = resourceName
	return out
}
