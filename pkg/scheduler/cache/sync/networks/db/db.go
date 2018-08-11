package db

import (
	"fmt"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/scheduler/db/models"
)

const (
	HostNetworkDescBuilderCache      = "HostNetworkDescBuilderCache"
	BaremetalNetworkDescBuilderCache = "BaremetalNetworkDescBuilderCache"
)

type NetworkDescBuilder interface {
	LoadAll() (map[string][]string, error)
	Load(ids []string) (map[string][]string, error)
	GetKey() string
	GetNetworkDesc(id string) ([]string, error)
}

func BuilderCacheKey(obj interface{}) (string, error) {
	builder, ok := obj.(NetworkDescBuilder)
	if !ok {
		return "", fmt.Errorf("Not a NetworkDescBuilder: %v", obj)
	}

	return builder.GetKey(), nil
}

func LoadNetworkDescBuilder() ([]interface{}, error) {
	builders := []NetworkDescBuilder{
		NewHostNetworkDescBuilder(),
		//NewBaremetalNetworkDescBuilder(),
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

func UpdateNetworkDescBuilder(keys []string) ([]interface{}, error) {
	builders := make([]NetworkDescBuilder, 0)

	for _, key := range keys {
		switch key {
		case HostNetworkDescBuilderCache:
			builders = append(builders, NewHostNetworkDescBuilder())
		case BaremetalNetworkDescBuilderCache:
			builders = append(builders, NewBaremetalNetworkDescBuilder())
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

type HostNetworkDescBuilder struct {
	data          map[string][]string
	host2Wires    map[string]string
	wire2Networks map[string]string
}

func NewHostNetworkDescBuilder() *HostNetworkDescBuilder {
	return &HostNetworkDescBuilder{
		data:          make(map[string][]string),
		host2Wires:    make(map[string]string),
		wire2Networks: make(map[string]string),
	}
}

func (b *HostNetworkDescBuilder) GetKey() string {
	return HostNetworkDescBuilderCache
}

func (b *HostNetworkDescBuilder) LoadAll() (map[string][]string, error) {
	return b.Load(nil)
}

func (b *HostNetworkDescBuilder) Load(ids []string) (map[string][]string, error) {
	// wireInfos, err := models.LoadAllWires()
	// if err != nil {
	// 	return nil, err
	// }
	// wiresMap := make(map[string]string, len(wireInfos))
	// for _, wire := range wireInfos {
	// 	wiresMap[wire.ID] = wire.Name
	// }

	hostAndWires, err := models.SelectHostHasWires()
	if err != nil {
		return nil, err
	}
	hostHasWires := make(map[string]string)
	for _, hostAndWire := range hostAndWires {
		if _, ok := hostHasWires[hostAndWire.HostID]; ok {
			if !strings.Contains(hostHasWires[hostAndWire.HostID], hostAndWire.WireID) {
				hostHasWires[hostAndWire.HostID] = fmt.Sprintf("%s;%s", hostHasWires[hostAndWire.HostID], hostAndWire.WireID)
			}
		} else {
			hostHasWires[hostAndWire.HostID] = hostAndWire.WireID
		}
	}
	b.host2Wires = hostHasWires

	wiresAndNetworks, err := models.SelectWireIDsHasNetworks()
	if err != nil {
		return nil, err
	}
	wireHasNetworks := make(map[string]string)
	for _, wireAndNetwork := range wiresAndNetworks {
		if _, ok := wireHasNetworks[wireAndNetwork.WireID]; ok {
			if !strings.Contains(wireHasNetworks[wireAndNetwork.WireID], wireAndNetwork.ID) {
				wireHasNetworks[wireAndNetwork.WireID] = fmt.Sprintf("%s;%s", wireHasNetworks[wireAndNetwork.WireID], wireAndNetwork.ID)
			}
		} else {
			wireHasNetworks[wireAndNetwork.WireID] = wireAndNetwork.ID
		}
	}
	b.wire2Networks = wireHasNetworks

	if len(ids) == 0 {
		hostIDs, err := models.AllHostIDs()
		if err != nil {
			log.Errorln(err)
		}
		for _, hostID := range hostIDs {
			networkResults, err := b.loadNetworks(hostID)
			if err != nil {
				log.Errorln(err)
			} else {
				b.data[hostID] = networkResults
			}
		}
	} else {
		for _, hostID := range ids {
			networkResults, err := b.loadNetworks(hostID)
			if err != nil {
				log.Errorln(err)
			} else {
				b.data[hostID] = networkResults
			}
		}
	}

	return b.data, nil
}

func (b *HostNetworkDescBuilder) loadNetworks(hostID string) (networkResults []string, err error) {
	wires := strings.Split(b.host2Wires[hostID], ";")
	for _, wire := range wires {
		networkResults = append(networkResults, strings.Split(b.wire2Networks[wire], ";")...)
	}
	return networkResults, nil
}

func (b *HostNetworkDescBuilder) GetNetworkDesc(id string) ([]string, error) {
	if r, ok := b.data[id]; ok {
		return r, nil
	}

	return nil, fmt.Errorf("can not find networks")
}

type BaremetalNetworkDescBuilder struct {
	data            map[string][]string
	baremetal2Wires map[string]string
	wire2Networks   map[string]string
}

// TODO:we should not new a object every time, the map memory leak, if want map
// was GCed, you must set b.data = nil and so on.
func NewBaremetalNetworkDescBuilder() *BaremetalNetworkDescBuilder {
	return &BaremetalNetworkDescBuilder{
		data:            make(map[string][]string, 30000), // the max number of baremetal if about 27000.
		baremetal2Wires: make(map[string]string, 30000),
		wire2Networks:   make(map[string]string, 0),
	}
}

func (b *BaremetalNetworkDescBuilder) GetKey() string {
	return BaremetalNetworkDescBuilderCache
}

func (b *BaremetalNetworkDescBuilder) LoadAll() (map[string][]string, error) {
	return b.Load(nil)
}

func (b *BaremetalNetworkDescBuilder) Load(ids []string) (map[string][]string, error) {
	// wireInfos, err := models.LoadAllWires()
	// if err != nil {
	// 	return nil, err
	// }
	// wiresMap := make(map[string]string, len(wireInfos))
	// for _, wire := range wireInfos {
	// 	wiresMap[wire.ID] = wire.Name
	// }

	baremetalsAndWires, err := models.SelectWiresAndBaremetals()
	if err != nil {
		return nil, err
	}
	baremetalHasWires := make(map[string]string)
	for _, baremetalsAndWire := range baremetalsAndWires {
		if _, ok := baremetalHasWires[baremetalsAndWire.BaremetalID]; ok {
			if !strings.Contains(baremetalHasWires[baremetalsAndWire.BaremetalID], baremetalsAndWire.WireID) {
				baremetalHasWires[baremetalsAndWire.BaremetalID] = fmt.Sprintf("%s;%s", baremetalHasWires[baremetalsAndWire.BaremetalID], baremetalsAndWire.WireID)
			}
		} else {
			baremetalHasWires[baremetalsAndWire.BaremetalID] = baremetalsAndWire.WireID
		}
	}
	b.baremetal2Wires = baremetalHasWires

	wiresAndNetworks, err := models.SelectWireIDsHasNetworks()
	if err != nil {
		return nil, err
	}
	wireHasNetworks := make(map[string]string)
	for _, wireAndNetwork := range wiresAndNetworks {
		if _, ok := wireHasNetworks[wireAndNetwork.WireID]; ok {
			if !strings.Contains(wireHasNetworks[wireAndNetwork.WireID], wireAndNetwork.ID) {
				wireHasNetworks[wireAndNetwork.WireID] = fmt.Sprintf("%s;%s", wireHasNetworks[wireAndNetwork.WireID], wireAndNetwork.ID)
			}
		} else {
			wireHasNetworks[wireAndNetwork.WireID] = wireAndNetwork.ID
		}
	}
	b.wire2Networks = wireHasNetworks

	if len(ids) == 0 {
		baremetalIDs, err := models.AllBaremetalIDs()
		if err != nil {
			log.Errorln(err)
		}

		for _, baremetalID := range baremetalIDs {
			networkResults, err := b.loadNetworks(baremetalID)
			if err != nil {
				log.Errorln(err)
			} else {
				b.data[baremetalID] = networkResults
			}
		}
	} else {
		for _, baremetalID := range ids {
			networkResults, err := b.loadNetworks(baremetalID)
			if err != nil {
				log.Errorln(err)
			} else {
				b.data[baremetalID] = networkResults
			}
		}
	}

	return b.data, nil
}

func (b *BaremetalNetworkDescBuilder) loadNetworks(baremetalID string) (networkResults []string, err error) {
	wires := strings.Split(b.baremetal2Wires[baremetalID], ";")
	for _, wire := range wires {
		networkResults = append(networkResults, strings.Split(b.wire2Networks[wire], ";")...)
	}
	return networkResults, nil
}

func (b *BaremetalNetworkDescBuilder) GetNetworkDesc(id string) ([]string, error) {
	if result, ok := b.data[id]; ok {
		return result, nil
	}

	return nil, fmt.Errorf("can not find networks")
}
