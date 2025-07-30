package cucloud

import (
	"net/url"

	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SStorage struct {
	multicloud.SResourceBase
}

func (region *SRegion) GetStorages() ([]SStorage, error) {
	params := url.Values{}
	params.Set("cloudRegionCode", region.CloudRegionCode)
	resp, err := region.list("/instance/v1/product/bss/diskTypes", params)
	if err != nil {
		return nil, err
	}
	ret := []SStorage{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
