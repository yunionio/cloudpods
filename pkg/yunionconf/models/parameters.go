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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/yunionconf"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/yunionconf/options"
)

const (
	NAMESPACE_USER       = "user"
	NAMESPACE_SERVICE    = "service"
	NAMESPACE_BUG_REPORT = "bug-report"
)

type SParameterManager struct {
	db.SResourceBaseManager
}

type SParameter struct {
	db.SResourceBase

	Id          int64                `primary:"true" auto_increment:"true" list:"user"`                                          // = Column(BigInteger, primary_key=True)
	CreatedBy   string               `width:"128" charset:"ascii" nullable:"false" create:"required" list:"user"`                // Column(VARCHAR(length=128, charset='ascii'), nullable=False)
	UpdatedBy   string               `width:"128" charset:"ascii" nullable:"false" create:"required" update:"user" list:"user"`  // Column(VARCHAR(length=128, charset='ascii'), nullable=False)  "user"/ serviceName/ "admin"
	Namespace   string               `width:"64" charset:"ascii" default:"user" nullable:"false" create:"required" list:"admin"` // Column(VARCHAR(length=128, charset='ascii'), nullable=False)  user_id / serviceid
	NamespaceId string               `width:"128" charset:"ascii" nullable:"false" index:"true" create:"required" list:"admin"`  // Column(VARCHAR(length=128, charset='ascii'), nullable=False)
	Name        string               `width:"128" charset:"ascii" nullable:"false" index:"true" create:"required" list:"user"`   // Column(VARCHAR(length=128, charset='ascii'), nullable=false)
	Value       jsonutils.JSONObject `charset:"utf8" create:"required" update:"user" update:"user" list:"user"`                  // Column(VARCHAR(charset='utf-8'))
}

var ParameterManager *SParameterManager

func init() {
	ParameterManager = &SParameterManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SParameter{},
			"paramters_tbl",
			"parameter",
			"parameters",
		),
	}
	ParameterManager.SetVirtualObject(ParameterManager)
}

func isAdminQuery(query jsonutils.JSONObject) bool {
	admin_fields := [3]string{"namespace_id", "user_id", "service_id"}

	for _, field := range admin_fields {
		if s, _ := query.GetString(field); len(s) > 0 {
			return true
		}
	}

	return false
}

func getUserId(user string) (string, error) {
	s := auth.GetAdminSession(context.Background(), options.Options.Region)
	userObj, err := modules.UsersV3.Get(s, user, nil)
	if err != nil {
		return "", err
	}

	uid, err := userObj.GetString("id")
	if err != nil {
		return "", err
	}

	return uid, nil
}

func getServiceId(service string) (string, error) {
	s := auth.GetAdminSession(context.Background(), options.Options.Region)
	serviceObj, err := modules.ServicesV3.Get(s, service, nil)
	if err != nil {
		return "", err
	}

	uid, err := serviceObj.GetString("id")
	if err != nil {
		return "", err
	}

	return uid, nil
}

func getNamespaceInContext(userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (namespace string, namespaceId string, err error) {
	// 优先匹配上线文中的参数, /users/<user_id>/parameters  /services/<service_id>/parameters
	if query != nil {
		if uid := jsonutils.GetAnyString(query, []string{"user", "user_id"}); len(uid) > 0 {
			uid, err := getUserId(uid)
			if err != nil {
				return "", "", err
			}
			return NAMESPACE_USER, uid, nil
		} else if sid := jsonutils.GetAnyString(query, []string{"service", "service_id"}); len(sid) > 0 {
			sid, err := getServiceId(sid)
			if err != nil {
				return "", "", err
			}
			return NAMESPACE_SERVICE, sid, nil
		}
	}

	// 匹配/parameters中的参数
	if uid := jsonutils.GetAnyString(data, []string{"user", "user_id"}); len(uid) > 0 {
		uid, err := getUserId(uid)
		if err != nil {
			return "", "", err
		}
		return NAMESPACE_USER, uid, nil
	} else if sid := jsonutils.GetAnyString(data, []string{"service", "service_id"}); len(sid) > 0 {
		sid, err := getServiceId(sid)
		if err != nil {
			return "", "", err
		}
		return NAMESPACE_SERVICE, sid, nil
	} else {
		return NAMESPACE_USER, userCred.GetUserId(), nil
	}
}

func getNamespace(userCred mcclient.TokenCredential, resource string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (string, string, error) {
	var namespace, namespace_id string
	if userCred.IsAllow(rbacscope.ScopeSystem, consts.GetServiceType(), resource, policy.PolicyActionList).Result.IsAllow() {
		if name, nameId, e := getNamespaceInContext(userCred, query, data); e != nil {
			return "", "", e
		} else {
			namespace = name
			namespace_id = nameId
		}
	} else {
		namespace = NAMESPACE_USER
		namespace_id = userCred.GetUserId()
	}

	return namespace, namespace_id, nil
}

func (manager *SParameterManager) CreateByInsertOrUpdate() bool {
	return false
}

func (manager *SParameterManager) NamespaceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeUser
}

func (manager *SParameterManager) ResourceScope() rbacscope.TRbacScope {
	return rbacscope.ScopeUser
}

func (manager *SParameterManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	// check duplication
	name, _ := data.GetString("name")
	uid := userCred.GetUserId()
	if len(uid) == 0 {
		return nil, httperrors.NewUserNotFoundError("user not found")
	}

	namespace, namespace_id, e := getNamespace(userCred, manager.KeywordPlural(), query, data)
	if e != nil {
		return nil, e
	}

	// check duplication, 同一个namespace下,name不能 重复
	q := manager.Query().Equals("name", name).Equals("namespace_id", namespace_id)
	cnt, err := q.CountWithError()
	if err != nil {
		return nil, httperrors.NewInternalServerError("check name duplication fail %s", err)
	}
	if cnt > 0 {
		return nil, httperrors.NewDuplicateNameError("paramter %s has been created", name)
	}

	_, err = data.Get("value")
	if err != nil {
		return nil, err
	}

	data.Add(jsonutils.NewString(uid), "created_by")
	data.Add(jsonutils.NewString(uid), "updated_by")
	data.Add(jsonutils.NewString(namespace), "namespace")
	data.Add(jsonutils.NewString(namespace_id), "namespace_id")
	return data, nil
}

func (manager *SParameterManager) FilterByOwner(q *sqlchemy.SQuery, man db.FilterByOwnerProvider, userCred mcclient.TokenCredential, owner mcclient.IIdentityProvider, scope rbacscope.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacscope.ScopeUser:
			if len(owner.GetUserId()) > 0 {
				q = q.Equals("namespace_id", owner.GetUserId()).Equals("namespace", NAMESPACE_USER)
			}
		}
	}
	return q
}

