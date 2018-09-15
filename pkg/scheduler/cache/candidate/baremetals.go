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
)

type BaremetalDesc struct {
	baseHostDesc

	Storages      []*baremetal.BaremetalStorage `json:"storages"`
	StorageType   string                        `json:"storage_type"`
	StorageSize   int64                         `json:"storage_size"`
	StorageInfo   string                        `json:"storage_info"`
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
		return bd.ID

	case "Name":
		return bd.Name

	case "Status":
		return bd.Status

	case "PoolID":
		return bd.PoolID

	case "ZoneID":
		return bd.ZoneID

	case "ServerID":
		return bd.ServerID

	case "CPUCount":
		return int64(bd.CPUCount)

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

func (bd *BaremetalDesc) XGet(key string, kind core.Kind) interface{} {
	return core.XGetCalculator(bd, key, kind)
}

func (bd *BaremetalDesc) IndexKey() string {
	return bd.ID
}

func (bd *BaremetalDesc) FreeCPUCount() int64 {
	if bd.ServerID == "" {
		return bd.CPUCount
	}
	return 0
}

func (bd *BaremetalDesc) FreeMemSize() int64 {
	if bd.ServerID == "" {
		return bd.MemSize
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
	desc := new(BaremetalDesc)
	desc.ID = bm.ID
	desc.Name = bm.Name
	desc.UpdatedAt = bm.UpdatedAt
	desc.Status = bm.Status
	desc.CPUCount = bm.CPUCount
	desc.NodeCount = bm.NodeCount
	desc.MemSize = int64(bm.MemSize)
	desc.StorageDriver = bm.StorageDriver
	desc.StorageType = bm.StorageType
	desc.StorageSize = int64(bm.StorageSize)
	desc.StorageInfo = bm.StorageInfo
	desc.PoolID = bm.PoolID
	desc.ZoneID = bb.getZoneID(bm)
	desc.Enabled = bm.Enabled
	desc.ClusterID = bm.ClusterID

	desc.HostStatus = bm.HostStatus
	desc.Enabled = bm.Enabled
	desc.HostType = bm.HostType
	desc.IsBaremetal = bm.IsBaremetal

	var baremetalStorages []*baremetal.BaremetalStorage
	err := fjson.Unmarshal([]byte(bm.StorageInfo), &baremetalStorages)
	if err != nil {
		// StorageInfo maybe is NULL
		if bm.StorageInfo != "" {
			log.Errorln(err)
		}
	}
	desc.Storages = baremetalStorages
	desc.Tenants = make(map[string]int64, 0)

	err = bb.fillServerID(desc, bm)
	if err != nil {
		return nil, err
	}

	err = bb.fillResidentTenants(desc, bm)
	if err != nil {
		return nil, err
	}

	// data from db
	err = bb.fillNetworks(desc, bm)
	if err != nil {
		return nil, err
	}

	err = desc.fillAggregates()
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

func (b *BaremetalBuilder) fillResidentTenants(desc *BaremetalDesc, host *models.Host) error {
	rets, err := HostResidentTenantCount(host.ID)
	if err != nil {
		return err
	}

	desc.Tenants = rets

	return nil
}

func (b *BaremetalBuilder) getZoneID(bm *models.Host) string {
	return bm.ZoneID
}
