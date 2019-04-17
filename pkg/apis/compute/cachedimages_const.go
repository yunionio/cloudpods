package compute

const (
	CACHED_IMAGE_STATUS_INIT         = "init"
	CACHED_IMAGE_STATUS_SAVING       = "saving"
	CACHED_IMAGE_STATUS_CACHING      = "caching"
	CACHED_IMAGE_STATUS_READY        = "ready"
	CACHED_IMAGE_STATUS_DELETING     = "deleting"
	CACHED_IMAGE_STATUS_CACHE_FAILED = "cache_fail"

	DOWNLOAD_SESSION_LENGTH = 3600 * 3 // 3 hour
)

const (
	CACHED_IMAGE_REFRESH_SECONDS                  = 900   // 15 minutes
	CACHED_IMAGE_REFERENCE_SESSION_EXPIRE_SECONDS = 86400 // 1 day
)
