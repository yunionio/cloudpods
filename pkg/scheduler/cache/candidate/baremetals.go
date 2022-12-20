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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
	computemodels "yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/core"
)

type baremetalGetter struct {
	*baseHostGetter
	bm *BaremetalDesc
}

func newBaremetalGetter(bm *BaremetalDesc) *baremetalGetter {
	return &baremetalGetter{
		baseHostGetter: newBaseHostGetter(bm.BaseHostDesc),
		bm:             bm,
	}
}

func (h baremetalGetter) FreeCPUCount(_ bool) int64 {
	return h.bm.FreeCPUCount()
}

func (h baremetalGetter) FreeMemorySize(_ bool) int64 {
	return h.bm.FreeMemSize()
}

func (h baremetalGetter) IsEmpty() bool {
	return h.bm.ServerID == ""
}

func (h baremetalGetter) StorageInfo() []*baremetal.BaremetalStorage {
	return h.bm.StorageInfo
}

func (h baremetalGetter) GetFreePort(netId string) int {
	cnt := h.h.GetFreePort(netId)
	if cnt < 0 {
		cnt = 0
	}
	nics := h.GetNics()
	for _, nic := range nics {
		if len(nic.IpAddr) > 0 && nic.NetId == netId {
			cnt += 1
		}
	}
	return cnt
}

type BaremetalDesc struct {
	*BaseHostDesc

	StorageInfo []*baremetal.BaremetalStorage `json:"storage_info"`
	StorageType string                        `json:"storage_type"`
	StorageSize int64                         `json:"storage_size"`
	ServerID    string                        `json:"server_id"`
}

type BaremetalBuilder struct {
	*baseBuilder
	baremetals []computemodels.SHost

	residentTenantDict map[string]map[string]interface{}
}

func (bd *BaremetalDesc) Getter() core.CandidatePropertyGetter {
	return newBaremetalGetter(bd)
}

func (bd *BaremetalDesc) String() string {
	return jsonutils.Marshal(bd).String()
}

func (bd *BaremetalDesc) Type() int {
	// Baremetal type
	return 1
}

func (bd *BaremetalDesc) GetGuestCount() int64 {
	if bd.ServerID == "" {
		return 0
	}
	return 1
}

func (bd *BaremetalDesc) IndexKey() string {
	return bd.Id
}

func (bd *BaremetalDesc) FreeCPUCount() int64 {
	if bd.ServerID == "" {
		return int64(bd.CpuCount)
	}
	return 0
}

func (bd *BaremetalDesc) FreeMemSize() int64 {
	if bd.ServerID == "" {
		return int64(bd.MemSize)
	}
	return 0
}

func (bd *BaremetalDesc) FreeStorageSize() int64 {
	if bd.ServerID == "" {
		return bd.StorageSize
	}
	return 0
}

func newBaremetalBuilder() *BaremetalBuilder {
	return &BaremetalBuilder{
		baseBuilder: newBaseBuilder(BaremetalDescBuilder),
	}
}

func (bb *BaremetalBuilder) init(ids []string) error {
	bms, err := FetchHostsByIds(ids)
	if err != nil {
		return err
	}

	//bb.baremetalAgents = agents
	bb.baremetals = bms

	wg := &WaitGroupWrapper{}
	errMessageChannel := make(chan error, 2)
	defer close(errMessageChannel)

	setFuncs := []func(){
		func() { bb.setIsolatedDevs(ids, errMessageChannel) },
	}

	for _, f := range setFuncs {
		wg.Wrap(f)
	}

	if ok := waitTimeOut(wg, time.Duration(20*time.Second)); !ok {
		log.Errorln("BaremetalBuilder waitgroup timeout.")
	}

	if len(errMessageChannel) != 0 {
		errMessages := make([]string, 0)
		lengthChan := len(errMessageChannel)
		for ; lengthChan >= 0; lengthChan-- {
			errMessages = append(errMessages, fmt.Sprintf("%s", <-errMessageChannel))
		}
		return fmt.Errorf("%s\n", strings.Join(errMessages, ";"))
	}

	return nil
}

func (bb *BaremetalBuilder) Clone() BuildActor {
	return &BaremetalBuilder{
		baseBuilder: newBaseBuilder(BaremetalDescBuilder),
	}
}

func (bb *BaremetalBuilder) AllIDs() ([]string, error) {
	q := computemodels.HostManager.Query("id")
	q = q.Filter(sqlchemy.Equals(q.Field("host_type"), computeapi.HOST_TYPE_BAREMETAL))
	return FetchModelIds(q)
}

func (bb *BaremetalBuilder) Do(ids []string) ([]interface{}, error) {
	err := bb.init(ids)
	if err != nil {
		return nil, err
	}
	netGetter := newNetworkGetter()
	descs, err := bb.build(netGetter)
	if err != nil {
		return nil, err
	}
	return descs, nil
}

func (bb *BaremetalBuilder) build(netGetter *networkGetter) ([]interface{}, error) {
	schedDescs := []interface{}{}
	for _, bm := range bb.baremetals {
		desc, err := bb.buildOne(&bm, netGetter)
		if err != nil {
			log.Errorf("BaremetalBuilder error: %v", err)
			continue
		}
		schedDescs = append(schedDescs, desc)
	}
	return schedDescs, nil
}

func (bb *BaremetalBuilder) buildOne(hostObj *computemodels.SHost, netGetter *networkGetter) (interface{}, error) {
	baseDesc, err := newBaseHostDesc(bb.baseBuilder, hostObj, netGetter)
	if err != nil {
		return nil, err
	}
	desc := &BaremetalDesc{
		BaseHostDesc: baseDesc,
	}

	desc.StorageDriver = hostObj.StorageDriver
	desc.StorageType = hostObj.StorageType
	desc.StorageSize = int64(hostObj.StorageSize)

	baremetalStorages := computemodels.ConvertStorageInfo2BaremetalStorages(hostObj.StorageInfo)
	desc.StorageInfo = baremetalStorages
	desc.Tenants = make(map[string]int64, 0)

	err = bb.fillServerID(desc, hostObj)
	if err != nil {
		return nil, err
	}

	return desc, nil
}

func (bb *BaremetalBuilder) fillServerID(desc *BaremetalDesc, b *computemodels.SHost) error {
	guest := b.GetBaremetalServer()
	srvId := ""
	if guest != nil {
		srvId = guest.GetId()
	}
	desc.ServerID = srvId

	return nil
}