func (manager *SParameterManager) FilterById(q *sqlchemy.SQuery, idStr string) *sqlchemy.SQuery {
	return q.Equals("id", idStr)
}

func (manager *SParameterManager) FilterByName(q *sqlchemy.SQuery, name string) *sqlchemy.SQuery {
	return q.Equals("name", name)
}

// 配置参数列表
func (manager *SParameterManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ParameterListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.ListItemFilter")
	}
	if len(query.Name) > 0 {
		q = q.In("name", query.Name)
	}
	if db.IsAdminAllowList(userCred, manager).Result.IsAllow() {
		if id := query.NamespaceId; len(id) > 0 {
			q = q.Equals("namespace_id", id)
		} else if id := query.ServiceId; len(id) > 0 {
			if sid, err := getServiceId(id); err != nil {
				return q, err
			} else {
				q = q.Equals("namespace_id", sid).Equals("namespace", NAMESPACE_SERVICE)
			}
		} else if id := query.UserId; len(id) > 0 {
			if uid, err := getUserId(id); err != nil {
				return q, err
			} else {
				q = q.Equals("namespace_id", uid).Equals("namespace", NAMESPACE_USER)
			}
		}
		/*else {
			//  not admin
			admin, _ := query.GetString("admin")
			if !utils.ToBool(admin) {
				q = q.Equals("namespace_id", userCred.GetUserId()).Equals("namespace", NAMESPACE_USER)
			}
		} */
	}
	return q, nil
}

func (manager *SParameterManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.ParameterListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SResourceBaseManager.OrderByExtraFielda")
	}

	return q, nil
}

func (manager *SParameterManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (model *SParameter) IsOwner(userCred mcclient.TokenCredential) bool {
	return model.CreatedBy == userCred.GetUserId() || (model.NamespaceId == userCred.GetUserId() && model.Namespace == NAMESPACE_USER)
}

func (model *SParameter) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	uid := userCred.GetUserId()
	if len(uid) == 0 {
		return nil, httperrors.NewUserNotFoundError("user not found")
	}

	namespace, namespace_id, e := getNamespace(userCred, model.KeywordPlural(), query, data)
	if e != nil {
		return nil, e
	}

	_, err := data.Get("value")
	if err != nil {
		return nil, err
	}

	data.Add(jsonutils.NewString(uid), "updated_by")
	data.Add(jsonutils.NewString(namespace_id), "namespace_id")
	data.Add(jsonutils.NewString(namespace), "namespace")
	return data, nil
}

func (model *SParameter) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return model.Delete(ctx, userCred)
}

func (model *SParameter) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	_, err := db.Update(model, func() error {
		model.Deleted = true
		model.DeletedAt = timeutils.UtcNow()
		return nil
	})
	if err != nil {
		log.Errorf("PendingDelete fail %s", err)
	}
	return err
}

func (model *SParameter) GetOwnerId() mcclient.IIdentityProvider {
	if model.Namespace == NAMESPACE_SERVICE {
		return nil
	}

	owner := db.SOwnerId{UserId: model.NamespaceId}
	return &owner
}

func (manager *SParameterManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return db.FetchUserInfo(ctx, data)
}

func (model *SParameter) GetId() string {
	return fmt.Sprintf("%d", model.Id)
}

func (model *SParameter) GetName() string {
	return model.Name
}

var bugReportEnable *bool = nil

func (manager *SParameterManager) GetBugReportEnabled() bool {
	if bugReportEnable != nil {
		return *bugReportEnable
	}
	enabled := manager.Query().Equals("namespace", NAMESPACE_BUG_REPORT).Count() > 0
	bugReportEnable = &enabled
	return enabled
}

func (manager *SParameterManager) EnableBugReport(ctx context.Context) bool {
	if manager.GetBugReportEnabled() {
		return true
	}
	res := &SParameter{
		Namespace:   NAMESPACE_BUG_REPORT,
		NamespaceId: NAMESPACE_BUG_REPORT,
		Name:        NAMESPACE_BUG_REPORT,
		Value:       jsonutils.NewDict(),
		CreatedBy:   api.SERVICE_TYPE,
	}
	res.SetModelManager(manager, res)
	err := manager.TableSpec().Insert(ctx, res)
	if err != nil {
		return false
	}
	enabled := true
	bugReportEnable = &enabled
	return true
}

func (manager *SParameterManager) DisableBugReport(ctx context.Context) error {
	if !manager.GetBugReportEnabled() {
		return nil
	}
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"delete from %s where namespace = ?",
			manager.TableSpec().Name(),
		), NAMESPACE_BUG_REPORT,
	)
	bugReportEnable = nil
	return err
}
