package models

import (
	"context"
	"sort"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
)

func (h *SHost) GetDetailsTapConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (api.SHostTapConfig, error) {
	conf := api.SHostTapConfig{}

	srvs, err := NetTapServiceManager.getEnabledTapServiceOnHost(h.Id)
	if err != nil {
		return conf, errors.Wrap(err, "NetTapServiceManager.getEnabledTapServiceOnHost")
	}
	for _, srv := range srvs {
		tapConf, err := srv.getConfig()
		if err != nil {
			return conf, errors.Wrap(err, "srv.getConfig")
		}
		conf.Taps = append(conf.Taps, tapConf)
	}

	flows, err := NetTapFlowManager.getEnabledTapFlowsOnHost(h.Id)
	if err != nil {
		return conf, errors.Wrap(err, "NetTapFlowManager.getEnabledTapFlowsOnHost")
	}
	mirrors := make([]api.SMirrorConfig, 0)
	for _, flow := range flows {
		mirror, err := flow.getMirrorConfig(true)
		if err != nil {
			if errors.Cause(err) == errors.ErrNotFound {
				continue
			} else {
				return conf, errors.Wrap(err, "flow.getMirrorConfig")
			}
		}
		mirrors = append(mirrors, mirror)
	}
	sort.Sort(sMirrorConfigs(mirrors))
	conf.Mirrors = mirrors // groupMirrorConfig(mirrors)

	return conf, nil
}

func (g *SGuest) getTapNicJsonDesc(ctx context.Context, p *api.GuestnetworkJsonDesc) *api.GuestnetworkJsonDesc {
	srv := NetTapServiceManager.getEnabledTapServiceByGuestId(g.Id)
	if srv == nil {
		return nil
	}
	var driver string
	var index int8
	if p == nil {
		driver = "virtio"
		index = 0
	} else {
		driver = p.Driver
		index = p.Index + 1
	}
	desc := &api.GuestnetworkJsonDesc{
		Mac:       srv.MacAddr,
		Virtual:   true,
		Ifname:    srv.Ifname,
		Driver:    driver,
		NumQueues: 1,
		Index:     index,
		Bridge:    api.HostTapBridge,
	}
	return desc
}
