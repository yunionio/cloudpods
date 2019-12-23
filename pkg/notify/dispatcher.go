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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/appsrv/dispatcher"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/notify/models"
	noutils "yunion.io/x/onecloud/pkg/notify/utils"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

// NotifyModelDispatcher is designed to complete some function that db.DBModelDispatcher can't.
// The apis of notify module has a certain degree of particularity so that we can't use common function.
type NotifyModelDispatcher struct {
	db.DBModelDispatcher
}

func NewNotifyModelDispatcher(manager db.IModelManager) *NotifyModelDispatcher {
	return &NotifyModelDispatcher{*db.NewModelHandler(manager)}
}

func (self *NotifyModelDispatcher) GetConfig(ctx context.Context, params map[string]string, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	listResult, err := self.List(ctx, mergeQueryParams(params, query), nil)
	if err != nil {
		return nil, err
	}
	keyVs := make(map[string]string)
	for _, ret := range listResult.Data {
		key, _ := ret.GetString("key_text")
		value, _ := ret.GetString("value_text")
		keyVs[key] = value
	}
	return jsonutils.Marshal(map[string]map[string]string{
		models.ConfigManager.Keyword(): keyVs,
	}), nil
}

func (self *NotifyModelDispatcher) DeleteConfig(ctx context.Context, params map[string]string) error {
	contactType := params["<type>"]
	configs, err := models.ConfigManager.GetConfigByType(contactType)
	if err != nil {
		return errors.Wrap(err, "Get Config by contactType failed")
	}
	userCred := policy.FetchUserCredential(ctx)
	for i := range configs {
		err = DeleteItem(&configs[i], ctx, userCred, jsonutils.JSONNull, jsonutils.JSONNull)
		if err != nil {
			return errors.Wrapf(err, "Delete part of old one, so please input new data again.")
		}
	}
	return nil
}

// UpdateConfig update config and restart corresponding send service.
func (self *NotifyModelDispatcher) UpdateConfig(ctx context.Context, body jsonutils.JSONObject) error {
	data := body.(*jsonutils.JSONDict)
	contactType := data.SortedKeys()[0]
	originData, err := models.ConfigManager.GetConfig(contactType)
	if err != nil {
		return err
	}
	tmp, _ := data.Get(contactType)
	data = tmp.(*jsonutils.JSONDict)
	userCred := policy.FetchUserCredential(ctx)
	// If no config of type 'contactType' in database, create news.
	// Else delete original ones and create news.
	if len(originData) != 0 {
		// delete original
		configs, err := models.ConfigManager.GetConfigByType(contactType)
		if err != nil {
			return errors.Wrap(err, "Get Config by contactType failed")
		}
		for i := range configs {
			err = DeleteItem(&configs[i], ctx, userCred, jsonutils.JSONNull, jsonutils.JSONNull)
			if err != nil {
				return errors.Wrapf(err, "Delete part of old one, so please input new data again.")
			}
		}
	}
	config := make(map[string]string)
	// create
	for _, key := range data.SortedKeys() {
		createData := jsonutils.NewDict()
		tmp, _ = data.Get(key)
		createData.Add(tmp, "value_text")
		createData.Add(jsonutils.NewString(key), "key_text")
		createData.Add(jsonutils.NewString(contactType), "type")
		_, err := self.Create(ctx, jsonutils.JSONNull, createData, nil)
		value, _ := tmp.GetString()
		config[key] = value
		if err != nil {
			return errors.Wrapf(err, "Create config (%s, %s, %s) failed", contactType, key, value)
		}
	}
	// update config
	models.RestartService(config, contactType)
	return nil
}

// CreateNotification create new notifications and send them through rpc.RpcService.
// If data contains 'gid' field, that means that send message to all users in group.
// Else send messager to user whose uid equals 'uid' in data.
func (self *NotifyModelDispatcher) CreateNotification(ctx context.Context, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	// Get all contacts info of group if data contains "gid".
	// If no contact, return ErrContactNotFound.
	contactType, _ := data.GetString("contact_type")
	group := false
	var ids []string
	if data.Contains("gid") {
		group = true
		ids = self.getIds(data, "gid")
	} else {
		ids = self.getIds(data, "uid")
	}
	contacts, err := models.ContactManager.GetAllNotify(ctx, ids, contactType, group)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "get all contacts error"))
	}

	notificationIDs, err := models.NotificationManager.BatchCreate(ctx, data, contacts)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	ret := jsonutils.NewDict()
	ret.Add(jsonutils.NewStringArray(notificationIDs), "notifications")
	return ret, nil
}

