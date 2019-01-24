package huawei

import "yunion.io/x/onecloud/pkg/util/huawei/client"

// https://support.huaweicloud.com/api-iam/zh-cn_topic_0057845574.html
// 租户列表
type SDomain struct {
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	ID          string `json:"id"`
	Name        string `json:"name"`
}

func (self *SHuaweiClient) getDomains() ([]SDomain, error) {
	huawei, _ := client.NewClientWithAccessKey("", "", self.accessKey, self.secret)
	domains := make([]SDomain, 0)
	err := DoList(huawei.Domains.List, nil, &domains)
	return domains, err
}

func (self *SHuaweiClient) getEnabledDomains() ([]SDomain, error) {
	huawei, _ := client.NewClientWithAccessKey("", "", self.accessKey, self.secret)
	domains := make([]SDomain, 0)
	err := DoList(huawei.Domains.List, nil, &domains)
	if err != nil {
		return domains, err
	}

	enabledDomains := make([]SDomain, 0)
	for i := range domains {
		domain := domains[i]
		if domain.Enabled {
			enabledDomains = append(enabledDomains, domain)
		}
	}

	return enabledDomains, err
}
