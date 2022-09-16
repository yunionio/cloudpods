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

package multicloud

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SBaseClouduser struct {
}

func (self *SBaseClouduser) PerformCreateAccessKey(description string) (jsonutils.JSONObject, error) {
	return nil, errors.Wrap(nil, "base PerformCreateAccessKey")
}

func (self *SBaseClouduser) GetDetailsAccessKeys() (jsonutils.JSONObject, error) {
	return nil, errors.Wrap(nil, "base GetDetailsAccessKeys")
}

func (self *SBaseClouduser) PerformDeleteAccessKey(accesskey string) (jsonutils.JSONObject, error) {
	return nil, errors.Wrap(nil, "base PerformDeleteAccessKey")
}