func (self *NotifyModelDispatcher) getIds(data jsonutils.JSONObject, key string) []string {
	var ids []string
	tmpIds, err := data.GetArray(key)
	if err != nil {
		id, _ := data.GetString(key)
		ids = make([]string, 1)
		ids[0] = id
	} else {
		ids = noutils.JsonArrayToStringArray(tmpIds)
	}
	// remove entry in which empty content
	ret := make([]string, 0, len(ids))
	for _, id := range ids {
		if len(id) == 0 {
			continue
		}
		ret = append(ret, id)
	}
	return ret[:len(ret):len(ret)]
}

// Verify process:
// 1.fetch verify by ID; 2.check that if verify is expired;
// 3.if not check that if token is correct and update status of contact whose id is verify's CID
// 4.otherwise generate a new verify and delete old one
func (self *NotifyModelDispatcher) Verify(ctx context.Context, params map[string]string, query jsonutils.JSONObject) error {
	processID := params["<id>"]
	token, _ := query.GetString("token")
	manager := models.VerifyManager
	verifys, err := manager.FetchByID(processID)
	if err != nil {
		return httperrors.NewGeneralError(err)
	}
	var verifition models.SVerify
	if len(verifys) == 0 {
		return httperrors.NewNotFoundError("%s verify record not found", processID)
	}
	current, have := time.Now(), false
	for i := range verifys {
		if current.Before(verifys[i].ExpireAt) {
			verifition = verifys[i]
			have = true
			break
		}
	}
	if !have {
		return httperrors.NewBadRequestError(models.VERIFICATION_TOKEN_EXPIRED)
	}
	if verifition.Token != token {
		return httperrors.NewBadRequestError(models.VERIFICATION_TOKEN_INVALID)
	}
	// modify contact's status and verified time.
	data := jsonutils.NewDict()
	data.Set("status", jsonutils.NewString(models.VERIFICATION_VERIFIED))
	data.Set("verified_at", jsonutils.NewTimeString(current))
	_, err = self.Update(ctx, verifition.CID, jsonutils.JSONNull, data, nil)
	if err != nil {
		return httperrors.NewGeneralError(err)
	}
	return nil
}

// VerifyTrigger process:
// 1.fetch contact by the information in data
// 2.if contact'status is 'init', make a new verify and send a verify message to the contact adress
// 3.if contact'status is 'verifying', fetch verify by CID, generate a new verify if it has expired
//   or return a error mention that "please don't try again".
func (self *NotifyModelDispatcher) VerifyTrigger(ctx context.Context, params map[string]string, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	uid := params["<uid>"]
	contact, _ := data.GetString("contact")
	contactType, _ := data.GetString("contact_type")
	contacts, err := models.ContactManager.FetchByMore(uid, contact, contactType)
	if err != nil || len(contacts) == 0 {
		return nil, errors.Error(fmt.Sprintf("uid '%s' don't have contact '%s' of contact_type '%s'", uid, contact, contactType))
	}
	userCred := policy.FetchUserCredential(ctx)
	scontact := contacts[0]

	makeNewVerify := func() (jsonutils.JSONObject, error) {
		verification := models.NewSVerify(contactType, scontact.ID)
		err = models.VerifyManager.Create(ctx, userCred, verification)
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
		// update contact state
		if scontact.Status != models.CONTACT_VERIFYING {
			scontact.SetStatus(userCred, models.CONTACT_VERIFYING, "")
		}

		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
		processID := verification.ID
		err := models.SendVerifyMessage(ctx, userCred, verification, &scontact)
		if err != nil {
			return nil, err
		}
		ret := map[string]map[string]string{
			"contact": {
				"process_id": processID,
			},
		}
		return jsonutils.Marshal(ret), nil
	}
	if scontact.Status == models.CONTACT_INIT {
		return makeNewVerify()
	}
	if scontact.Status == models.CONTACT_VERIFYING {
		verifications, err := models.VerifyManager.FetchByCID(scontact.ID, func(q *sqlchemy.SQuery) *sqlchemy.SQuery {
			q = q.In("status", []string{models.VERIFICATION_SENT, "init"}).Desc("created_at")
			return q
		})
		if err != nil {
			return nil, httperrors.NewGeneralError(err)
		}
		if len(verifications) == 0 {
			// no verifications in status "sent"
			return makeNewVerify()
		}
		current := time.Now()
		for _, verification := range verifications {
			if current.After(verification.ExpireAt) {
				//delete old one
				err = DeleteItem(&verification, ctx, userCred, jsonutils.JSONNull, jsonutils.JSONNull)
				if err != nil {
					return nil, httperrors.NewGeneralError(err)
				}
				return makeNewVerify()
			}
		}
		return nil, httperrors.NewGeneralError(models.ErrVeritying)
	}

	return jsonutils.JSONNull, nil
}

