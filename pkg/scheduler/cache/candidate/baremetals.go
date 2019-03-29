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

	fjson "github.com/json-iterator/go"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/compute/baremetal"
	"yunion.io/x/onecloud/pkg/scheduler/cache"
	"yunion.io/x/onecloud/pkg/scheduler/cache/db"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	"yunion.io/x/onecloud/pkg/scheduler/db/models"

	computemodels "yunion.io/x/onecloud/pkg/compute/models"
)

type BaremetalDesc struct {
	*BaseHostDesc

	StorageInfo   []*baremetal.BaremetalStorage `json:"storage_info"`
	StorageType   string                        `json:"storage_type"`
	StorageSize   int64                         `json:"storage_size"`
	StorageDriver string                        `json:"storage_driver"`
	ServerID      string                        `json:"server_id"`
}

type BaremetalBuilder struct {
	baremetalAgents cache.Cache
	baremetals      []interface{}

	residentTenantDict map[string]map[string]interface{}
}

func (bd *BaremetalDesc) String() string {
	s, _ := fjson.Marshal(bd)
	return string(s)
}

func (bd *BaremetalDesc) Type() int {
	// Baremetal type
	return 1
}

func (bd *BaremetalDesc) Get(key string) interface{} {
	switch key {
	case "ID":
		return bd.Id

	case "Name":
		return bd.Name

	case "Status":
		return bd.Status

	case "ZoneID":
		return bd.ZoneId

	case "ServerID":
		return bd.ServerID

	case "CPUCount":
		return int64(bd.CpuCount)

	case "FreeCPUCount":
		return bd.FreeCPUCount()

	case "NodeCount":
		return int64(bd.NodeCount)

	case "MemSize":
		return bd.MemSize

	case "FreeMemSize":
		return bd.FreeMemSize()

	case "Storages":
		return bd.StorageType

	case "StorageSize":
		return bd.StorageSize

	case "StorageType":
		return bd.StorageType

	case "StorageInfo":
		return bd.StorageInfo

	case "StorageDriver":
		return bd.StorageDriver

	case "FreeStorageSize":
		return bd.FreeStorageSize()

	case "HostStatus":
		return bd.HostStatus

	default:
		return nil
	}
}

func (bd *BaremetalDesc) GetGuestCount() int64 {
	if bd.ServerID == "" {
		return 0
	}
	return 1
}

func (bd *BaremetalDesc) XGet(key string, kind core.Kind) interface{} {
	return core.XGetCalculator(bd, key, kind)
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

func (bb *BaremetalBuilder) init(ids []string, dbCache DBGroupCacher, syncCache SyncGroupCacher) error {
	agents, err := dbCache.Get(db.BaremetalAgentDBCache)
	if err != nil {
		return err
	}

	bms, err := models.FetchBaremetalHostByIDs(ids)
	if err != nil {
		return err
	}

	bb.baremetalAgents = agents
	bb.baremetals = bms

	wg := &WaitGroupWrapper{}
	errMessageChannel := make(chan error, 2)
	defer close(errMessageChannel)

	setFuncs := []func(){}

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
	return &BaremetalBuilder{}
}

func (bb *BaremetalBuilder) Type() string {
	return BaremetalDescBuilder
}

func (bb *BaremetalBuilder) AllIDs() ([]string, error) {
	return models.AllBaremetalIDs()
}

func (bb *BaremetalBuilder) Do(ids []string, dbCache DBGroupCacher, syncCache SyncGroupCacher) ([]interface{}, error) {
	err := bb.init(ids, dbCache, syncCache)
	if err != nil {
		return nil, err
	}

	descs, err := bb.build()
	if err != nil {
		return nil, err
	}
	return descs, nil
}

func (bb *BaremetalBuilder) build() ([]interface{}, error) {
	schedDescs := []interface{}{}
	for _, bm := range bb.baremetals {
		desc, err := bb.buildOne(bm.(*models.Host))
		if err != nil {
			log.Errorf("BaremetalBuilder error: %v", err)
			continue
		}
		schedDescs = append(schedDescs, desc)
	}
	return schedDescs, nil
}

func (bb *BaremetalBuilder) buildOne(bm *models.Host) (interface{}, error) {
	hostObj := computemodels.HostManager.FetchHostById(bm.ID)
	baseDesc, err := newBaseHostDesc(hostObj)
	if err != nil {
		return nil, err
	}
	desc := &BaremetalDesc{
		BaseHostDesc: baseDesc,
	}

	desc.StorageDriver = bm.StorageDriver
	desc.StorageType = bm.StorageType
	desc.StorageSize = int64(bm.StorageSize)

	var baremetalStorages []*baremetal.BaremetalStorage
	err = fjson.Unmarshal([]byte(bm.StorageInfo), &baremetalStorages)
	if err != nil {
		// StorageInfo maybe is NULL
		if bm.StorageInfo != "" {
			log.Errorln(err)
		}
	}
	desc.StorageInfo = baremetalStorages
	desc.Tenants = make(map[string]int64, 0)

	err = bb.fillServerID(desc, bm)
	if err != nil {
		return nil, err
	}

	return desc, nil
}

func (bb *BaremetalBuilder) fillServerID(desc *BaremetalDesc, b *models.Host) error {
	guests, err := models.FetchGuestByHostIDsWithCond([]string{b.ID},
		map[string]interface{}{
			"hypervisor": "baremetal",
		})
	if err != nil {
		return err
	}

	if len(guests) == 0 {
		desc.ServerID = ""
	} else if len(guests) == 1 {
		desc.ServerID = guests[0].(*models.Guest).ID
	} else {
		return fmt.Errorf("One baremetal %q contains %d guests, %v", b.Name, len(guests), guests)
	}

	return nil
}

func (bb *BaremetalBuilder) fillNetworks(desc *BaremetalDesc, b *models.Host) error {
	return desc.fillNetworks(b.ID)
}

func (b *BaremetalBuilder) getZoneID(bm *models.Host) string {
	return bm.ZoneID
}
