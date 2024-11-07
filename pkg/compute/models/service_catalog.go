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
	"database/sql"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	computeapis "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=servicecatalog
// +onecloud:swagger-gen-model-plural=servicecatalogs
type SServiceCatalogManager struct {
	db.SSharableVirtualResourceBaseManager
}

type SServiceCatalog struct {
	db.SSharableVirtualResourceBase

	IconUrl         string `charset:"ascii" create:"optional" list:"user" get:"user" update:"user"`
	GuestTemplateID string `width:"128" charset:"ascii" create:"optional" list:"user" get:"user" update:"user"`
}

var ServiceCatalogManager *SServiceCatalogManager

func init() {
	ServiceCatalogManager = &SServiceCatalogManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SServiceCatalog{},
			"servicecatalogs_tbl",
			"servicecatalog",
			"servicecatalogs",
		),
	}
	ServiceCatalogManager.SetVirtualObject(ServiceCatalogManager)
}

func (scm *SServiceCatalog) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, input *computeapis.ServiceCatalogUpdateInput) (*jsonutils.JSONDict, error) {

	data := jsonutils.NewDict()
	if len(input.GuestTemplate) > 0 {
		// check
		model, err := GuestTemplateManager.FetchByIdOrName(ctx, userCred, input.GuestTemplate)
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError("no such guest template")
		}
		if err != nil {
			return nil, err
		}
		data.Add(jsonutils.NewString(model.GetId()), "guest_template_id")
	}
	if len(input.Name) > 0 {
		// no need to check name
		data.Add(jsonutils.NewString(input.Name), "name")
	}
	if len(input.IconUrl) > 0 {
		//check icon url
		url, err := url.Parse(input.IconUrl)
		if err != nil {
			return nil, httperrors.NewInputParameterError("fail to parse icon url '%s'", input.IconUrl)
		}
		data.Add(jsonutils.NewString(url.String()), "icon_url")

	}
	return data, nil
}

func (scm *SServiceCatalogManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *computeapis.ServiceCatalogCreateInput) (*jsonutils.JSONDict,
	error) {

	if len(input.GuestTemplate) == 0 {
		return nil, httperrors.NewMissingParameterError("guest_template")
	}

	model, err := GuestTemplateManager.FetchByIdOrName(ctx, userCred, input.GuestTemplate)
	if errors.Cause(err) == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError("no such guest template")
	}
	if err != nil {
		return nil, err
	}
	/*
		gt := model.(*SGuestTemplate)
		//scope := rbacutils.String2Scope(gt.PublicScope)
		//if !gt.IsPublic || scope != rbacutils.ScopeSystem {
		//	return nil, httperrors.NewForbiddenError("guest template must be public in scope system")
		//}
		if userCred.GetProjectId() != gt.ProjectId {
			return nil, httperrors.NewForbiddenError("guest template must has same project id with the request")
		}
	*/

	data := input.JSON(input)
	data.Remove("guest_template")
	data.Add(jsonutils.NewString(model.GetId()), "guest_template_id")

	// check url
	if len(input.IconUrl) == 0 {
		return data, nil
	}
	url, err := url.Parse(input.IconUrl)
	if err != nil {
		return nil, httperrors.NewInputParameterError("fail to parse icon url '%s'", input.IconUrl)
	}
	data.Add(jsonutils.NewString(url.String()), "icon_url")
	return data, nil
}

func (sc *SServiceCatalog) PerformDeploy(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, input *computeapis.ServiceCatalogDeploy) (jsonutils.JSONObject, error) {

	if len(input.Name) == 0 && len(input.GenerateName) == 0 {
		return nil, httperrors.NewMissingParameterError("name or generate_name")
	}
	model, err := GuestTemplateManager.FetchById(sc.GuestTemplateID)
	if errors.Cause(err) == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError("no such guest_template %s", sc.GuestTemplateID)
	}
	if err != nil {
		return nil, err
	}
	guestTempalte := model.(*SGuestTemplate)
	content := guestTempalte.Content
	contentDict := content.(*jsonutils.JSONDict)
	if len(input.GenerateName) != 0 {
		contentDict.Add(jsonutils.NewString(input.GenerateName), "generate_name")
	} else {
		contentDict.Add(jsonutils.NewString(input.Name), "name")
	}
	if input.Count != 0 {
		input.Count = 1
	}
	contentDict.Add(jsonutils.NewInt(int64(input.Count)), "count")
	s := auth.GetSession(ctx, userCred, options.Options.Region)
	_, err = compute.Servers.Create(s, content)
	if err != nil {
		return nil, errors.Wrap(err, "fail to create guest")
	}
	return nil, err
}

// 服务目录列表
func (manager *SServiceCatalogManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input computeapis.ServiceCatalogListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemFilter")
	}
	if len(input.GuestTemplateId) > 0 {
		gtObj, err := GuestTemplateManager.FetchByIdOrName(ctx, userCred, input.GuestTemplateId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(GuestTemplateManager.Keyword(), input.GuestTemplateId)
			} else {
				return nil, errors.Wrap(err, "GuestTemplateManager.FetchByIdOrName")
			}
		}
		q = q.Equals("guest_template_id", gtObj.GetId())
	}
	return q, nil
}

func (manager *SServiceCatalogManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input computeapis.ServiceCatalogListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SSharableVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SServiceCatalogManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SServiceCatalogManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []computeapis.ServiceCatalogDetails {
	rows := make([]computeapis.ServiceCatalogDetails, len(objs))

	virtRows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = computeapis.ServiceCatalogDetails{
			SharableVirtualResourceDetails: virtRows[i],
		}
	}

	return rows
}
