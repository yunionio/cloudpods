package sync

import (
	"github.com/yunionio/onecloud/pkg/scheduler/db/models"
	"github.com/yunionio/pkg/utils"
)

const (
	HostNetworkDescBuilderCache      = "HostNetworkDescBuilderCache"
	BaremetalNetworkDescBuilderCache = "BaremetalNetworkDescBuilderCache"
)

func avaliableAddress(network *models.WireNetwork) (int, error) {
	totalAddress := utils.IpRangeCount(network.GuestIpStart, network.GuestIpEnd)
	guestNicCount, err := models.GuestNicCountsWithNetworkID(network.ID)
	if err != nil {
		return 0, err
	}
	groupNicCount, err := models.GroupNicCountsWithNetworkID(network.ID)
	if err != nil {
		return 0, err
	}
	baremetalNicCount, err := models.BaremetalNicCountsWithNetworkID(network.ID)
	if err != nil {
		return 0, err
	}
	reserveNicCount, err := models.ReserveNicCountsWithNetworkID(network.ID)
	if err != nil {
		return 0, err
	}

	return totalAddress - guestNicCount.Count - groupNicCount.Count - baremetalNicCount.Count - reserveNicCount.Count, nil
}
