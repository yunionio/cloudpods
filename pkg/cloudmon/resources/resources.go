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

package resources

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/cloudmon/providerdriver"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/influxdb"
)

type sBaseInfo struct {
	Id         string
	ExternalId string
	ManagerId  string
	CreatedAt  time.Time
	ImportedAt time.Time
	DeletedAt  time.Time
	UpdatedAt  time.Time
}

type SBaseResources struct {
	manager modulebase.Manager

	importedAt time.Time
	createdAt  time.Time
	deletedAt  time.Time
	updatedAt  time.Time

	resourceLock sync.Mutex
	Resources    map[string]jsonutils.JSONObject

	providerLock      sync.Mutex
	ProviderResources map[string]map[string]jsonutils.JSONObject
}

func (self *SBaseResources) getResources(managerId string) map[string]jsonutils.JSONObject {
	ret := map[string]jsonutils.JSONObject{}
	if len(managerId) == 0 {
		return self.Resources
	}
	res, ok := self.ProviderResources[managerId]
	if ok {
		return res
	}
	return ret
}

func (self *SBaseResources) init() error {
	s := auth.GetAdminSession(context.Background(), options.Options.Region)
	query := map[string]interface{}{
		"limit":          20,
		"scope":          "system",
		"details":        true,
		"order_by.0":     "created_at",
		"order_by.1":     "imported_at",
		"order":          "asc",
		"pending_delete": "all",
		"filter.0":       "external_id.isnotempty()",
	}
	if self.manager.GetKeyword() == compute.Hosts.GetKeyword() { // private and vmware
		query["cloud_env"] = "private_or_onpremise"
	}
	offset := 0
	for {
		query["offset"] = offset
		resp, err := self.manager.List(s, jsonutils.Marshal(query))
		if err != nil {
			return errors.Wrapf(err, "%s.List", self.manager.GetKeyword())
		}
		offset += len(resp.Data)
		for i := range resp.Data {
			baseInfo := struct {
				Id         string
				ExternalId string
				ManagerId  string
				CreatedAt  time.Time
				ImportedAt time.Time
			}{}
			resp.Data[i].Unmarshal(&baseInfo)
			if len(baseInfo.ExternalId) == 0 && (self.manager.GetKeyword() != compute.Cloudproviders.GetKeyword() &&
				self.manager.GetKeyword() != compute.Cloudaccounts.GetKeyword()) {
				continue
			}
			key := baseInfo.ExternalId
			if len(key) == 0 {
				key = baseInfo.Id
			}
			self.resourceLock.Lock()
			self.Resources[key] = resp.Data[i]
			self.resourceLock.Unlock()
			if len(baseInfo.ManagerId) > 0 {
				if _, ok := self.ProviderResources[baseInfo.ManagerId]; !ok {
					self.ProviderResources[baseInfo.ManagerId] = map[string]jsonutils.JSONObject{}
				}
				self.providerLock.Lock()
				self.ProviderResources[baseInfo.ManagerId][key] = resp.Data[i]
				self.providerLock.Unlock()
			}
			if self.importedAt.IsZero() || self.importedAt.Before(baseInfo.ImportedAt) {
				self.importedAt = baseInfo.ImportedAt
			}
			if self.createdAt.IsZero() || self.createdAt.Before(baseInfo.CreatedAt) {
				self.createdAt = baseInfo.CreatedAt
			}
		}
		if offset >= resp.Total {
			break
		}
	}
	self.deletedAt = time.Now()
	self.updatedAt = time.Now()
	log.Infof("init %d %s importedAt: %s createdAt: %s", len(self.Resources), self.manager.GetKeyword(), self.importedAt, self.createdAt)
	return nil
}

