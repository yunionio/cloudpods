package cgrouputils

import (
	"encoding/json"
	"sync"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

const (
	utilHistoryFile        = "/tmp/util.history"
	MAX_HISTORY_UTIL_COUNT = 5
)

var (
	rebalanceProcessesLock    = sync.Mutex{}
	rebalanceProcessesRunning = false

	utilHistory map[string][]float64
)

func RebalanceProcesses(pids string) {
	rebalanceProcessesLock.Lock()
	defer rebalanceProcessesLock.Unlock()

	if rebalanceProcessesRunning {
		return
	}

	rebalanceProcesses(pids)
}

func rebalanceProcesses(pids string) {
	FetchHistoryUtil()
	// ...
	SaveHistoryUtil()
}

func FetchHistoryUtil() map[string][]float64 {
	if utilHistory == nil {
		utilHistory = make(map[string][]float64)
		if fileutils2.Exists(utilHistoryFile) {
			contents, err := fileutils2.FileGetContents(utilHistoryFile)
			if err != nil {
				log.Errorf("FetchHistoryUtil error: %s", err)
				return utilHistory
			}
			var objmap map[string]*json.RawMessage
			if err := json.Unmarshal([]byte(contents), objmap); err != nil {
				log.Errorf("FetchHistoryUtil error: %s", err)
				return utilHistory
			}
			for k, v := range objmap {
				var s = []float64{}
				if err := json.Unmarshal(*v, &s); err != nil {
					log.Errorf("FetchHistoryUtil error: %s", err)
					break
				}
				utilHistory[k] = s
			}
		}
	}
	return utilHistory
}

func SaveHistoryUtil() {
	content, err := json.Marshal(utilHistory)
	if err != nil {
		log.Errorf("SaveHistoryUtil error: %s", err)
	} else {
		fileutils2.FilePutContents(utilHistoryFile, string(content), false)
	}
}
