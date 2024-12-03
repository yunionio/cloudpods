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
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/tsdb"
	"yunion.io/x/onecloud/pkg/cloudmon/options"
	"yunion.io/x/onecloud/pkg/cloudmon/providerdriver"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/util/influxdb"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type sBaseInfo struct {
	Id         string
	ExternalId string
	ManagerId  string
	CreatedAt  time.Time
	ImportedAt time.Time
	DeletedAt  time.Time
	UpdatedAt  time.Time
	Metadata   map[string]string
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

func (self *SBaseResources) getResources(ctx context.Context, managerId string) map[string]jsonutils.JSONObject {
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

func (self *SBaseResources) init(ctx context.Context) error {
	s := auth.GetAdminSession(ctx, options.Options.Region)
	query := map[string]interface{}{
		"limit":          20,
		"scope":          "system",
		"details":        true,
		"order_by.0":     "created_at",
		"order_by.1":     "imported_at",
		"order":          "asc",
		"pending_delete": "all",
	}
	if self.manager.GetKeyword() == compute.Hosts.GetKeyword() { // private and vmware
		query["cloud_env"] = "private_or_onpremise"
	}
	if self.manager.GetKeyword() != compute.Cloudproviders.GetKeyword() {
		query["filter.0"] = "external_id.isnotempty()"
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
			baseInfo := sBaseInfo{}
			resp.Data[i].Unmarshal(&baseInfo)
			if len(baseInfo.ExternalId) == 0 && (self.manager.GetKeyword() != compute.Cloudproviders.GetKeyword() &&
				self.manager.GetKeyword() != compute.Cloudaccounts.GetKeyword() &&
				self.manager.GetKeyword() != identity.Projects.GetKeyword()) {
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

func (self *SBaseResources) increment(ctx context.Context) error {
	s := auth.GetAdminSession(ctx, options.Options.Region)
	timeFilter := fmt.Sprintf("imported_at.gt('%s')", self.importedAt.Format(time.RFC3339))
	if self.importedAt.IsZero() {
		timeFilter = fmt.Sprintf("created_at.gt('%s')", self.createdAt.Format(time.RFC3339))
	}
	query := map[string]interface{}{
		"limit":      20,
		"scope":      "system",
		"details":    true,
		"order_by.0": "created_at",
		"order_by.1": "imported_at",
		"order":      "asc",
		"filter.0":   timeFilter,
	}
	if self.manager.GetKeyword() != compute.Cloudproviders.GetKeyword() {
		query["filter.1"] = "external_id.isnotempty()"
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
			self.manager.GetKeyword() != identity.Projects.GetKeyword() &&
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

func (self *SBaseResources) decrement(ctx context.Context) error {
	s := auth.GetAdminSession(ctx, options.Options.Region)
	timeFilter := fmt.Sprintf("deleted_at.gt('%s')", self.deletedAt.Format(time.RFC3339))
	query := map[string]interface{}{
		"limit":      20,
		"scope":      "system",
		"details":    true,
		"order_by.0": "deleted_at",
		"order":      "asc",
		"delete":     "all",
		"@deleted":   "true",
		"filter.0":   timeFilter,
	}
	if self.manager.GetKeyword() != compute.Cloudproviders.GetKeyword() {
		query["filter.1"] = "external_id.isnotempty()"
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
		if len(baseInfo.ExternalId) == 0 && self.manager.GetKeyword() != compute.Cloudproviders.GetKeyword() &&
			self.manager.GetKeyword() != compute.Cloudaccounts.GetKeyword() &&
			self.manager.GetKeyword() != identity.Projects.GetKeyword() {
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

func (self *SBaseResources) update(ctx context.Context) error {
	s := auth.GetAdminSession(ctx, options.Options.Region)
	timeFilter := fmt.Sprintf("updated_at.gt('%s')", self.updatedAt.Format(time.RFC3339))
	query := map[string]interface{}{
		"limit":          20,
		"scope":          "system",
		"details":        true,
		"order_by.0":     "updated_at",
		"order":          "asc",
		"pending_delete": "all",
		"filter.0":       timeFilter,
	}
	if self.manager.GetKeyword() != compute.Cloudproviders.GetKeyword() {
		query["filter.1"] = "external_id.isnotempty()"
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
		key := baseInfo.ExternalId
		if len(key) == 0 {
			key = baseInfo.Id
		}
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
	init(ctx context.Context) error
	increment(ctx context.Context) error
	decrement(ctx context.Context) error
	update(ctx context.Context) error
	getResources(ctx context.Context, managerId string) map[string]jsonutils.JSONObject
}

type SResources struct {
	init           bool
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
	Wires          TResource
	Projects       TResource
	ElasticIps     TResource
}

func (self *SResources) IsInit() bool {
	return self.init
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
		Wires:          NewBaseResources(&compute.Wires),
		Projects:       NewBaseResources(&identity.Projects),
		ElasticIps:     NewBaseResources(&compute.Elasticips),
	}
}

func (self *SResources) Init(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if isStart {
		err := func() error {
			errs := []error{}
			err := self.Cloudaccounts.init(ctx)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "Cloudaccount.init"))
			}
			err = self.Projects.init(ctx)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "Projects.init"))
			}
			err = self.Cloudproviders.init(ctx)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "Cloudproviders.init"))
			}
			err = self.DBInstances.init(ctx)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "DBInstances.init"))
			}
			err = self.Servers.init(ctx)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "Servers.init"))
			}
			err = self.Hosts.init(ctx)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "Hosts.init"))
			}
			err = self.Storages.init(ctx)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "Storages.init"))
			}
			err = self.Redis.init(ctx)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "Redis.init"))
			}
			err = self.Loadbalancers.init(ctx)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "Loadbalancers.init"))
			}
			err = self.Buckets.init(ctx)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "Buckets.init"))
			}
			err = self.KubeClusters.init(ctx)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "KubeClusters.init"))
			}
			err = self.ModelartsPool.init(ctx)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "ModelartsPool.init"))
			}
			err = self.ElasticIps.init(ctx)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "ElasticIps.init"))
			}
			err = self.Wires.init(ctx)
			if err != nil {
				errs = append(errs, errors.Wrapf(err, "Wires.init"))
			}
			return errors.NewAggregate(errs)
		}()
		if err != nil {
			log.Errorf("Resource init error: %v", err)
		}
		self.init = true
	}
}