func (self *SBaseResources) increment() error {
	s := auth.GetAdminSession(context.Background(), options.Options.Region)
	timeFilter := fmt.Sprintf("imported_at.gt('%s')", self.importedAt)
	if self.importedAt.IsZero() {
		timeFilter = fmt.Sprintf("created_at.gt('%s')", self.createdAt)
	}
	query := map[string]interface{}{
		"limit":      20,
		"scope":      "system",
		"details":    true,
		"order_by.0": "created_at",
		"order_by.1": "imported_at",
		"order":      "asc",
		"filter.0":   timeFilter,
		"filter.1":   "external_id.isnotempty()",
	}
	if self.manager.GetKeyword() == compute.Hosts.GetKeyword() {
		query["cloud_env"] = "private_or_onpremise"
	}
	ret := []jsonutils.JSONObject{}
	for {
		query["offset"] = len(ret)
		resp, err := self.manager.List(s, jsonutils.Marshal(query))
		if err != nil {
			return errors.Wrapf(err, "%s.List", self.manager.GetKeyword())
		}
		ret = append(ret, resp.Data...)
		if len(ret) >= resp.Total {
			break
		}
	}
	for i := range ret {
		baseInfo := sBaseInfo{}
		ret[i].Unmarshal(&baseInfo)
		if len(baseInfo.ExternalId) == 0 && (self.manager.GetKeyword() != compute.Cloudproviders.GetKeyword() &&
			self.manager.GetKeyword() != compute.Cloudaccounts.GetKeyword()) {
			continue
		}
		key := baseInfo.ExternalId
		if len(key) == 0 {
			key = baseInfo.Id
		}
		self.resourceLock.Lock()
		self.Resources[key] = ret[i]
		self.resourceLock.Unlock()
		if len(baseInfo.ManagerId) > 0 {
			if _, ok := self.ProviderResources[baseInfo.ManagerId]; !ok {
				self.ProviderResources[baseInfo.ManagerId] = map[string]jsonutils.JSONObject{}
			}
			self.providerLock.Lock()
			self.ProviderResources[baseInfo.ManagerId][key] = ret[i]
			self.providerLock.Unlock()
		}
		if self.importedAt.IsZero() || self.importedAt.Before(baseInfo.ImportedAt) {
			self.importedAt = baseInfo.ImportedAt
		}
		if self.createdAt.IsZero() || self.createdAt.Before(baseInfo.CreatedAt) {
			self.createdAt = baseInfo.CreatedAt
		}
	}
	log.Infof("increment %d %s", len(ret), self.manager.GetKeyword())
	return nil
}

func (self *SBaseResources) decrement() error {
	s := auth.GetAdminSession(context.Background(), options.Options.Region)
	timeFilter := fmt.Sprintf("deleted_at.gt('%s')", self.deletedAt)
	query := map[string]interface{}{
		"limit":      20,
		"scope":      "system",
		"details":    true,
		"order_by.0": "deleted_at",
		"order":      "asc",
		"delete":     "all",
		"@deleted":   "true",
		"filter.0":   timeFilter,
		"filter.1":   "external_id.isnotempty()",
	}
	if self.manager.GetKeyword() == compute.Hosts.GetKeyword() {
		query["cloud_env"] = "private_or_onpremise"
	}

	ret := []jsonutils.JSONObject{}
	for {
		query["offset"] = len(ret)
		resp, err := self.manager.List(s, jsonutils.Marshal(query))
		if err != nil {
			return errors.Wrapf(err, "%s.List", self.manager.GetKeyword())
		}
		ret = append(ret, resp.Data...)
		if len(ret) >= resp.Total {
			break
		}
	}
	for i := range ret {
		baseInfo := sBaseInfo{}
		ret[i].Unmarshal(&baseInfo)
		if len(baseInfo.ExternalId) == 0 && self.manager.GetKeyword() != compute.Cloudproviders.GetKeyword() {
			continue
		}
		key := baseInfo.ExternalId
		if len(key) == 0 {
			key = baseInfo.Id
		}
		delete(self.Resources, key)
		if len(baseInfo.ManagerId) > 0 {
			providerInfo, ok := self.ProviderResources[baseInfo.ManagerId]
			if ok {
				delete(providerInfo, key)
				self.providerLock.Lock()
				self.ProviderResources[baseInfo.ManagerId] = providerInfo
				self.providerLock.Unlock()
			}
		}
		if self.deletedAt.Before(baseInfo.DeletedAt) {
			self.deletedAt = baseInfo.DeletedAt
		}
	}
	log.Infof("decrement %d %s", len(ret), self.manager.GetKeyword())
	return nil
}

func (self *SBaseResources) update() error {
	s := auth.GetAdminSession(context.Background(), options.Options.Region)
	timeFilter := fmt.Sprintf("updated_at.gt('%s')", self.updatedAt)
	query := map[string]interface{}{
		"limit":          20,
		"scope":          "system",
		"details":        true,
		"order_by.0":     "updated_at",
		"order":          "asc",
		"pending_delete": "all",
		"filter.0":       timeFilter,
		"filter.1":       "external_id.isnotempty()",
	}
	if self.manager.GetKeyword() == compute.Hosts.GetKeyword() {
		query["cloud_env"] = "private_or_onpremise"
	}

	ret := []jsonutils.JSONObject{}
	for {
		query["offset"] = len(ret)
		resp, err := self.manager.List(s, jsonutils.Marshal(query))
		if err != nil {
			return errors.Wrapf(err, "%s.List", self.manager.GetKeyword())
		}
		ret = append(ret, resp.Data...)
		if len(ret) >= resp.Total {
			break
		}
	}
	for i := range ret {
		baseInfo := sBaseInfo{}
		ret[i].Unmarshal(&baseInfo)
		if len(baseInfo.ExternalId) == 0 {
			continue
		}
		key := baseInfo.ExternalId
		self.resourceLock.Lock()
		self.Resources[key] = ret[i]
		self.resourceLock.Unlock()
		if len(baseInfo.ManagerId) > 0 {
			_, ok := self.ProviderResources[baseInfo.ManagerId]
			if ok {
				self.providerLock.Lock()
				self.ProviderResources[baseInfo.ManagerId][key] = ret[i]
				self.providerLock.Unlock()
			}
		}
	}
	self.updatedAt = time.Now()
	log.Infof("update %d %s", len(ret), self.manager.GetKeyword())
	return nil
}

