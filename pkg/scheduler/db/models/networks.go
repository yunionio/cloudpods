package models

import (
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"

	"yunion.io/x/pkg/utils"
)

const (
	NetworkResourceName = "network"

	GuestNicCountC      = "GuestNiCount"
	GroupNicCountC      = "GroupNicCount"
	BaremetalNicCountC  = "BaremetalNicCount"
	ReserveDipNicCountC = "ReserveDipNicCount"
)

type Network struct {
	StandaloneModel
	Status        string `json:"status,omitempty" gorm:"not null"`
	TenantID      string `json:"tenant_id,omitempty" gorm:"not null"`
	UserId        string `json:"user_id,omitempty" gorm:"not null"`
	IsPublic      int    `json:"is_public,omitempty" gorm:"not null"`
	GuestIpStart  string `json:"guest_ip_start,omitempty" gorm:"not null"`
	GuestIpEnd    string `json:"guest_ip_end,omitempty" gorm:"not null"`
	GuestIpMask   int    `json:"guest_ip_mask,omitempty" gorm:"not null"`
	GuestGateway  string `json:"guest_gateway,omitempty"`
	GuestDns      string `json:"guest_dns,omitempty"`
	GuestIp6Start string `json:"guest_ip6_start,omitempty"`
	GuestIp6End   string `json:"guest_ip6_end,omitempty"`
	GuestIp6Mask  int    `json:"guest_ip6_mask,omitempty"`
	GuestGateway6 string `json:"guest_gateway6,omitempty"`
	GuestDns6     string `json:"guest_dns6,omitempty"`
	GuestDomain6  string `json:"guest_domain6,omitempty"`
	VlanId        int64  `json:"vlan_id,omitempty" gorm:"not null"`
	DhcpHostId    string `json:"dhcp_host_id,omitempty"`
	WireID        string `json:"wire_id,omitempty"`
	IsChanged     int    `json:"is_changed,omitempty" gorm:"not null"`
	IsSystem      int    `json:"is_system,omitempty"`
	GuestDhcp     string `json:"guest_dhcp,omitempty"`
	BillingType   string `json:"billing_type,omitempty"`
	ServerType    string `json:"server_type,omitempty"`
	VpcId         string `json:"vpc_id,omitempty"`
	ZoneId        string `json:"zone_id,omitempty"`
	AcSubnetId    string `json:"ac_subnet_id,omitempty"`
}

func (n Network) TableName() string {
	return networksTable
}

func (n Network) String() string {
	str, _ := JsonString(n)
	return str
}

func NewNetworksResource(db *gorm.DB) (Resourcer, error) {
	model := func() interface{} {
		return &Network{}
	}
	models := func() interface{} {
		networks := []Network{}
		return &networks
	}

	return newResource(db, networksTable, model, models)
}

func SelectNetworksWithByWireIDs(wireIDs []string) ([]WireNetwork, error) {
	networks := []WireNetwork{}
	err := Networks.DB().Table(networksTable).
		Select("distinct id").
		Where(fmt.Sprintf("wire_id in ('%s') and deleted=0", strings.Join(wireIDs, "','"))).
		Scan(&networks).Error

	return networks, err
}

func SelectWireIDsHasNetworks() ([]WireNetwork, error) {
	networks := []WireNetwork{}
	err := Networks.DB().Table(networksTable).
		Select("id,wire_id").
		Where("deleted=0").
		Scan(&networks).Error

	return networks, err
}

type WireNetwork struct {
	ID           string `json:"id,omitempty" gorm:"not null"`
	TenantID     string `json:"tenant_id,omitempty" gorm:"not null"`
	GuestIpStart string `json:"guest_ip_start,omitempty" gorm:"not null"`
	GuestIpEnd   string `json:"guest_ip_end,omitempty" gorm:"not null"`
	IsPublic     int    `json:"is_public,omitempty" gorm:"not null"`
	WireID       string `json:"wire_id,omitempty"`
	ServerType   string `json:"server_type,omitempty"`
}

func (c WireNetwork) First() string {
	return c.ID
}

