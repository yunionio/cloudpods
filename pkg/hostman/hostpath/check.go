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

package hostpath

import (
	"fmt"
	"path/filepath"
	"strings"

	"yunion.io/x/onecloud/pkg/apis"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

var runHostPathCheckCommand = func(name string, args ...string) error {
	return procutils.NewRemoteCommandAsFarAsPossible(name, args...).Run()
}

func Check(input hostapi.HostPathCheckInput) (*hostapi.HostPathCheckOutput, error) {
	output := &hostapi.HostPathCheckOutput{
		Results: make([]hostapi.HostPathCheckResult, 0, len(input.Paths)),
	}
	for _, item := range input.Paths {
		if err := validateItem(item); err != nil {
			return nil, err
		}
		output.Results = append(output.Results, checkOne(item))
	}
	return output, nil
}

func validateItem(item hostapi.HostPathCheckItem) error {
	if item.Path == "" {
		return fmt.Errorf("host path is empty")
	}
	if strings.ContainsRune(item.Path, 0) {
		return fmt.Errorf("host path %q contains NUL byte", item.Path)
	}
	if !filepath.IsAbs(item.Path) {
		return fmt.Errorf("host path %q must be absolute", item.Path)
	}
	switch item.Type {
	case apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY, apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE:
		return nil
	default:
		return fmt.Errorf("unsupported host path type %q", item.Type)
	}
}

func checkOne(item hostapi.HostPathCheckItem) hostapi.HostPathCheckResult {
	result := hostapi.HostPathCheckResult{
		Path: item.Path,
		Type: item.Type,
	}

	if err := runHostPathCheckCommand("test", "-e", item.Path); err != nil {
		result.Error = fmt.Sprintf("%s does not exist", item.Path)
		return result
	}

	result.Exists = true
	switch item.Type {
	case apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY:
		result.TypeMatched = runHostPathCheckCommand("test", "-d", item.Path) == nil
	case apis.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE:
		result.TypeMatched = runHostPathCheckCommand("test", "-f", item.Path) == nil
	}
	if !result.TypeMatched {
		result.Error = fmt.Sprintf("%s is not %s", item.Path, item.Type)
	}
	return result
}
