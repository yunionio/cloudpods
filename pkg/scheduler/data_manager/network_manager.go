package data_manager

import (
	"sync"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/scheduler/cache"
	synccache "yunion.io/x/onecloud/pkg/scheduler/cache/sync"
	networks_db "yunion.io/x/onecloud/pkg/scheduler/cache/sync/networks/db"
)

// ---------------------------------------------------

type CandidateIdMap map[string]int

type VpcNetwork struct {
	Data    CandidateIdMap
	Network *synccache.SchedNetworkBuildResult
}

func NewVpcNetwork() *VpcNetwork {
	return &VpcNetwork{Data: make(CandidateIdMap)}
}

// ---------------------------------------------------

type VpcNetworks struct {
	Data           map[string]*VpcNetwork
	vpcNetworkLock sync.Mutex
}

func NewVpcNetworks() *VpcNetworks {
	return &VpcNetworks{
		Data:           make(map[string]*VpcNetwork),
		vpcNetworkLock: sync.Mutex{},
	}
}

func (vns *VpcNetworks) Append(candidateId string,
	networks []*synccache.SchedNetworkBuildResult) {

	vns.vpcNetworkLock.Lock()
	defer vns.vpcNetworkLock.Unlock()

	for _, network := range networks {
		for _, idx := range []string{network.ID, network.Name} {
			vpcNetwork, ok := vns.Data[idx]
			if !ok {
				vpcNetwork = NewVpcNetwork()
				vns.Data[idx] = vpcNetwork
			}

			vpcNetwork.Network = network
			vpcNetwork.Data[candidateId] = 0
		}
	}
}

func (vns *VpcNetworks) Exists(networkId, candidateId string,
) *synccache.SchedNetworkBuildResult {
	vns.vpcNetworkLock.Lock()
	defer vns.vpcNetworkLock.Unlock()

	if vpcNetwork, ok := vns.Data[networkId]; ok {
		if _, ok := vpcNetwork.Data[candidateId]; ok {
			return vpcNetwork.Network
		}
	}

	return nil
}

func (vns *VpcNetworks) Get(networkId string) *VpcNetwork {
	vns.vpcNetworkLock.Lock()
	defer vns.vpcNetworkLock.Unlock()

	if vpcNetwork, ok := vns.Data[networkId]; ok {
		return vpcNetwork
	}

	return nil
}

// ---------------------------------------------------

type NetworkManager struct {
	dataManager  *DataManager
	vpcNetworks  *VpcNetworks
	networksPool *ReservedPool
}

func NewNetworkManager(dataManager *DataManager, reservedPoolManager *ReservedPoolManager) *NetworkManager {
	networksPool, err := reservedPoolManager.GetPool("networks")
	if err != nil {
		log.Errorln(err)
	}

	return &NetworkManager{
		dataManager:  dataManager,
		vpcNetworks:  NewVpcNetworks(),
		networksPool: networksPool,
	}
}

func (m *NetworkManager) CleanVpc() {
	m.vpcNetworks = NewVpcNetworks()
}

func (m *NetworkManager) IsUnknown(id string) bool {
	return false
}

func (m *NetworkManager) GetReservecPorts(id string) int64 {
	// TODO: impl reserve network resource
	//if m.networksPool.GetReservedItem(id) != nil {
	//return m.networksPool.GetReservedItem(id).Get("Ports", int64(0)).(int64)
	//} else {
	return 0
	//}
}

func (m *NetworkManager) LoadUnknownNetworks(ids []string) {
	if len(ids) == 0 {
		return
	}

	builder, err := m.getHostNetworkDBDescBuilder()
	if err != nil {
		log.Errorf("Reload network error: %v", err)
		return
	}

	builder.Load(ids)
}

func (m *NetworkManager) GetVpcNetwork(networkId string) *VpcNetwork {
	return m.vpcNetworks.Get(networkId)
}

func (m *NetworkManager) ExistsVpcNetwork(networkId, candidateId string,
) *synccache.SchedNetworkBuildResult {
	return m.vpcNetworks.Exists(networkId, candidateId)
}

func (m *NetworkManager) getHostNetworkDBDescBuilder() (networks_db.NetworkDescBuilder, error) {
	cache, err := m.getNetworkCache()
	if err != nil {
		return nil, err
	}

	builder, err := cache.Get(synccache.HostNetworkDescBuilderCache)
	if err != nil {
		return nil, err
	}

	return builder.(networks_db.NetworkDescBuilder), nil
}

func (m *NetworkManager) getNetworkCache() (cache.Cache, error) {
	cache, err := m.dataManager.SyncCacheGroup.Get(synccache.NetworkSyncCache)
	if err != nil {
		return nil, err
	}

	cache.WaitForReady()
	return cache, nil
}
