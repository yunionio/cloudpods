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
	"context"

	api "yunion.io/x/onecloud/pkg/apis/notify"
)

type ISenderDriver interface {
	GetSenderType() string
	Send(ctx context.Context, args api.SendParams) error
	ValidateConfig(context.Context, api.NotifyConfig) (string, error)
	UpdateConfig(context.Context, api.NotifyConfig) error
	AddConfig(context.Context, api.NotifyConfig) error
	DeleteConfig(context.Context, api.NotifyConfig) error
	ContactByMobile(ctx context.Context, mobile, domainId string) (string, error)
	IsRobot() bool
	IsPersonal() bool
	IsSystemConfigContactType() bool
	IsValid() bool
	IsPullType() bool
}

var (
	driverTable = make(map[string]ISenderDriver)
)

func Register(driver ISenderDriver) {
	driverTable[driver.GetSenderType()] = driver
}

func GetSenderTypes() []string {
	ret := []string{}
	for k := range driverTable {
		ret = append(ret, k)
	}
	return ret
}

func GetRobotTypes() []string {
	ret := []string{}
	for k := range driverTable {
		if driverTable[k].IsRobot() {
			ret = append(ret, k)
		}
	}
	return ret
}

func GetValidPersonalSenderTypes() []string {
	ret := []string{}
	for k := range driverTable {
		if driverTable[k].IsValid() && driverTable[k].IsPersonal() {
			ret = append(ret, k)
		}
	}
	return ret
}

func GetDriver(sendType string) ISenderDriver {
	driver, _ := driverTable[sendType]
	return driver
}
