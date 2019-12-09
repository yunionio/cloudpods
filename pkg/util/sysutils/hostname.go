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

package sysutils

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func setHostname7(name string) error {
	return procutils.NewCommand("hostnamectl", "set-hostname", name).Run()
}

func setHostname6(name string) error {
	err := procutils.NewCommand("hostname", name).Run()
	if err != nil {
		return err
	}
	cont := fmt.Sprintf("NETWORKING=yes\nHOSTNAME=%s\n", name)
	return fileutils2.FileSetContents("/etc/sysconfig/network", cont)
}

func SetHostname(name string) error {
	err := setHostname7(name)
	if err != nil {
		err = setHostname6(name)
	}
	return err
}
