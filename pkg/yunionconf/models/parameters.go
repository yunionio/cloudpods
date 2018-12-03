package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/pkg/util/timeutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"
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

func getNamespaceInContext(userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (namespace string, namespaceId string, err error) {
	// 优先匹配上线文中的参数, /users/<user_id>/parameters  /services/<service_id>/parameters
	if uid, _ := query.GetString("user_id"); len(uid) > 0 {
		return NAMESPACE_USER, uid, nil
	} else if sid, _ := query.GetString("service_id"); len(sid) > 0 {
		return NAMESPACE_SERVICE, sid, nil
	}

	// 匹配/parameters中的参数
	if uid, _ := data.GetString("user_id"); len(uid) > 0 {
		return NAMESPACE_USER, uid, nil
	} else if sid, _ := data.GetString("service_id"); len(sid) > 0 {
		return NAMESPACE_SERVICE, sid, nil
	} else {
		return NAMESPACE_USER, userCred.GetUserId(), nil
	}
}

func getNamespace(userCred mcclient.TokenCredential, resource string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (string, string, error) {
	var namespace, namespace_id string
	if userCred.IsAdminAllow(consts.GetServiceType(), resource, policy.PolicyActionList) {
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

func (manager *SParameterManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	if !isAdminQuery(query) {
		return true
	}

	return db.IsAdminAllowCreate(userCred, manager)
}

func (manager *SParameterManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
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
	if q.Count() > 0 {
		return nil, httperrors.NewDuplicateNameError("paramter %s has been created", name)
	}

	data.Add(jsonutils.NewString(uid), "created_by")
	data.Add(jsonutils.NewString(uid), "updated_by")
	data.Add(jsonutils.NewString(namespace), "namespace")
	data.Add(jsonutils.NewString(namespace_id), "namespace_id")
	return data, nil
}

func (manager *SParameterManager) GetOwnerId(userCred mcclient.IIdentityProvider) string {
	return userCred.GetUserId()
}

func (manager *SParameterManager) FilterByOwner(q *sqlchemy.SQuery, owner string) *sqlchemy.SQuery {
	return q.Equals("created_by", owner)
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
		} else if id, _ := query.GetString("service_id"); len(id) > 0 {
			q = q.Equals("namespace_id", id).Equals("namespace", NAMESPACE_SERVICE)
		} else if id, _ := query.GetString("user_id"); len(id) > 0 {
			q = q.Equals("namespace_id", id).Equals("namespace", NAMESPACE_USER)
		} else {
			//  not admin
			admin, _ := query.GetString("admin")
			if !utils.ToBool(admin) {
				q = q.Equals("namespace_id", userCred.GetUserId()).Equals("namespace", NAMESPACE_USER)
			}
		}

		return q, nil
	}
	return q.Equals("namespace_id", userCred.GetUserId()).Equals("namespace", NAMESPACE_USER), nil
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
	_, err := model.GetModelManager().TableSpec().Update(model, func() error {
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
