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
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	ptem "text/template"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	notifyv2 "yunion.io/x/onecloud/pkg/notify"
	"yunion.io/x/onecloud/pkg/notify/options"
	"yunion.io/x/onecloud/pkg/notify/rpc/apis"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type STemplateManager struct {
	db.SStandaloneAnonResourceBaseManager
}

var TemplateManager *STemplateManager

func init() {
	TemplateManager = &STemplateManager{
		SStandaloneAnonResourceBaseManager: db.NewStandaloneAnonResourceBaseManager(
			STemplate{},
			"template_tbl",
			"notifytemplate",
			"notifytemplates",
		),
	}
	TemplateManager.SetVirtualObject(TemplateManager)
}

const (
	CONTACTTYPE_ALL = "all"
)

type STemplate struct {
	db.SStandaloneAnonResourceBase

	ContactType string `width:"16" nullable:"false" create:"required" update:"user" list:"user"`
	Topic       string `width:"20" nullable:"false" create:"required" update:"user" list:"user"`

	// title | content | remote
	TemplateType string `width:"10" nullable:"false" create:"required" update:"user" list:"user"`
	Content      string `length:"text" nullable:"false" create:"required" get:"user" list:"user" update:"user"`
	Lang         string `width:"8" charset:"ascii" nullable:"false" list:"user" update:"user" create:"optional"`
	Example      string `nullable:"false" create:"optional" get:"user" list:"user" update:"user"`
}

const (
	verifyUrlPath = "/email-verification/id/{0}/token/{1}?region=%s"
	templatePath  = "/opt/yunion/share/template"
)

func (tm *STemplateManager) GetEmailUrl() string {
	return httputils.JoinPath(options.Options.ApiServer, fmt.Sprintf(verifyUrlPath, options.Options.Region))
}

func (tm *STemplateManager) defaultTemplate() ([]STemplate, error) {
	templates := make([]STemplate, 0, 4)

	for _, templateType := range []string{"title", "content"} {
		for _, lang := range []string{api.TEMPLATE_LANG_CN, api.TEMPLATE_LANG_EN} {
			contactType, topic := CONTACTTYPE_ALL, ""
			titleTemplatePath := fmt.Sprintf("%s/%s@%s", templatePath, templateType, lang)
			files, err := ioutil.ReadDir(titleTemplatePath)
			if err != nil {
				return templates, errors.Wrapf(err, "Read Dir '%s'", titleTemplatePath)
			}
			for _, file := range files {
				if file.IsDir() {
					continue
				}
				spliteName := strings.Split(file.Name(), ".")
				topic = spliteName[0]
				if len(spliteName) > 1 {
					contactType = spliteName[1]
				}
				fullPath := filepath.Join(titleTemplatePath, file.Name())
				content, err := ioutil.ReadFile(fullPath)
				if err != nil {
					return templates, err
				}
				templates = append(templates, STemplate{
					ContactType:  contactType,
					Topic:        topic,
					Lang:         lang,
					TemplateType: templateType,
					Content:      string(content),
				})
			}
		}
	}
	return templates, nil
}

type SCompanyInfo struct {
	LoginLogo       string `json:"login_logo"`
	LoginLogoFormat string `json:"login_logo_format"`
	Copyright       string `json:"copyright"`
	Name            string `json:"name"`
}

func (tm *STemplateManager) GetCompanyInfo(ctx context.Context) (SCompanyInfo, error) {
	// fetch copyright and logo
	session := auth.GetAdminSession(ctx, "", "")
	obj, err := modules.Info.Get(session, "info", jsonutils.NewDict())
	if err != nil {
		return SCompanyInfo{}, err
	}
	var info SCompanyInfo
	err = obj.Unmarshal(&info)
	if err != nil {
		return SCompanyInfo{}, err
	}
	return info, nil
}

var (
	forceInitTopic = []string{
		"VERIFY",
		"USER_LOGIN_EXCEPTION",
	}
	defaultLang = api.TEMPLATE_LANG_CN
)

func getTemplateLangFromCtx(ctx context.Context) string {
	return notifyclientI18nTable.Lookup(ctx, tempalteLang)
}

