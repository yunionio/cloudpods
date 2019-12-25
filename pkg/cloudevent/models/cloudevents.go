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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/cloudevent"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudevent/options"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
)

type SCloudeventManager struct {
	db.SVirtualResourceBaseManager
}

var CloudeventManager *SCloudeventManager
var mods map[string]modulebase.Manager

func init() {
	CloudeventManager = &SCloudeventManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SCloudevent{},
			"cloudevents_tbl",
			"cloudevent",
			"cloudevents",
		),
	}
	CloudeventManager.SetVirtualObject(CloudeventManager)
}

type SCloudevent struct {
	db.SVirtualResourceBase

	Service      string               `width:"64" charset:"utf8" nullable:"true" list:"user"`
	ResourceType string               `width:"64" charset:"utf8" nullable:"true" list:"user"`
	Action       string               `width:"64" charset:"utf8" nullable:"true" list:"user"`
	RequestId    string               `width:"128" charset:"utf8" nullable:"true" list:"user"`
	Request      jsonutils.JSONObject `charset:"utf8" nullable:"true" list:"user"`
	Account      string               `width:"64" charset:"utf8" nullable:"true" list:"user"`
	Success      bool                 `nullable:"false" list:"user"`

	CloudproviderId string `width:"64" charset:"utf8" nullable:"true" list:"user"`
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
	q, err := manager.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.JSON(input))
	if err != nil {
		return nil, err
	}
	if len(input.Cloudprovider) > 0 {
		providerObj, err := CloudproviderManager.FetchByIdOrName(userCred, input.Cloudprovider)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(CloudproviderManager.Keyword(), input.Cloudprovider)
			} else {
				return nil, httperrors.NewGeneralError(err)
			}
		}
		q = q.Equals("cloudprovider_id", providerObj.GetId())
	}

	if len(input.Providers) > 0 {
		sq := CloudproviderManager.Query().SubQuery()
		q = q.Join(sq, sqlchemy.Equals(q.Field("cloudprovider_id"), sq.Field("id"))).
			Filter(sqlchemy.In(sq.Field("provider"), input.Providers))
	}
	//过滤已删除的cloudprovider日志
	sq := CloudproviderManager.Query("id").SubQuery()
	q = q.In("cloudprovider_id", sq)
	return q, nil
}

func (self *SCloudevent) GetCloudprovider() (*SCloudprovider, error) {
	cloudprovider, err := CloudproviderManager.FetchById(self.CloudproviderId)
	if err != nil {
		return nil, err
	}
	return cloudprovider.(*SCloudprovider), nil
}

func (self *SCloudevent) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := self.SStatusStandaloneResourceBase.GetCustomizeColumns(ctx, userCred, query)
	extra, _ = self.getMoreDetails(ctx, userCred, query, extra)
	return extra
}

func (self *SCloudevent) getMoreDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, extra *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	cloudprovider, err := self.GetCloudprovider()
	if err != nil {
		return nil, err
	}
	info := jsonutils.Marshal(map[string]string{
		"provider": cloudprovider.Provider,
		"manager":  cloudprovider.Name,
	})
	extra.Update(info)
	return extra, nil
}

func (self *SCloudevent) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*jsonutils.JSONDict, error) {
	extra, err := self.SVirtualResourceBase.GetExtraDetails(ctx, userCred, query)
	if err != nil {
		return nil, err
	}
	return self.getMoreDetails(ctx, userCred, query, extra)
}

func (self *SCloudeventManager) fetchMods(ctx context.Context, userCred mcclient.TokenCredential) {
	if len(mods) > 0 {
		return
	}
	s := auth.GetAdminSession(ctx, options.Options.Region, "v2")
	mods = map[string]modulebase.Manager{}
	rms, _ := modulebase.GetRegisterdModules()
	for _, _mods := range rms {
		for _, _mod := range _mods {
			if _, ok := mods[_mod]; !ok {
				mod, err := modulebase.GetModule(s, _mod)
				if err != nil {
					log.Errorf("failed to get mod %s error: %v", _mod, err)
					continue
				}
				mods[mod.GetKeyword()] = mod
			}
		}
	}
	return
}

func (manager *SCloudeventManager) setEventInfo(session *mcclient.ClientSession, mod modulebase.Manager, event *SCloudevent) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(fmt.Sprintf("external_id.equals(%s)", event.Name)), "filter")
	result, err := mod.List(session, params)
	if err != nil {
		return errors.Wrapf(err, "mod.List for %s by externalId: %s", mod.KeyString(), event.Name)
	}
	if len(result.Data) != 1 {
		return errors.Wrapf(err, "found %d %s by externalId: %s", len(result.Data), mod.KeyString(), event.Name)
	}

	data := struct {
		Name     string
		TenantId string
	}{}
	err = result.Data[0].Unmarshal(&data)
	if err != nil {
		return errors.Wrapf(err, "result.Data[0].Unmarshal %s", result.Data[0])
	}
	if len(data.Name) > 0 {
		event.Name = event.Name
	}
	if len(data.TenantId) > 0 {
		event.ProjectId = data.TenantId
	}
	return nil
}

func (manager *SCloudeventManager) SyncCloudevent(ctx context.Context, userCred mcclient.TokenCredential, cloudprovider *SCloudprovider, iEvents []cloudprovider.ICloudEvent) int {
	count := 0
	for _, iEvent := range iEvents {
		event := &SCloudevent{
			Service:         iEvent.GetService(),
			ResourceType:    iEvent.GetResourceType(),
			Action:          iEvent.GetAction(),
			Account:         iEvent.GetAccount(),
			RequestId:       iEvent.GetRequestId(),
			Request:         iEvent.GetRequest(),
			Success:         iEvent.IsSuccess(),
			CloudproviderId: cloudprovider.Id,
		}

		event.Name = iEvent.GetName()
		event.Status = "ready"
		event.ProjectId = userCred.GetProjectId()
		event.CreatedAt = iEvent.GetCreatedAt()
		event.ProjectId = userCred.GetProjectId()
		event.DomainId = userCred.GetDomainId()

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
