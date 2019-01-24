package huawei

import "fmt"

type SResource struct {
	ResourceID       string `json:"resourceId"`
	CloudServiceType string `json:"cloudServiceType"`
	RegionCode       string `json:"regionCode"`
	ResourceType     string `json:"resourceType"`
	ResourceSpecCode string `json:"resourceSpecCode"`
	Status           int64  `json:"status"`
}

/*
获取订单信息  https://support.huaweicloud.com/api-oce/api_order_00001.html
*/
func (self *SRegion) GetAllResByOrderId(orderId string) ([]SResource, error) {
	domains, err := self.client.getEnabledDomains()
	if err != nil {
		return nil, err
	}

	if domains == nil || len(domains) == 0 {
		return nil, fmt.Errorf("GetAllResByOrderId domain is empty")
	} else if len(domains) > 1 {
		// not supported??
		return nil, fmt.Errorf("GetAllResByOrderId mutliple domain(%d) found", len(domains))
	}

	err = self.ecsClient.Orders.SetDomainId(domains[0].ID)
	if err != nil {
		return nil, err
	}

	resp, err := self.ecsClient.Orders.Get(orderId, nil)
	if err != nil {
		return nil, err
	}

	resources := make([]SResource, 0)
	err = resp.Unmarshal(&resources, "resources")
	return resources, err
}

func (self *SRegion) getAllResByType(orderId string, resourceType string) ([]SResource, error) {
	res, err := self.GetAllResByOrderId(orderId)
	if err != nil {
		return nil, err
	}

	ret := make([]SResource, 0)
	for i := range res {
		r := res[i]
		if r.ResourceType == resourceType {
			ret = append(ret, r)
		}
	}

	return ret, nil
}

func (self *SRegion) getAllResIdsByType(orderId string, resourceType string) ([]string, error) {
	res, err := self.getAllResByType(orderId, resourceType)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0)
	for _, r := range res {
		ids = append(ids, r.ResourceID)
	}

	return ids, nil
}
