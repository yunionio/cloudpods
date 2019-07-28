package multicloud

import "yunion.io/x/jsonutils"

type SBaseBucket struct{}

func (b *SBaseBucket) GetId() string {
	return ""
}

func (b *SBaseBucket) GetName() string {
	return ""
}

func (b *SBaseBucket) GetGlobalId() string {
	return ""
}

func (b *SBaseBucket) GetStatus() string {
	return ""
}

func (b *SBaseBucket) Refresh() error {
	return nil
}

func (b *SBaseBucket) IsEmulated() bool {
	return false
}

func (b *SBaseBucket) GetMetadata() *jsonutils.JSONDict {
	return nil
}
