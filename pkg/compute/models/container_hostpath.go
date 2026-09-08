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
	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

// ValidateHostBindPath cleans an absolute host bind path and rejects
// restricted host locations for non-admin callers.
func ValidateHostBindPath(userCred mcclient.TokenCredential, path string) (string, error) {
	clean, err := fileutils2.CleanHostBindPath(path)
	if err != nil {
		return "", httperrors.NewInputParameterError("%v", err)
	}
	if userCred == nil || !userCred.HasSystemAdminPrivilege() {
		if fileutils2.IsRestrictedHostBindPath(clean) {
			return "", httperrors.NewForbiddenError("host path %s is not allowed", clean)
		}
	}
	return clean, nil
}

// ValidateHostPathAutoCreate checks auto-create options for a host bind path.
func ValidateHostPathAutoCreate(autoCreate bool, cfg *apis.ContainerVolumeMountHostPathAutoCreateConfig) error {
	if !autoCreate && cfg == nil {
		return nil
	}
	if cfg == nil {
		return nil
	}
	if cfg.Permissions != "" {
		_, canon, err := fileutils2.ParseUnixFileMode(cfg.Permissions)
		if err != nil {
			return httperrors.NewInputParameterError("invalid auto_create permissions: %v", err)
		}
		cfg.Permissions = canon
	}
	return nil
}

// ValidateRelSubpath cleans a relative path that must stay inside a parent dir.
func ValidateRelSubpath(path string) (string, error) {
	clean, err := fileutils2.CleanRelSubpath(path)
	if err != nil {
		return "", httperrors.NewInputParameterError("%v", err)
	}
	return clean, nil
}
