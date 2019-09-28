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

package notify

import (
	"context"
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/notify/cache"
	"yunion.io/x/onecloud/pkg/notify/models"
	"yunion.io/x/onecloud/pkg/notify/options"
	"yunion.io/x/onecloud/pkg/notify/utils"
)

func InitHandlers(app *appsrv.Application) {
	db.AddProjectResourceCountHandler("api/v1", app)
	db.RegisterModelManager(models.ContactManager)
	db.RegisterModelManager(models.VerifyManager)
	db.RegisterModelManager(models.NotificationManager)
	db.RegisterModelManager(models.ConfigManager)
	db.RegisterModelManager(cache.UserCacheManager)
	db.RegisterModelManager(cache.UserGroupCacheManager)
	AddNotifyDispatcher("/api/v1/", app)
}

func AddNotifyDispatcher(prefix string, app *appsrv.Application) {
	var metadata map[string]interface{}
	var tags map[string]string

	// Contact Handler
	modelDispatcher := NewNotifyModelDispatcher(models.ContactManager)
	metadata, tags = map[string]interface{}{"manager": modelDispatcher}, map[string]string{"resource": modelDispatcher.KeywordPlural()}
	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/<uid>/update-contact", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(contactUpdateHandler), metadata, "contact_update", tags)
	// List
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(listHandler), metadata, "list_contacts", tags)

	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/users", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(keyStoneUserListHandler), metadata, "list_users", tags)

	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<uid>", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(getHandler), metadata, "list_by_uid", tags)

	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/delete-contact", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(deleteContactHandler), metadata, "delete", tags)

	// verify-trigger
	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/<uid>/verify", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(verifyTriggerHandler), metadata, "verify_trigger", tags)

	// Verify Handler, this modelDispatcher need db.DBModelDispatcher'Create function to create Contact so this modelDispatcher is
	// NotifyModelDispatcher whose DBModelDispatcher has modelManager models.ContactManager
	metadata, tags = map[string]interface{}{"manager": modelDispatcher}, map[string]string{"resource": models.VerifyManager.KeywordPlural()}
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<id>", prefix, models.VerifyManager.KeywordPlural()),
		modelDispatcher.Filter(verifyHandler), metadata, "verify", tags)

	// notification Handler
	modelDispatcher = NewNotifyModelDispatcher(models.NotificationManager)
	metadata, tags = map[string]interface{}{"manager": modelDispatcher}, map[string]string{"resource": modelDispatcher.KeywordPlural()}
	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(notificationHandler), metadata, "send_notifications", tags)
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(listHandler), metadata, "send_notifications", tags)
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<id>", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(listHandler), metadata, "list_notification_by_id", tags)

	// config Handler
	modelDispatcher = NewNotifyModelDispatcher(models.ConfigManager)
	metadata, tags = map[string]interface{}{"manager": modelDispatcher}, map[string]string{"resource": modelDispatcher.KeywordPlural()}
	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(configUpdateHandler), metadata, "update_configs", tags)
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<type>", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(configGetHandler), metadata, "get_configs", tags)
	app.AddHandler2("DELETE",
		fmt.Sprintf("%s/%s/<type>", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(configDeleteHandler), metadata, "delete_configs", tags)

	// email handler for being compatible
	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/", prefix, EMAIL_KEYWORDPLURAL),
		modelDispatcher.Filter(emailConfigUpdateHandler), metadata, "", tags)
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<type>", prefix, EMAIL_KEYWORDPLURAL),
		modelDispatcher.Filter(emailConfigGetHandler), metadata, "", tags)
	app.AddHandler2("DELETE",
		fmt.Sprintf("%s/%s/<type>", prefix, EMAIL_KEYWORDPLURAL),
		modelDispatcher.Filter(emailConfigDeleteHandler), metadata, "", tags)
	app.AddHandler2("PUT",
		fmt.Sprintf("%s/%s/<type>", prefix, EMAIL_KEYWORDPLURAL),
		modelDispatcher.Filter(emailConfigUpdateHandler), metadata, "", tags)

	app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/", prefix, SMS_KEYWORDPLURAL),
		modelDispatcher.Filter(smsConfigUpdateHandler), metadata, "", tags)
	app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<type>", prefix, SMS_KEYWORDPLURAL),
		modelDispatcher.Filter(smsConfigGetHandler), metadata, "", tags)
	app.AddHandler2("DELETE",
		fmt.Sprintf("%s/%s/<type>", prefix, SMS_KEYWORDPLURAL),
		modelDispatcher.Filter(smsConfigDeleteHandler), metadata, "", tags)
	app.AddHandler2("PUT",
		fmt.Sprintf("%s/%s/<type>", prefix, SMS_KEYWORDPLURAL),
		modelDispatcher.Filter(smsConfigUpdateHandler), metadata, "", tags)

}

func configDeleteHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, _, _ := fetchEnv(ctx, w, r)
	err := manager.DeleteConfig(ctx, params)
	if err != nil {
		httperrors.GeneralServerError(w, err)
	}
}

func configGetHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchEnv(ctx, w, r)
	ret, err := manager.GetConfig(ctx, params, query)
	if err != nil {
		httperrors.GeneralServerError(w, err)
	}
	appsrv.SendJSON(w, ret)
}

func configUpdateHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, _, _, body := fetchEnv(ctx, w, r)
	body, err := body.Get(models.ConfigManager.Keyword())
	if err != nil {
		httperrors.GeneralServerError(w, httperrors.NewInputParameterError("need config or configs"))
		return
	}
	err = manager.UpdateConfig(ctx, body)
	if err != nil {
		httperrors.GeneralServerError(w, err)
	}
}

func notificationHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, _, _, body := fetchEnv(ctx, w, r)
	data, err := body.Get(manager.Keyword())
	if err != nil {
		httperrors.BadRequestError(w, "request body should contain %s", manager.Keyword())
	}
	ret, err := manager.CreateNotification(ctx, data)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, ret)
}

// verify handler
func verifyHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchEnv(ctx, w, r)
	err := manager.Verify(ctx, params, query)
	if err != nil {
		httperrors.GeneralServerError(w, err)
	}
}

// contact update handler
func contactUpdateHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, body := fetchEnv(ctx, w, r)

	var data []jsonutils.JSONObject
	data, err := body.GetArray(manager.Keyword(), manager.KeywordPlural())
	if err != nil {
		httperrors.GeneralServerError(w, httperrors.NewInputParameterError("need %s or %s", manager.Keyword(),
			manager.KeywordPlural()))
		return
	}
	// check that if the uid is exist
	uid := params["<uid>"]
	_, err = utils.GetUserByID(ctx, uid)
	if err != nil {
		log.Errorf(`uid %q not found`, uid)
		httperrors.NotFoundError(w, "Uid Not Found")
		return
	}
	queryDict := mergeQueryParams(params, query)
	update, _ := body.Bool(manager.Keyword(), "update_dingtalk")
	if update {
		dict := queryDict.(*jsonutils.JSONDict)
		dict.Add(jsonutils.JSONTrue, "update_dingtalk")
	}
	err = manager.UpdateContacts(ctx, uid, queryDict, data, nil)
	if err != nil {
		log.Errorf(err.Error())
		httperrors.BadRequestError(w, "")
		return
	}
	return
}

// delete contact handler
func deleteContactHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, _, _, body := fetchEnv(ctx, w, r)

	var data []jsonutils.JSONObject
	var err error
	if body != nil {
		data, err = body.GetArray(manager.KeywordPlural())
		if err != nil {
			httperrors.BadRequestError(w, "request body should have %s", manager.KeywordPlural())
		}
	}
	err = manager.DeleteContacts(ctx, data)

	if err != nil {
		log.Errorf("delete contact of %s failed, error: %s", data, err)
		httperrors.GeneralServerError(w, errors.Error("delete failed"))
	}
}

// verify trigger handler
func verifyTriggerHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, _, body := fetchEnv(ctx, w, r)
	data, err := body.Get(models.ContactManager.Keyword())
	if err != nil {
		httperrors.BadRequestError(w, "request body should have %s", manager.KeywordPlural())
	}
	ret, err := manager.VerifyTrigger(ctx, params, data)
	if err != nil {
		log.Errorf("verifyTrigger failed beacause %s", err)
		httperrors.GeneralServerError(w, err)
	}
	appsrv.SendJSON(w, ret)
}

