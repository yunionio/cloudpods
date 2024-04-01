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

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudid/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/informer"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

// +onecloud:swagger-gen-ignore
type SCloudproviderManager struct {
	db.SStandaloneResourceBaseManager
}

var CloudproviderManager *SCloudproviderManager

func init() {
	CloudproviderManager = &SCloudproviderManager{
		SStandaloneResourceBaseManager: db.NewStandaloneResourceBaseManager(
			SCloudprovider{},
			"cloudproviders_tbl",
			"cloudprovider",
			"cloudproviders",
		),
	}
	CloudproviderManager.SetVirtualObject(CloudproviderManager)
}

type SCloudprovider struct {
	db.SStandaloneResourceBase

	Provider       string `width:"64" charset:"ascii" list:"domain"`
	CloudaccountId string `width:"36" charset:"ascii" nullable:"false" list:"user"`
}

func (m *SCloudproviderManager) FetchProvier(id string) (*SCloudprovider, error) {
	ret, err := m.FetchById(id)
	if err != nil {
		return nil, err
	}
	return ret.(*SCloudprovider), nil
}

func (self *SCloudprovider) GetRole(ctx context.Context, userId string) (*SCloudrole, error) {
	groups := CloudgroupManager.Query().Equals("manager_id", self.Id).SubQuery()
	sq := SamluserManager.Query("cloudrole_id").Equals("owner_id", userId)
	sq = sq.Join(groups, sqlchemy.Equals(groups.Field("id"), sq.Field("cloudgroup_id")))
	ret := &SCloudrole{}
	ret.SetModelManager(CloudroleManager, ret)
	err := CloudroleManager.Query().In("id", sq.SubQuery()).IsNotEmpty("external_id").First(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SCloudprovider) GetSamlUser(userId string) (*SSamluser, error) {
	groups := CloudgroupManager.Query("id").Equals("manager_id", self.Id).SubQuery()
	q := SamluserManager.Query().Equals("owner_id", userId).In("cloudgroup_id", groups)
	ret := &SSamluser{}
	ret.SetModelManager(SamluserManager, ret)
	err := q.First(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SCloudprovider) GetSamlProvider() (*SSAMLProvider, error) {
	q := SAMLProviderManager.Query().Equals("status", apis.STATUS_AVAILABLE).
		Equals("entity_id", options.Options.ApiServer).Equals("manager_id", self.Id).IsNotEmpty("external_id")
	ret := &SSAMLProvider{}
	ret.SetModelManager(SAMLProviderManager, ret)
	err := q.First(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (self *SCloudprovider) GetProvider() (cloudprovider.ICloudProvider, error) {
	ctx := context.Background()
	s := auth.GetAdminSession(ctx, options.Options.Region)
	return modules.Cloudproviders.GetProvider(ctx, s, self.Id)
}

func (self *SCloudprovider) GetCloudaccount() (*SCloudaccount, error) {
	account, err := CloudaccountManager.FetchById(self.CloudaccountId)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchById(%s)", self.CloudaccountId)
	}
	return account.(*SCloudaccount), nil
}

func (self *SCloudprovider) GetDriver() (IProviderDriver, error) {
	return GetProviderDriver(self.Provider)
}

func (self *SCloudprovider) StartCloudproviderSyncResourcesTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "CloudproviderSyncResourcesTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (manager *SCloudproviderManager) GetCloudproviders() ([]SCloudprovider, error) {
	ret := []SCloudprovider{}
	q := manager.Query()
	err := db.FetchModelObjects(manager, q, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return ret, nil
}

func (manager *SCloudproviderManager) SyncCloudproviderResources(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	managers, err := manager.GetCloudproviders()
	if err != nil {
		log.Errorf("GetCloudproviders error: %v", err)
		return
	}
	for i := range managers {
		err = managers[i].StartCloudproviderSyncResourcesTask(ctx, userCred, "")
		if err != nil {
			log.Errorf("StartCloudproviderSyncResourcesTask for manager %s(%s) error: %v", managers[i].Name, managers[i].Provider, err)
		}
	}
}

func (m *SCloudproviderManager) StartWatchInRegion() error {
	adminSession := auth.GetAdminSession(context.Background(), "")
	watchMan, err := informer.NewWatchManagerBySession(adminSession)
	if err != nil {
		return err
	}
	resMan := &modules.Cloudproviders
	return watchMan.For(resMan).AddEventHandler(context.Background(), m)
}

func (m *SCloudproviderManager) OnAdd(obj *jsonutils.JSONDict) {
	model := &SCloudprovider{}
	model.SetModelManager(m, model)
	ctx := context.Background()
	err := obj.Unmarshal(model)
	if err != nil {
		return
	}
	if len(model.Provider) == 0 || len(model.CloudaccountId) == 0 {
		s := auth.GetAdminSession(ctx, options.Options.Region)
		data, err := modules.Cloudproviders.GetById(s, model.Id, nil)
		if err != nil {
			return
		}
		err = data.Unmarshal(model)
		if err != nil {
			return
		}
	}

	err = m.TableSpec().InsertOrUpdate(ctx, model)
	if err != nil {
		return
	}

	model.StartCloudproviderSyncResourcesTask(ctx, auth.AdminCredential(), "")
}

func (self *SCloudprovider) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return self.purge(ctx)
}

func (m *SCloudproviderManager) OnDelete(obj *jsonutils.JSONDict) {
	provider, err := func() (*SCloudprovider, error) {
		id, err := obj.GetString("id")
		if err != nil {
			return nil, errors.Wrapf(err, "get id")
		}
		provider, err := m.FetchById(id)
		if err != nil {
			return nil, errors.Wrapf(err, "fetch by id %s", id)
		}
		return provider.(*SCloudprovider), nil
	}()
	if err != nil {
		log.Errorf("fetch manager error: %v", err)
		return
	}
	err = provider.Delete(context.Background(), nil)
	if err != nil {
		log.Errorf("purge account error: %v", err)
		return
	}
}

func (m *SCloudproviderManager) OnUpdate(oldObj, newObj *jsonutils.JSONDict) {
	info := struct {
		Id   string
		Name string
	}{}
	newObj.Unmarshal(&info)
	log.Debugf("OnUpdate %s", jsonutils.Marshal(info).String())
	_, err := m.FetchProvier(info.Id)
	if err != nil && errors.Cause(err) == sql.ErrNoRows {
		m.OnAdd(newObj)
		return
	}
}
