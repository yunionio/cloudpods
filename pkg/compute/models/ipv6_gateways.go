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

type SIPv6GatewayManager struct {
	db.SSharableVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SVpcResourceBaseManager
}

var IPv6GatewayManager *SIPv6GatewayManager

func init() {
	IPv6GatewayManager = &SIPv6GatewayManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SIPv6Gateway{},
			"ipv6_gateways_tbl",
			"ipv6_gateway",
			"ipv6_gateways",
		),
	}
	IPv6GatewayManager.SetVirtualObject(IPv6GatewayManager)
}

type SIPv6Gateway struct {
	db.SSharableVirtualResourceBase
	db.SExternalizedResourceBase

	SVpcResourceBase `width:"36" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	InstanceType string `width:"64" charset:"utf8" nullable:"true" list:"user" create:"optional"`
}

func (manager *SIPv6GatewayManager) GetContextManagers() [][]db.IModelManager {
	return [][]db.IModelManager{
		{VpcManager},
	}
}

func (self *SIPv6Gateway) ValidateDeleteCondition(ctx context.Context, data *api.IPv6GatewayDetails) error {
	return self.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx, nil)
}

func (self *SVpc) GetIPv6Gateways() ([]SIPv6Gateway, error) {
	q := IPv6GatewayManager.Query().Equals("vpc_id", self.Id)
	ret := []SIPv6Gateway{}
	err := db.FetchModelObjects(IPv6GatewayManager, q, &ret)
	return ret, err
}

func (self *SVpc) SyncIPv6Gateways(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	exts []cloudprovider.ICloudIPv6Gateway,
	provider *SCloudprovider,
	xor bool,
) compare.SyncResult {
	lockman.LockRawObject(ctx, IPv6GatewayManager.Keyword(), self.Id)
	defer lockman.ReleaseRawObject(ctx, IPv6GatewayManager.Keyword(), self.Id)

	result := compare.SyncResult{}

	dbRes, err := self.GetIPv6Gateways()
	if err != nil {
		result.Error(err)
		return result
	}

	removed := make([]SIPv6Gateway, 0)
	commondb := make([]SIPv6Gateway, 0)
	commonext := make([]cloudprovider.ICloudIPv6Gateway, 0)
	added := make([]cloudprovider.ICloudIPv6Gateway, 0)

	err = compare.CompareSets(dbRes, exts, &removed, &commondb, &commonext, &added)
	if err != nil {
		result.Error(err)
		return result
	}

	for i := 0; i < len(removed); i += 1 {
		err = removed[i].syncRemoveCloudIPv6Gateway(ctx, userCred)
		if err != nil {
			result.DeleteError(err)
		} else {
			result.Delete()
		}
	}
	if !xor {
		for i := 0; i < len(commondb); i += 1 {
			err = commondb[i].SyncWithCloudIPv6Gateway(ctx, userCred, commonext[i], provider)
			if err != nil {
				result.UpdateError(err)
				continue
			}
			result.Update()
		}
	}
	for i := 0; i < len(added); i += 1 {
		_, err := self.newFromCloudIPv6Gateway(ctx, userCred, added[i], provider)
		if err != nil {
			result.AddError(err)
			continue
		}
		result.Add()
	}

	return result
}

func (self *SIPv6Gateway) syncRemoveCloudIPv6Gateway(ctx context.Context, userCred mcclient.TokenCredential) error {
	lockman.LockObject(ctx, self)
	defer lockman.ReleaseObject(ctx, self)

	err := self.ValidateDeleteCondition(ctx, nil)
	if err != nil { // cannot delete
		self.SetStatus(ctx, userCred, api.NETWORK_STATUS_UNKNOWN, "Sync to remove")
		return err
	}
	return self.RealDelete(ctx, userCred)
}

func (self *SIPv6Gateway) SyncWithCloudIPv6Gateway(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudIPv6Gateway, provider *SCloudprovider) error {
	diff, err := db.Update(self, func() error {
		if options.Options.EnableSyncName {
			newName, _ := db.GenerateAlterName(self, ext.GetName())
			if len(newName) > 0 {
				self.Name = newName
			}
		}

		self.Status = ext.GetStatus()
		self.InstanceType = ext.GetInstanceType()
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

func (self *SVpc) newFromCloudIPv6Gateway(ctx context.Context, userCred mcclient.TokenCredential, ext cloudprovider.ICloudIPv6Gateway, provider *SCloudprovider) (*SIPv6Gateway, error) {
	ret := &SIPv6Gateway{}
	ret.SetModelManager(IPv6GatewayManager, ret)

	ret.Status = ext.GetStatus()
	ret.ExternalId = ext.GetGlobalId()
	ret.VpcId = self.Id
	ret.InstanceType = ext.GetInstanceType()

	if createdAt := ext.GetCreatedAt(); !createdAt.IsZero() {
		ret.CreatedAt = createdAt
	}

	var err = func() error {
		lockman.LockRawObject(ctx, IPv6GatewayManager.Keyword(), "name")
		defer lockman.ReleaseRawObject(ctx, IPv6GatewayManager.Keyword(), "name")

		newName, err := db.GenerateName(ctx, IPv6GatewayManager, provider.GetOwnerId(), ext.GetName())
		if err != nil {
			return err
		}
		ret.Name = newName
		return IPv6GatewayManager.TableSpec().Insert(ctx, ret)
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

func (manager *SIPv6GatewayManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.IPv6GatewayDetails {
	rows := make([]api.IPv6GatewayDetails, len(objs))

	virtRows := manager.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	vpcRows := manager.SVpcResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	for i := range rows {
		rows[i] = api.IPv6GatewayDetails{
			SharableVirtualResourceDetails: virtRows[i],
			VpcResourceInfo:                vpcRows[i],
		}
	}
	return rows
}

func (manager *SIPv6GatewayManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input api.IPv6GatewayCreateInput) (api.IPv6GatewayCreateInput, error) {
	return input, httperrors.NewNotImplementedError("ValidateCreateData")
}

func (self *SIPv6Gateway) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.IPv6GatewayUpdateInput) (api.IPv6GatewayUpdateInput, error) {
	return input, httperrors.NewNotImplementedError("ValidateCreateData")
}

func (self *SIPv6Gateway) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	self.SetStatus(ctx, userCred, apis.STATUS_DELETING, "")
	return nil
}

func (self *SIPv6Gateway) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	db.OpsLog.LogEvent(self, db.ACT_DELOCATE, self.GetShortDesc(ctx), userCred)
	return self.SSharableVirtualResourceBase.Delete(ctx, userCred)
}

// IPv6网关列表
func (manager *SIPv6GatewayManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.IPv6GatewayListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemFilter")
	}
	q, err = manager.SVpcResourceBaseManager.ListItemFilter(ctx, q, userCred, input.VpcFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemFilter")
	}

	q, err = manager.SExternalizedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.ExternalizedResourceBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SExternalizedResourceBaseManager.ListItemFilter")
	}

	return q, nil
}

func (manager *SIPv6GatewayManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.IPv6GatewayListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SSharableVirtualResourceBaseManager.OrderByExtraFields(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SIPv6GatewayManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SSharableVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (self *SIPv6Gateway) ValidateUpdateCondition(ctx context.Context) error {
	return self.SSharableVirtualResourceBase.ValidateUpdateCondition(ctx)
}

func (self *SIPv6Gateway) GetChangeOwnerCandidateDomainIds() []string {
	candidates := [][]string{}
	vpc, _ := self.GetVpc()
	if vpc != nil {
		candidates = append(candidates, vpc.GetChangeOwnerCandidateDomainIds())
	}
	return db.ISharableMergeChangeOwnerCandidateDomainIds(self, candidates...)
}

func (manager *SIPv6GatewayManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SSharableVirtualResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SSharableVirtualResourceBaseManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SVpcResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SVpcResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SVpcResourceBaseManager.ListItemExportKeys")
		}
	}
	return q, nil
}

func (manager *SIPv6GatewayManager) AllowScope(userCred mcclient.TokenCredential) rbacscope.TRbacScope {
	scope, _ := policy.PolicyManager.AllowScope(userCred, api.SERVICE_TYPE, IPv6GatewayManager.KeywordPlural(), policy.PolicyActionGet)
	return scope
}