var incrementSync = false

func (self *SResources) IncrementSync(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if isStart || incrementSync {
		return
	}
	incrementSync = true
	defer func() {
		incrementSync = false
	}()
	err := func() error {
		errs := []error{}
		err := self.Cloudaccounts.increment(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Cloudaccounts.increment"))
		}
		err = self.Projects.increment(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Projects.increment"))
		}
		err = self.Cloudproviders.increment(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Cloudproviders.increment"))
		}
		err = self.DBInstances.increment(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "DBInstances.increment"))
		}
		err = self.Servers.increment(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Servers.increment"))
		}
		err = self.Hosts.increment(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Hosts.increment"))
		}
		err = self.Storages.increment(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Storages.increment"))
		}
		err = self.Redis.increment(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Redis.increment"))
		}
		err = self.Loadbalancers.increment(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Loadbalancers.increment"))
		}
		err = self.Buckets.increment(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Buckets.increment"))
		}
		err = self.KubeClusters.increment(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "KubeClusters.increment"))
		}
		err = self.ModelartsPool.increment(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "ModelartsPool.increment"))
		}
		err = self.ElasticIps.increment(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Elasticips.increment"))
		}
		err = self.Wires.increment(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Wires.increment"))
		}
		return errors.NewAggregate(errs)
	}()
	if err != nil {
		log.Errorf("Increment error: %v", err)
	}
}

var decrementSync = false

func (self *SResources) DecrementSync(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if isStart || decrementSync {
		return
	}
	decrementSync = true
	defer func() {
		decrementSync = false
	}()
	err := func() error {
		errs := []error{}
		err := self.Cloudaccounts.decrement(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Cloudaccounts.decrement"))
		}
		err = self.Cloudproviders.decrement(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Cloudproviders.decrement"))
		}
		err = self.DBInstances.decrement(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "DBInstances.decrement"))
		}
		err = self.Servers.decrement(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Servers.decrement"))
		}
		err = self.Hosts.decrement(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Hosts.decrement"))
		}
		err = self.Storages.decrement(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Storages.decrement"))
		}
		err = self.Redis.decrement(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Redis.decrement"))
		}
		err = self.Loadbalancers.decrement(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Loadbalancers.decrement"))
		}
		err = self.Buckets.decrement(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Buckets.decrement"))
		}
		err = self.KubeClusters.decrement(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "KubeClusters.decrement"))
		}
		err = self.ModelartsPool.decrement(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "ModelartsPool.decrement"))
		}
		err = self.Wires.decrement(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "ModelartsPool.decrement"))
		}
		err = self.ElasticIps.decrement(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "ElasticIps.decrement"))
		}
		err = self.Projects.decrement(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Projects.decrement"))
		}
		return errors.NewAggregate(errs)
	}()
	if err != nil {
		log.Errorf("Increment error: %v", err)
	}
}

