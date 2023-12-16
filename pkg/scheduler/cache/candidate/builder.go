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

package candidate

import (
	"fmt"
	"strings"
	gosync "sync"
	"sync/atomic"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"
	"yunion.io/x/pkg/util/workqueue"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	computemodels "yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/data_manager/cloudaccount"
	"yunion.io/x/onecloud/pkg/scheduler/data_manager/cloudprovider"
	o "yunion.io/x/onecloud/pkg/scheduler/options"
)

type baseBuilder struct {
	resourceType string
	builder      IResourceBuilder

	hosts    []computemodels.SHost
	hostDict map[string]*computemodels.SHost

	isolatedDevicesDict map[string][]interface{}

	hostCloudproviers map[string]*computemodels.SCloudprovider
	hostCloudaccounts map[string]*computemodels.SCloudaccount
}

type InitFunc func(ids []computemodels.SHost, errChan chan error)

type IResourceBuilder interface {
	FetchHosts(ids []string) ([]computemodels.SHost, error)
	InitFuncs() []InitFunc
	BuildOne(host *computemodels.SHost, getter *networkGetter, desc *BaseHostDesc) (interface{}, error)
}

func newBaseBuilder(resourceType string, builder IResourceBuilder) *baseBuilder {
	return &baseBuilder{
		resourceType: resourceType,
		builder:      builder,
	}
}

func (b *baseBuilder) Type() string {
	return b.resourceType
}

func (b *baseBuilder) Do(ids []string) ([]interface{}, error) {
	err := b.init(ids)
	if err != nil {
		return nil, err
	}
	netGetter := newNetworkGetter()
	descs, err := b.build(netGetter)
	if err != nil {
		return nil, err
	}
	return descs, nil
}

func (b *baseBuilder) init(ids []string) error {
	if err := b.setHosts(ids); err != nil {
		return errors.Wrap(err, "set host objects")
	}
	wg := &WaitGroupWrapper{}
	errMessageChannel := make(chan error, 12)
	defer close(errMessageChannel)
	setFuncs := []func(){
		// func() { b.setHosts(ids, errMessageChannel) },
		func() {
			b.setIsolatedDevs(ids, errMessageChannel)
		},
		func() {
			b.setCloudproviderAccounts(b.hosts, errMessageChannel)
		},
		func() {
			for _, f := range b.builder.InitFuncs() {
				f(b.hosts, errMessageChannel)
			}
		},
	}

	for _, f := range setFuncs {
		wg.Wrap(f)
	}

	if ok := waitTimeOut(wg, time.Duration(20*time.Second)); !ok {
		log.Errorln("HostBuilder waitgroup timeout.")
	}

	if len(errMessageChannel) != 0 {
		errMessages := make([]string, 0)
		lengthChan := len(errMessageChannel)
		for ; lengthChan > 0; lengthChan-- {
			msg := fmt.Sprintf("%s", <-errMessageChannel)
			log.Errorf("Get error from chan: %s", msg)
			errMessages = append(errMessages, msg)
		}
		return fmt.Errorf("%s\n", strings.Join(errMessages, ";"))
	}

	return nil
}

func (b *baseBuilder) build(netGetter *networkGetter) ([]interface{}, error) {
	schedDescs := make([]interface{}, len(b.hosts))
	errs := []error{}
	var descResultLock gosync.Mutex
	var descedLen int32

	buildOne := func(i int) {
		if i >= len(b.hosts) {
			log.Errorf("invalid host index[%d] in b.hosts: %v", i, b.hosts)
			return
		}
		host := b.hosts[i]
		desc, err := b.buildOne(&host, netGetter)
		if err != nil {
			descResultLock.Lock()
			errs = append(errs, err)
			descResultLock.Unlock()
			return
		}
		descResultLock.Lock()
		schedDescs[atomic.AddInt32(&descedLen, 1)-1] = desc
		descResultLock.Unlock()
	}

	workqueue.Parallelize(o.Options.HostBuildParallelizeSize, len(b.hosts), buildOne)
	schedDescs = schedDescs[:descedLen]
	if len(errs) > 0 {
		//return nil, errors.NewAggregate(errs)
		err := errors.NewAggregate(errs)
		log.Errorf("Build schedule desc of %s error: %s", b.resourceType, err)
	}

	return schedDescs, nil
}

func (b *baseBuilder) buildOne(host *computemodels.SHost, netGetter *networkGetter) (interface{}, error) {
	baseDesc, err := newBaseHostDesc(b, host, netGetter)
	if err != nil {
		return nil, err
	}

	return b.builder.BuildOne(host, netGetter, baseDesc)
}

func (b *baseBuilder) setHosts(ids []string) error {
	hostObjs, err := b.builder.FetchHosts(ids)
	if err != nil {
		return errors.Wrap(err, "FetchHosts")
	}

	hostDict := ToDict(hostObjs)
	b.hosts = hostObjs
	b.hostDict = hostDict
	return nil
}