func (tm *STemplateManager) InitializeData() error {
	// init lang
	q := tm.Query().IsEmpty("lang")
	var noLangTemplates []STemplate
	err := db.FetchModelObjects(tm, q, &noLangTemplates)
	if err != nil {
		return errors.Wrap(err, "unable to fetch templates")
	}
	for i := range noLangTemplates {
		t := &noLangTemplates[i]
		_, err := db.Update(t, func() error {
			t.Lang = defaultLang
			return nil
		})
		if err != nil {
			return err
		}
	}
	templates, err := tm.defaultTemplate()
	if err != nil {
		return err
	}
	for _, template := range templates {
		q := tm.Query().Equals("contact_type", template.ContactType).Equals("topic", template.Topic).Equals("template_type", template.TemplateType).Equals("lang", template.Lang)
		count, _ := q.CountWithError()
		if count > 0 && !utils.IsInStringArray(template.Topic, forceInitTopic) {
			continue
		}
		if count == 0 {
			err := tm.TableSpec().Insert(context.TODO(), &template)
			if err != nil {
				return errors.Wrap(err, "sqlchemy.TableSpec.Insert")
			}
			continue
		}
		oldTemplates := make([]STemplate, 0, 1)
		err := db.FetchModelObjects(tm, q, &oldTemplates)
		if err != nil {
			return errors.Wrap(err, "db.FetchModelObjects")
		}
		// delete addtion
		var (
			ctx      = context.Background()
			userCred = auth.AdminCredential()
		)
		for i := 1; i < len(oldTemplates); i++ {
			err := oldTemplates[i].Delete(ctx, userCred)
			if err != nil {
				return errors.Wrap(err, "STemplate.Delete")
			}
		}
		// update
		oldTemplate := &oldTemplates[0]
		_, err = db.Update(oldTemplate, func() error {
			oldTemplate.Content = template.Content
			return nil
		})
		if err != nil {
			return errors.Wrap(err, "db.Update")
		}
	}
	return nil
}

// FillWithTemplate will return the title and content generated by corresponding template.
// Local cache about common template will be considered in case of performance issues.
func (tm *STemplateManager) FillWithTemplate(ctx context.Context, lang string, no notifyv2.SNotification) (params apis.SendParams, err error) {
	if len(lang) == 0 {
		params.Title = no.Topic
		params.Message = no.Message
		return
	}
	params.Topic = no.Topic
	templates := make([]STemplate, 0, 3)
	var q *sqlchemy.SQuery
	q = tm.Query().Equals("topic", strings.ToUpper(no.Topic)).Equals("lang", lang).In("contact_type", []string{CONTACTTYPE_ALL, no.ContactType})
	err = db.FetchModelObjects(tm, q, &templates)
	if errors.Cause(err) == sql.ErrNoRows || len(templates) == 0 {
		// no such template, return as is
		params.Title = no.Topic
		params.Message = no.Message
		return
	}
	if err != nil {
		err = errors.Wrap(err, "db.FetchModelObjects")
		return
	}
	for _, template := range tm.chooseTemplate(no.ContactType, templates) {
		var title, content string
		switch template.TemplateType {
		case api.TEMPLATE_TYPE_TITLE:
			title, err = template.Execute(no.Message)
			if err != nil {
				return
			}
			params.Title = title
		case api.TEMPLATE_TYPE_CONTENT:
			content, err = template.Execute(no.Message)
			if err != nil {
				return
			}
			params.Message = content
		case api.TEMPLATE_TYPE_REMOTE:
			params.RemoteTemplate = template.Content
			params.Message = no.Message
		default:
			err = errors.Error("no support template type")
			return
		}
	}
	return
}

func (tm *STemplateManager) chooseTemplate(contactType string, tempaltes []STemplate) []*STemplate {
	var titleTemplate, contentTemplate *STemplate
	// contactType first
	for i := range tempaltes {
		switch tempaltes[i].TemplateType {
		case api.TEMPLATE_TYPE_REMOTE:
			if tempaltes[i].ContactType == contactType {
				return []*STemplate{&tempaltes[i]}
			}
		case api.TEMPLATE_TYPE_TITLE:
			if tempaltes[i].ContactType == contactType {
				titleTemplate = &tempaltes[i]
			} else if titleTemplate == nil {
				titleTemplate = &tempaltes[i]
			}
		case api.TEMPLATE_TYPE_CONTENT:
			if tempaltes[i].ContactType == contactType {
				contentTemplate = &tempaltes[i]
			} else if contentTemplate == nil {
				contentTemplate = &tempaltes[i]
			}
		}
	}
	ret := make([]*STemplate, 0, 2)
	if titleTemplate != nil {
		ret = append(ret, titleTemplate)
	}
	if contentTemplate != nil {
		ret = append(ret, contentTemplate)
	}
	return ret
}

func (tm *STemplate) Execute(str string) (string, error) {
	tem, err := ptem.New("tmp").Parse(tm.Content)
	if err != nil {
		return "", errors.Wrapf(err, "Template.Parse for template %s", tm.GetId())
	}
	var buffer bytes.Buffer
	tmpMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(str), &tmpMap)
	if err != nil {
		return "", errors.Wrap(err, "json.Unmarshal")
	}
	err = tem.Execute(&buffer, tmpMap)
	if err != nil {
		return "", errors.Wrap(err, "template,Execute")
	}
	return buffer.String(), nil
}

func (tm *STemplateManager) AllowPerformSave(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsAdminAllowPerform(userCred, tm, "save")
}

