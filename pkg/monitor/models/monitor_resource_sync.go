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

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo/hostconsts"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	mc_mds "yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	resourceSyncMap   map[string]IResourceSync
	guestResourceSync IResourceSync
	hostResourceSync  IResourceSync
)

func RegistryResourceSync(sync IResourceSync) error {
	if resourceSyncMap == nil {
		resourceSyncMap = make(map[string]IResourceSync)
	}
	if _, ok := resourceSyncMap[sync.SyncType()]; ok {
		return errors.Errorf(fmt.Sprintf("syncType:%s has registered", sync.SyncType()))
	}
	resourceSyncMap[sync.SyncType()] = sync
	return nil
}

func GetResourceSyncByType(syncType string) IResourceSync {
	if resourceSyncMap == nil {
		resourceSyncMap = make(map[string]IResourceSync)
	}
	return resourceSyncMap[syncType]
}

func GetResourceSyncMap() map[string]IResourceSync {
	if resourceSyncMap == nil {
		resourceSyncMap = make(map[string]IResourceSync)
	}
	return resourceSyncMap
}

type SyncObject struct {
	sync IResourceSync
}

type IResourceSync interface {
	SyncResources(ctx context.Context, userCred mcclient.TokenCredential, param jsonutils.JSONObject) error
	SyncType() string
}

type GuestResourceSync struct {
	SyncObject
}

func NewGuestResourceSync() IResourceSync {
	if guestResourceSync == nil {
		sync := new(GuestResourceSync)
		obj := newSyncObj(sync)
		sync.SyncObject = obj
		guestResourceSync = sync
	}

	return guestResourceSync
}

func (g *GuestResourceSync) SyncType() string {
	return monitor.METRIC_RES_TYPE_GUEST
}

type HostResourceSync struct {
	SyncObject
}

func (self *HostResourceSync) SyncType() string {
	return monitor.METRIC_RES_TYPE_HOST
}

func NewHostResourceSync() IResourceSync {
	if hostResourceSync == nil {
		sync := new(HostResourceSync)
		obj := newSyncObj(sync)
		sync.SyncObject = obj
		hostResourceSync = sync
	}
	return hostResourceSync
}

func newSyncObj(sync IResourceSync) SyncObject {
	return SyncObject{sync: sync}
}

func (self *SyncObject) SyncResources(ctx context.Context, userCred mcclient.TokenCredential,
	param jsonutils.JSONObject) error {
	log.Errorf("start sync %s", self.sync.SyncType())
	resources, err := GetOnecloudResources(self.sync.SyncType())
	if err != nil {
		return errors.Wrapf(err, fmt.Sprintf("syncType:%s GetOnecloudResources err", self.sync.SyncType()))
	}
	input := monitor.MonitorResourceListInput{
		OnlyResId: true,
		ResType:   self.sync.SyncType(),
	}
	monResources, err := MonitorResourceManager.GetMonitorResources(input)
	if err != nil {
		return errors.Wrap(err, "GetMonitorResources err")
	}
	errs := make([]error, 0)
monLoop:
	for i, _ := range monResources {
		for index, res := range resources {
			resId, _ := res.GetString("id")
			if resId == monResources[i].ResId {
				resource, err := MonitorResourceManager.GetMonitorResourceById(monResources[i].GetId())
				if err != nil {
					errs = append(errs, err)
					continue monLoop
				}
				_, err = db.Update(resource, func() error {
					res.(*jsonutils.JSONDict).Remove("id")
					res.Unmarshal(resource)
					return nil
				})
				if err != nil {
					errs = append(errs, errors.Wrapf(err, "monitorResource:%s Update err", resource.Name))
					continue monLoop
				}
				resource.UpdateAlertState()
				if index == len(resources)-1 {
					resources = resources[0:index]
				} else {
					resources = append(resources[0:index], resources[index+1:]...)
				}

				index--
				continue monLoop
			}
		}
		resource, _ := MonitorResourceManager.GetMonitorResourceById(monResources[i].GetId())
		err := resource.RealDelete(ctx, userCred)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "delete monitorResource:%s err", resource.GetId()))
		}
	}
	for _, res := range resources {
		createData := self.newMonitorResourceCreateInput(res)
		_, err = db.DoCreate(MonitorResourceManager, ctx, userCred, nil, createData,
			userCred)
		if err != nil {
			name, _ := createData.GetString("name")
			errs = append(errs, errors.Wrapf(err, "monitorResource:%s resType:%s DoCreate err", name, self.sync.SyncType()))
		}
	}
	return errors.NewAggregate(errs)
}

