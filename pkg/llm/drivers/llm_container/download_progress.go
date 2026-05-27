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

func reportInstantModelStepProgress(callback func(progress float32), completed, total int) {
	if callback == nil || total <= 0 {
		return
	}
	if completed < 0 {
		completed = 0
	}
	if completed > total {
		completed = total
	}
	callback(float32(completed) * 100 / float32(total))
}

func instantModelStepDownloadProgress(callback func(progress float32), completed, total int) func(downloaded, fileTotal int64) {
	return func(downloaded, fileTotal int64) {
		if callback == nil || total <= 0 {
			return
		}
		if completed < 0 {
			completed = 0
		}
		if completed > total {
			completed = total
		}
		fileProgress := float32(0)
		if fileTotal > 0 {
			if downloaded < 0 {
				downloaded = 0
			}
			if downloaded > fileTotal {
				downloaded = fileTotal
			}
			fileProgress = float32(downloaded) / float32(fileTotal)
		}
		progress := (float32(completed) + fileProgress) * 100 / float32(total)
		if progress > 100 {
			progress = 100
		}
		callback(progress)
	}
}
