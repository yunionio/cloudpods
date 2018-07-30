package candidate

import (
	"time"

	"github.com/yunionio/log"
	"github.com/yunionio/onecloud/pkg/scheduler/api"
	"github.com/yunionio/onecloud/pkg/scheduler/db/models"
	"github.com/yunionio/pkg/utils"
)

type baseDesc struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (b *baseDesc) UUID() string {
	return b.ID
}

type baseHostDesc struct {
	baseDesc

	Status         string                       `json:"status"`
	CPUCount       int64                        `json:"cpu_count"`
	MemSize        int64                        `json:"mem_size"`
	Networks       []*models.NetworkSchedResult `json:"networks"`
	HostStatus     string                       `json:"host_status"`
	Enabled        bool                         `json:"enabled"`
	HostType       string                       `json:"host_type"`
	IsBaremetal    bool                         `json:"is_baremetal"`
	NodeCount      int64                        `json:"node_count"`
	Tenants        map[string]int64             `json:"tenants"`
	ZoneID         string                       `json:"zone_id"`
	PoolID         string                       `json:"pool_id"`
	ClusterID      string                       `json:"cluster_id"`
	Aggregates     []*models.Aggregate          `json:"aggregates"`
	HostAggregates []*models.Aggregate          `json:"host_aggregates"`
}

func (b *baseHostDesc) fillAggregates() error {
	b.Aggregates = make([]*models.Aggregate, 0)
	objs, err := models.All(models.Aggregates)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		agg := obj.(*models.Aggregate)
		b.Aggregates = append(b.Aggregates, agg)
	}

	aggs, err := models.HostAggregates(b.ID)
	if err != nil {
		return err
	}
	b.HostAggregates = aggs
	return nil
}

func (b *baseHostDesc) GetAggregates() []*models.Aggregate {
	return b.Aggregates
}

func (b *baseHostDesc) GetHostAggregates() []*models.Aggregate {
	return b.HostAggregates
}

func (b *baseHostDesc) fillNetworks(hostID string) error {
	net, err := models.HostNetworkSchedResults(hostID)
	if err != nil {
		return err
	}
	b.Networks = net
	return nil
}

func (h *baseHostDesc) GetEnableStatus() string {
	if h.Enabled {
		return "enable"
	}
	return "disable"
}

func (h *baseHostDesc) GetHostType() string {
	if h.HostType == api.HostTypeBaremetal && h.IsBaremetal {
		return api.HostTypeBaremetal
	}
	return h.HostType
}

func HostsResidentTenantStats(hostIDs []string) (map[string]map[string]interface{}, error) {
	residentTenantStats, err := models.ResidentTenantsInHosts(hostIDs)
	if err != nil {
		return nil, err
	}
	stat3 := make([]utils.StatItem3, len(residentTenantStats))
	for i, item := range residentTenantStats {
		stat3[i] = item
	}
	return utils.ToStatDict3(stat3)
}

func HostResidentTenantCount(id string) (map[string]int64, error) {
	residentTenantDict, err := HostsResidentTenantStats([]string{id})
	if err != nil {
		return nil, err
	}
	tenantMap, ok := residentTenantDict[id]
	if !ok {
		log.V(10).Infof("Not found host ID: %s when fill resident tenants, may be no guests on it.", id)
		return nil, nil
	}
	rets := make(map[string]int64, len(tenantMap))
	for tenantID, countObj := range tenantMap {
		rets[tenantID] = countObj.(int64)
	}
	return rets, nil
}

type DescBuilder struct {
	dbGroupCache   DBGroupCacher
	syncGroupCache SyncGroupCacher
	actor          BuildActor
}

func NewDescBuilder(db DBGroupCacher, sync SyncGroupCacher, act BuildActor) *DescBuilder {
	return &DescBuilder{
		dbGroupCache:   db,
		syncGroupCache: sync,
		actor:          act,
	}
}

func (d *DescBuilder) Build(ids []string) ([]interface{}, error) {
	return d.actor.Do(ids, d.dbGroupCache, d.syncGroupCache)
}