func SelectNetworksWithByWireIDsi(wireIDs []string) ([]WireNetwork, error) {
	networks := []WireNetwork{}
	err := Networks.DB().Table(networksTable).
		Select("distinct id,wire_id,tenant_id,is_public,server_type,guest_ip_start,guest_ip_end").
		Where(fmt.Sprintf("wire_id in ('%s') and deleted=0", strings.Join(wireIDs, "','"))).
		Scan(&networks).Error
	return networks, err
}

type NetworkSchedResult struct {
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

func HostNetworkSchedResults(hostID string) ([]*NetworkSchedResult, error) {
	hostAndWires, err := SelectWiresWithHostID(hostID)
	if err != nil {
		return nil, err
	}

	if len(hostAndWires) == 0 {
		return nil, fmt.Errorf("Host %q not in wire.", hostID)
	}

	wireIDs := []string{}
	for _, hostWire := range hostAndWires {
		wireIDs = append(wireIDs, hostWire.WireID)
	}

	hostNets, err := FetchByWireIDs(Networks, wireIDs)
	if err != nil {
		return nil, err
	}

	netRes := []*NetworkSchedResult{}
	for _, n := range hostNets {
		r, err := NewNetworkSchedResult(n.(*Network))
		if err != nil {
			return nil, fmt.Errorf("NewNetworkBuildResult err: %v", err)
		}
		netRes = append(netRes, r)
	}
	return netRes, nil
}

func NewNetworkSchedResult(net *Network) (*NetworkSchedResult, error) {
	if net == nil {
		return nil, fmt.Errorf("empty network model resource")
	}

	wire, err := FetchByID(Wires, net.WireID)
	if err != nil {
		return nil, fmt.Errorf("fetch wire %q err: %v", net.WireID, err)
	}

	res := &NetworkSchedResult{
		ID:         net.ID,
		WireID:     net.WireID,
		Name:       net.Name,
		Wire:       wire.(*Wire).Name,
		TenantID:   net.TenantID,
		ServerType: net.ServerType,
		IsExit:     utils.IsExitAddress(net.GuestIpStart),
	}
	res.IsPublic = net.IsPublic == 1
	ports, err := NetworkAvaliableAddress(net)
	if err != nil {
		return nil, err
	}
	res.Ports = ports
	return res, nil
}

func NetworkAvaliableAddress(net *Network) (ports int, err error) {
	totalAddress := utils.IpRangeCount(net.GuestIpStart, net.GuestIpEnd)
	guestNicCount, err := NicCount(GuestNicCountC)
	if err != nil {
		return
	}

	groupNicCount, err := NicCount(GroupNicCountC)
	if err != nil {
		return
	}

	baremetalNicCount, err := NicCount(BaremetalNicCountC)
	if err != nil {
		return
	}

	reserveDipNicCount, err := NicCount(ReserveDipNicCountC)
	if err != nil {
		return
	}

	ports = totalAddress - guestNicCount[net.ID] - groupNicCount[net.ID] - baremetalNicCount[net.ID] - reserveDipNicCount[net.ID]
	return
}

func NicCount(nicName string) (map[string]int, error) {
	countsMap := make(map[string]int)
	switch nicName {
	case GuestNicCountC:
		counts, err := GuestNicCounts()
		if err != nil {
			return nil, err
		}

		for _, count := range counts {
			countsMap[count.NetworkID] = count.Count
		}
	case GroupNicCountC:
		counts, err := GroupNicCounts()
		if err != nil {
			return nil, err
		}

		for _, count := range counts {
			countsMap[count.NetworkID] = count.Count
		}

	case BaremetalNicCountC:
		counts, err := BaremetalNicCounts()
		if err != nil {
			return nil, err
		}

		for _, count := range counts {
			countsMap[count.NetworkID] = count.Count
		}

	case ReserveDipNicCountC:
		counts, err := ReserveNicCounts()
		if err != nil {
			return nil, err
		}

		for _, count := range counts {
			countsMap[count.NetworkID] = count.Count
		}
	}

	return countsMap, nil
}
