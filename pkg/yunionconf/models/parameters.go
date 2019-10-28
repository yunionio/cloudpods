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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/sqlchemy" // "yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/yunionconf/options"
)

const (
	NAMESPACE_USER    = "user"
	NAMESPACE_SERVICE = "service"
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
	ParameterManager = &SParameterManager{SResourceBaseManager: db.NewResourceBaseManager(SParameter{}, "paramters_tbl", "parameter", "parameters")}
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
	s := auth.GetAdminSession(context.Background(), options.Options.Region, "")
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
	s := auth.GetAdminSession(context.Background(), options.Options.Region, "")
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
	if userCred.IsAllow(rbacutils.ScopeSystem, consts.GetServiceType(), resource, policy.PolicyActionList) {
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

func (manager *SParameterManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	if !isAdminQuery(query) {
		return true
	}

	return db.IsAdminAllowList(userCred, manager)
}

func (manager *SParameterManager) NamespaceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeUser
}

func (manager *SParameterManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeUser
}

func (manager *SParameterManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	if !isAdminQuery(query) {
		return true
	}

	return db.IsAdminAllowCreate(userCred, manager)
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

func (manager *SParameterManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	if owner != nil {
		switch scope {
		case rbacutils.ScopeUser:
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

func (manager *SParameterManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	if db.IsAdminAllowList(userCred, manager) {
		if id, _ := query.GetString("namespace_id"); len(id) > 0 {
			q = q.Equals("namespace_id", id)
		} else if id := jsonutils.GetAnyString(query, []string{"service", "service_id"}); len(id) > 0 {
			if sid, err := getServiceId(id); err != nil {
				return q, err
			} else {
				q = q.Equals("namespace_id", sid).Equals("namespace", NAMESPACE_SERVICE)
			}
		} else if id := jsonutils.GetAnyString(query, []string{"user", "user_id"}); len(id) > 0 {
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

func (model *SParameter) IsOwner(userCred mcclient.TokenCredential) bool {
	return model.CreatedBy == userCred.GetUserId() || (model.NamespaceId == userCred.GetUserId() && model.Namespace == NAMESPACE_USER)
}

func (model *SParameter) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return model.IsOwner(userCred) || db.IsAdminAllowUpdate(userCred, model)
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

func (model *SParameter) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return model.IsOwner(userCred) || db.IsAdminAllowDelete(userCred, model)
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

func (model *SParameter) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return model.IsOwner(userCred) || db.IsAdminAllowGet(userCred, model)
}

func (model *SParameter) GetOwnerId() mcclient.IIdentityProvider {
	if model.Namespace == NAMESPACE_SERVICE {
		return nil
	}

	owner := db.SOwnerId{UserId: model.NamespaceId}
	return &owner
}
