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
	"time"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/appsrv"
)

const (
	MIN_REFRESH_INTERVAL_SECONDS = 30
)

type TOptionsChangeFunc func(oldOpts, newOpts interface{}) bool

type SOptionManager struct {
	serviceType    string
	serviceVersion string

	options interface{}

	session IServiceConfigSession

	refreshInterval time.Duration

	onOptionsChange TOptionsChangeFunc
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
		refreshInterval: refreshInterval,
		onOptionsChange: onChange,
	}
	OptionManager.firstSync()
}

func (manager *SOptionManager) newOptions() interface{} {
	optType := reflect.ValueOf(manager.options).Elem().Type()
	return reflect.New(optType).Interface()
}

func copyOptions(dst, src interface{}) {
	dstValue := reflect.ValueOf(dst).Elem()
	dstValue.Set(reflect.ValueOf(src).Elem())
}

func (manager *SOptionManager) doSync(first bool) {
	newOpts := manager.newOptions()
	copyOptions(newOpts, manager.options)
	merged := manager.session.Merge(newOpts, manager.serviceType, manager.serviceVersion)

	if merged && !reflect.DeepEqual(newOpts, manager.options) {
		log.Infof("Service config changed ...")
		if !first && manager.onOptionsChange != nil && manager.onOptionsChange(manager.options, newOpts) {
			log.Infof("Option changes detected and going to restart the program...")
			appsrv.SetExitFlag()
		}
		copyOptions(manager.options, newOpts)
		manager.session.Upload()
	}
}

func (manager *SOptionManager) firstSync() {
	manager.doSync(true)
	time.AfterFunc(manager.refreshInterval, manager.sync)
}

func (manager *SOptionManager) sync() {
	manager.doSync(false)
	time.AfterFunc(manager.refreshInterval, manager.sync)
}
