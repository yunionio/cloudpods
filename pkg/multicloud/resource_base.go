package multicloud

import "yunion.io/x/jsonutils"

type SResourceBase struct{}

func (self *SResourceBase) IsEmulated() bool {
	return false
}

func (self *SResourceBase) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SResourceBase) Refresh() error {
	return nil
}
