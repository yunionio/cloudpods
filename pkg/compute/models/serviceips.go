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

	"yunion.io/x/onecloud/pkg/cloudprovider"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/compare"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/sqlchemy"
)

type SServiceIpManager struct {
	db.SResourceBaseManager
}

var ServiceIpManager *SServiceIpManager

func init() {
	ServiceIpManager = &SServiceIpManager{
		SResourceBaseManager: db.NewResourceBaseManager(
			SServiceIp{},
			"serviceips_tbl",
			"serviceip",
			"serviceips",
		),
	}
}

func (model *SServiceIp) BeforeInsert() {
	if len(model.Id) == 0 {
		model.Id = db.DefaultUUIDGenerator()
	}
}

type SServiceIp struct {
	db.SResourceBase
	Id          string `width:"128" charset:"ascii" primary:"true" list:"user" get:"user"`
	ServiceId   string `width:"36" charset:"ascii" nullable:"false" list:"user" `
	IpAddr      string `width:"16" charset:"ascii" nullable:"false" list:"user"`
	URL         string `width:"256" charset:"ascii" nullable:"true" list:"user"`
	ServiceType string `width:"16" charset:"ascii" nullable:"false" list:"user"`
}

func (manager *SServiceIpManager) AllowListItems(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return true
}

func (manager *SServiceIpManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	return manager.SResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
}

func (manager *SServiceIpManager) GetServiceIps(resource db.IModel) ([]SServiceIp, error) {
	q := manager.Query().Equals("service_id", resource.GetId()).Equals("service_type", resource.Keyword())
	ips := []SServiceIp{}
	err := db.FetchModelObjects(manager, q, &ips)
	if err != nil {
		return nil, errors.Wrapf(err, "GetServiceIps.FetchModelObjects")
	}
	return ips, nil
}

func (manager *SServiceIpManager) SyncExtraIps(ctx context.Context, userCred mcclient.TokenCredential, resource db.IModel, ips []cloudprovider.SExtraIp) compare.SyncResult {
	result := compare.SyncResult{}
	dbServiceIps, err := manager.GetServiceIps(resource)
	if err != nil {
		result.Error(err)
		return result
	}
	localIps := map[string]*SServiceIp{}
	for i := 0; i < len(dbServiceIps); i++ {
		localIps[dbServiceIps[i].IpAddr] = &dbServiceIps[i]
	}
	remoteIps := map[string]cloudprovider.SExtraIp{}
	for i := 0; i < len(ips); i++ {
		remoteIps[ips[i].IP] = ips[i]
	}

	for _, ip := range ips {
		if _, exist := localIps[ip.IP]; !exist {
			err = manager.newServiceIp(ctx, userCred, resource, ip)
			if err != nil {
				result.AddError(err)
			} else {
				result.Add()
			}
		}
	}

	for ip, serviceIp := range localIps {
		if _, exist := remoteIps[ip]; !exist {
			err := serviceIp.Delete(ctx, userCred)
			if err != nil {
				result.DeleteError(err)
			} else {
				result.Delete()
			}
		}
	}

	return result
}

func (manager *SServiceIpManager) newServiceIp(ctx context.Context, userCred mcclient.TokenCredential, resource db.IModel, ip cloudprovider.SExtraIp) error {
	lockman.LockClass(ctx, manager, db.GetLockClassKey(manager, userCred))
	defer lockman.ReleaseClass(ctx, manager, db.GetLockClassKey(manager, userCred))

	serviceIp := SServiceIp{}
	serviceIp.SetModelManager(manager, &serviceIp)

	serviceIp.ServiceId = resource.GetId()
	serviceIp.ServiceType = resource.Keyword()
	serviceIp.IpAddr = ip.IP
	serviceIp.URL = ip.URL

	err := manager.TableSpec().Insert(&serviceIp)
	if err != nil {
		return errors.Wrapf(err, "newServiceIp.Insert")
	}
	return nil
}
