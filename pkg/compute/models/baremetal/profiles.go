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

package baremetal

import (
	"context"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	baremetalapi "yunion.io/x/onecloud/pkg/apis/compute/baremetal"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SBaremetalProfileManager struct {
	db.SStandaloneAnonResourceBaseManager
}

type SBaremetalProfile struct {
	db.SStandaloneAnonResourceBase

	// 品牌名称（English)
	OemName string `width:"64" charset:"ascii" nullable:"false" default:"" index:"true" list:"user" create:"required"`

	Model string `width:"64" charset:"utf8" list:"user" create:"optional"`

	LanChannel uint8 `list:"user" update:"user" create:"optional"`

	LanChannel2 uint8 `list:"user" update:"user" create:"optional"`

	LanChannel3 uint8 `list:"user" update:"user" create:"optional"`

	// BMC Root账号名称，默认为 root
	RootName string `width:"64" charset:"ascii" nullable:"false" list:"user" update:"user" create:"required" default:"root"`
	// BMC Root账号ID，默认为 2
	RootId int `list:"user" update:"user" create:"optional" default:"2"`
	// 是否要求强密码
	StrongPass bool `list:"user" update:"user" create:"optional"`
}

var BaremetalProfileManager *SBaremetalProfileManager

func init() {
	BaremetalProfileManager = &SBaremetalProfileManager{
		SStandaloneAnonResourceBaseManager: db.NewStandaloneAnonResourceBaseManager(
			SBaremetalProfile{},
			"baremetal_profiles_tbl",
			"baremetal_profile",
			"baremetal_profiles",
		),
	}
	BaremetalProfileManager.SetVirtualObject(BaremetalProfileManager)
}

func (bpm *SBaremetalProfileManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query baremetalapi.BaremetalProfileListInput,
) (*sqlchemy.SQuery, error) {
	q, err := bpm.SStandaloneAnonResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneAnonResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneAnonResourceBaseManager.ListItemFilter")
	}

	query.Normalize()

	if len(query.OemName) > 0 {
		q = q.In("oem_name", query.OemName)
	}
	if len(query.Model) > 0 {
		q = q.In("model", query.Model)
	}

	return q, nil
}

func (bpm *SBaremetalProfileManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query baremetalapi.BaremetalProfileListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = bpm.SStandaloneAnonResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.StandaloneAnonResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneAnonResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (bpm *SBaremetalProfileManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = bpm.SStandaloneAnonResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, nil
}

func (bpm *SBaremetalProfileManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	var err error
	q, err = bpm.SStandaloneAnonResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (bpm *SBaremetalProfileManager) InitializeData() error {
	for _, spec := range baremetalapi.PredefinedProfiles {
		bp := SBaremetalProfile{}
		bp.Id = strings.ToLower(spec.OemName)
		if len(bp.Id) == 0 {
			bp.Id = baremetalapi.DefaultBaremetalProfileId
		}
		bp.OemName = strings.ToLower(spec.OemName)
		if len(spec.LanChannels) > 0 {
			bp.LanChannel = spec.LanChannels[0]
		}
		if len(spec.LanChannels) > 1 {
			bp.LanChannel2 = spec.LanChannels[1]
		}
		if len(spec.LanChannels) > 2 {
			bp.LanChannel3 = spec.LanChannels[2]
		}
		bp.RootName = spec.RootName
		bp.RootId = spec.RootId
		bp.StrongPass = spec.StrongPass
		bp.SetModelManager(bpm, &bp)

		err := bpm.TableSpec().InsertOrUpdate(context.Background(), &bp)
		if err != nil {
			return errors.Wrapf(err, "insert default baremetal profile %s", jsonutils.Marshal(bp))
		}
	}
	return nil
}

func (bpm *SBaremetalProfileManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input baremetalapi.BaremetalProfileCreateInput,
) (baremetalapi.BaremetalProfileCreateInput, error) {
	var err error
	input.StandaloneAnonResourceCreateInput, err = bpm.SStandaloneAnonResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StandaloneAnonResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "StandaloneAnonResourceBaseManager.ValidateCreateData")
	}
	input.OemName = strings.ToLower(strings.TrimSpace(input.OemName))
	input.Model = strings.ToLower(strings.TrimSpace(input.Model))
	// ensure uniquess
	uniqQuery := bpm.fetchByOemNameAndModelQuery(input.OemName, input.Model, true)
	cnt, err := uniqQuery.CountWithError()
	if err != nil {
		return input, errors.Wrap(err, "CountWithError")
	}
	if cnt > 0 {
		return input, errors.Wrapf(httperrors.ErrConflict, "%s/%s already exists", input.OemName, input.Model)
	}
	if input.LanChannel == 0 {
		return input, errors.Wrap(httperrors.ErrInputParameter, "lan_channel must be set")
	}
	if len(input.RootName) == 0 {
		return input, errors.Wrap(httperrors.ErrInputParameter, "root_name must be set")
	}

	return input, nil
}

func (bpm *SBaremetalProfileManager) fetchByOemNameAndModelQuery(oemName, model string, exact bool) *sqlchemy.SQuery {
	q := bpm.Query()
	if exact {
		q = q.Equals("oem_name", oemName)
	} else {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.IsEmpty(q.Field("oem_name")),
			sqlchemy.Equals(q.Field("oem_name"), oemName),
		))
	}
	if exact {
		q = q.Equals("model", model)
	} else {
		q = q.Filter(sqlchemy.OR(
			sqlchemy.IsEmpty(q.Field("model")),
			sqlchemy.Equals(q.Field("model"), model),
		))
	}
	return q
}

func (bp *SBaremetalProfile) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input baremetalapi.BaremetalProfileUpdateInput,
) (baremetalapi.BaremetalProfileUpdateInput, error) {
	var err error
	input.StandaloneAnonResourceBaseUpdateInput, err = bp.SStandaloneAnonResourceBase.ValidateUpdateData(ctx, userCred, query, input.StandaloneAnonResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "SStandaloneAnonResourceBase.ValidateUpdateData")
	}
	return input, nil
}
