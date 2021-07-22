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
	"database/sql"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SMetricFieldManager struct {
	db.SEnabledResourceBaseManager
	db.SStatusStandaloneResourceBaseManager
	db.SScopedResourceBaseManager
}

type SMetricField struct {
	//db.SVirtualResourceBase
	db.SEnabledResourceBase
	db.SStatusStandaloneResourceBase
	db.SScopedResourceBase

	DisplayName string `width:"256" list:"user" update:"user"`
	Unit        string `width:"32" list:"user" update:"user"`
	ValueType   string `width:"32" list:"user" update:"user"`
	Score       int    `width:"32" list:"user" update:"user" default:"99"`
}

var MetricFieldManager *SMetricFieldManager

func init() {
	MetricFieldManager = &SMetricFieldManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SMetricField{},
			"metricfield_tbl",
			"metricfield",
			"metricfields",
		),
	}
	MetricFieldManager.SetVirtualObject(MetricFieldManager)
}

func (manager *SMetricFieldManager) NamespaceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeSystem
}

func (manager *SMetricFieldManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	q, err := manager.SStatusStandaloneResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SStatusStandaloneResourceBaseManager.ListItemExportKeys")
	}
	q, err = manager.SScopedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (man *SMetricFieldManager) ValidateCreateData(
	ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject,
	data monitor.MetricFieldCreateInput) (monitor.MetricFieldCreateInput, error) {
	return data, nil
}

func (field *SMetricField) ValidateUpdateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	data monitor.MetricFieldUpdateInput,
) (monitor.MetricFieldUpdateInput, error) {
	if len(data.DisplayName) == 0 {
		return data, errors.Wrap(httperrors.ErrNotEmpty, "display_name")
	}
	if len(data.Unit) == 0 {
		return data, errors.Wrap(httperrors.ErrNotEmpty, "unit")
	}
	if !utils.IsInStringArray(data.Unit, monitor.MetricUnit) {
		return data, errors.Wrap(httperrors.ErrBadRequest, "unit")
	}
	return data, nil
}

func (manager *SMetricFieldManager) ListItemFilter(
	ctx context.Context, q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query monitor.MetricFieldListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SEnabledResourceBaseManager.ListItemFilter(ctx, q, userCred, query.EnabledResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SEnabledResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SScopedResourceBaseManager.ListItemFilter(ctx, q, userCred,
		query.ScopedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.ListItemFilter")
	}
	if len(query.DisplayName) != 0 {
		//q = q.Equals("display_name", query.DisplayName)
		q = q.Filter(sqlchemy.Like(q.Field("display_name"), query.DisplayName))
	}
	if len(query.Unit) != 0 {
		q = q.Equals("unit", query.Unit)
	}
	return q, nil
}

func (man *SMetricFieldManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []monitor.MetricFieldDetail {
	rows := make([]monitor.MetricFieldDetail, len(objs))
	return rows
}

func (man *SMetricFieldManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input monitor.AlertListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = man.SStatusStandaloneResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SStandaloneResourceBaseManager.OrderByExtraFields")
	}
	q, err = man.SScopedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.ScopedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SScopedResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SMetricFieldManager) SaveMetricField(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, fieldInput monitor.MetricFieldCreateInput) (*SMetricField, error) {
	field := new(SMetricField)
	field.Name = fieldInput.Name
	field.DisplayName = fieldInput.DisplayName
	field.Description = fieldInput.Description
	field.Unit = fieldInput.Unit
	field.ValueType = fieldInput.ValueType
	field.Enabled = tristate.True
	field.SetModelManager(manager, field)
	if err := manager.TableSpec().Insert(ctx, field); err != nil {
		return nil, errors.Wrapf(err, "insert config %#v", field)
	}

	return field, nil
}

func (man *SMetricFieldManager) GetFieldByIdOrName(id string, userCred mcclient.TokenCredential) (*SMetricField, error) {
	obj, err := man.FetchByIdOrName(userCred, id)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return obj.(*SMetricField), nil
}

func (self *SMetricField) CustomizeDelete(
	ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	metricJoint, err := self.getMetricJoint()
	if err != nil {
		return err
	}
	for _, joint := range metricJoint {
		if err := joint.Detach(ctx, userCred); err != nil {
			return err
		}
	}
	return nil
}

func (self *SMetricField) getMetricJoint() ([]SMetric, error) {
	metricJoint := make([]SMetric, 0)
	q := MetricManager.Query().Equals(MetricManager.GetSlaveFieldName(), self.Id)
	if err := db.FetchModelObjects(MetricManager, q, &metricJoint); err != nil {
		return nil, err
	}
	return metricJoint, nil
}
