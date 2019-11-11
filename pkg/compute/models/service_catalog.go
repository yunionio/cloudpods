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

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SServiceCatalogManager struct {
	db.SSharableVirtualResourceBaseManager
}

type SServiceCatalog struct {
	db.SSharableVirtualResourceBase

	IconUrl         string `charset:"ascii" create:"optional"`
	GuestTemplateID string `width:"128" charset:"ascii" create:"optional"`
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

func (scm *SServiceCatalogManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {

	if !data.Contains("guest_template") {
		return nil, httperrors.NewMissingParameterError("guest_template")
	}
	guestTemplate, _ := data.GetString("guest_template")
	model, err := GuestTemplateManager.FetchByIdOrName(userCred, guestTemplate)
	if errors.Cause(err) == sql.ErrNoRows {
		return nil, httperrors.NewResourceNotFoundError("no such guest template")
	}
	if err != nil {
		return nil, err
	}
	gt := model.(*SGuestTemplate)
	scope := rbacutils.String2Scope(gt.PublicScope)
	if !gt.IsPublic || scope != rbacutils.ScopeSystem {
		return nil, httperrors.NewForbiddenError("guest template must be public in scope system")
	}
	data.Add(jsonutils.NewString(model.GetId()), "guest_template_id")

	// check url
	if !data.Contains("icon_url") {
		return data, nil
	}
	urlstr, _ := data.GetString("icon_url")
	url, err := url.Parse(urlstr)
	if err != nil {
		return nil, httperrors.NewInputParameterError("fail to parse icon url '%s'", urlstr)
	}
	data.Add(jsonutils.NewString(url.String()), "icon_url")
	return data, nil
}

func (sc *SServiceCatalog) AllowPerformDeploy(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) bool {

	return sc.IsOwner(userCred) || db.IsAdminAllowPerform(userCred, sc, "deploy")
}

func (sc *SServiceCatalog) PerformDeploy(ctx context.Context, userCred mcclient.TokenCredential,
	query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {

	if !data.Contains("name") && !data.Contains("generate_name") {
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
	contentDict, dataDict := content.(*jsonutils.JSONDict), data.(*jsonutils.JSONDict)
	for _, k := range dataDict.SortedKeys() {
		v, _ := dataDict.Get(k)
		contentDict.Add(v, k)
	}
	if !contentDict.Contains("count") {
		contentDict.Add(jsonutils.NewInt(1), "count")
	}
	s := auth.GetSession(ctx, userCred, options.Options.Region, "")
	_, err = modules.Servers.Create(s, content)
	if err != nil {
		return nil, errors.Wrap(err, "fail to create guest")
	}
	return nil, err
}
