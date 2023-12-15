package common

var cacheManager CacheManager

type CacheManager interface {
	ReloadHosts(ids []string) ([]interface{}, error)
}

func RegisterCacheManager(man CacheManager) {
	cacheManager = man
}

func GetCacheManager() CacheManager {
	return cacheManager
}
