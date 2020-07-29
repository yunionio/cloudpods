package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
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
	obj, err := db.DoCreate(manager, ctx, userCred, nil, fieldInput.JSON(&fieldInput), userCred)
	if err != nil {
		return nil, errors.Wrapf(err, "SaveMetricField error input: %s", fieldInput.JSON(&fieldInput))
	}
	return obj.(*SMetricField), nil
}
