package huawei

import "yunion.io/x/onecloud/pkg/util/huawei/client"

// https://support.huaweicloud.com/api-iam/zh-cn_topic_0057845574.html
// 租户列表
type SDomain struct {
	Contacts       string `json:"contacts"`
	Description    string `json:"description"`
	Enabled        bool   `json:"enabled"`
	EnterpriseName string `json:"enterpriseName"`
	ID             string `json:"id"`
	Name           string `json:"name"`
	Tagflag        int    `json:"tagflag"`
}

func (self *SHuaweiClient) GetDomains() ([]SDomain, error) {
	huawei, _ := client.NewClientWithAccessKey("", "", self.accessKey, self.secret, self.debug)
	domains := make([]SDomain, 0)
	err := doListAll(huawei.Domains.List, nil, &domains)
	return domains, err
}

func (self *SHuaweiClient) getEnabledDomains() ([]SDomain, error) {
	domains, err := self.GetDomains()

	enabledDomains := make([]SDomain, 0)
	for i := range domains {
		if domains[i].Enabled {
			enabledDomains = append(enabledDomains, domains[i])
		}
	}

	return enabledDomains, err
}