// DeleteContacts delete a group of contacts
func (self *NotifyModelDispatcher) DeleteContacts(ctx context.Context, uids2 []jsonutils.JSONObject) error {
	// Get all id of uid
	uids := make([]string, len(uids2))
	for i := range uids2 {
		uids[i] = strings.Trim(uids2[i].String(), `"`)
	}
	uname := false
	if v := ctx.Value("uname"); v != nil {
		uname = true
	}
	contacts, err := models.ContactManager.FetchByUIDs(ctx, uids, uname)
	if err != nil {
		return httperrors.NewGeneralError(err)
	}
	userCred := policy.FetchUserCredential(ctx)
	deleteFailed := make([]string, 0, 1)
	for _, contact := range contacts {
		err = DeleteItem(&contact, ctx, userCred, jsonutils.JSONNull, jsonutils.JSONNull)
		if err != nil {
			deleteFailed = append(deleteFailed, contact.ID)
		}
	}
	if len(deleteFailed) != 0 {
		errInfo := strings.Join(deleteFailed, ", ") + " ; these contact delete failed."
		return errors.Error(errInfo)
	}
	return nil
}

// UpdateContacts analysis the data and update corresponding contacts if they exist in the database create new ones.
func (self *NotifyModelDispatcher) UpdateContacts(ctx context.Context, idstr string, query jsonutils.JSONObject,
	datas []jsonutils.JSONObject, ctxIds []dispatcher.SResourceContext) error {

	type pair struct {
		contact string
		enabled string
	}

	// contactInfos will be used to find all contact info need to update.
	// And others will be created.
	contactInfos := make(map[string]pair)
	contactTypes := make([]string, len(datas))
	for i := range datas {
		contactType, _ := datas[i].GetString("contact_type")
		if _, ok := models.UpdateNotAllow[contactType]; ok {
			continue
		}
		contact, _ := datas[i].GetString("contact")
		enabled := "-1"
		if datas[i].Contains("enabled") {
			enabled, _ = datas[i].GetString("enabled")
		}
		contactInfos[contactType] = pair{contact, enabled}
		contactTypes[i] = contactType
	}

	records, err := models.ContactManager.FetchByUIDAndCType(idstr, contactTypes)
	if err != nil {
		return httperrors.NewGeneralError(err)
	}

	// updateFailed record the information of failed update
	updateFailed := make([]string, 0, 1)
	deleteFailed := make([]string, 0, 1)
	// UpdateItem contact info
	userCred := policy.FetchUserCredential(ctx)
	for i := range records {
		contactType := records[i].ContactType
		pairUpdate := contactInfos[contactType]
		if len(pairUpdate.contact) == 0 {
			// delete
			err = DeleteItem(&records[i], ctx, userCred, jsonutils.JSONNull, jsonutils.JSONNull)
			if err != nil {
				deleteFailed = append(deleteFailed, fmt.Sprintf(`uid:%q, contact_type:%q`, idstr, contactType))
			}
			continue
		}
		updateData := jsonutils.NewDict()
		updateData.Set("contact", jsonutils.NewString(pairUpdate.contact))
		if pairUpdate.enabled != "-1" {
			updateData.Set("enabled", jsonutils.NewString(pairUpdate.enabled))
		}
		if records[i].Contact != pairUpdate.contact {
			updateData.Set("status", jsonutils.NewString(models.CONTACT_INIT))
		}
		// update is not relational
		//updateData.Set("status", jsonutils.NewString("init"))
		err = UpdateItem(models.ContactManager, &records[i], ctx, userCred, jsonutils.JSONNull, updateData)
		if err != nil {
			updateFailed = append(updateFailed, fmt.Sprintf(`uid:%q, contact_type:%q, contact:%q`, idstr, contactType, pairUpdate.contact))
		}
		delete(contactInfos, contactType)
	}

	// createFailed record the information of failed creation
	createFailed := make([]string, 0, 1)
	// Create contact info
	newDatas := make([]map[string]interface{}, 0, len(contactInfos))
	for conType, conPair := range contactInfos {
		tmpMap := map[string]interface{}{
			"uid":          idstr,
			"contact_type": conType,
			"contact":      conPair.contact,
		}
		if conPair.enabled != "-1" {
			tmpMap["enabled"] = conPair.enabled
		}
		newDatas = append(newDatas, tmpMap)
	}

	for _, newData := range newDatas {
		_, err := self.Create(ctx, jsonutils.JSONNull, jsonutils.Marshal(newData), ctxIds)
		if err != nil {
			createFailed = append(createFailed, fmt.Sprintf(`uid:%q, contact_type:%q, contact:%q`, idstr,
				newData["contact_type"], newData["contact"]))
		}
	}

	// generate error through updateFailed and createFailed
	if len(updateFailed) != 0 || len(createFailed) != 0 || len(deleteFailed) != 0 {
		var errInfoBuffer strings.Builder
		if len(updateFailed) != 0 {
			errInfoBuffer.WriteString(strings.Join(updateFailed, "; "))
			errInfoBuffer.WriteString(" update failed. ")
		}
		if len(deleteFailed) != 0 {
			errInfoBuffer.WriteString(strings.Join(updateFailed, "; "))
			errInfoBuffer.WriteString(" delete failed. ")
		}
		if len(createFailed) != 0 {
			errInfoBuffer.WriteString(strings.Join(createFailed, "; "))
			errInfoBuffer.WriteString(" create failed. ")
		}
		errInfo := errInfoBuffer.String()
		return httperrors.NewGeneralError(errors.Error(errInfo))
	}

	if query.Contains("update_dingtalk") {
		models.UpdateDingtalk(idstr)
	}
	return nil
}