func (tm *STemplateManager) PerformSave(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.TemplateManagerSaveInput) (jsonutils.JSONObject, error) {
	q := tm.Query().Equals("contact_type", input.ContactType)
	templates := []STemplate{}
	err := db.FetchModelObjects(tm, q, &templates)
	if err != nil {
		return nil, err
	}
	tempaltesMap := make(map[string]*api.TemplateCreateInput, len(input.Templates))
	for i := range input.Templates {
		template := &input.Templates[i]
		if template.ContactType != input.ContactType {
			continue
		}
		input.Templates[i], err = tm.ValidateCreateData(ctx, userCred, userCred, nil, input.Templates[i])
		if err != nil {
			return nil, err
		}
		key := fmt.Sprintf("%s-%s-%s", template.Topic, template.TemplateType, template.Lang)
		tempaltesMap[key] = template
	}
	for i := range templates {
		key := fmt.Sprintf("%s-%s-%s", templates[i].Topic, templates[i].TemplateType, templates[i].Lang)
		if _, ok := tempaltesMap[key]; !ok {
			continue
		}
		if input.Force {
			err := templates[i].Delete(ctx, userCred)
			if err != nil {
				return nil, errors.Wrapf(err, "unable to delete template %s", templates[i].Id)
			}
		} else {
			delete(tempaltesMap, key)
		}
	}
	for _, template := range tempaltesMap {
		t := STemplate{
			ContactType:  input.ContactType,
			Topic:        template.Topic,
			TemplateType: template.TemplateType,
			Lang:         template.Lang,
			Example:      template.Example,
			Content:      template.Content,
		}
		err = tm.TableSpec().Insert(ctx, &t)
		if err != nil {
			return nil, errors.Wrap(err, "unable to insert template")
		}
	}
	return nil, nil
}

func (tm *STemplateManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.TemplateCreateInput) (api.TemplateCreateInput, error) {
	var err error
	input.StandaloneAnonResourceCreateInput, err = tm.SStandaloneAnonResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.StandaloneAnonResourceCreateInput)
	if err != nil {
		return input, err
	}
	if !utils.IsInStringArray(input.TemplateType, []string{
		api.TEMPLATE_TYPE_CONTENT, api.TEMPLATE_TYPE_REMOTE, api.TEMPLATE_TYPE_TITLE,
	}) {
		return input, httperrors.NewInputParameterError("no such support for tempalte type %s", input.TemplateType)
	}
	if input.TemplateType != api.TEMPLATE_TYPE_REMOTE {
		if err := tm.validate(input.Content, input.Example); err != nil {
			return input, httperrors.NewInputParameterError(err.Error())
		}
	}
	if input.Lang == "" {
		input.Lang = api.TEMPLATE_LANG_CN
	}
	if !utils.IsInStringArray(input.Lang, []string{api.TEMPLATE_LANG_EN, api.TEMPLATE_LANG_CN}) {
		return input, httperrors.NewInputParameterError("no such lang %s", input.Lang)
	}
	return input, nil
}

func (tm *STemplateManager) validate(template string, example string) error {
	// check example availability
	tem, err := ptem.New("tmp").Parse(template)
	if err != nil {
		return errors.Wrap(err, "invalid template")
	}
	var buffer bytes.Buffer
	tmpMap := make(map[string]interface{})
	err = json.Unmarshal([]byte(example), &tmpMap)
	if err != nil {
		return errors.Wrap(err, "invalid example")
	}
	err = tem.Execute(&buffer, tmpMap)
	if err != nil {
		return errors.Wrap(err, "invalid example")
	}
	return nil
}

func (tm *STemplateManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.TemplateListInput) (*sqlchemy.SQuery, error) {
	q, err := tm.SStandaloneAnonResourceBaseManager.ListItemFilter(ctx, q, userCred, input.StandaloneAnonResourceListInput)
	if err != nil {
		return nil, err
	}
	if len(input.Topic) > 0 {
		q = q.Equals("topic", input.Topic)
	}
	if len(input.TemplateType) > 0 {
		q = q.Equals("template_type", input.TemplateType)
	}
	if len(input.ContactType) > 0 {
		q = q.Equals("contact_type", input.ContactType)
	}
	if len(input.Lang) > 0 {
		q = q.Equals("lang", input.Lang)
	}
	return q, nil
}

func (t *STemplate) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.TemplateUpdateInput) (api.TemplateUpdateInput, error) {
	var err error
	input.StandaloneAnonResourceBaseUpdateInput, err = t.SStandaloneAnonResourceBase.ValidateUpdateData(ctx, userCred, query, input.StandaloneAnonResourceBaseUpdateInput)
	if err != nil {
		return input, err
	}
	if t.TemplateType == api.TEMPLATE_TYPE_REMOTE {
		return input, nil
	}
	if err := TemplateManager.validate(input.Content, input.Example); err != nil {
		return input, httperrors.NewInputParameterError(err.Error())
	}
	return input, nil
}
