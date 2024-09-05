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
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"

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
type SCloudaccountManager struct {
	db.SDomainLevelResourceBaseManager
}

var CloudaccountManager *SCloudaccountManager

func init() {
	CloudaccountManager = &SCloudaccountManager{
		SDomainLevelResourceBaseManager: db.NewDomainLevelResourceBaseManager(
			SCloudaccount{},
			"cloudaccounts_tbl",
			"cloudaccount",
			"cloudaccounts",
		),
	}
	CloudaccountManager.SetVirtualObject(CloudaccountManager)
}

type SCloudaccount struct {
	db.SStandaloneResourceBase
	db.SDomainizedResourceBase

	AccountId   string            `width:"128" charset:"utf8" nullable:"true" list:"domain" create:"domain_optional"`
	Provider    string            `width:"64" charset:"ascii" list:"domain"`
	Brand       string            `width:"64" charset:"utf8" nullable:"true" list:"domain"`
	IamLoginUrl string            `width:"512" charset:"ascii"`
	SAMLAuth    tristate.TriState `list:"domain" default:"false"`

	AccessUrl string `width:"64" charset:"ascii" nullable:"true" list:"domain" update:"domain" create:"domain_optional"`

	ReadOnly bool `default:"false" create:"domain_optional" list:"domain" update:"domain"`
}

func (manager *SCloudaccountManager) GetResourceCount() ([]db.SScopeResourceCount, error) {
	return []db.SScopeResourceCount{}, nil
}

