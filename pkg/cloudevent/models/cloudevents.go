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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/cloudevent"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SCloudeventManager struct {
	db.SModelBaseManager
	db.SDomainizedResourceBaseManager
}

var CloudeventManager *SCloudeventManager

func init() {
	CloudeventManager = &SCloudeventManager{
		SModelBaseManager: db.NewModelBaseManagerWithSplitable(
			SCloudevent{},
			"cloudevents_tbl",
			"cloudevent",
			"cloudevents",
			"event_id",
			"created_at",
			consts.SplitableMaxDuration(),
			consts.SplitableMaxKeepSegments(),
		),
	}
	CloudeventManager.SetVirtualObject(CloudeventManager)
}

type SCloudevent struct {
	db.SModelBase
	db.SDomainizedResourceBase

	EventId      int64                `primary:"true" auto_increment:"true" list:"user"`
	Name         string               `width:"128" charset:"utf8" nullable:"false" index:"true" list:"user"`
	Service      string               `width:"64" charset:"utf8" nullable:"true" list:"user"`
	ResourceType string               `width:"64" charset:"utf8" nullable:"true" list:"user"`
	Action       string               `width:"64" charset:"utf8" nullable:"true" list:"user"`
	RequestId    string               `width:"128" charset:"utf8" nullable:"true" list:"user"`
	Request      jsonutils.JSONObject `charset:"utf8" nullable:"true" list:"user"`
	Account      string               `width:"64" charset:"utf8" nullable:"true" list:"user"`
	Success      bool                 `nullable:"false" list:"user"`
	CreatedAt    time.Time            `nullable:"false" created_at:"true" index:"true" get:"user" list:"user"`

	CloudproviderId string `width:"64" charset:"utf8" nullable:"true" list:"user"`
	Manager         string `width:"128" charset:"utf8" nullable:"false" index:"true" list:"user"`
	Provider        string `width:"64" charset:"ascii" nullable:"false" list:"user"`
	Brand           string `width:"64" charset:"ascii" list:"domain"`
}

func (self *SCloudeventManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsDomainAllowList(userCred, self)
}

func (self *SCloudeventManager) AllowCreateItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (self *SCloudevent) AllowDeleteItem(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return false
}

func (self *SCloudevent) AllowUpdateItem(ctx context.Context, userCred mcclient.TokenCredential) bool {
	return false
}

func (self *SCloudevent) AllowGetDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return db.IsDomainAllowGet(userCred, self)
}

// 云平台操作日志列表
func (manager *SCloudeventManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	input api.CloudeventListInput,
) (*sqlchemy.SQuery, error) {
	q, err := manager.SModelBaseManager.ListItemFilter(ctx, q, userCred, input.ModelBaseListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SModelBaseManager.ListItemFilter")
	}
	q, err = manager.SDomainizedResourceBaseManager.ListItemFilter(ctx, q, userCred, input.DomainizedResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SDomainizedResourceBaseManager.ListItemFilter")
	}

	if len(input.Providers) > 0 {
		q = q.In("provider", input.Providers)
	}

	if len(input.Brands) > 0 {
		q = q.In("brand", input.Brands)
	}

	if len(input.Service) > 0 {
		q = q.In("service", input.Service)
	}

	if len(input.Manager) > 0 {
		q = q.In("manager", input.Manager)
	}

	if len(input.Account) > 0 {
		q = q.In("account", input.Account)
	}

	if len(input.Action) > 0 {
		q = q.In("action", input.Action)
	}

	if len(input.ResourceType) > 0 {
		q = q.In("resource_type", input.ResourceType)
	}

	if input.Success != nil {
		q = q.Equals("success", *input.Success)
	}

	if !input.Since.IsZero() {
		q = q.GT("created_at", input.Since)
	}
	if !input.Until.IsZero() {
		q = q.LE("created_at", input.Until)
	}

	return q, nil
}

func (manager *SCloudeventManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.CloudeventDetails {
	rows := make([]api.CloudeventDetails, len(objs))
	base := manager.SModelBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	domainRows := manager.SDomainizedResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	for i := range rows {
		rows[i].ModelBaseDetails = base[i]
		rows[i].DomainizedResourceInfo = domainRows[i]
	}
	return rows
}

func (self *SCloudevent) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return self.SModelBase.CustomizeCreate(ctx, userCred, ownerId, query, data)
}

func (manager *SCloudeventManager) NamespaceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeDomain
}

func (manager *SCloudeventManager) ResourceScope() rbacutils.TRbacScope {
	return rbacutils.ScopeDomain
}

func (self *SCloudevent) GetOwnerId() mcclient.IIdentityProvider {
	owner := db.SOwnerId{DomainId: self.DomainId}
	return &owner
}

func (manager *SCloudeventManager) FilterByOwner(q *sqlchemy.SQuery, owner mcclient.IIdentityProvider, scope rbacutils.TRbacScope) *sqlchemy.SQuery {
	return manager.SDomainizedResourceBaseManager.FilterByOwner(q, owner, scope)
}

