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
	"fmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SMiscResourceManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

var MiscResourceManager *SMiscResourceManager

func init() {
	MiscResourceManager = &SMiscResourceManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SMiscResource{},
			"misc_resources_tbl",
			"misc_resource",
			"misc_resources",
		),
	}
	MiscResourceManager.SetVirtualObject(MiscResourceManager)
}

type SMiscResource struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase

	SManagedResourceBase

	SCloudregionResourceBase `width:"36" charset:"ascii" nullable:"false" list:"domain" create:"domain_required" default:"default"`

	ResourceType string               `width:"32" charset:"utf8" list:"user"`
	MiscConf     jsonutils.JSONObject `nullable:"true" get:"domain" list:"domain" update:"domain"`
}

func (manager *SMiscResourceManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
	}
}

func (self *SMiscResource) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	return self.SVirtualResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (manager *SMiscResourceManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.MiscResourceDetails {
	rows := make([]api.MiscResourceDetails, len(objs))
	virtRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.MiscResourceDetails{
			VirtualResourceDetails:  virtRows[i],
			ManagedResourceInfo:     managerRows[i],
			CloudregionResourceInfo: regionRows[i],
		}
	}
	return rows
}

func (self *SCloudregion) GetMiscResources() ([]SMiscResource, error) {
	misc := []SMiscResource{}
	q := MiscResourceManager.Query().Equals("cloudregion_id", self.Id)
	err := db.FetchModelObjects(MiscResourceManager, q, &misc)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return misc, nil
}

func (self *SCloudregion) SyncMiscResources(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	provider *SCloudprovider,
	exts []cloudprovider.ICloudMiscResource,
	xor bool,
) compare.SyncResult {
	lockman.LockRawObject(ctx, CloudproviderManager.Keyword(), fmt.Sprintf("%s-%s", provider.Id, self.Id))
	defer lockman.ReleaseRawObject(ctx, CloudproviderManager.Keyword(), fmt.Sprintf("%s-%s", provider.Id, self.Id))

	result := compare.SyncResult{}

	dbRes, err := self.GetMiscResources()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SMiscResource, 0)
	commondb := make([]SMiscResource, 0)
	commonext := make([]cloudprovider.ICloudMiscResource, 0)
	added := make([]cloudprovider.ICloudMiscResource, 0)

	err = compare.CompareSets(dbRes, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveCloudMiscResource(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
			continue
		}
		result.Delete()
	}
	if !xor {
		for i := 0; i < len(commondb); i += 1 {
			err = commondb[i].SyncWithCloudMiscResource(ctx, userCred, commonext[i], provider)
			if err != nil {
				result.UpdateError(err)
				continue
			}
			result.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		_, err := self.newFromCloudMiscResource(ctx, userCred, added[i], provider)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (self *SMiscResource) syncRemoveCloudMiscResource(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	return self.RealDelete(ctx, userCred)
}

func (self *SMiscResource) SyncWithCloudMiscResource(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudMiscResource, provider *SCloudprovider) error {
	diff, err := db.UpdateWithLock(ctx, self, func() error {
		self.Status = ext.GetStatus()
		self.IsEmulated = ext.IsEmulated()
		self.MiscConf = ext.GetConfig()

		if createdAt := ext.GetCreatedAt(); !createdAt.IsZero() {
			self.CreatedAt = createdAt
		}

		return nil
	})
	if err != nil {
		return err
	}
	if account := self.GetCloudaccount(); account != nil {
		syncVirtualResourceMetadata(ctx, userCred, self, ext, account.ReadOnly)
	}
	SyncCloudProject(ctx, userCred, self, provider.GetOwnerId(), ext, provider)

	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	return nil
}

func (self *SCloudregion) newFromCloudMiscResource(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudMiscResource, provider *SCloudprovider) (*SMiscResource, error) {
	misc := SMiscResource{}
	misc.SetModelManager(MiscResourceManager, &misc)

	misc.Status = ext.GetStatus()
	misc.ExternalId = ext.GetGlobalId()
	misc.CloudregionId = self.Id
	misc.ManagerId = provider.Id
	misc.ResourceType = ext.GetResourceType()
	misc.MiscConf = ext.GetConfig()
	if createdAt := ext.GetCreatedAt(); !createdAt.IsZero() {
		misc.CreatedAt = createdAt
	}
	misc.IsEmulated = ext.IsEmulated()

	var err = func() error {
		lockman.LockRawObject(ctx, MiscResourceManager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, MiscResourceManager.Keyword(), "name")

		newName, err := db.GenerateName(ctx, MiscResourceManager, userCred, ext.GetName())
		if err != nil {
			return err
		}
		misc.Name = newName

		return MiscResourceManager.TableSpec().Insert(ctx, &misc)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	syncVirtualResourceMetadata(ctx, userCred, &misc, ext, false)
	SyncCloudProject(ctx, userCred, &misc, provider.GetOwnerId(), ext, provider)

	db.OpsLog.LogEvent(&misc, db.ACT_CREATE, misc.GetShortDesc(ctx), userCred)

	return &misc, nil
}

func (self *SMiscResource) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	desc := self.SVirtualResourceBase.GetShortDesc(ctx)
	desc.Update(self.MiscConf)
	region, _ := self.GetRegion()
	provider := self.GetCloudprovider()
	info := MakeCloudProviderInfo(region, nil, provider)
	desc.Update(jsonutils.Marshal(&info))
	desc.Set("resource_type", jsonutils.NewString(self.ResourceType))
	return desc
}

func (manager *SMiscResourceManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.MiscResourceCreateInput,
) (api.MiscResourceCreateInput, error) {
	var err error
	input.VirtualResourceCreateInput, err = manager.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return input, err
	}

	return input, nil
}

func (self *SMiscResource) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	log.Infof("SMiscResource delete do nothing")
	return self.RealDelete(ctx, userCred)
}

func (self *SMiscResource) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}

// 列出Misc Resource
func (manager *SMiscResourceManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.MiscResourceListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SMiscResourceManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	q, err = manager.SManagedResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	q, err = manager.SCloudregionResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}

	return q, httperrors.ErrNotFound
}

func (manager *SMiscResourceManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.MiscResourceListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}
	return q, nil
}

func (manager *SMiscResourceManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SCloudregionResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SCloudregionResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemExportKeys")
		}
	}
	if keys.ContainsAny(manager.SManagedResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SManagedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}
