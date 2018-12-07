package candidate

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	cloudmodels "yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/db/models"
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
	models.BillingResourceBase

	ManagerID      *string                      `json:"manager_id"`
	Status         string                       `json:"status"`
	CPUCount       int64                        `json:"cpu_count"`
	MemSize        int64                        `json:"mem_size"`
	Networks       []*models.NetworkSchedResult `json:"networks"`
	HostStatus     string                       `json:"host_status"`
	Enabled        bool                         `json:"enabled"`
	HostType       string                       `json:"host_type"`
	IsBaremetal    bool                         `json:"is_baremetal"`
	IsMaintenance  bool                         `json:"is_maintenance"`
	NodeCount      int64                        `json:"node_count"`
	Tenants        map[string]int64             `json:"tenants"`
	ZoneID         string                       `json:"zone_id"`
	Zone           string                       `json:"zone"`
	PoolID         string                       `json:"pool_id"`
	ClusterID      string                       `json:"cluster_id"`
	Aggregates     []*models.Aggregate          `json:"aggregates"`
	HostAggregates []*models.Aggregate          `json:"host_aggregates"`
	Cloudprovider  *models.Cloudprovider        `json:"cloudprovider"`
	ResourceType   string                       `json:"resource_type"`
	RealExternalId string                       `json:"real_external_id"`
}

func reviseResourceType(resType string) string {
	if resType == "" {
		return cloudmodels.HostResourceTypeDefault
	}
	return resType
}

func newBaseHostDesc(host *models.Host) (*baseHostDesc, error) {
	desc := &baseHostDesc{
		baseDesc: baseDesc{
			ID:        host.ID,
			Name:      host.Name,
			UpdatedAt: host.UpdatedAt,
		},
		BillingResourceBase: host.BillingResourceBase,
		ManagerID:           host.ManagerID,
		Status:              host.Status,
		CPUCount:            host.CPUCount,
		MemSize:             host.MemSize,
		HostStatus:          host.HostStatus,
		Enabled:             host.Enabled,
		HostType:            host.HostType,
		IsBaremetal:         host.IsBaremetal,
		NodeCount:           host.NodeCount,
		IsMaintenance:       host.IsMaintenance,
		ResourceType:        reviseResourceType(host.ResourceType),
		RealExternalId:      host.RealExternalId,
	}

	if err := desc.fillCloudProvider(host); err != nil {
		return nil, fmt.Errorf("Fill cloudprovider info error: %v", err)
	}

	if err := desc.fillNetworks(desc.ID); err != nil {
		return nil, fmt.Errorf("Fill networks error: %v", err)
	}

	if err := desc.fillZone(host); err != nil {
		return nil, fmt.Errorf("Fill zone error: %v", err)
	}

	if err := desc.fillResidentTenants(host); err != nil {
		return nil, fmt.Errorf("Fill resident tenants error: %v", err)
	}

	if err := desc.fillAggregates(); err != nil {
		return nil, fmt.Errorf("Fill schetags error: %v", err)
	}
	return desc, nil
}

func (b baseHostDesc) GetSchedDesc() *jsonutils.JSONDict {
	desc := jsonutils.NewDict()

	desc.Add(jsonutils.NewString(b.ID), "id")
	desc.Add(jsonutils.NewString(b.Name), "name")
	desc.Add(jsonutils.NewInt(b.CPUCount), "cpu_count")
	desc.Add(jsonutils.NewInt(b.MemSize), "mem_size")
	desc.Add(jsonutils.NewString(b.HostType), "host_type")
	desc.Add(jsonutils.NewString(b.ZoneID), "zone_id")
	desc.Add(jsonutils.NewString(b.Zone), "zone")
	desc.Add(jsonutils.NewString(b.ResourceType), "resource_type")

	if b.Cloudprovider != nil {
		p := b.Cloudprovider
		cloudproviderDesc := jsonutils.NewDict()
		cloudproviderDesc.Add(jsonutils.NewString(p.ProjectId), "tenant_id")
		cloudproviderDesc.Add(jsonutils.NewString(p.Provider), "provider")
		desc.Add(cloudproviderDesc, "cloudprovider")
	}

	return desc
}

func (b baseHostDesc) GetResourceType() string {
	return b.ResourceType
}

func (b *baseHostDesc) fillCloudProvider(host *models.Host) error {
	if host.ManagerID == nil {
		log.Debugf("Host %q manager id is empty, no cloud provider", host.Name)
		return nil
	}
	provider, err := models.FetchCloudproviderById(*(host.ManagerID))
	if err != nil {
		return err
	}
	b.Cloudprovider = provider
	return nil
}

func (b *baseHostDesc) fillZone(host *models.Host) error {
	zone, err := models.FetchZoneByID(host.ZoneID)
	if err != nil {
		return err
	}
	b.ZoneID = zone.ID
	b.Zone = zone.Name
	return nil
}

func (b *baseHostDesc) fillResidentTenants(host *models.Host) error {
	rets, err := HostResidentTenantCount(host.ID)
	if err != nil {
		return err
	}

	b.Tenants = rets

	return nil
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
