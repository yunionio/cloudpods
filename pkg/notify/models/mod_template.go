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
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/notify/options"
	"yunion.io/x/onecloud/pkg/notify/rpc/apis"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type STemplateManager struct {
	SStandaloneResourceBaseManager
}

var TemplateManager *STemplateManager

func init() {
	TemplateManager = &STemplateManager{
		SStandaloneResourceBaseManager: NewStandaloneResourceBaseManager(
			STemplate{},
			"notify_t_template",
			"notifytemplate",
			"notifytemplates",
		),
	}
}

const (
	TEMPLATE_TYPE_TITLE   = "title"
	TEMPLATE_TYPE_CONTENT = "content"
	TEMPLATE_TYPE_REMOTE  = "remote"
	CONTACTTYPE_ALL       = "all"
)

type STemplate struct {
	SStandaloneResourceBase

	ContactType string `width:"16" nullable:"false" create:"required" update:"user" list:"user"`
	Topic       string `width:"20" nullable:"false" create:"required" update:"user" list:"user"`

	// title | content | remote
	TemplateType string `width:"10" nullable:"false" create:"required" update:"user" list:"user"`
	Content      string `length:"text" nullable:"false" create:"required" get:"user" list:"user" update:"user"`
}

const (
	verifyUrlPath = "/email-verification/id/{0}/token/{1}?region=%s"
)

func (tm *STemplateManager) GetEmailUrl() string {
	if len(options.Options.ApiServer) > 0 {
		return httputils.JoinPath(options.Options.ApiServer, fmt.Sprintf(verifyUrlPath, options.Options.Region))
	}
	return options.Options.VerifyEmailUrl
}

var templatePath = "/opt/yunion/share/template"

func (tm *STemplateManager) defaultTemplate() ([]STemplate, error) {
	templates := make([]STemplate, 0, 4)

	for _, templateType := range []string{"title", "content", "remote"} {
		contactType, topic := CONTACTTYPE_ALL, ""
		titleTemplatePath := fmt.Sprintf("%s/%s", templatePath, templateType)
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
				TemplateType: templateType,
				Content:      string(content),
			})
		}
	}
	return templates, nil
}

var (
	InitVerifyEmailOver = false
)

type CompanyInfo struct {
	Logo       string
	LogoFormat string
	Copyright  string
}

func (tm *STemplateManager) TryInitVerifyEmail(ctx context.Context) error {
	if InitVerifyEmailOver {
		return nil
	}
	// fetch copyright and logo
	session := auth.GetAdminSession(ctx, "", "")
	obj, err := modules.Info.Get(session, "info", jsonutils.NewDict())
	if err != nil {
		return err
	}
	var info CompanyInfo
	err = obj.Unmarshal(&info)
	if err != nil {
		return err
	}

	// fetch verify email template
	q := tm.Query().Equals("contact_type", "email").Equals("topic", "VERIFY").Equals("template_type", "content")
	tem := &STemplate{}
	err = q.First(tem)
	if err != nil {
		return err
	}

	tem.SetModelManager(TemplateManager, tem)

	content, err := tem.Execute(jsonutils.Marshal(info).String())
	if err != nil {
		return err
	}

	_, err = db.Update(tem, func() error {
		tem.Content = content
		return nil
	})
	return err
}

func (tm *STemplateManager) InitializeData() error {
	templates, err := tm.defaultTemplate()
	if err != nil {
		return err
	}
	for _, template := range templates {
		q := tm.Query().Equals("contact_type", template.ContactType).Equals("topic", template.Topic).Equals("template_type", template.TemplateType)
		count, _ := q.CountWithError()
		if count > 0 {
			continue
		}
		err := tm.TableSpec().InsertOrUpdate(&template)
		if err != nil {
			return errors.Wrap(err, "sqlchemy.TableSpec.InsertOrUpdate")
		}
	}
	return nil
}

// NotifyFilter will return the title and content generated by corresponding template.
// Local cache about common template will be considered in case of performance issues.
func (tm *STemplateManager) NotifyFilter(contactType, topic, msg string) (params apis.SendParams, err error) {
	params.Topic = topic
	templates := make([]STemplate, 0, 3)
	q := tm.Query().Equals("topic", strings.ToUpper(topic)).In("contact_type", []string{CONTACTTYPE_ALL, contactType})
	err = db.FetchModelObjects(tm, q, &templates)
	if errors.Cause(err) == sql.ErrNoRows || len(templates) == 0 {
		// no such template, return as is
		params.Title = topic
		params.Message = msg
		return
	}
	if err != nil {
		err = errors.Wrap(err, "db.FetchModelObjects")
		return
	}
	for _, template := range templates {
		var title, content string
		switch template.TemplateType {
		case TEMPLATE_TYPE_TITLE:
			title, err = template.Execute(msg)
			if err != nil {
				return
			}
			params.Title = title
		case TEMPLATE_TYPE_CONTENT:
			content, err = template.Execute(msg)
			if err != nil {
				return
			}
			params.Message = content
		case TEMPLATE_TYPE_REMOTE:
			params.RemoteTemplate = template.Content
			params.Message = msg
		default:
			err = errors.Error("no support template type")
			return
		}
	}
	return
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

func (manager *STemplateManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {

	ty, _ := data.GetString("template_type")
	if ty != TEMPLATE_TYPE_TITLE && ty != TEMPLATE_TYPE_CONTENT && ty != TEMPLATE_TYPE_REMOTE {
		return nil, httperrors.NewInputParameterError("no such support for tempalte type %s", ty)
	}
	return data, nil
}

func (self *STemplateManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	queryDict := query.(*jsonutils.JSONDict)
	if queryDict.Contains("topic") {
		val, _ := queryDict.GetString("topic")
		q = q.Equals("topic", val)
	}
	if queryDict.Contains("template_type") {
		val, _ := queryDict.GetString("template_type")
		q = q.Equals("template_type", val)
	}
	if queryDict.Contains("contact_type") {
		val, _ := queryDict.GetString("contact_type")
		q = q.Equals("contact_type", val)
	}
	return q, nil
}
