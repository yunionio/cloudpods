package storageman

// not need to impl, using esxi agent
type SAgentImageCacheManager struct {
	storagemanager *SStorageManager
}

func NewAgentImageCacheManager(storagemanager *SStorageManager) *SAgentImageCacheManager {
	return &SAgentImageCacheManager{storagemanager}
}
