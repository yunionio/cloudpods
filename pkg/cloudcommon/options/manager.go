// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package options

import (
	"reflect"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/cloudcommon/syncman/watcher"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

const (
	MIN_REFRESH_INTERVAL_SECONDS = 30
)

type TOptionsChangeFunc func(oldOpts, newOpts interface{}) bool

type SOptionManager struct {
	watcher.SInformerSyncManager

	serviceType    string
	serviceVersion string

	options interface{}

	session IServiceConfigSession

	onOptionsChange TOptionsChangeFunc

	refreshInterval time.Duration
}

var (
	OptionManager *SOptionManager
)

func StartOptionManager(option interface{}, refreshSeconds int, serviceType, serviceVersion string, onChange TOptionsChangeFunc) {
	StartOptionManagerWithSessionDriver(option, refreshSeconds, serviceType, serviceVersion, onChange, newServiceConfigSession())
}

func StartOptionManagerWithSessionDriver(options interface{}, refreshSeconds int, serviceType, serviceVersion string, onChange TOptionsChangeFunc, session IServiceConfigSession) {
	log.Infof("OptionManager start to fetch service configs ...")
	if refreshSeconds <= MIN_REFRESH_INTERVAL_SECONDS {
		// a minimal 30 seconds refresh interval
		refreshSeconds = MIN_REFRESH_INTERVAL_SECONDS
	}
	refreshInterval := time.Duration(refreshSeconds) * time.Second
	OptionManager = &SOptionManager{
		serviceType:     serviceType,
		serviceVersion:  serviceVersion,
		options:         options,
		session:         session,
		onOptionsChange: onChange,
		refreshInterval: refreshInterval,
	}

	OptionManager.InitSync(OptionManager)

	OptionManager.FirstSync()

	if session.IsRemote() {
		OptionManager.StartWatching(&modules.ServicesV3)
	}
}

func (manager *SOptionManager) newOptions() interface{} {
	optType := reflect.ValueOf(manager.options).Elem().Type()
	return reflect.New(optType).Interface()
}

func copyOptions(dst, src interface{}) {
	dstValue := reflect.ValueOf(dst).Elem()
	dstValue.Set(reflect.ValueOf(src).Elem())
}

func optionsEquals(newOpts interface{}, oldOpts interface{}) bool {
	newOptsDict := jsonutils.Marshal(newOpts).(*jsonutils.JSONDict)
	oldOptsDict := jsonutils.Marshal(oldOpts).(*jsonutils.JSONDict)

	deleted, diff, _, added := jsonutils.Diff(oldOptsDict, newOptsDict)

	if deleted.Length() > 0 {
		log.Infof("Options removed: %s", deleted)
		return false
	}
	if diff.Length() > 0 {
		log.Infof("Options changed: %s", diff)
		return false
	}
	if added.Length() > 0 {
		log.Infof("Options added: %s", added)
		return false
	}
	return true
}

func (manager *SOptionManager) DoSync(first bool) (time.Duration, error) {
	newOpts := manager.newOptions()
	copyOptions(newOpts, manager.options)
	merged := manager.session.Merge(newOpts, manager.serviceType, manager.serviceVersion)

	if merged && !optionsEquals(newOpts, manager.options) {
		if manager.onOptionsChange != nil && manager.onOptionsChange(manager.options, newOpts) && !first {
			log.Infof("Option changes detected and going to restart the program...")
			appsrv.SetExitFlag()
		}
		copyOptions(manager.options, newOpts)
		// if first {
		// upload config
		manager.session.Upload()
		// }
	}
	return manager.refreshInterval, nil
}

func (manager *SOptionManager) NeedSync(dat *jsonutils.JSONDict) bool {
	serviceType, _ := dat.GetString("type")
	if strings.HasPrefix(serviceType, manager.serviceType) || serviceType == consts.COMMON_SERVICE {
		return true
	}
	return false
}

func (manager *SOptionManager) Name() string {
	return "ServiceConfigManager"
}