//speciallist hander for contact records
func listManyHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchEnv(ctx, w, r)
	listResult, err := manager.List(ctx, mergeQueryParams(params, query), nil)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	listResult = arrangeList(ctx, listResult)
	appsrv.SendJSON(w, modulebase.ListResult2JSONWithKey(listResult, manager.KeywordPlural()))
}

// list handler for all resource in notify module
func listHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchEnv(ctx, w, r)
	listResult, err := manager.List(ctx, mergeQueryParams(params, query), nil)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, modulebase.ListResult2JSONWithKey(listResult, manager.KeywordPlural()))
}

func getHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchEnv(ctx, w, r)
	listResult, err := manager.List(ctx, mergeQueryParams(params, query), nil)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	var data jsonutils.JSONObject
	if len(listResult.Data) == 0 {
		data = jsonutils.NewDict()
	} else {
		data = listResult.Data[0]
	}
	appsrv.SendJSON(w, wrap(data, manager.Keyword()))
}

func listOneHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchEnv(ctx, w, r)
	listResult, err := manager.List(ctx, mergeQueryParams(params, query), nil)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, wrap(arrangeOne(ctx, listResult), manager.Keyword()))
}

func wrap(data jsonutils.JSONObject, key string) jsonutils.JSONObject {
	ret := jsonutils.NewDict()
	ret.Add(data, key)
	return ret
}

// For limit option, there is a bug but don't fix it for now.
// This limit point to contact record, but these contact records whose uid are same
// are considered as one record.
func arrangeList(ctx context.Context, listResult *modulebase.ListResult) *modulebase.ListResult {
	ret := make(map[string]*jsonutils.JSONArray)
	for _, data := range listResult.Data {
		uid, _ := data.GetString("uid")
		_, ok := ret[uid]
		if !ok {
			ret[uid] = jsonutils.NewArray()
		}
		ret[uid].Add(data)
	}
	data := make([]jsonutils.JSONObject, len(ret))
	index := 0
	for uid, value := range ret {
		cr := models.NewSContactResponse(ctx, uid, value.String())
		data[index] = jsonutils.Marshal(cr)
		index++
	}
	listResult.Data = data
	listResult.Total = len(ret)
	return listResult
}

func arrangeOne(ctx context.Context, listResult *modulebase.ListResult) jsonutils.JSONObject {
	if len(listResult.Data) == 0 {
		return jsonutils.NewDict()
	}
	uid, _ := listResult.Data[0].GetString("uid")
	details := jsonutils.NewArray()
	for _, data := range listResult.Data {
		details.Add(data)
	}
	return jsonutils.Marshal(models.NewSContactResponse(ctx, uid, details.String()))
}

// offset ang limit is not useable for here
func keyStoneUserListHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, _, query, _ := fetchEnv(ctx, w, r)
	haveContacts := jsonutils.QueryBoolean(query, "have_contacts", false)

	userCred := policy.FetchUserCredential(ctx)
	s := auth.GetSession(ctx, userCred, options.Options.Region, "")
	users, err := modules.UsersV3.List(s, query)
	if err != nil {
		log.Errorf("keystone list error: %s", err)
		httperrors.InternalServerError(w, err.Error())
		return
	}
	q := models.ContactManager.Query("uid").GroupBy("uid")
	row, err := q.Rows()
	if err != nil {
		log.Errorf("get contact's uid error: %s", err)
		httperrors.InternalServerError(w, err.Error())
		return
	}
	defer row.Close()
	uidSet, uid := make(map[string]struct{}), ""
	for row.Next() {
		row.Scan(&uid)
		uidSet[uid] = struct{}{}
	}

	type sPair struct {
		ID   string
		Name string
	}
	newDatas := make([]sPair, 0, len(users.Data))

	for _, data := range users.Data {
		id, _ := data.GetString("id")
		name, _ := data.GetString("name")
		if _, ok := uidSet[id]; ok {
			if haveContacts {
				newDatas = append(newDatas, sPair{id, name})
			}
			continue
		}
		if haveContacts {
			continue
		}
		newDatas = append(newDatas, sPair{id, name})
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.Marshal(newDatas), manager.Keyword())
	appsrv.SendJSON(w, ret)
}
