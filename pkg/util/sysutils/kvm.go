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
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

const (
	KVM_MODULE_INTEL     = "kvm-intel"
	KVM_MODULE_AMD       = "kvm-amd"
	KVM_MODULE           = "kvm"
	KVM_MODULE_UNSUPPORT = "unsupport"
	KVM_MODULE_BUILDIN   = "buildin"

	HOST_NEST_UNSUPPORT = "0"
	HOST_NEST_SUPPORT   = "1"
	HOST_NEST_ENABLE    = "3"
)

var (
	kvmModuleSupport string
	nestStatus       string
)

func GetKVMModuleSupport() string {
	if len(kvmModuleSupport) == 0 {
		kvmModuleSupport = detectiveKVMModuleSupport()
	}
	return kvmModuleSupport
}

func IsKvmSupport() bool {
	GetKVMModuleSupport()
	if kvmModuleSupport == KVM_MODULE_UNSUPPORT {
		return false
	}
	return true
}

func IsProcessorIntel() bool {
	GetKVMModuleSupport()
	if kvmModuleSupport == KVM_MODULE_INTEL {
		return true
	}
	return false
}

func IsProcessorAmd() bool {
	GetKVMModuleSupport()
	if kvmModuleSupport == KVM_MODULE_AMD {
		return true
	}
	return false
}

func detectiveKVMModuleSupport() string {
	var km = KVM_MODULE_UNSUPPORT
	if ModprobeKvmModule(KVM_MODULE_INTEL, false, false) {
		km = KVM_MODULE_INTEL
	} else if ModprobeKvmModule(KVM_MODULE_AMD, false, false) {
		km = KVM_MODULE_AMD
	} else if ModprobeKvmModule(KVM_MODULE, false, false) {
		km = KVM_MODULE
	}
	if km == KVM_MODULE_UNSUPPORT {
		if fileutils2.Exists("/dev/kvm") {
			km = KVM_MODULE_BUILDIN
		}
	}
	return km
}

func ModprobeKvmModule(name string, remove, nest bool) bool {
	var params = []string{"modprobe"}
	if remove {
		params = append(params, "-r")
	}
	params = append(params, name)
	if nest {
		params = append(params, "nested=1")
	}
	if err := procutils.NewCommand(params[0], params[1:]...).Run(); err != nil {
		return false
	}
	return true
}

func IsNestEnabled() bool {
	return GetNestSupport() == HOST_NEST_ENABLE
}

func GetNestSupport() string {
	if len(nestStatus) == 0 {
		nestStatus = detectNestSupport()
	}
	return nestStatus
}

func detectNestSupport() string {
	moduleName := GetKVMModuleSupport()
	nestStatus := HOST_NEST_UNSUPPORT

	if moduleName != KVM_MODULE_UNSUPPORT && isNestSupport(moduleName) {
		nestStatus = HOST_NEST_SUPPORT
	}

	if nestStatus == HOST_NEST_SUPPORT && loadKvmModuleWithNest(moduleName) {
		nestStatus = HOST_NEST_ENABLE
	}
	return nestStatus
}

func isNestSupport(name string) bool {
	output, err := procutils.NewCommand("modinfo", name).Output()
	if err != nil {
		log.Errorln(err)
		return false
	}

	// TODO Test
	var re = regexp.MustCompile(`parm:\s*nested:`)
	for _, line := range strings.Split(string(output), "\n") {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

func loadKvmModuleWithNest(name string) bool {
	var notload = true
	if IsKernelModuleLoaded(name) {
		nest := GetKernelModuleParameter(name, "nested")
		if nest == "Y" {
			return true
		}
		notload = unloadKvmModule(name)
	}
	if notload {
		if ModprobeKvmModule(name, false, true) {
			return true
		}
	}
	return false
}

func unloadKvmModule(name string) bool {
	return ModprobeKvmModule(name, true, false)
}

func GetKernelModuleParameter(name, moduel string) string {
	pa := path.Join("/sys/module/", strings.Replace(name, "-", "_", -1), "/parameters/", moduel)
	if f, err := os.Stat(pa); err == nil {
		if f.IsDir() {
			return ""
		}
		cont, err := fileutils2.FileGetContents(pa)
		if err != nil {
			log.Errorln(err)
			return ""
		}
		return strings.TrimSpace(cont)
	}
	return ""
}

func IsKernelModuleLoaded(name string) bool {
	output, err := procutils.NewCommand("lsmod").Output()
	if err != nil {
		log.Errorln(err)
		return false
	}
	for _, line := range strings.Split(string(output), "\n") {
		lm := strings.Split(line, " ")
		if len(lm) > 0 && utils.IsInStringArray(strings.Replace(name, "-", "_", -1), lm) {
			return true
		}
	}
	return false
}

func SetSysConfig(cpath, val string) bool {
	if fileutils2.Exists(cpath) {
		oval, err := ioutil.ReadFile(cpath)
		if err != nil {
			log.Errorln(err)
			return false
		}
		if string(oval) != val {
			err = fileutils2.FilePutContents(cpath, val, false)
			if err == nil {
				return true
			}
			log.Errorln(err)
		}
	}
	return false
}

func IsHypervisor() bool {
	cont, _ := fileutils2.FileGetContents("/proc/cpuinfo")
	if strings.Index(cont, " hypervisor") > 0 {
		return true
	}
	return false
}
