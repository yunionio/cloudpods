package multicloud

type SStorageBase struct {
	SResourceBase
	STagBase
}

func (s *SStorageBase) DisableSync() bool {
	return false
}
