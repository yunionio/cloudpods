package llm

import "time"

const (
	INSTANT_MODEL_STATUS_PACKAGING = "packaging"

	InstantModelImportDownloadProgressEnd float32 = 90
	InstantModelImportArchiveProgress     float32 = 95
	InstantModelImportUploadProgress      float32 = 98
	InstantModelImportCompleteProgress    float32 = 100
	InstantModelImportProgressMinDelta    float32 = 1
	InstantModelImportProgressMinInterval         = 2 * time.Second
)