func (manager *SCloudeventManager) FetchOwnerId(ctx context.Context, data jsonutils.JSONObject) (mcclient.IIdentityProvider, error) {
	return manager.SDomainizedResourceBaseManager.FetchOwnerId(ctx, data)
}

func (manager *SCloudeventManager) ListItemExportKeys(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, keys stringutils2.SSortedStrings) (*sqlchemy.SQuery, error) {
	return manager.SDomainizedResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
}

func (manager *SCloudeventManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	return manager.SDomainizedResourceBaseManager.QueryDistinctExtraField(q, field)
}

func (manager *SCloudeventManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.CloudeventListInput,
) (*sqlchemy.SQuery, error) {
	return manager.SDomainizedResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.DomainizedResourceListInput)
}

func (manager *SCloudeventManager) SyncCloudevent(ctx context.Context, userCred mcclient.TokenCredential, cloudprovider *SCloudprovider, iEvents []cloudprovider.ICloudEvent) int {
	count := 0
	for _, iEvent := range iEvents {
		event := &SCloudevent{
			Name:            iEvent.GetName(),
			Service:         iEvent.GetService(),
			ResourceType:    iEvent.GetResourceType(),
			Action:          iEvent.GetAction(),
			Account:         iEvent.GetAccount(),
			RequestId:       iEvent.GetRequestId(),
			Request:         iEvent.GetRequest(),
			Success:         iEvent.IsSuccess(),
			Manager:         cloudprovider.Name,
			Provider:        cloudprovider.Provider,
			Brand:           cloudprovider.Brand,
			CloudproviderId: cloudprovider.Id,
		}
		event.DomainId = cloudprovider.DomainId
		if len(event.Brand) == 0 {
			event.Brand = event.Provider
		}

		for k, v := range map[string]string{
			"service":       event.Service,
			"resoruce_type": event.ResourceType,
			"action":        event.Action,
			"account":       event.Account,
			"manager":       event.Manager,
			"provider":      event.Provider,
			"brand":         event.Brand,
		} {
			db.DistinctFieldManager.InsertOrUpdate(ctx, manager, k, v)
		}

		event.CreatedAt = iEvent.GetCreatedAt()
		event.SetModelManager(manager, event)
		err := manager.TableSpec().Insert(ctx, event)
		if err != nil {
			log.Errorf("failed to insert event: %s for cloudprovider: %s(%s) error: %v", jsonutils.Marshal(event).PrettyString(), cloudprovider.Name, cloudprovider.Id, err)
			continue
		}
		count += 1
	}
	return count
}

func (manager *SCloudeventManager) GetPagingConfig() *db.SPagingConfig {
	return &db.SPagingConfig{
		Order:        sqlchemy.SQL_ORDER_DESC,
		MarkerFields: []string{"created_at", "event_id"},
		DefaultLimit: 20,
	}
}

func (manager *SCloudeventManager) GetPropertyDistinctField(ctx context.Context, userCred mcclient.TokenCredential, input apis.DistinctFieldInput) (jsonutils.JSONObject, error) {
	fields, err := db.DistinctFieldManager.GetObjectDistinctFields(manager.Keyword())
	if err != nil {
		return nil, errors.Wrapf(err, "DistinctFieldManager.GetObjectDistinctFields")
	}
	fieldMaps := map[string][]string{}
	for _, field := range fields {
		_, ok := fieldMaps[field.Key]
		if !ok {
			fieldMaps[field.Key] = []string{}
		}
		fieldMaps[field.Key] = append(fieldMaps[field.Key], field.Value)
	}
	ret := map[string][]string{}
	for _, key := range input.Field {
		ret[key], _ = fieldMaps[key]
	}
	return jsonutils.Marshal(ret), nil
}

func (manager *SCloudeventManager) initDistinctField() error {
	fileds, err := db.DistinctFieldManager.GetObjectDistinctFields(manager.Keyword())
	if err != nil {
		return errors.Wrapf(err, "GetObjectDistinctFields")
	}
	if len(fileds) > 0 {
		return nil
	}
	for _, key := range []string{"service", "resource_type", "action", "account", "manager", "provider", "brand"} {
		values, err := db.FetchDistinctField(manager, key)
		if err != nil {
			return errors.Wrapf(err, "db.FetchDistinctField")
		}
		for _, value := range values {
			if len(value) > 0 {
				err = db.DistinctFieldManager.InsertOrUpdate(nil, manager, key, value)
				if err != nil {
					return errors.Wrapf(err, "DistinctFieldManager.InsertOrUpdate(%s, %s)", key, value)
				}
			}
		}
	}
	return nil
}

func (manager *SCloudeventManager) InitializeData() error {
	events := []SCloudevent{}
	q := manager.Query().IsNullOrEmpty("brand")
	err := db.FetchModelObjects(manager, q, &events)
	if err != nil {
		return err
	}
	for i := range events {
		_, err = db.Update(&events[i], func() error {
			events[i].Brand = events[i].Provider
			return nil
		})
		if err != nil {
			return err
		}
	}
	return manager.initDistinctField()
}
