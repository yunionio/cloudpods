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
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type STablestoreManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SCloudregionResourceBaseManager
	SManagedResourceBaseManager
}

var TablestoreManager *STablestoreManager

func init() {
	TablestoreManager = &STablestoreManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			STablestore{},
			"tablestores_tbl",
			"tablestore",
			"tablestores",
		),
	}
	TablestoreManager.SetVirtualObject(TablestoreManager)
}

type STablestore struct {
	db.SVirtualResourceBase
	db.SExternalizedResourceBase
	SCloudregionResourceBase
	SManagedResourceBase
}

func (manager *STablestoreManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{CloudregionManager},
	}
}

func (self *STablestore) ValidateDeleteCondition(ctx context.Context, data *api.TablestoreDetails) error {
	return self.SVirtualResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SCloudregion) GetTablestores() ([]STablestore, error) {
	q := TablestoreManager.Query().Equals("cloudregion_id", self.Id)
	ret := []STablestore{}
	err := db.FetchModelObjects(TablestoreManager, q, &ret)
	return ret, err
}

func (self *SCloudregion) SyncTablestores(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	exts []cloudprovider.ICloudTablestore,
	provider *SCloudprovider,
	xor bool,
) compare.SyncResult {
	lockman.LockRawObject(ctx, TablestoreManager.Keyword(), self.Id)
	defer lockman.ReleaseRawObject(ctx, TablestoreManager.Keyword(), self.Id)

	result := compare.SyncResult{}

	dbRes, err := self.GetTablestores()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]STablestore, 0)
	commondb := make([]STablestore, 0)
	commonext := make([]cloudprovider.ICloudTablestore, 0)
	added := make([]cloudprovider.ICloudTablestore, 0)

	err = compare.CompareSets(dbRes, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveCloudTablestore(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}
	if !xor {
		for i := 0; i < len(commondb); i += 1 {
			err = commondb[i].SyncWithCloudTablestore(ctx, userCred, commonext[i], provider)
			if err != nil {
				result.UpdateError(err)
				continue
			}
			result.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		_, err := self.newFromCloudTablestore(ctx, userCred, added[i], provider)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (self *STablestore) syncRemoveCloudTablestore(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx, nil)
	if err != nil { // cannot delete
		self.SetStatus(ctx, userCred, api.TABLESTORE_STATUS_UNKNOWN, "Sync to remove")
		return err
	}
	return self.RealDelete(ctx, userCred)
}

func (self *STablestore) SyncWithCloudTablestore(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudTablestore, provider *SCloudprovider) error {
	diff, err := db.Update(self, func() error {
		if options.Options.EnableSyncName {
			newName, _ := db.GenerateAlterName(self, ext.GetName())
			if len(newName) > 0 {
				self.Name = newName
			}
		}

		self.Status = ext.GetStatus()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	db.OpsLog.LogSyncUpdate(self, diff, userCred)
	if len(diff) > 0 {
		notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
			Obj:    self,
			Action: notifyclient.ActionSyncUpdate,
		})
	}

	if account, _ := provider.GetCloudaccount(); account != nil {
		syncVirtualResourceMetadata(ctx, userCred, self, ext, account.ReadOnly)
	}

	SyncCloudProject(ctx, userCred, self, provider.GetOwnerId(), ext, provider)
	return nil
}

func (self *SCloudregion) newFromCloudTablestore(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudTablestore, provider *SCloudprovider) (*STablestore, error) {
	ret := &STablestore{}
	ret.SetModelManager(TablestoreManager, ret)

	ret.Status = ext.GetStatus()
	ret.ExternalId = ext.GetGlobalId()
	ret.CloudregionId = self.Id
	ret.ManagerId = provider.Id

	if createdAt := ext.GetCreatedAt(); !createdAt.IsZero() {
		ret.CreatedAt = createdAt
	}

	var err = func() error {
		lockman.LockRawObject(ctx, TablestoreManager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, TablestoreManager.Keyword(), "name")

		newName, err := db.GenerateName(ctx, TablestoreManager, provider.GetOwnerId(), ext.GetName())
		if err != nil {
			return err
		}
		ret.Name = newName
		return TablestoreManager.TableSpec().Insert(ctx, ret)
	}()
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}

	syncVirtualResourceMetadata(ctx, userCred, ret, ext, false)
	SyncCloudProject(ctx, userCred, ret, provider.GetOwnerId(), ext, provider)

	db.OpsLog.LogEvent(ret, db.ACT_CREATE, ret.GetShortDesc(ctx), userCred)
	notifyclient.EventNotify(ctx, userCred, notifyclient.SEventNotifyParam{
		Obj:    ret,
		Action: notifyclient.ActionSyncCreate,
	})

	return ret, nil
}

func (manager *STablestoreManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.TablestoreDetails {
	rows := make([]api.TablestoreDetails, len(objs))
	virRows := manager.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	managerRows := manager.SManagedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	regionRows := manager.SCloudregionResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.TablestoreDetails{
			VirtualResourceDetails:  virRows[i],
			ManagedResourceInfo:     managerRows[i],
			CloudregionResourceInfo: regionRows[i],
		}
	}
	return rows
}

func (manager *STablestoreManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.TablestoreCreateInput) (api.TablestoreCreateInput, error) {
	return input, httperrors.NewNotImplementedError("ValidateCreateData")
}

func (self *STablestore) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.TablestoreUpdateInput) (api.TablestoreUpdateInput, error) {
	return input, httperrors.NewNotImplementedError("ValidateCreateData")
}

func (self *STablestore) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	self.SetStatus(ctx, userCred, apis.STATUS_DELETING, "")
	return nil
}

func (self *STablestore) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	db.OpsLog.LogEvent(self, db.ACT_DELOCATE, self.GetShortDesc(ctx), userCred)
	return self.SVirtualResourceBase.Delete(ctx, userCred)
}

// Tablestore列表
func (manager *STablestoreManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.TablestoreListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SManagedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SCloudregionResourceBaseManager.ListItemFilter(ctx, q, userCred, input.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *STablestoreManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.TablestoreListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SManagedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.ManagedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SManagedResourceBaseManager.OrderByExtraFields")
	}
	q, err = manager.SCloudregionResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.RegionalFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SCloudregionResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *STablestoreManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
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

func (self *STablestore) ValidateUpdateCondition(ctx context.Context) error {
	return self.SVirtualResourceBase.ValidateUpdateCondition(ctx)
}

func (manager *STablestoreManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
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

func (manager *STablestoreManager) AllowScope(userCred mcclient.TokenCredential) rbacscope.TRbacScope {
	scope, _ := policy.PolicyManager.AllowScope(userCred, api.SERVICE_TYPE, TablestoreManager.KeywordPlural(), policy.PolicyActionGet)
	return scope
}