func NewBaseResources(manager modulebase.Manager) *SBaseResources {
	return &SBaseResources{
		manager:           manager,
		Resources:         map[string]jsonutils.JSONObject{},
		ProviderResources: map[string]map[string]jsonutils.JSONObject{},
	}
}

type TResource interface {
	init() error
	increment() error
	decrement() error
	update() error
	getResources(managerId string) map[string]jsonutils.JSONObject
}

type SResources struct {
	Cloudaccounts  TResource
	Cloudproviders TResource
	DBInstances    TResource
	Servers        TResource
	Hosts          TResource
	Redis          TResource
	Loadbalancers  TResource
	Buckets        TResource
	KubeClusters   TResource
	Storages       TResource
	ModelartsPool  TResource
}

func NewResources() *SResources {
	return &SResources{
		Cloudaccounts:  NewBaseResources(&compute.Cloudaccounts),
		Cloudproviders: NewBaseResources(&compute.Cloudproviders),
		DBInstances:    NewBaseResources(&compute.DBInstance),
		Servers:        NewBaseResources(&compute.Servers),
		Hosts:          NewBaseResources(&compute.Hosts),
		Storages:       NewBaseResources(&compute.Storages),
		Redis:          NewBaseResources(&compute.ElasticCache),
		Loadbalancers:  NewBaseResources(&compute.Loadbalancers),
		Buckets:        NewBaseResources(&compute.Buckets),
		KubeClusters:   NewBaseResources(&compute.KubeClusters),
		ModelartsPool:  NewBaseResources(&compute.ModelartsPools),
	}
}

func (self *SResources) Init(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if isStart {
		err := func() error {
			errs := []error{}
			err := self.Cloudaccounts.init()
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "Cloudaccount.init"))
			}
			err = self.Cloudproviders.init()
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "Cloudproviders.init"))
			}
			err = self.DBInstances.init()
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "DBInstances.init"))
			}
			err = self.Servers.init()
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "Servers.init"))
			}
			err = self.Hosts.init()
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "Hosts.init"))
			}
			err = self.Storages.init()
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "Storages.init"))
			}
			err = self.Redis.init()
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "Redis.init"))
			}
			err = self.Loadbalancers.init()
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "Loadbalancers.init"))
			}
			err = self.Buckets.init()
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "Buckets.init"))
			}
			err = self.KubeClusters.init()
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "KubeClusters.init"))
			}
			err = self.ModelartsPool.init()
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "ModelartsPool.init"))
			}
			return errors.NewAggregate(errs)
		}()
		if err != nil {
			log.Errorf("Resource init error: %v", err)
		}
	}
}

func (self *SResources) IncrementSync(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if isStart {
		return
	}
	err := func() error {
		errs := []error{}
		err := self.Cloudaccounts.increment()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Cloudaccounts.increment"))
		}
		err = self.Cloudproviders.increment()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Cloudproviders.increment"))
		}
		err = self.DBInstances.increment()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "DBInstances.increment"))
		}
		err = self.Servers.increment()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Servers.increment"))
		}
		err = self.Hosts.increment()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Hosts.increment"))
		}
		err = self.Storages.increment()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Storages.increment"))
		}
		err = self.Redis.increment()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Redis.increment"))
		}
		err = self.Loadbalancers.increment()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Loadbalancers.increment"))
		}
		err = self.Buckets.increment()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Buckets.increment"))
		}
		err = self.KubeClusters.increment()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "KubeClusters.increment"))
		}
		err = self.ModelartsPool.increment()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "ModelartsPool.increment"))
		}
		return errors.NewAggregate(errs)
	}()
	if err != nil {
		log.Errorf("Increment error: %v", err)
	}
}

