package bingocloud

import (
	"yunion.io/x/log"
)

// type SEip struct {
// }

func (self *SRegion) GetEips() (string, int, error) {
	resp, err := self.invoke("DescribeIpTypes", nil)
	if err != nil {
		return "", 0, err
	}
	log.Errorf("resp=:%s", resp)
	return resp.String(), 0, nil
}