func (self *NotifyModelDispatcher) UpdateTemplate(ctx context.Context, ctype string, query jsonutils.JSONObject,
	datas []jsonutils.JSONObject) error {

	type sTemplate struct {
		ContactType  string
		Topic        string
		TemplateType string
		Content      string
	}
	templates := make([]sTemplate, 0, len(datas))
	topics := sets.NewString()
	for _, data := range datas {
		var tem sTemplate
		err := data.Unmarshal(&tem)
		if err != nil {
			return errors.Wrap(err, "data.Unmarshal")
		}
		if tem.TemplateType != models.TEMPLATE_TYPE_REMOTE && tem.TemplateType != models.
			TEMPLATE_TYPE_CONTENT && tem.TemplateType != models.TEMPLATE_TYPE_TITLE {

			return httperrors.NewInputParameterError("no support for such template type '%s'", tem.TemplateType)
		}
		tem.ContactType = ctype
		tem.Topic = strings.ToUpper(tem.Topic)
		templates = append(templates, tem)
		topics.Insert(tem.Topic)
	}

	q := models.TemplateManager.Query().Equals("contact_type", ctype).In("topic", topics.List())
	templateModels := make([]models.STemplate, 0, 1)
	err := db.FetchModelObjects(models.ContactManager, q, &templateModels)
	if err != nil {
		log.Errorf("db.FetchModelObjects sql: %s", q.String())
		return errors.Wrap(err, "db.FetchModelObjects")
	}

	templateMaps := make(map[string]*models.STemplate)
	for i := range templateModels {
		k := fmt.Sprintf("%s/%s/%s", templateModels[i].ContactType, templateModels[i].Topic, templateModels[i].TemplateType)
		templateMaps[k] = &templateModels[i]
	}

	userCred := policy.FetchUserCredential(ctx)
	for _, tem := range templates {
		k := fmt.Sprintf("%s/%s/%s", tem.ContactType, tem.Topic, tem.TemplateType)
		if tmod, ok := templateMaps[k]; ok {
			updateData := jsonutils.NewDict()
			updateData.Add(jsonutils.NewString(tem.Content), "content")
			err = UpdateItem(models.TemplateManager, tmod, ctx, userCred, query, updateData)
			if err != nil {
				return errors.Wrapf(err, "fail to update template '%s'", tmod.ID)
			}
			continue
		}
		_, err = self.Create(ctx, query, jsonutils.Marshal(tem), nil)
		if err != nil {
			return errors.Wrapf(err, "fail to create template(contact_type: %s, topic: %s, template_type: %s)",
				tem.ContactType, tem.Topic, tem.TemplateType)
		}
	}
	return nil
}

func (self *NotifyModelDispatcher) DeleteTemplate(ctx context.Context, query jsonutils.JSONObject, ctype, topic string) error {

	q := models.TemplateManager.Query().Equals("contact_type", ctype)
	if len(topic) != 0 {
		q = q.Equals("topic", topic)
	}
	templates := make([]models.STemplate, 0, 1)
	err := db.FetchModelObjects(models.TemplateManager, q, &templates)
	if err != nil {
		log.Errorf("db.FetchModelObjects sql: %s", q.String())
		return errors.Wrap(err, "db.FetchModelObjects")
	}
	userCred := policy.FetchUserCredential(ctx)
	for i := range templates {
		err = DeleteItem(&templates[i], ctx, userCred, query, jsonutils.JSONNull)
		if err != nil {
			return errors.Wrapf(err, "fail to delete template '%s'", templates[i].ID)
		}
	}
	return nil
}

