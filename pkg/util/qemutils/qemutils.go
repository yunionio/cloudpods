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

package qemutils

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/version"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

const (
	USR_LOCAL_BIN = "/usr/local/bin"
	HOMEBREW_BIN  = "/opt/homebrew/bin/"
	USR_BIN       = "/usr/bin"
)

var qemuSystemCmd = "qemu-system-x86_64"

func UseAarch64() {
	qemuSystemCmd = "qemu-system-aarch64"
}

func UseLoongarch64() {
	qemuSystemCmd = "qemu-system-loongarch64"
}

func GetQemu(version string) string {
	return getQemuCmd(qemuSystemCmd, version)
}

func GetQemuNbd() string {
	return getQemuCmd("qemu-nbd", "")
}

func GetQemuImg() string {
	return getQemuCmd("qemu-img", "")
}

func getQemuCmd(cmd, version string) string {
	if len(version) > 0 {
		return getQemuCmdByVersion(cmd, version)
	} else {
		return getQemuDefaultCmd(cmd)
	}
}

func getQemuCmdByVersion(cmd, version string) string {
	p := path.Join(fmt.Sprintf("/usr/local/qemu-%s/bin", version), cmd)
	if _, err := procutils.RemoteStat(p); err == nil {
		return p
	} else {
		log.Errorf("stat %s: %s", p, err)
	}
	cmd = cmd + "_" + version
	p = path.Join(USR_LOCAL_BIN, cmd)
	if _, err := procutils.RemoteStat(p); err == nil {
		return p
	} else {
		log.Errorf("stat %s: %s", p, err)
	}
	p = path.Join(USR_BIN, cmd)
	if _, err := procutils.RemoteStat(p); err == nil {
		return p
	} else {
		log.Errorf("stat %s: %s", p, err)
	}
	p = path.Join(HOMEBREW_BIN, cmd)
	if _, err := procutils.RemoteStat(p); err == nil {
		return p
	} else {
		log.Errorf("stat %s: %s", p, err)
	}
	return ""
}

func getQemuVersion(verString string) string {
	s := regexp.MustCompile(`qemu-(?P<ver>\d+(\.\d+)+)$`).FindString(verString)
	if len(s) == 0 {
		return ""
	}
	return s[len("qemu-"):]
}

func getCmdVersion(cmd string) string {
	s := regexp.MustCompile(`_(?P<ver>\d+(\.\d+)+)$`).FindString(cmd)
	if len(s) == 0 {
		return ""
	}
	return s[1:]
}

func getQemuDefaultCmd(cmd string) string {
	var qemus = make([]string, 0)
	if files, err := procutils.RemoteReadDir("/usr/local"); err == nil {
		for i := 0; i < len(files); i++ {
			if strings.HasPrefix(files[i].Name(), "qemu-") {
				qemus = append(qemus, files[i].Name())
			}
		}
		if len(qemus) > 0 {
			sort.Slice(qemus, func(i, j int) bool {
				return version.LT(getQemuVersion(qemus[i]),
					getQemuVersion(qemus[j]))
			})
			p := fmt.Sprintf("/usr/local/%s/bin/%s", qemus[len(qemus)-1], cmd)
			if _, err := procutils.RemoteStat(p); err == nil {
				return p
			} else {
				log.Errorf("stat %s: %s", p, err)
			}
		}
	}

	cmds := make([]string, 0)
	for _, dir := range []string{USR_LOCAL_BIN, USR_BIN, HOMEBREW_BIN} {
		if files, err := procutils.RemoteReadDir(dir); err == nil {
			for i := 0; i < len(files); i++ {
				if strings.HasPrefix(files[i].Name(), cmd) {
					cmds = append(cmds, files[i].Name())
				}
			}
			if len(cmds) > 0 {
				sort.Slice(cmds, func(i, j int) bool {
					return version.LT(getCmdVersion(cmds[i]),
						getCmdVersion(cmds[j]))
				})
				p := path.Join(dir, cmds[len(cmds)-1])
				if _, err := procutils.RemoteStat(p); err == nil {
					return p
				} else {
					log.Errorf("stat %s: %s", p, err)
				}
			}
		}
	}
	return ""
}
