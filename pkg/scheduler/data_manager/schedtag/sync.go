package schedtag

import (
	"fmt"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/wait"
)

var (
	globalDataManager *dataManager
)

func Start(refreshInterval time.Duration) error {
	var err error
	globalDataManager, err = newDataManager(refreshInterval)
	if err != nil {
		panic(errors.Wrap(err, "newDataManager"))
	}
	globalDataManager.sync()
	return nil
}

func GetAllSchedtags(resType string) ([]ISchedtag, error) {
	return globalDataManager.tagMan.getAllSchedtags(resType)
}

func GetEnabledDynamicSchedtagsByResource(resType string) ([]IDynamicschedtag, error) {
	ret, ok := globalDataManager.tagMan.dynamicSchedtagsByType.rawGet(resType)
	if !ok {
		return nil, nil
	}
	return ret.([]IDynamicschedtag), nil
}

func GetCandidateSchedtags(resType, id string) []ISchedtag {
	return globalDataManager.tagMan.getCandidateSchedtags(resType, id)
}

type dataManager struct {
	tagMan *schedtagManager

	refreshInterval time.Duration
}

func newDataManager(refreshInterval time.Duration) (*dataManager, error) {
	man := &dataManager{
		refreshInterval: refreshInterval,
	}
	return man, nil
}

func (m *dataManager) sync() {
	wait.Forever(m.syncOnce, m.refreshInterval)
}

func (m *dataManager) syncOnce() {
	log.Infof("Schedtag data start sync")
	startTime := time.Now()

	if err := func() error {
		m.tagMan = newSchedtagManagerWithoutInit()
		if err := m.tagMan.initAllSchedtags(); err != nil {
			return errors.Wrap(err, "initAllSchedtags")
		}
		if err := m.tagMan.initDynamicschedtags(); err != nil {
			return errors.Wrap(err, "initResourceSchedtags")
		}
		return nil
	}(); err != nil {
		log.Errorf("Schedtag sync data error: %v", err)
		return
	}

	log.Infof("Schedtag finish sync, elapsed %s", time.Since(startTime))
}
