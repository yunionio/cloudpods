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
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/cloudevent"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCloudeventManager struct {
	db.SModelBaseManager
}

var CloudeventManager *SCloudeventManager

func init() {
	CloudeventManager = &SCloudeventManager{
		SModelBaseManager: db.NewModelBaseManager(
			SCloudevent{},
			"cloudevents_tbl",
			"cloudevent",
			"cloudevents",
		),
	}
	CloudeventManager.SetVirtualObject(CloudeventManager)
}

type SCloudevent struct {
	db.SModelBase

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

func (manager *SCloudeventManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.CloudeventListInput) (*sqlchemy.SQuery, error) {
	q, err := manager.SModelBaseManager.ListItemFilter(ctx, q, userCred, input.JSON(input))
	if err != nil {
		return nil, err
	}

	if len(input.Providers) > 0 {
		q = q.In("provider", input.Providers)
	}

	if !input.Since.IsZero() {
		q = q.GT("created_at", input.Since)
	}
	if !input.Until.IsZero() {
		q = q.LE("created_at", input.Until)
	}

	return q, nil
}

func (self *SCloudevent) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	return self.SModelBase.GetCustomizeColumns(ctx, userCred, query)
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
			CloudproviderId: cloudprovider.Id,
		}

		event.CreatedAt = iEvent.GetCreatedAt()
		event.SetModelManager(manager, event)
		err := manager.TableSpec().Insert(event)
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
		MarkerField:  "created_at",
		DefaultLimit: 20,
	}
}
