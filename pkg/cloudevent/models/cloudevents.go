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

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudevent/options"
	"yunion.io/x/onecloud/pkg/cloudprovider"
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
