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

package apparmorutils

import (
	"fmt"
	"os"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

const (
	AppArmorEnabledFile = "/sys/module/apparmor/parameters/enabled"
)

func IsEnabled() bool {
	content, err := os.ReadFile(AppArmorEnabledFile)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(content)) == "Y"
}

func Parser(profile string) error {
	tmpFileName := fmt.Sprintf("/tmp/apparmor-profile-%s.tmp", utils.GenRequestId(12))

	err := procutils.FilePutContents(tmpFileName, profile)
	if err != nil {
		return errors.Wrap(err, "FilePutContents")
	}
	defer func() {
		procutils.NewRemoteCommandAsFarAsPossible("rm", "-f", tmpFileName).Run()
	}()

	err = procutils.NewRemoteCommandAsFarAsPossible("apparmor_parser", "-r", "-W", tmpFileName).Run()
	if err != nil {
		return errors.Wrap(err, "apparmor_parser")
	}

	return nil
}