func (b *baseBuilder) getIsolatedDevices(hostID string) (devs []computemodels.SIsolatedDevice) {
	devObjs, ok := b.isolatedDevicesDict[hostID]
	devs = make([]computemodels.SIsolatedDevice, 0)
	if !ok {
		return
	}
	for _, obj := range devObjs {
		dev := obj.(computemodels.SIsolatedDevice)
		devs = append(devs, dev)
	}
	return
}

func (b *baseBuilder) setIsolatedDevs(ids []string, errMessageChannel chan error) {
	devs := computemodels.IsolatedDeviceManager.FindByHosts(ids)
	dict, err := utils.GroupBy(devs, func(obj interface{}) (string, error) {
		dev, ok := obj.(computemodels.SIsolatedDevice)
		if !ok {
			return "", utils.ConvertError(obj, "computemodels.SIsolatedDevice")
		}
		return dev.HostId, nil
	})
	if err != nil {
		errMessageChannel <- err
		return
	}
	b.isolatedDevicesDict = dict
}

func (b *baseBuilder) setCloudproviderAccounts(hosts []computemodels.SHost, errCh chan error) {
	providerSets := sets.NewString()
	for _, host := range hosts {
		mId := host.ManagerId
		if mId != "" {
			providerSets.Insert(mId)
		}
	}
	providerObjs := make([]computemodels.SCloudprovider, 0)
	for _, pId := range providerSets.List() {
		pObj, ok := cloudprovider.GetManager().GetResource(pId)
		if !ok {
			errCh <- errors.Errorf("Not found cloudprovider by id: %q", pId)
			return
		}
		providerObjs = append(providerObjs, pObj)
	}
	providerDict := ToDict(providerObjs)

	accountSets := sets.NewString()
	for _, provider := range providerObjs {
		accountSets.Insert(provider.CloudaccountId)
	}
	accountObjs := make([]computemodels.SCloudaccount, 0)
	for _, aId := range accountSets.List() {
		aObj, ok := cloudaccount.Manager.GetResource(aId)
		if !ok {
			errCh <- errors.Errorf("Not found cloudaccount by id: %q", aId)
			return
		}
		accountObjs = append(accountObjs, aObj)
	}
	accountDict := ToDict(accountObjs)

	b.hostCloudproviers = make(map[string]*computemodels.SCloudprovider, 0)
	b.hostCloudaccounts = make(map[string]*computemodels.SCloudaccount, 0)
	for _, host := range hosts {
		pId := host.ManagerId
		provider, ok := providerDict[pId]
		if !ok {
			continue
		}
		b.hostCloudproviers[host.GetId()] = provider
		aId := provider.CloudaccountId
		account, ok := accountDict[aId]
		if !ok {
			continue
		}
		b.hostCloudaccounts[host.GetId()] = account
	}
}

func FetchModelIds(q *sqlchemy.SQuery) ([]string, error) {
	rs, err := q.Rows()
	if err != nil {
		return nil, err
	}
	ret := []string{}
	defer rs.Close()
	for rs.Next() {
		var id string
		if err := rs.Scan(&id); err != nil {
			return nil, err
		}
		ret = append(ret, id)
	}
	return ret, nil
}

func FetchHostsByIds(ids []string) ([]computemodels.SHost, error) {
	hosts := computemodels.HostManager.Query()
	q := hosts.In("id", ids)
	hostObjs := make([]computemodels.SHost, 0)
	if err := db.FetchModelObjects(computemodels.HostManager, q, &hostObjs); err != nil {
		return nil, err
	}
	return hostObjs, nil
}

type UpdateStatus struct {
	Id        string    `json:"id"`
	UpdatedAt time.Time `json:"updated_at"`
}

func FetchModelUpdateStatus(man db.IStandaloneModelManager, cond sqlchemy.ICondition) ([]UpdateStatus, error) {
	ret := make([]UpdateStatus, 0)
	err := man.Query("id", "updated_at").Filter(cond).All(&ret)
	return ret, err
}

func FetchHostsUpdateStatus(isBaremetal bool) ([]UpdateStatus, error) {
	q := computemodels.HostManager.Query("id", "updated_at")
	if isBaremetal {
		q = q.Equals("host_type", computeapi.HOST_TYPE_BAREMETAL)
	} else {
		q = q.NotEquals("host_type", computeapi.HOST_TYPE_BAREMETAL)
	}
	ret := make([]UpdateStatus, 0)
	err := q.All(&ret)
	return ret, err
}

type ResidentTenant struct {
	HostId      string `json:"host_id"`
	TenantId    string `json:"tenant_id"`
	TenantCount int64  `json:"tenant_count"`
}

func (t ResidentTenant) First() string {
	return t.HostId
}

func (t ResidentTenant) Second() string {
	return t.TenantId
}

func (t ResidentTenant) Third() interface{} {
	return t.TenantCount
}

func FetchHostsResidentTenants(hostIds []string) ([]ResidentTenant, error) {
	guests := computemodels.GuestManager.Query().SubQuery()
	q := guests.Query(
		guests.Field("host_id"),
		guests.Field("tenant_id"),
		sqlchemy.COUNT("tenant_count", guests.Field("tenant_id")),
	).In("host_id", hostIds).GroupBy("tenant_id", "host_id")
	ret := make([]ResidentTenant, 0)
	err := q.All(&ret)
	return ret, err
}
