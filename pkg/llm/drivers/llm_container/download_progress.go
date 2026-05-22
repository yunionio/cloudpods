package llm_container

func reportInstantModelDownloadProgress(callback func(progress float32), downloaded, total int64) {
	if callback == nil || total <= 0 {
		return
	}
	if downloaded < 0 {
		downloaded = 0
	}
	if downloaded > total {
		downloaded = total
	}
	callback(float32(downloaded) * 100 / float32(total))
}

func instantModelFileDownloadProgress(callback func(progress float32), completed, total, expectedSize int64) func(downloaded, fileTotal int64) {
	return func(downloaded, fileTotal int64) {
		if expectedSize <= 0 {
			expectedSize = fileTotal
		}
		if expectedSize <= 0 {
			return
		}
		if downloaded > expectedSize {
			downloaded = expectedSize
		}
		reportInstantModelDownloadProgress(callback, completed+downloaded, total)
	}
}
