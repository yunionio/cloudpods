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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-model-singular=webappenvironment
// +onecloud:swagger-gen-model-plural=webappenvironments
type SAppEnvironmentManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
}

type SAppEnvironment struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	AppId string `width:"36" charset:"ascii" index:"true"`
}

var AppEnvironmentManager *SAppEnvironmentManager

func init() {
	AppEnvironmentManager = &SAppEnvironmentManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SAppEnvironment{},
			"appenvironments_tbl",
			"webappenvironment",
			"webappenvironments",
		),
	}
	AppEnvironmentManager.SetVirtualObject(AppEnvironmentManager)
}

func (aem *SAppEnvironmentManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.AppEnvironmentListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = aem.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = aem.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	if input.AppId != "" {
		q = q.Equals("app_id", input.AppId)
	}
	if input.InstanceType != "" {
		q = q.Equals("instance_type", input.InstanceType)
	}
	return q, nil
}

func (aem *SAppEnvironmentManager) OrderByExtraFields(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.AppEnvironmentListInput) (*sqlchemy.SQuery, error) {
	return aem.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
}

func (aem *SAppEnvironmentManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	var err error
	q, err = aem.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
	}
	return q, nil
}

func (aem *SAppEnvironmentManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := aem.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	switch field {
	case "instance_type":
		q = aem.Query("instance_type").Distinct()
	}
	return q, nil
}

func (a *SApp) SyncAppEnvironments(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, exts []cloudprovider.ICloudAppEnvironment) compare.SyncResult {
	lockman.LockRawObject(ctx, AppEnvironmentManager.KeywordPlural(), a.Id)
	defer lockman.ReleaseRawObject(ctx, AppEnvironmentManager.KeywordPlural(), a.Id)
	result := compare.SyncResult{}
	aes, err := a.GetAppEnvironments()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SAppEnvironment, 0)
	commondb := make([]SAppEnvironment, 0)
	commonext := make([]cloudprovider.ICloudAppEnvironment, 0)
	added := make([]cloudprovider.ICloudAppEnvironment, 0)
	// compare
	err = compare.CompareSets(aes, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	// remove
	for i := 0; i < len(removed); i++ {
		err := removed[i].syncRemoveCloudAppEnvironment(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}

	// sync with cloud
	for i := 0; i < len(commondb); i++ {
		err := commondb[i].SyncWithCloudAppEnvironment(ctx, userCred, commonext[i])
		if err != nil {
			result.UpdateError(err)
			continue
		}
		result.Update()
	}

	// new one
	for i := 0; i < len(added); i++ {
		_, err := a.newFromCloudAppEnvironment(ctx, userCred, provider, added[i])
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}
	return result
}

func (a *SApp) newFromCloudAppEnvironment(ctx context.Context, userCred mcclient.TokenCredential, provider *SCloudprovider, ext cloudprovider.ICloudAppEnvironment) (*SAppEnvironment, error) {
	var err error
	appEnvironment := SAppEnvironment{}
	appEnvironment.SetModelManager(AppEnvironmentManager, &appEnvironment)

	appEnvironment.ExternalId = ext.GetGlobalId()
	appEnvironment.IsEmulated = ext.IsEmulated()
	appEnvironment.Status = ext.GetStatus()

	appEnvironment.Name = ext.GetName()
	appEnvironment.AppId = a.Id
	err = AppEnvironmentManager.TableSpec().Insert(ctx, &appEnvironment)
	if err != nil {
		return nil, errors.Wrapf(err, "newFromCloudAppEnvironment.Insert")
	}

	SyncCloudProject(ctx, userCred, &appEnvironment, provider.GetOwnerId(), ext, provider)
	db.OpsLog.LogEvent(&appEnvironment, db.ACT_CREATE, appEnvironment.GetShortDesc(ctx), userCred)
	return nil, nil
}

func (ae *SAppEnvironment) syncRemoveCloudAppEnvironment(ctx context.Context, userCred mcclient.TokenCredential) error {
	return ae.Delete(ctx, userCred)
}

func (ae *SAppEnvironment) SyncWithCloudAppEnvironment(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudAppEnvironment) error {
	_, err := db.UpdateWithLock(ctx, ae, func() error {
		ae.ExternalId = ext.GetGlobalId()
		ae.Status = ext.GetStatus()
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "db.Update")
	}
	return nil
}

func (aem *SAppEnvironmentManager) FetchCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, objs []interface{}, fields stringutils2.SSortedStrings, isList bool) []api.AppEnvironmentDetails {
	rows := make([]api.AppEnvironmentDetails, len(objs))
	virRows := aem.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i].VirtualResourceDetails = virRows[i]
	}
	return rows
}
