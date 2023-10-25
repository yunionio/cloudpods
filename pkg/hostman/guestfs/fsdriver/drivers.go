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

package fsdriver

import (
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

type newRootFsDriverFunc func(part IDiskPartition) IRootFsDriver

var (
	privatePrefixes    []string
	rootfsDrivers      = make([]newRootFsDriverFunc, 0)
	hostCpuArch        string
	cloudrootDirectory string
)

func GetRootfsDrivers() []newRootFsDriverFunc {
	return rootfsDrivers
}

func Init(initPrivatePrefixes []string, cloudrootDir string) error {
	if len(initPrivatePrefixes) > 0 {
		privatePrefixes = make([]string, len(initPrivatePrefixes))
		copy(privatePrefixes, initPrivatePrefixes)
	}

	linuxFsDrivers := []newRootFsDriverFunc{
		NewFangdeRootFs, NewUnionOSRootFs,
		NewAnolisRootFs, NewRockyRootFs,
		NewGalaxyKylinRootFs, NewNeoKylinRootFs,
		NewFangdeDeskRootfs, NewUKylinRootfs,
		NewCentosRootFs, NewFedoraRootFs,
		NewRhelRootFs,
		NewOpenSuseRootFs,
		NewDebianRootFs, NewCirrosRootFs, NewCirrosNewRootFs, NewUbuntuRootFs,
		NewGentooRootFs, NewArchLinuxRootFs, NewOpenWrtRootFs, NewCoreOsRootFs,
		NewOpenEulerRootFs,
	}
	rootfsDrivers = append(rootfsDrivers, linuxFsDrivers...)
	rootfsDrivers = append(rootfsDrivers, NewMacOSRootFs)
	rootfsDrivers = append(rootfsDrivers, NewEsxiRootFs)
	rootfsDrivers = append(rootfsDrivers, NewWindowsRootFs)

	androidFsDrivers := []newRootFsDriverFunc{
		NewAndroidRootFs,
		NewPhoenixOSRootFs,
	}

	rootfsDrivers = append(rootfsDrivers, androidFsDrivers...)

	cpuArch, err := procutils.NewCommand("uname", "-m").Output()
	if err != nil {
		return errors.Wrap(err, "get cpu architecture")
	}
	hostCpuArch = strings.TrimSpace(string(cpuArch))
	cloudrootDirectory = cloudrootDir
	if len(cloudrootDirectory) == 0 {
		cloudrootDirectory = "/opt"
	}
	return nil
}
