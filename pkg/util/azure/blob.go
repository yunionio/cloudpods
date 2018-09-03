package azure

import (
	"fmt"

	"yunion.io/x/jsonutils"
)

type SStorageAccount struct {
	region *SRegion
}

func (self *SStorageAccount) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SStorageAccount) GetId() string {
	return fmt.Sprintf("%s-%s", self.region.client.providerId, self.region.GetId())
}