var updateSync = false

func (self *SResources) UpdateSync(ctx context.Context, userCred mcclient.TokenCredential, isStart bool) {
	if isStart || updateSync {
		return
	}
	updateSync = true
	defer func() {
		updateSync = false
	}()
	err := func() error {
		errs := []error{}
		err := self.Cloudaccounts.update(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Cloudacconts.update"))
		}
		err = self.Projects.update(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Projects.update"))
		}
		err = self.DBInstances.update(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "DBInstances.update"))
		}
		err = self.Servers.update(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Servers.update"))
		}
		err = self.Hosts.update(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Hosts.update"))
		}
		err = self.Storages.update(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Storages.update"))
		}
		err = self.Redis.update(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Redis.update"))
		}
		err = self.Loadbalancers.update(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "Loadbalancers.update"))
		}
		err = self.ModelartsPool.update(ctx)
		if err != nil {
			errs = append(errs, errors.Wrapf(err, "ModelartsPool.update"))
		}
		return errors.NewAggregate(errs)
	}()
	if err != nil {
		log.Errorf("Update error: %v", err)
	}
}

type sMetricProvider struct {
	api.CloudproviderDetails
}

func (p sMetricProvider) GetId() string {
	return p.Id
}

func (p sMetricProvider) GetName() string {
	return p.Name
}

func (p sMetricProvider) Keyword() string {
	return "cloudprovider"
}

