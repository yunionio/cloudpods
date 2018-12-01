package db

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/sqlchemy"
)

type SSharableVirtualResourceBase struct {
	SVirtualResourceBase

	IsPublic bool `default:"false" nullable:"false" create:"admin_optional" list:"user"`
}

type SSharableVirtualResourceBaseManager struct {
	SVirtualResourceBaseManager
}

func NewSharableVirtualResourceBaseManager(dt interface{}, tableName string, keyword string, keywordPlural string) SSharableVirtualResourceBaseManager {
	return SSharableVirtualResourceBaseManager{SVirtualResourceBaseManager: NewVirtualResourceBaseManager(dt, tableName, keyword, keywordPlural)}
}

func (manager *SSharableVirtualResourceBaseManager) FilterByOwner(q *sqlchemy.SQuery, owner string) *sqlchemy.SQuery {
	q = q.Filter(sqlchemy.OR(sqlchemy.Equals(q.Field("tenant_id"), owner), sqlchemy.IsTrue(q.Field("is_public"))))
	q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("pending_deleted")), sqlchemy.IsFalse(q.Field("pending_deleted"))))
	q = q.Filter(sqlchemy.OR(sqlchemy.IsNull(q.Field("is_system")), sqlchemy.IsFalse(q.Field("is_system"))))
	return q
}

func (model *SSharableVirtualResourceBase) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return model.IsOwner(userCred) || model.IsPublic || IsAdminAllowGet(userCred, model)
}

func (model *SSharableVirtualResourceBase) IsSharable() bool {
	return model.IsPublic
}

func (model *SSharableVirtualResourceBase) AllowPerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return IsAdminAllowPerform(userCred, model, "public")
}

func (model *SSharableVirtualResourceBase) AllowPerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return IsAdminAllowPerform(userCred, model, "private")
}

func (model *SSharableVirtualResourceBase) PerformPublic(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if !model.IsPublic {
		_, err := model.GetModelManager().TableSpec().Update(model, func() error {
			model.IsPublic = true
			return nil
		})
		return nil, err
	}
	return nil, nil
}

func (model *SSharableVirtualResourceBase) PerformPrivate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if model.IsPublic {
		_, err := model.GetModelManager().TableSpec().Update(model, func() error {
			model.IsPublic = false
			return nil
		})
		return nil, err
	}
	return nil, nil
}
