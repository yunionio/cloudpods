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
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/notify/models"
	"yunion.io/x/onecloud/pkg/notify/utils"
)

func InitHandlers(app *appsrv.Application) {
	db.RegisterModelManager(models.ContactManager)
	db.RegisterModelManager(models.VerifyManager)
	db.RegisterModelManager(models.NotificationManager)
	db.RegisterModelManager(models.ConfigManager)
	AddNotifyDispatcher("/api/v1/", app)
}

func AddNotifyDispatcher(prefix string, app *appsrv.Application) {
	var metadata map[string]interface{}
	var tags map[string]string

	// Contact Handler
	modelDispatcher := NewNotifyModelDispatcher(models.ContactManager)
	metadata, tags = map[string]interface{}{"manager": modelDispatcher}, map[string]string{"resource": modelDispatcher.KeywordPlural()}
	h := app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/<uid>/update-contact", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(contactUpdateHandler), metadata, "contact_update", tags)
	modelDispatcher.CustomizeHandlerInfo(h)
	// List
	h = app.AddHandler2("GET",
		fmt.Sprintf("%s/%s", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(listManyHandler), metadata, "list_contacts", tags)
	modelDispatcher.CustomizeHandlerInfo(h)

	h = app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<uid>", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(listOneHandler), metadata, "list_by_uid", tags)
	modelDispatcher.CustomizeHandlerInfo(h)

	h = app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/delete-contact", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(deleteContactHandler), metadata, "delete", tags)
	modelDispatcher.CustomizeHandlerInfo(h)

	// verify-trigger
	h = app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/<uid>/verify", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(verifyTriggerHandler), metadata, "verify_trigger", tags)
	modelDispatcher.CustomizeHandlerInfo(h)

	// Verify Handler, this modelDispatcher need db.DBModelDispatcher'Create function to create Contact so this modelDispatcher is
	// NotifyModelDispatcher whose DBModelDispatcher has modelManager models.ContactManager
	metadata, tags = map[string]interface{}{"manager": modelDispatcher}, map[string]string{"resource": models.VerifyManager.KeywordPlural()}
	h = app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<id>", prefix, models.VerifyManager.KeywordPlural()),
		modelDispatcher.Filter(verifyHandler), metadata, "verify", tags)

	// notification Handler
	modelDispatcher = NewNotifyModelDispatcher(models.NotificationManager)
	metadata, tags = map[string]interface{}{"manager": modelDispatcher}, map[string]string{"resource": modelDispatcher.KeywordPlural()}
	h = app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(notificationHandler), metadata, "send_notifications", tags)
	modelDispatcher.CustomizeHandlerInfo(h)
	h = app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(listHandler), metadata, "send_notifications", tags)
	modelDispatcher.CustomizeHandlerInfo(h)
	h = app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<id>", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(listHandler), metadata, "list_notification_by_id", tags)
	modelDispatcher.CustomizeHandlerInfo(h)

	// config Handler
	modelDispatcher = NewNotifyModelDispatcher(models.ConfigManager)
	metadata, tags = map[string]interface{}{"manager": modelDispatcher}, map[string]string{"resource": modelDispatcher.KeywordPlural()}
	h = app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(configUpdateHandler), metadata, "update_configs", tags)
	modelDispatcher.CustomizeHandlerInfo(h)
	h = app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<type>", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(configGetHandler), metadata, "get_configs", tags)
	modelDispatcher.CustomizeHandlerInfo(h)
	h = app.AddHandler2("DELETE",
		fmt.Sprintf("%s/%s/<type>", prefix, modelDispatcher.KeywordPlural()),
		modelDispatcher.Filter(configDeleteHandler), metadata, "delete_configs", tags)
	modelDispatcher.CustomizeHandlerInfo(h)

	// email handler for being compatible
	h = app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/", prefix, EMAIL_KEYWORDPLURAL),
		modelDispatcher.Filter(emailConfigUpdateHandler), metadata, "", tags)
	modelDispatcher.CustomizeHandlerInfo(h)
	h = app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<type>", prefix, EMAIL_KEYWORDPLURAL),
		modelDispatcher.Filter(emailConfigGetHandler), metadata, "", tags)
	modelDispatcher.CustomizeHandlerInfo(h)
	h = app.AddHandler2("DELETE",
		fmt.Sprintf("%s/%s/<type>", prefix, EMAIL_KEYWORDPLURAL),
		modelDispatcher.Filter(emailConfigDeleteHandler), metadata, "", tags)
	modelDispatcher.CustomizeHandlerInfo(h)
	h = app.AddHandler2("PUT",
		fmt.Sprintf("%s/%s/<type>", prefix, EMAIL_KEYWORDPLURAL),
		modelDispatcher.Filter(emailConfigUpdateHandler), metadata, "", tags)
	modelDispatcher.CustomizeHandlerInfo(h)

	h = app.AddHandler2("POST",
		fmt.Sprintf("%s/%s/", prefix, SMS_KEYWORDPLURAL),
		modelDispatcher.Filter(smsConfigUpdateHandler), metadata, "", tags)
	modelDispatcher.CustomizeHandlerInfo(h)
	h = app.AddHandler2("GET",
		fmt.Sprintf("%s/%s/<type>", prefix, SMS_KEYWORDPLURAL),
		modelDispatcher.Filter(smsConfigGetHandler), metadata, "", tags)
	modelDispatcher.CustomizeHandlerInfo(h)
	h = app.AddHandler2("DELETE",
		fmt.Sprintf("%s/%s/<type>", prefix, SMS_KEYWORDPLURAL),
		modelDispatcher.Filter(smsConfigDeleteHandler), metadata, "", tags)
	modelDispatcher.CustomizeHandlerInfo(h)
	h = app.AddHandler2("PUT",
		fmt.Sprintf("%s/%s/<type>", prefix, SMS_KEYWORDPLURAL),
		modelDispatcher.Filter(smsConfigUpdateHandler), metadata, "", tags)
	modelDispatcher.CustomizeHandlerInfo(h)

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
	if body.Contains("config") {
		body, _ = body.Get("config")
	}
	if body.Contains("configs") {
		body, _ = body.Get("config")
	}
	err := manager.UpdateConfig(ctx, body)
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
	manager, params, _, body := fetchEnv(ctx, w, r)

	var data jsonutils.JSONObject
	if body != nil {
		if body.Contains(manager.Keyword()) {
			data, _ = body.Get(manager.Keyword())
			if data == nil {
				data = body.(*jsonutils.JSONDict)
			}
		} else {
			data = body
		}
	} else {
		data = jsonutils.NewDict()
	}

	// check that if the uid is exist
	uid := params["<uid>"]
	_, err := utils.GetUserByID(uid)
	if err != nil {
		log.Errorf(`uid %q not found`, uid)
		httperrors.NotFoundError(w, "Uid Not Found")
		return
	}
	_, err = manager.UpdateContacts(ctx, uid, jsonutils.JSONNull, data, nil)
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
	listResult = arrangeList(listResult)
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

func listOneHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	manager, params, query, _ := fetchEnv(ctx, w, r)
	listResult, err := manager.List(ctx, mergeQueryParams(params, query), nil)
	if err != nil {
		httperrors.GeneralServerError(w, err)
		return
	}
	appsrv.SendJSON(w, wrap(arrangeOne(listResult), manager.Keyword()))
}

func wrap(data jsonutils.JSONObject, key string) jsonutils.JSONObject {
	ret := jsonutils.NewDict()
	ret.Add(data, key)
	return ret
}

// For limit option, there is a bug but don't fix it for now.
// This limit point to contact record, but these contact records whose uid are same
// are considered as one record.
func arrangeList(listResult *modulebase.ListResult) *modulebase.ListResult {
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
		cr := models.NewSContactResponse(uid, value.String())
		data[index] = jsonutils.Marshal(cr)
		index++
	}
	listResult.Data = data
	listResult.Total = len(ret)
	return listResult
}

func arrangeOne(listResult *modulebase.ListResult) jsonutils.JSONObject {
	if len(listResult.Data) == 0 {
		return jsonutils.NewDict()
	}
	uid, _ := listResult.Data[0].GetString("uid")
	details := jsonutils.NewArray()
	for _, data := range listResult.Data {
		details.Add(data)
	}
	return jsonutils.Marshal(models.NewSContactResponse(uid, details.String()))
}
