package multicloud

type SStorageBase struct {
	SResourceBase
}

func (s *SStorageBase) DisableSync() bool {
	return false
}
