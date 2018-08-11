package sync

import (
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/scheduler/db/models"
	"yunion.io/x/pkg/utils"
)

const (
	NetworksBuilderCache = "NetworksBuilderCache"

	GuestNiCount       = "GuestNiCount"
	GroupNicCount      = "GroupNicCount"
	BaremetalNicCount  = "BaremetalNicCount"
	ReserveDipNicCount = "ReserveDipNicCount"
)

type SchedNetworkBuildResult struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	TenantID   string `json:"tenant_id"`
	IsPublic   bool   `json:"is_public"`
	ServerType string `json:"server_type"`
	Ports      int    `json:"ports"`
	IsExit     bool   `json:"is_exit"`
	Wire       string `json:"wire_name"`
	WireID     string `json:"wire_id"`
}

func BuilderNetworkCacheKey(obj interface{}) (string, error) {
	builder, ok := obj.(NetworksBuilder)
	if !ok {
		return "", fmt.Errorf("Not a NetworksDataSyncCache: %v", obj)
	}

	return builder.GetKey(), nil
}

func updateNetworksBuilder(keys []string) ([]interface{}, error) {
	builders := make([]NetworksBuilder, 0)

	for _, key := range keys {
		switch key {
		case NetworksBuilderCache:
			builders = append(builders, NewNetworksBuilder())
		default:
			return nil, fmt.Errorf("Not support update, key: %v", key)
		}
	}

	ret := []interface{}{}
	for _, builder := range builders {
		_, err := builder.LoadAll()
		if err != nil {
			log.Errorf("Network load error: %v", err)
		}
		ret = append(ret, builder)
	}

	return ret, nil
}

type NetworksBuilder interface {
	LoadAll() (map[string]*SchedNetworkBuildResult, error)
	Load(ids []string) (map[string]*SchedNetworkBuildResult, error)
	GetKey() string
	GetNetworksData(ids []string) []*SchedNetworkBuildResult
}

func loadNetworksBuilder() ([]interface{}, error) {
	builders := []NetworksBuilder{
		NewNetworksBuilder(),
	}

	rets := []interface{}{}

	for _, builder := range builders {
		_, err := builder.LoadAll()
		if err != nil {
			log.Errorf("Network load error: %v", err)
		} else {
			rets = append(rets, builder)
		}
	}

	return rets, nil
}

type NetworksDataBuilder struct {
	GuestNicCount      map[string]int
	GroupNicCount      map[string]int
	BaremetalNicCount  map[string]int
	ReserveDipNicCount map[string]int
	Wires              map[string]string
	data               map[string]*SchedNetworkBuildResult
}

func NewNetworksBuilder() *NetworksDataBuilder {
	return &NetworksDataBuilder{
		GuestNicCount:      make(map[string]int),
		GroupNicCount:      make(map[string]int),
		BaremetalNicCount:  make(map[string]int),
		ReserveDipNicCount: make(map[string]int),
		data:               make(map[string]*SchedNetworkBuildResult),
	}
}

func (b *NetworksDataBuilder) GetKey() string {
	return NetworksBuilderCache
}

func (b *NetworksDataBuilder) LoadAll() (map[string]*SchedNetworkBuildResult, error) {
	return b.Load(nil)
}

func (b *NetworksDataBuilder) toSchedNetworkBuildResult(network *models.Network) (*SchedNetworkBuildResult, error) {
	if network == nil {
		return nil, fmt.Errorf("empty network resource.")
	}

	res := new(SchedNetworkBuildResult)
	res.WireID = network.WireID
	res.Wire = b.Wires[network.WireID]
	res.ID = network.ID
	ports, err := b.avaliableAddress(network)
	if err != nil {
		return nil, err
	} else {
		res.Ports = ports
	}
	res.Name = network.Name
	res.TenantID = network.TenantID
	res.IsPublic = network.IsPublic == 1
	res.ServerType = network.ServerType
	res.IsExit = utils.IsExitAddress(network.GuestIpStart)

	return res, nil
}

func (b *NetworksDataBuilder) avaliableAddress(network *models.Network) (int, error) {
	totalAddress := utils.IpRangeCount(network.GuestIpStart, network.GuestIpEnd)

	return totalAddress - b.GuestNicCount[network.ID] - b.GroupNicCount[network.ID] - b.BaremetalNicCount[network.ID] - b.ReserveDipNicCount[network.ID], nil
}

func getNiCount(nicName string) (map[string]int, error) {
	countsMap := make(map[string]int)
	switch nicName {
	case GuestNiCount:
		counts, err := models.GuestNicCounts()
		if err != nil {
			return nil, err
		}

		for _, count := range counts {
			countsMap[count.NetworkID] = count.Count
		}
	case GroupNicCount:
		counts, err := models.GroupNicCounts()
		if err != nil {
			return nil, err
		}

		for _, count := range counts {
			countsMap[count.NetworkID] = count.Count
		}

	case BaremetalNicCount:
		counts, err := models.BaremetalNicCounts()
		if err != nil {
			return nil, err
		}

		for _, count := range counts {
			countsMap[count.NetworkID] = count.Count
		}

	case ReserveDipNicCount:
		counts, err := models.ReserveNicCounts()
		if err != nil {
			return nil, err
		}

		for _, count := range counts {
			countsMap[count.NetworkID] = count.Count
		}
	}

	return countsMap, nil
}

func (b *NetworksDataBuilder) Load(ids []string) (map[string]*SchedNetworkBuildResult, error) {
	wireInfos, err := models.LoadAllWires()
	if err != nil {
		return nil, err
	}
	wiresMap := make(map[string]string, len(wireInfos))
	for _, wire := range wireInfos {
		wiresMap[wire.ID] = wire.Name
	}
	b.Wires = wiresMap

	b.GuestNicCount, err = getNiCount(GuestNiCount)
	if err != nil {
		log.Errorln(err)
	}
	b.GroupNicCount, err = getNiCount(GroupNicCount)
	if err != nil {
		log.Errorln(err)
	}
	b.BaremetalNicCount, err = getNiCount(BaremetalNicCount)
	if err != nil {
		log.Errorln(err)
	}
	b.ReserveDipNicCount, err = getNiCount(ReserveDipNicCount)
	if err != nil {
		log.Errorln(err)
	}

	networks, err := models.All(models.Networks)
	if err != nil {
		return nil, err
	}

	for _, network := range networks {
		n := network.(*models.Network)
		b.data[n.ID], err = b.toSchedNetworkBuildResult(n)
		if err != nil {
			log.Errorln(err)
		}
	}

	return b.data, nil
}

func (b *NetworksDataBuilder) GetNetworksData(ids []string) (networks []*SchedNetworkBuildResult) {
	for _, networkID := range ids {
		if network, ok := b.data[networkID]; ok {
			networks = append(networks, network)
		}
	}
	return
}
