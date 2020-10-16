package qcloud

import (
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SElasticcacheDeal struct {
	region *SRegion

	DealID      string   `json:"DealId"`
	DealName    string   `json:"DealName"`
	ZoneID      int64    `json:"ZoneId"`
	GoodsNum    int64    `json:"GoodsNum"`
	Creater     string   `json:"Creater"`
	CreatTime   string   `json:"CreatTime"`
	OverdueTime string   `json:"OverdueTime"`
	EndTime     string   `json:"EndTime"`
	Status      int64    `json:"Status"`
	Description string   `json:"Description"`
	Price       int64    `json:"Price"`
	InstanceIDS []string `json:"InstanceIds"`
}

// https://cloud.tencent.com/document/product/239/30602
func (self *SRegion) GetElasticcacheIdByDeal(dealId string) (string, error) {
	params := map[string]string{}
	params["DealIds.0"] = dealId
	resp, err := self.redisRequest("DescribeInstanceDealDetail", params)
	if err != nil {
		return "", errors.Wrap(err, "DescribeInstanceDealDetail")
	}

	ret := []SElasticcacheDeal{}
	err = resp.Unmarshal(&ret, "DealDetails")
	if err != nil {
		return "", errors.Wrap(err, "DealDetails")
	}

	if len(ret) == 0 {
		return "", cloudprovider.ErrNotFound
	} else if len(ret) > 1 {
		log.Infof("%#v", ret)
		return "", cloudprovider.ErrDuplicateId
	} else {
		if ret[0].InstanceIDS != nil && len(ret[0].InstanceIDS) == 1 {
			return ret[0].InstanceIDS[0], nil
		} else {
			log.Infof("%#v", ret)
			return "", cloudprovider.ErrNotFound
		}
	}
}
