package bingocloud

type SQuotas struct {
	OwnerId    string
	Resource   string
	ResourceEn string
	ResourceZh string
	HardLimit  int
	InUse      int
}

func (self *SRegion) GetQuotas() ([]SQuotas, error) {
	resp, err := self.invoke("DescribeQuotas", nil)
	if err != nil {
		return nil, err
	}
	var ret []SQuotas
	return ret, resp.Unmarshal(&ret, "quotaSet")
}