func (self *SyncObject) newMonitorResourceCreateInput(input jsonutils.JSONObject) jsonutils.JSONObject {
	monitorResource := jsonutils.DeepCopy(input).(*jsonutils.JSONDict)
	id, _ := monitorResource.GetString("id")
	monitorResource.Add(jsonutils.NewString(id), "res_id")
	monitorResource.Remove("id")
	monitorResource.Add(jsonutils.NewString(self.sync.SyncType()), "res_type")

	return monitorResource
}

func GetOnecloudResources(resTyep string) ([]jsonutils.JSONObject, error) {
	var err error
	allResources := make([]jsonutils.JSONObject, 0)

	query := jsonutils.NewDict()
	query.Add(jsonutils.NewStringArray([]string{"running", "ready"}), "status")
	query.Add(jsonutils.NewString("true"), "admin")
	switch resTyep {
	case monitor.METRIC_RES_TYPE_HOST:
		//query.Set("host-type", jsonutils.NewString(hostconsts.TELEGRAF_TAG_KEY_HYPERVISOR))
		allResources, err = ListAllResources(&mc_mds.Hosts, query)
	case monitor.METRIC_RES_TYPE_GUEST:
		allResources, err = ListAllResources(&mc_mds.Servers, query)
	case monitor.METRIC_RES_TYPE_AGENT:
		allResources, err = ListAllResources(&mc_mds.Servers, query)
	case monitor.METRIC_RES_TYPE_RDS:
		allResources, err = ListAllResources(&mc_mds.DBInstance, query)
	case monitor.METRIC_RES_TYPE_REDIS:
		allResources, err = ListAllResources(&mc_mds.ElasticCache, query)
	case monitor.METRIC_RES_TYPE_OSS:
		allResources, err = ListAllResources(&mc_mds.Buckets, query)
	case monitor.METRIC_RES_TYPE_CLOUDACCOUNT:
		query.Remove("status")
		query.Add(jsonutils.NewBool(true), "enabled")
		allResources, err = ListAllResources(&mc_mds.Cloudaccounts, query)
	case monitor.METRIC_RES_TYPE_TENANT:
		allResources, err = ListAllResources(&mc_mds.Projects, query)
	case monitor.METRIC_RES_TYPE_DOMAIN:
		allResources, err = ListAllResources(&mc_mds.Domains, query)
	case monitor.METRIC_RES_TYPE_STORAGE:
		query.Remove("status")
		allResources, err = ListAllResources(&mc_mds.Storages, query)
	default:
		query := jsonutils.NewDict()
		query.Set("brand", jsonutils.NewString(hostconsts.TELEGRAF_TAG_ONECLOUD_BRAND))
		query.Set("host-type", jsonutils.NewString(hostconsts.TELEGRAF_TAG_KEY_HYPERVISOR))
		allResources, err = ListAllResources(&mc_mds.Hosts, query)
	}

	if err != nil {
		return nil, errors.Wrap(err, "NoDataQueryCondition Host list error")
	}
	return allResources, nil
}

func ListAllResources(manager modulebase.Manager, params *jsonutils.JSONDict) ([]jsonutils.JSONObject, error) {
	if params == nil {
		params = jsonutils.NewDict()
	}
	params.Add(jsonutils.NewString("system"), "scope")
	params.Add(jsonutils.NewInt(0), "limit")
	params.Add(jsonutils.NewBool(true), "details")
	var count int
	session := auth.GetAdminSession(context.Background(), "", "")
	objs := make([]jsonutils.JSONObject, 0)
	for {
		params.Set("offset", jsonutils.NewInt(int64(count)))
		result, err := manager.List(session, params)
		if err != nil {
			return nil, errors.Wrapf(err, "list %s resources with params %s", manager.KeyString(), params.String())
		}
		for _, data := range result.Data {
			objs = append(objs, data)
		}
		total := result.Total
		count = count + len(result.Data)
		if count >= total {
			break
		}
	}
	return objs, nil
}
