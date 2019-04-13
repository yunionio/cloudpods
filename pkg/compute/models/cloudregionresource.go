package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SCloudregionResourceBase struct {
	CloudregionId string `width:"36" charset:"ascii" nullable:"false" list:"admin" default:"default" create:"optional"`
}

func (self *SCloudregionResourceBase) GetRegion() *SCloudregion {
	region, err := CloudregionManager.FetchById(self.CloudregionId)
	if err != nil {
		log.Errorf("failed to find cloudregion %s error: %v", self.CloudregionId, err)
		return nil
	}
	return region.(*SCloudregion)
}

func (self *SCloudregionResourceBase) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	region := self.GetRegion()
	if region == nil {
		return nil
	}
	info := map[string]string{
		"region":    region.GetName(),
		"region_id": region.GetId(),
	}
	if len(region.ExternalId) > 0 {
		info["region_ext_id"] = fetchExternalId(region.ExternalId)
	}
	return jsonutils.Marshal(info).(*jsonutils.JSONDict)
}