func (self *SResources) DecrementSync(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if isStart {
		return
	}
	err := func() error {
		errs := []error{}
		err := self.Cloudaccounts.decrement()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Cloudaccounts.decrement"))
		}
		err = self.Cloudproviders.decrement()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Cloudproviders.decrement"))
		}
		err = self.DBInstances.decrement()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "DBInstances.decrement"))
		}
		err = self.Servers.decrement()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Servers.decrement"))
		}
		err = self.Hosts.decrement()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Hosts.decrement"))
		}
		err = self.Storages.decrement()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Storages.decrement"))
		}
		err = self.Redis.decrement()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Redis.decrement"))
		}
		err = self.Loadbalancers.decrement()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Loadbalancers.decrement"))
		}
		err = self.Buckets.decrement()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Buckets.decrement"))
		}
		err = self.KubeClusters.decrement()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "KubeClusters.decrement"))
		}
		err = self.ModelartsPool.decrement()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "ModelartsPool.decrement"))
		}
		return errors.NewAggregate(errs)
	}()
	if err != nil {
		log.Errorf("Increment error: %v", err)
	}
}

func (self *SResources) UpdateSync(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if isStart {
		return
	}
	err := func() error {
		errs := []error{}
		err := self.Cloudaccounts.update()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Cloudacconts.update"))
		}
		err = self.DBInstances.update()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "DBInstances.update"))
		}
		err = self.Servers.update()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Servers.update"))
		}
		err = self.Hosts.update()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Hosts.update"))
		}
		err = self.Storages.update()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Storages.update"))
		}
		err = self.Redis.update()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Redis.update"))
		}
		err = self.Loadbalancers.update()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Loadbalancers.update"))
		}
		err = self.ModelartsPool.update()
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "ModelartsPool.update"))
		}
		return errors.NewAggregate(errs)
	}()
	if err != nil {
		log.Errorf("Update error: %v", err)
	}
}

