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
	"time"

	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	computemodels "yunion.io/x/onecloud/pkg/compute/models"
)

type baseBuilder struct {
	resourceType string

	isolatedDevicesDict map[string][]interface{}
}

func newBaseBuilder(resourceType string) *baseBuilder {
	return &baseBuilder{
		resourceType: resourceType,
	}
}

func (b *baseBuilder) Type() string {
	return b.resourceType
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