// fetchEnv fetch handler, params, query and body from ctx(context.Context)
func fetchEnv(ctx context.Context, w http.ResponseWriter, r *http.Request) (*NotifyModelDispatcher, map[string]string, jsonutils.JSONObject, jsonutils.JSONObject) {
	params, query, body := appsrv.FetchEnv(ctx, w, r)
	metadata := appctx.AppContextMetadata(ctx)
	manager, ok := metadata["manager"].(*NotifyModelDispatcher)
	if !ok {
		log.Fatalf("No manager found for URL: %s", r.URL)
	}
	return manager, params, query, body
}

func mergeQueryParams(params map[string]string, query jsonutils.JSONObject, excludes ...string) jsonutils.JSONObject {
	if query == nil {
		query = jsonutils.NewDict()
	}
	queryDict := query.(*jsonutils.JSONDict)
	for k, v := range params {
		if !utils.IsInStringArray(k, excludes) {
			queryDict.Add(jsonutils.NewString(v), k[1:len(k)-1])
		}
	}
	return queryDict
}

// DeleteItem delete a database record corresponding to model
func DeleteItem(model db.IModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lockman.LockObject(ctx, model)
	defer lockman.ReleaseObject(ctx, model)
	err := model.ValidateDeleteCondition(ctx)
	if err != nil {
		log.Errorf("validate delete condition error: %s", err)
		return err
	}
	err = db.CustomizeDelete(model, ctx, userCred, query, data)
	if err != nil {
		log.Errorf("customize delete error: %s", err)
		return httperrors.NewNotAcceptableError(err.Error())
	}
	model.PreDelete(ctx, userCred)
	err = model.Delete(ctx, userCred)
	if err != nil {
		log.Errorf("Delete error %s", err)
		return err
	}
	model.PostDelete(ctx, userCred)
	return nil
}

// UpdateItem update a database record corresponding to model whose update fields are in data
func UpdateItem(manager db.IModelManager, item db.IModel, ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	lockman.LockObject(ctx, item)
	defer lockman.ReleaseObject(ctx, item)
	var err error
	err = item.ValidateUpdateCondition(ctx)

	if err != nil {
		log.Errorf("validate update condition error: %s", err)
		return httperrors.NewGeneralError(err)
	}

	dataDict, ok := data.(*jsonutils.JSONDict)
	if !ok {
		return httperrors.NewInternalServerError("Invalid data JSONObject")
	}

	dataDict, err = db.ValidateUpdateData(item, ctx, userCred, query, dataDict)
	if err != nil {
		errMsg := fmt.Sprintf("validate update data error: %s", err)
		log.Errorf(errMsg)
		return httperrors.NewGeneralError(err)
	}

	item.PreUpdate(ctx, userCred, query, dataDict)

	diff, err := db.Update(item, func() error {
		filterData := dataDict.CopyIncludes(updateFields(manager, userCred)...)
		err = filterData.Unmarshal(item)
		if err != nil {
			errMsg := fmt.Sprintf("unmarshal fail: %s", err)
			log.Errorf(errMsg)
			return httperrors.NewGeneralError(err)
		}
		return nil
	})

	if err != nil {
		log.Errorf("save update error: %s", err)
		return httperrors.NewGeneralError(err)
	}
	db.OpsLog.LogEvent(item, db.ACT_UPDATE, diff, userCred)

	item.PostUpdate(ctx, userCred, query, data)

	return nil

}

// get the field of model which is d
func updateFields(manager db.IModelManager, userCred mcclient.TokenCredential) []string {
	ret := make([]string, 0)
	for _, col := range manager.TableSpec().Columns() {
		tags := col.Tags()
		update := tags["update"]
		if allowAction(manager, userCred, update, db.IsAllowUpdate) {
			ret = append(ret, col.Name())
		}
	}
	return ret
}

func allowAction(manager db.IResource, userCred mcclient.TokenCredential, action string, testfunc func(scope rbacutils.TRbacScope, userCred mcclient.TokenCredential, manager db.IResource) bool) bool {
	if action == "user" {
		return true
	}
	if action == "domain" && (testfunc(rbacutils.ScopeDomain, userCred, manager) || testfunc(rbacutils.ScopeSystem, userCred, manager)) {
		return true
	}
	if action == "admin" && testfunc(rbacutils.ScopeSystem, userCred, manager) {
		return true
	}
	return false
}