func (manager *SCloudaccountManager) GetCloudaccounts() ([]SCloudaccount, error) {
	accounts := []SCloudaccount{}
	q := manager.Query()
	err := db.FetchModelObjects(manager, q, &accounts)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func (self *SCloudaccount) GetCloudproviderId() string {
	return ""
}

func (self *SCloudaccount) GetSamlusers() ([]SSamluser, error) {
	q := SamluserManager.Query().Equals("cloudaccount_id", self.Id)
	users := []SSamluser{}
	err := db.FetchModelObjects(SamluserManager, q, &users)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return users, nil
}

func (self *SCloudaccount) GetProvider() (cloudprovider.ICloudProvider, error) {
	ctx := context.Background()
	s := auth.GetAdminSession(ctx, options.Options.Region)
	return modules.Cloudaccounts.GetProvider(ctx, s, self.Id)
}

func (self *SCloudaccount) GetDriver() (IProviderDriver, error) {
	return GetProviderDriver(self.Provider)
}

func (self *SCloudaccount) GetCloudpolicies(managerId string) ([]SCloudpolicy, error) {
	q := CloudpolicyManager.Query().Equals("cloudaccount_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	policies := []SCloudpolicy{}
	err := db.FetchModelObjects(CloudpolicyManager, q, &policies)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return policies, nil
}

func (self *SCloudaccount) GetCloudproviders() ([]SCloudprovider, error) {
	q := CloudproviderManager.Query().Equals("cloudaccount_id", self.Id)
	providers := []SCloudprovider{}
	err := db.FetchModelObjects(CloudproviderManager, q, &providers)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return providers, nil
}

func (self *SCloudaccount) GetCloudgroups(managerId string) ([]SCloudgroup, error) {
	groups := []SCloudgroup{}
	q := CloudgroupManager.Query().Equals("cloudaccount_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	err := db.FetchModelObjects(CloudgroupManager, q, &groups)
	if err != nil {
		return nil, errors.Wrap(err, "db.FetchModelObjects")
	}
	return groups, nil
}

func (manager *SCloudaccountManager) SyncCloudaccountResources(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	accounts, err := manager.GetCloudaccounts()
	if err != nil {
		log.Errorf("GetCloudaccounts error: %v", err)
		return
	}
	for i := range accounts {
		err = accounts[i].StartSyncCloudaccountResourcesTask(ctx, userCred, "")
		if err != nil {
			log.Errorf("StartCloudaccountSyncResourcesTask for account %s(%s) error: %v", accounts[i].Name, accounts[i].Provider, err)
		}
	}
}

func (self *SCloudaccount) IsSAMLProviderValid() (*SSAMLProvider, bool) {
	provider, err := self.RegisterSAMProvider()
	if err != nil {
		return provider, false
	}
	if len(provider.ExternalId) == 0 {
		return provider, false
	}
	return provider, true
}

func (self *SCloudaccount) RegisterSAMProvider() (*SSAMLProvider, error) {
	if len(options.Options.ApiServer) == 0 {
		return nil, fmt.Errorf("empty api server")
	}
	sps, err := self.GetSAMLProviders("")
	if err != nil {
		return nil, errors.Wrapf(err, "GetSAMLProviders")
	}
	for i := range sps {
		if sps[i].EntityId == options.Options.ApiServer {
			return &sps[i], nil
		}
	}
	sp := &SSAMLProvider{}
	sp.SetModelManager(SAMLProviderManager, sp)
	sp.Name = func() string {
		name := strings.TrimPrefix(options.Options.ApiServer, "https://")
		name = strings.TrimPrefix(name, "http://")
		return name
	}()
	sp.EntityId = options.Options.ApiServer
	sp.CloudaccountId = self.Id
	sp.DomainId = self.DomainId
	sp.Status = apis.STATUS_CREATING
	metadata := SamlIdpInstance().GetMetadata(self.Id).String()
	sp.MetadataDocument = metadata
	err = SAMLProviderManager.TableSpec().Insert(context.TODO(), sp)
	if err != nil {
		return nil, errors.Wrapf(err, "Insert")
	}
	return sp, nil
}

func (self *SCloudaccount) StartSAMLProviderCreateTask(ctx context.Context, userCred mcclient.TokenCredential) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "SAMLProviderCreateTask", self, userCred, params, "", "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (self *SCloudaccount) GetSAMLProviders(managerId string) ([]SSAMLProvider, error) {
	q := SAMLProviderManager.Query().Equals("cloudaccount_id", self.Id).Desc("external_id")
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	samls := []SSAMLProvider{}
	err := db.FetchModelObjects(SAMLProviderManager, q, &samls)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return samls, nil
}

func (self *SCloudaccount) StartSyncCloudaccountResourcesTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	params := jsonutils.NewDict()
	task, err := taskman.TaskManager.NewTask(ctx, "CloudaccountSyncResourcesTask", self, userCred, params, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (self *SCloudaccount) GetCloudroles(managerId string) ([]SCloudrole, error) {
	roles := []SCloudrole{}
	q := CloudroleManager.Query().Equals("cloudaccount_id", self.Id)
	if len(managerId) > 0 {
		q = q.Equals("manager_id", managerId)
	}
	err := db.FetchModelObjects(CloudroleManager, q, &roles)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return roles, nil
}

func (m *SCloudaccountManager) StartWatchSAMLInRegion() error {
	adminSession := auth.GetAdminSession(context.Background(), "")
	watchMan, err := informer.NewWatchManagerBySession(adminSession)
	if err != nil {
		return err
	}
	resMan := &modules.Cloudaccounts
	return watchMan.For(resMan).AddEventHandler(context.Background(), m)
}

func (m *SCloudaccountManager) OnAdd(obj *jsonutils.JSONDict) {
	account := &SCloudaccount{}
	obj.Unmarshal(account)
	log.Debugf("OnAdd provider %s %s(%s)", account.Provider, account.Name, account.Id)
	account.SetModelManager(m, account)
	ctx := context.Background()
	if len(account.Provider) == 0 {
		s := auth.GetAdminSession(ctx, options.Options.Region)
		data, err := modules.Cloudaccounts.GetById(s, account.Id, nil)
		if err != nil {
			return
		}
		err = data.Unmarshal(account)
		if err != nil {
			return
		}
	}
	err := m.TableSpec().Insert(ctx, account)
	if err != nil {
		return
	}
	account.StartSyncCloudaccountResourcesTask(ctx, auth.AdminCredential(), "")
}

func (m *SCloudaccountManager) FetchAccount(id string) (*SCloudaccount, error) {
	account, err := m.FetchById(id)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch by id %s", id)
	}
	return account.(*SCloudaccount), nil
}

func (self *SCloudaccount) GetSamlProvider() (*SSAMLProvider, error) {
	q := SAMLProviderManager.Query().Equals("status", apis.STATUS_AVAILABLE).
		Equals("entity_id", options.Options.ApiServer).Equals("cloudaccount_id", self.Id).IsNotEmpty("external_id")
	ret := &SSAMLProvider{}
	ret.SetModelManager(SAMLProviderManager, ret)
	err := q.First(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (m *SCloudaccountManager) OnDelete(obj *jsonutils.JSONDict) {
	log.Debugf("OnDelete %s", obj)
	account, err := func() (*SCloudaccount, error) {
		id, err := obj.GetString("id")
		if err != nil {
			return nil, errors.Wrapf(err, "get id")
		}
		return m.FetchAccount(id)
	}()
	if err != nil {
		log.Errorf("fetch account error: %v", err)
		return
	}
	err = account.purge(context.Background())
	if err != nil {
		log.Errorf("purge account error: %v", err)
		return
	}
}

func (m *SCloudaccountManager) OnUpdate(oldObj, newObj *jsonutils.JSONDict) {
	id, err := newObj.GetString("id")
	if err != nil {
		return
	}
	account, err := m.FetchAccount(id)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			m.OnAdd(newObj)
			return
		}
		return
	}
	db.Update(account, func() error {
		newObj.Unmarshal(account)
		return nil
	})
	oSaml, _ := oldObj.Bool("saml_auth")
	nSaml, _ := newObj.Bool("saml_auth")
	if !oSaml && nSaml {
		log.Debugf("OnUpdate provider %s %s(%s) enable saml auth", account.Provider, account.Name, account.Id)
		account.StartSAMLProviderCreateTask(context.Background(), auth.AdminCredential())
	}
}
