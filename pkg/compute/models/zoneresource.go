package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SZoneResourceBase struct {
	ZoneId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
}

func (self *SZoneResourceBase) GetZone() *SZone {
	return ZoneManager.FetchZoneById(self.ZoneId)
}

func (self *SZoneResourceBase) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	zone := self.GetZone()
	if zone == nil {
		return nil
	}
	info := map[string]string{
		"zone":    zone.GetName(),
		"zone_id": zone.GetId(),
	}
	if len(zone.ExternalId) > 0 {
		info["zone_ext_id"] = fetchExternalId(zone.ExternalId)
	}
	if region := zone.GetRegion(); region != nil {
		info["region"] = region.GetName()
		info["region_id"] = region.GetId()
		if len(region.ExternalId) > 0 {
			info["region_ext_id"] = fetchExternalId(region.ExternalId)
		}
	}
	return jsonutils.Marshal(info).(*jsonutils.JSONDict)
}
