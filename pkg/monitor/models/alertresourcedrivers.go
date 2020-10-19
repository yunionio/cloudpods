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

package models

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
)

const (
	ErrAlertResourceDriverNotFound       = errors.Error("Alert resource driver not found")
	ErrAlertResourceDriverDuplicateMatch = errors.Error("Alert resource driver duplicate match")
)

var (
	alertResourceDriverFs = make(map[monitor.AlertResourceType]IAlertResourceDriverFactory, 0)
)

type IAlertResourceDriverFactory interface {
	// GetType return the driver type
	GetType() monitor.AlertResourceType

	// IsEvalMatched match driver by monitor.EvalMatch
	IsEvalMatched(input monitor.EvalMatch) bool

	GetDriver(input monitor.EvalMatch) IAlertResourceDriver
}

type IAlertResourceDriver interface {
	// GetType return the driver type
	GetType() monitor.AlertResourceType
	// GetUniqCond get uniq match conditions from eval match to find AlertResource
	GetUniqCond() *AlertResourceUniqCond
}

func RegisterAlertResourceDriverFactory(drvs ...IAlertResourceDriverFactory) {
	for _, drv := range drvs {
		alertResourceDriverFs[drv.GetType()] = drv
	}
}

func GetAlertResourceDriver(match monitor.EvalMatch) (IAlertResourceDriver, error) {
	var matchedType monitor.AlertResourceType
	matchedDrvs := make(map[monitor.AlertResourceType]IAlertResourceDriverFactory)
	for _, drv := range alertResourceDriverFs {
		if ok := drv.IsEvalMatched(match); ok {
			matchedDrvs[drv.GetType()] = drv
			matchedType = drv.GetType()
		}
	}
	if len(matchedDrvs) == 0 {
		return nil, errors.Wrapf(ErrAlertResourceDriverNotFound, "match by %v", match)
	}
	if len(matchedDrvs) > 1 {
		return nil, errors.Wrapf(ErrAlertResourceDriverDuplicateMatch, "match by %v", match)
	}
	return matchedDrvs[matchedType].GetDriver(match), nil
}
