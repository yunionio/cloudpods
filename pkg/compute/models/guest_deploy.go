// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or authorized to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"yunion.io/x/pkg/util/sets"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

var deployConfigActions = sets.NewString("", "create", "append")

func ValidateDeployConfigs(deploys []*api.DeployConfig) error {
	for i, d := range deploys {
		if d == nil {
			continue
		}
		if !deployConfigActions.Has(d.Action) {
			return httperrors.NewInputParameterError("invalid deploy action %q", d.Action)
		}
		clean, err := fileutils2.CleanGuestDeployPath(d.Path)
		if err != nil {
			return httperrors.NewInputParameterError("deploy_configs[%d].path: %v", i, err)
		}
		d.Path = clean
	}
	return nil
}
