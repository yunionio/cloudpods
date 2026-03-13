package cucloud

import (
	"net/url"

	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDisk struct {
	multicloud.SResourceBase
}

func (region *SRegion) GetDisks() ([]SDisk, error) {
	params := url.Values{}
	params.Set("cloudRegionCode", region.CloudRegionCode)
	resp, err := region.list("/instance/v1/product/bss/cloudstorage", params)
	if err != nil {
		return nil, err
	}
	ret := []SDisk{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