func (res *SResources) CollectMetrics(ctx context.Context, userCred mcclient.TokenCredential, taskStartTime time.Time, isStart bool) {
	if isStart {
		return
	}
	ch := make(chan struct{}, options.Options.CloudAccountCollectMetricsBatchCount)
	defer close(ch)
	s := auth.GetAdminSession(ctx, options.Options.Region)
	resources := res.Cloudproviders.getResources(ctx, "")
	cloudproviders := map[string]api.CloudproviderDetails{}
	jsonutils.Update(&cloudproviders, resources)
	az, _ := time.LoadLocation(options.Options.TimeZone)
	_endTime := taskStartTime.In(az)
	_startTime := _endTime.Add(-1 * time.Minute * time.Duration(options.Options.CollectMetricInterval))
	var wg sync.WaitGroup
	for i := range cloudproviders {
		ch <- struct{}{}
		wg.Add(1)
		goctx := context.WithValue(ctx, appctx.APP_CONTEXT_KEY_START_TIME, time.Now().UTC())
		go func(ctx context.Context, manager api.CloudproviderDetails) {
			succ := true
			msgs := make([]string, 0)
			defer func() {
				if len(msgs) > 0 {
					logclient.AddActionLogWithContext(ctx, &sMetricProvider{manager}, logclient.ACT_COLLECT_METRICS, strings.Join(msgs, ";"), userCred, succ)
				}
				wg.Done()
				<-ch
			}()

			if strings.Contains(strings.ToLower(options.Options.SkipMetricPullProviders), strings.ToLower(manager.Provider)) {
				logmsg := fmt.Sprintf("skip %s metric pull with options: %s", manager.Provider, options.Options.SkipMetricPullProviders)
				log.Infoln(logmsg)
				return
			}

			driver, err := providerdriver.GetDriver(manager.Provider)
			if err != nil {
				logmsg := fmt.Sprintf("failed get provider %s(%s) driver %v", manager.Name, manager.Provider, err)
				log.Errorln(logmsg)
				msgs = append(msgs, logmsg)
				succ = false
				return
			}

			if !driver.IsSupportMetrics() {
				logmsg := fmt.Sprintf("%s not support metrics, skip", driver.GetProvider())
				log.Infoln(logmsg)
				return
			}

			provider, err := compute.Cloudproviders.GetProvider(ctx, s, manager.Id)
			if err != nil {
				logmsg := fmt.Sprintf("failed get provider %s(%s) driver %v", manager.Name, manager.Provider, err)
				log.Errorln(logmsg)
				msgs = append(msgs, logmsg)
				succ = false
				return
			}
			duration := driver.GetDelayDuration()
			endTime := _endTime.Add(-1 * duration)
			startTime := _startTime.Add(-1 * duration).Add(time.Second * -59)

			resources = res.DBInstances.getResources(ctx, manager.Id)
			dbinstances := map[string]api.DBInstanceDetails{}
			err = jsonutils.Update(&dbinstances, resources)
			if err != nil {
				logmsg := fmt.Sprintf("unmarshal rds resources error: %v", err)
				log.Errorln(logmsg)
				msgs = append(msgs, logmsg)
				succ = false
			}
			if len(dbinstances) > 0 {
				err = driver.CollectDBInstanceMetrics(ctx, manager, provider, dbinstances, startTime, endTime)
				if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					logmsg := fmt.Sprintf("CollectDBInstanceMetrics for %s(%s) error: %v", manager.Name, manager.Provider, err)
					log.Errorln(logmsg)
					msgs = append(msgs, logmsg)
					succ = false
				}
			}

			resources = res.Servers.getResources(ctx, manager.Id)
			servers := map[string]api.ServerDetails{}
			err = jsonutils.Update(&servers, resources)
			if err != nil {
				logmsg := fmt.Sprintf("unmarsha server resources error: %v", err)
				log.Errorln(logmsg)
				msgs = append(msgs, logmsg)
				succ = false
			}

			if len(servers) > 0 {
				err = driver.CollectServerMetrics(ctx, manager, provider, servers, startTime, endTime)
				if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					logmsg := fmt.Sprintf("CollectServerMetrics for %s(%s) error: %v", manager.Name, manager.Provider, err)
					log.Errorf(logmsg)
					msgs = append(msgs, logmsg)
					succ = false
				}
			}

			resources = res.Hosts.getResources(ctx, manager.Id)
			hosts := map[string]api.HostDetails{}
			err = jsonutils.Update(&hosts, resources)
			if err != nil {
				logmsg := fmt.Sprintf("unmarsha host resources error: %v", err)
				log.Errorln(logmsg)
				msgs = append(msgs, logmsg)
				succ = false
			}

			if len(hosts) > 0 {
				err = driver.CollectHostMetrics(ctx, manager, provider, hosts, startTime, endTime)
				if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					logmsg := fmt.Sprintf("CollectHostMetrics for %s(%s) error: %v", manager.Name, manager.Provider, err)
					log.Errorln(logmsg)
					msgs = append(msgs, logmsg)
					succ = false
				}
			}

			resources = res.Storages.getResources(ctx, manager.Id)
			storages := map[string]api.StorageDetails{}
			err = jsonutils.Update(&storages, resources)
			if err != nil {
				logmsg := fmt.Sprintf("unmarsha storage resources error: %v", err)
				log.Errorln(logmsg)
				msgs = append(msgs, logmsg)
				succ = false
			}
			if len(storages) > 0 {
				err = driver.CollectStorageMetrics(ctx, manager, provider, storages, startTime, endTime)
				if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					logmsg := fmt.Sprintf("CollectStorageMetrics for %s(%s) error: %v", manager.Name, manager.Provider, err)
					log.Errorln(logmsg)
					msgs = append(msgs, logmsg)
					succ = false
				}
			}

			resources = res.Redis.getResources(ctx, manager.Id)
			caches := map[string]api.ElasticcacheDetails{}
			err = jsonutils.Update(&caches, resources)
			if err != nil {
				logmsg := fmt.Sprintf("unmarsha redis resources error: %v", err)
				log.Errorln(logmsg)
				msgs = append(msgs, logmsg)
				succ = false
			}

			if len(caches) > 0 {
				err = driver.CollectRedisMetrics(ctx, manager, provider, caches, startTime, endTime)
				if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					logmsg := fmt.Sprintf("CollectRedisMetrics for %s(%s) error: %v", manager.Name, manager.Provider, err)
					log.Errorf(logmsg)
					msgs = append(msgs, logmsg)
					succ = false
				}
			}

			resources = res.Loadbalancers.getResources(ctx, manager.Id)
			lbs := map[string]api.LoadbalancerDetails{}
			err = jsonutils.Update(&lbs, resources)
			if err != nil {
				logmsg := fmt.Sprintf("unmarsha lb resources error: %v", err)
				log.Errorln(logmsg)
				msgs = append(msgs, logmsg)
				succ = false
			}

			if len(lbs) > 0 {
				err = driver.CollectLoadbalancerMetrics(ctx, manager, provider, lbs, startTime, endTime)
				if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					logmsg := fmt.Sprintf("CollectLoadbalancerMetrics for %s(%s) error: %v", manager.Name, manager.Provider, err)
					log.Errorf(logmsg)
					msgs = append(msgs, logmsg)
					succ = false
				}
			}

			resources = res.Buckets.getResources(ctx, manager.Id)
			buckets := map[string]api.BucketDetails{}
			err = jsonutils.Update(&buckets, resources)
			if err != nil {
				logmsg := fmt.Sprintf("unmarsha bucket resources error: %v", err)
				log.Errorln(logmsg)
				msgs = append(msgs, logmsg)
				succ = false
			}

			if len(buckets) > 0 {
				err = driver.CollectBucketMetrics(ctx, manager, provider, buckets, startTime, endTime)
				if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					logmsg := fmt.Sprintf("CollectBucketMetrics for %s(%s) error: %v", manager.Name, manager.Provider, err)
					log.Errorln(logmsg)
					msgs = append(msgs, logmsg)
					succ = false
				}
			}

			resources = res.KubeClusters.getResources(ctx, manager.Id)
			clusters := map[string]api.KubeClusterDetails{}
			err = jsonutils.Update(&clusters, resources)
			if err != nil {
				logmsg := fmt.Sprintf("unmarsha k8s resources error: %v", err)
				log.Errorln(logmsg)
				msgs = append(msgs, logmsg)
				succ = false
			}

			if len(clusters) > 0 {
				err = driver.CollectK8sMetrics(ctx, manager, provider, clusters, startTime, endTime)
				if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					logmsg := fmt.Sprintf("CollectK8sMetrics for %s(%s) error: %v", manager.Name, manager.Provider, err)
					log.Errorln(logmsg)
					msgs = append(msgs, logmsg)
					succ = false
				}
			}

			resources = res.ModelartsPool.getResources(ctx, manager.Id)
			pools := map[string]api.ModelartsPoolDetails{}
			err = jsonutils.Update(&pools, resources)
			if err != nil {
				logmsg := fmt.Sprintf("unmarsha modelarts resources error: %v", err)
				log.Errorln(logmsg)
				msgs = append(msgs, logmsg)
				succ = false
			}

			if len(pools) > 0 {
				err = driver.CollectModelartsPoolMetrics(ctx, manager, provider, pools, startTime, endTime)
				if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					logmsg := fmt.Sprintf("CollectModelartsPoolMetrics for %s(%s) error: %v", manager.Name, manager.Provider, err)
					log.Errorln(logmsg)
					msgs = append(msgs, logmsg)
					succ = false
				}
			}

			resources = res.Wires.getResources(ctx, manager.Id)
			wires := map[string]api.WireDetails{}
			err = jsonutils.Update(&wires, resources)
			if err != nil {
				logmsg := fmt.Sprintf("unmarsha wires resources error: %v", err)
				log.Errorln(logmsg)
				msgs = append(msgs, logmsg)
				succ = false
			}

			if len(wires) > 0 {
				err = driver.CollectWireMetrics(ctx, manager, provider, wires, startTime, endTime)
				if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					logmsg := fmt.Sprintf("CollectWireMetrics for %s(%s) error: %v", manager.Name, manager.Provider, err)
					log.Errorln(logmsg)
					msgs = append(msgs, logmsg)
					succ = false
				}
			}

			resources = res.ElasticIps.getResources(ctx, manager.Id)
			eips := map[string]api.ElasticipDetails{}
			err = jsonutils.Update(&eips, resources)
			if err != nil {
				logmsg := fmt.Sprintf("unmarsha eips resources error: %v", err)
				log.Errorln(logmsg)
				msgs = append(msgs, logmsg)
				succ = false
			}

			if len(eips) > 0 {
				err = driver.CollectEipMetrics(ctx, manager, provider, eips, startTime, endTime)
				if err != nil && errors.Cause(err) != cloudprovider.ErrNotImplemented && errors.Cause(err) != cloudprovider.ErrNotSupported {
					logmsg := fmt.Sprintf("CollectEipMetrics for %s(%s) error: %v", manager.Name, manager.Provider, err)
					log.Errorln(logmsg)
					msgs = append(msgs, logmsg)
					succ = false
				}
			}

		}(goctx, cloudproviders[i])
	}
	wg.Wait()

	resources = res.Cloudaccounts.getResources(ctx, "")
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
	urls, err := tsdb.GetDefaultServiceSourceURLs(s, options.Options.SessionEndpointType)
	if err != nil {
		log.Errorf("Get influxdb %s service url: %v", options.Options.SessionEndpointType, err)
		return
	}
	if err := influxdb.SendMetrics(urls, "meter_db", metrics, true); err != nil {
		log.Errorf("SendMetrics to meter_db: %v", err)
		return
	}
}
