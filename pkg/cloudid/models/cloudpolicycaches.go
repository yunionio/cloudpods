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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SCloudpolicycacheManager struct {
	db.SStatusStandaloneResourceBaseManager
	db.SExternalizedResourceBaseManager
	SCloudaccountResourceBaseManager
	SCloudproviderResourceBaseManager
}

var CloudpolicycacheManager *SCloudpolicycacheManager

func init() {
	CloudpolicycacheManager = &SCloudpolicycacheManager{
		SStatusStandaloneResourceBaseManager: db.NewStatusStandaloneResourceBaseManager(
			SCloudpolicycache{},
			"cloudpolicycaches_tbl",
			"cloudpolicycache",
			"cloudpolicycaches",
		),
	}
	CloudpolicycacheManager.SetVirtualObject(CloudpolicycacheManager)
}

type SCloudpolicycache struct {
	db.SStatusStandaloneResourceBase
	db.SExternalizedResourceBase
	SCloudaccountResourceBase
	SCloudproviderResourceBase

	// 权限Id
	CloudpolicyId string `width:"36" charset:"ascii" nullable:"true" list:"user" index:"true" json:"cloudpolicy_id"`
}

func (manager *SCloudpolicycacheManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

// 公有云权限缓存列表
func (manager *SCloudpolicycacheManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query api.CloudpolicycacheListInput) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SStatusStandaloneResourceBaseManager.ListItemFilter(ctx, q, userCred, query.StatusStandaloneResourceListInput)
	if err != nil {
		return nil, err
	}
	if len(query.CloudpolicyId) > 0 {
		policy, err := CloudpolicyManager.FetchByIdOrName(nil, query.CloudpolicyId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("cloudpolicy", query.CloudpolicyId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		q = q.Equals("cloudpolicy_id", policy.GetId())
	}
	if len(query.CloudaccountId) > 0 {
		account, err := CloudaccountManager.FetchByIdOrName(nil, query.CloudaccountId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2("cloudaccount", query.CloudaccountId)
			}
			return nil, httperrors.NewGeneralError(err)
		}
		q = q.Equals("cloudaccount_id", account.GetId())
	}
	return q, nil
}

// 获取公有云权限缓存详情
func (manager *SCloudpolicycacheManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CloudpolicycacheDetails {
	rows := make([]api.CloudpolicycacheDetails, len(objs))
	stdRows := manager.SStatusStandaloneResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	acRows := manager.SCloudaccountResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	apRows := manager.SCloudproviderResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i] = api.CloudpolicycacheDetails{
			StatusStandaloneResourceDetails: stdRows[i],
			CloudaccountResourceDetails:     acRows[i],
			CloudproviderResourceDetails:    apRows[i],
		}
	}
	return rows
}

func (manager *SCloudpolicycacheManager) Register(ctx context.Context, accountId, providerId, policyId string) (*SCloudpolicycache, error) {
	policy, err := CloudpolicyManager.FetchById(policyId)
	if err != nil {
		return nil, errors.Wrapf(err, "CloudpolicyManager.FetchById(%s)", policyId)
	}
	q := manager.Query().Equals("cloudaccount_id", accountId).Equals("cloudpolicy_id", policyId)
	if len(providerId) > 0 {
		q = q.Equals("cloudprovider_id", providerId)
	}
	caches := []SCloudpolicycache{}
	err = db.FetchModelObjects(manager, q, &caches)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	for i := range caches {
		if len(caches[i].ExternalId) > 0 {
			return &caches[i], nil
		}
	}
	if len(caches) > 0 {
		return &caches[0], nil
	}
	cache := &SCloudpolicycache{}
	cache.SetModelManager(manager, cache)
	cache.CloudaccountId = accountId
	cache.SCloudproviderResourceBase.CloudproviderId = providerId
	cache.CloudpolicyId = policyId
	cache.Name = policy.GetName()
	cache.Status = api.CLOUD_POLICY_CACHE_STATUS_CACHING
	return cache, manager.TableSpec().Insert(ctx, cache)
}

func (self *SCloudpolicycache) GetCloudprovider() (*SCloudprovider, error) {
	provider, err := CloudproviderManager.FetchById(self.CloudproviderId)
	if err != nil {
		return nil, errors.Wrapf(err, "CloudproviderManager.FetchById(%s)", self.CloudproviderId)
	}
	return provider.(*SCloudprovider), nil
}

func (self *SCloudpolicycache) GetProvider() (cloudprovider.ICloudProvider, error) {
	if len(self.CloudproviderId) > 0 {
		provider, err := self.GetCloudprovider()
		if err != nil {
			return nil, errors.Wrap(err, "GetCloudprovider")
		}
		return provider.GetProvider()
	}
	account, err := self.GetCloudaccount()
	if err != nil {
		return nil, errors.Wrap(err, "GetCloudaccount")
	}
	return account.GetProvider()
}

func (self *SCloudpolicycache) GetCloudpolicy() (*SCloudpolicy, error) {
	policy, err := CloudpolicyManager.FetchById(self.CloudpolicyId)
	if err != nil {
		return nil, errors.Wrapf(err, "CloudpolicyManager.FetchById(%s)", self.CloudpolicyId)
	}
	return policy.(*SCloudpolicy), nil
}

func (self *SCloudpolicycache) cacheCustomCloudpolicy() error {
	provider, err := self.GetProvider()
	if err != nil {
		return errors.Wrapf(err, "GetProvide")
	}
	if len(self.ExternalId) > 0 {
		policies, err := provider.GetICustomCloudpolicies()
		if err != nil {
			return errors.Wrapf(err, "GetICustomCloudpolicies")
		}
		for i := range policies {
			if policies[i].GetGlobalId() == self.ExternalId {
				return nil
			}
		}
	}
	policy, err := self.GetCloudpolicy()
	if err != nil {
		return errors.Wrapf(err, "GetCloudpolicy")
	}
	if policy.Document == nil {
		return fmt.Errorf("nil document for custom policy %s(%s)", policy.Name, policy.Provider)
	}
	opts := cloudprovider.SCloudpolicyCreateOptions{
		Name:     policy.Name,
		Desc:     policy.Description,
		Document: policy.Document,
	}
	iPolicy, err := provider.CreateICloudpolicy(&opts)
	if err != nil {
		return errors.Wrap(err, "CreateICloudpolicy")
	}
	_, err = db.Update(self, func() error {
		self.ExternalId = iPolicy.GetGlobalId()
		self.Name = iPolicy.GetName()
		return nil
	})
	return err
}
