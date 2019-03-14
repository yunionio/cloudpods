package candidate

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/scheduler/core"
	"yunion.io/x/onecloud/pkg/scheduler/db/models"

	computedb "yunion.io/x/onecloud/pkg/cloudcommon/db"
	computemodels "yunion.io/x/onecloud/pkg/compute/models"
)

type BaseHostDesc struct {
	*computemodels.SHost
	Region        *computemodels.SCloudregion   `json:"region"`
	Zone          *computemodels.SZone          `json:"zone"`
	Cloudprovider *computemodels.SCloudprovider `json:"cloudprovider"`
	Networks      []computemodels.SNetwork      `json:"networks"`
	Storages      []*api.CandidateStorage       `json:"storages"`

	Tenants       map[string]int64          `json:"tenants"`
	HostSchedtags []computemodels.SSchedtag `json:"schedtags"`
}

type baseHostGetter struct {
	h *BaseHostDesc
}

func newBaseHostGetter(h *BaseHostDesc) *baseHostGetter {
	return &baseHostGetter{h}
}

func (b *BaseHostDesc) Getter() core.CandidatePropertyGetter {
	return newBaseHostGetter(b)
}

func (b baseHostGetter) Id() string {
	return b.h.GetId()
}

func (b baseHostGetter) Name() string {
	return b.h.GetName()
}

func (b baseHostGetter) Zone() *computemodels.SZone {
	return b.h.Zone
}

func (b baseHostGetter) Region() *computemodels.SCloudregion {
	return b.h.Region
}

func (b baseHostGetter) HostType() string {
	return b.h.HostType
}

func (b baseHostGetter) HostSchedtags() []computemodels.SSchedtag {
	return b.h.HostSchedtags
}

func (b baseHostGetter) Storages() []*api.CandidateStorage {
	return b.h.Storages
}

func reviseResourceType(resType string) string {
	if resType == "" {
		return computemodels.HostResourceTypeDefault
	}
	return resType
}

func newBaseHostDesc(host *computemodels.SHost) (*BaseHostDesc, error) {
	host.ResourceType = reviseResourceType(host.ResourceType)
	desc := &BaseHostDesc{
		SHost: host,
	}

	if err := desc.fillCloudProvider(host); err != nil {
		return nil, fmt.Errorf("Fill cloudprovider info error: %v", err)
	}

	if err := desc.fillNetworks(desc.Id); err != nil {
		return nil, fmt.Errorf("Fill networks error: %v", err)
	}

	if err := desc.fillZone(host); err != nil {
		return nil, fmt.Errorf("Fill zone error: %v", err)
	}

	if err := desc.fillRegion(host); err != nil {
		return nil, fmt.Errorf("Fill region error: %v", err)
	}

	if err := desc.fillResidentTenants(host); err != nil {
		return nil, fmt.Errorf("Fill resident tenants error: %v", err)
	}

	return desc, nil
}

func (b BaseHostDesc) GetSchedDesc() *jsonutils.JSONDict {
	desc := jsonutils.Marshal(b.SHost).(*jsonutils.JSONDict)

	if b.Cloudprovider != nil {
		p := b.Cloudprovider
		cloudproviderDesc := jsonutils.NewDict()
		cloudproviderDesc.Add(jsonutils.NewString(p.ProjectId), "tenant_id")
		cloudproviderDesc.Add(jsonutils.NewString(p.Provider), "provider")
		desc.Add(cloudproviderDesc, "cloudprovider")
	}

	return desc
}

func (b BaseHostDesc) GetResourceType() string {
	return b.ResourceType
}

func (b *BaseHostDesc) fillCloudProvider(host *computemodels.SHost) error {
	b.Cloudprovider = host.GetCloudprovider()
	return nil
}

func (b *BaseHostDesc) fillRegion(host *computemodels.SHost) error {
	b.Region = host.GetRegion()
	return nil
}

func (b *BaseHostDesc) fillZone(host *computemodels.SHost) error {
	zone := host.GetZone()
	b.Zone = zone
	b.ZoneId = host.ZoneId
	return nil
}

func (b *BaseHostDesc) fillResidentTenants(host *computemodels.SHost) error {
	rets, err := HostResidentTenantCount(host.Id)
	if err != nil {
		return err
	}

	b.Tenants = rets

	return nil
}

func (b *BaseHostDesc) fillSchedtags(allSchedtags []computemodels.SSchedtag) error {
	//b.Schedtags = allSchedtags
	b.HostSchedtags = b.SHost.GetSchedtags()
	return nil
}

func (b *BaseHostDesc) fillNetworks(hostID string) error {
	hostwires := computemodels.HostwireManager.Query().SubQuery()
	sq := hostwires.Query(sqlchemy.DISTINCT("wire_id", hostwires.Field("wire_id"))).Equals("host_id", hostID)
	networks := computemodels.NetworkManager.Query().SubQuery()
	q := networks.Query().In("wire_id", sq)

	nets := make([]computemodels.SNetwork, 0)
	err := computedb.FetchModelObjects(computemodels.NetworkManager, q, &nets)

	if err != nil {
		return err
	}
	b.Networks = nets
	return nil
}

func (b *BaseHostDesc) fillStorages(host *computemodels.SHost) error {
	ss := make([]*api.CandidateStorage, 0)
	//log.Errorf("====host %s append storages: %#v", b.Name, storages)
	for _, s := range host.GetHoststorages() {
		storage := s.GetStorage()
		ss = append(ss, &api.CandidateStorage{
			SStorage:  storage,
			Schedtags: storage.GetSchedtags(),
		})
	}
	b.Storages = ss
	return nil
}

func (h *BaseHostDesc) GetEnableStatus() string {
	if h.Enabled {
		return "enable"
	}
	return "disable"
}

func (h *BaseHostDesc) GetHostType() string {
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