func (self *SResources) CollectMetrics(ctx context.Context, userCred mcclient.TokenCredential, taskStartTime time.Time, isStart bool) {
	if isStart {
		return
	}
	ch := make(chan struct{}, options.Options.CloudAccountCollectMetricsBatchCount)
	defer close(ch)
	s := auth.GetAdminSession(context.Background(), options.Options.Region)
	resources := self.Cloudproviders.getResources("")
	cloudproviders := map[string]api.CloudproviderDetails{}
	jsonutils.Update(&cloudproviders, resources)
	sh, _ := time.LoadLocation("Asia/Shanghai")
	_endTime := taskStartTime.In(sh)
	_startTime := _endTime.Add(-1 * time.Minute * time.Duration(options.Options.CollectMetricInterval))
	var wg sync.WaitGroup
	for i := range cloudproviders {
		ch <- struct{}{}
		wg.Add(1)
		go func(manager api.CloudproviderDetails) {
			defer func() {
				wg.Done()
				<-ch
			}()

			if strings.Contains(strings.ToLower(options.Options.SkipMetricPullProviders), strings.ToLower(manager.Provider)) {
				log.Infof("skip %s metric pull with options: %s", manager.Provider, options.Options.SkipMetricPullProviders)
				return
			}

			driver, err := providerdriver.GetDriver(manager.Provider)
			if err != nil {
				log.Errorf("failed get provider %s(%s) driver %v", manager.Name, manager.Provider, err)
				return
			}

			if !driver.IsSupportMetrics() {
				log.Infof("%s not support metrics, skip", driver.GetProvider())
				return
			}

			provider, err := compute.Cloudproviders.GetProvider(ctx, s, manager.Id)
			if err != nil {
				log.Errorf("failed get provider %s(%s) driver %v", manager.Name, manager.Provider, err)
				return
			}
			duration := driver.GetDelayDuration()
			endTime := _endTime.Add(-1 * duration)
			startTime := _startTime.Add(-1 * duration).Add(time.Second * -59)

			resources = self.DBInstances.getResources(manager.Id)
			dbinstances := map[string]api.DBInstanceDetails{}
			err = jsonutils.Update(&dbinstances, resources)
			if err != nil {
				log.Errorf("unmarsha rds resources error: %v", err)
			}
			err = driver.CollectDBInstanceMetrics(ctx, manager, provider, dbinstances, startTime, endTime)
			if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
				log.Errorf("CollectDBInstanceMetrics for %s(%s) error: %v", manager.Name, manager.Provider, err)
			}

			resources = self.Servers.getResources(manager.Id)
			servers := map[string]api.ServerDetails{}
			err = jsonutils.Update(&servers, resources)
			if err != nil {
				log.Errorf("unmarsha server resources error: %v", err)
			}
			err = driver.CollectServerMetrics(ctx, manager, provider, servers, startTime, endTime)
			if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
				log.Errorf("CollectServerMetrics for %s(%s) error: %v", manager.Name, manager.Provider, err)
			}

			resources = self.Hosts.getResources(manager.Id)
			hosts := map[string]api.HostDetails{}
			err = jsonutils.Update(&hosts, resources)
			if err != nil {
				log.Errorf("unmarsha host resources error: %v", err)
			}

			err = driver.CollectHostMetrics(ctx, manager, provider, hosts, startTime, endTime)
			if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
				log.Errorf("CollectHostMetrics for %s(%s) error: %v", manager.Name, manager.Provider, err)
			}

			resources = self.Storages.getResources(manager.Id)
			storages := map[string]api.StorageDetails{}
			err = jsonutils.Update(&storages, resources)
			if err != nil {
				log.Errorf("unmarsha storage resources error: %v", err)
			}
			err = driver.CollectStorageMetrics(ctx, manager, provider, storages, startTime, endTime)
			if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
				log.Errorf("CollectStorageMetrics for %s(%s) error: %v", manager.Name, manager.Provider, err)
			}

			resources = self.Redis.getResources(manager.Id)
			caches := map[string]api.ElasticcacheDetails{}
			err = jsonutils.Update(&caches, resources)
			if err != nil {
				log.Errorf("unmarsha redis resources error: %v", err)
			}

			err = driver.CollectRedisMetrics(ctx, manager, provider, caches, startTime, endTime)
			if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
				log.Errorf("CollectRedisMetrics for %s(%s) error: %v", manager.Name, manager.Provider, err)
			}

			resources = self.Loadbalancers.getResources(manager.Id)
			lbs := map[string]api.LoadbalancerDetails{}
			err = jsonutils.Update(&lbs, resources)
			if err != nil {
				log.Errorf("unmarsha lb resources error: %v", err)
			}

			err = driver.CollectLoadbalancerMetrics(ctx, manager, provider, lbs, startTime, endTime)
			if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
				log.Errorf("CollectLoadbalancerMetrics for %s(%s) error: %v", manager.Name, manager.Provider, err)
			}

			resources = self.Buckets.getResources(manager.Id)
			buckets := map[string]api.BucketDetails{}
			err = jsonutils.Update(&buckets, resources)
			if err != nil {
				log.Errorf("unmarsha bucket resources error: %v", err)
			}

			err = driver.CollectBucketMetrics(ctx, manager, provider, buckets, startTime, endTime)
			if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
				log.Errorf("CollectBucketMetrics for %s(%s) error: %v", manager.Name, manager.Provider, err)
			}

			resources = self.KubeClusters.getResources(manager.Id)
			clusters := map[string]api.KubeClusterDetails{}
			err = jsonutils.Update(&clusters, resources)
			if err != nil {
				log.Errorf("unmarsha k8s resources error: %v", err)
			}

			err = driver.CollectK8sMetrics(ctx, manager, provider, clusters, startTime, endTime)
			if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
				log.Errorf("CollectK8sMetrics for %s(%s) error: %v", manager.Name, manager.Provider, err)
			}

			resources = self.ModelartsPool.getResources(manager.Id)
			pools := map[string]api.ModelartsPoolDetails{}
			err = jsonutils.Update(&pools, resources)
			if err != nil {
				log.Errorf("unmarsha modelarts resources error: %v", err)
			}

			err = driver.CollectModelartsPoolMetrics(ctx, manager, provider, pools, startTime, endTime)
			if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
				log.Errorf("CollectModelartsPoolMetrics for %s(%s) error: %v", manager.Name, manager.Provider, err)
			}
		}(cloudproviders[i])
	}
	wg.Wait()

	resources = self.Cloudaccounts.getResources("")
	accounts := map[string]api.CloudaccountDetail{}
	jsonutils.Update(&accounts, resources)

	metrics := []influxdb.SMetricData{}
	for _, account := range accounts {
		driver, err := providerdriver.GetDriver(account.Provider)
		if err != nil {
			log.Errorf("failed get account %s(%s) driver %v", account.Name, account.Provider, err)
			return
		}

		metric, err := driver.CollectAccountMetrics(ctx, account)
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
				log.Errorf("CollectAccountMetrics for %s(%s) error: %v", account.Name, account.Provider, err)
				continue
			}
			continue
		}
		metrics = append(metrics, metric)
	}
	urls, err := s.GetServiceURLs(apis.SERVICE_TYPE_INFLUXDB, options.Options.SessionEndpointType, "")
	if err != nil {
		return
	}
	influxdb.SendMetrics(urls, "meter_db", metrics, false)
	return
}
