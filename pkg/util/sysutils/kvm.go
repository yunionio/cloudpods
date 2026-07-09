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
)

func GetKVMModuleSupport() string {
	if len(kvmModuleSupport) == 0 {
		kvmModuleSupport = detectKVMModuleSupport()
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

func isDevKVMExists() bool {
	return fileutils2.Exists("/dev/kvm")
}

func detectKVMModuleSupport() string {
	var km = KVM_MODULE_UNSUPPORT
	if isDevKVMExists() {
		if IsKernelModuleLoaded(KVM_MODULE_INTEL) {
			km = KVM_MODULE_INTEL
		} else if IsKernelModuleLoaded(KVM_MODULE_AMD) {
			km = KVM_MODULE_AMD
		} else if IsKernelModuleLoaded(KVM_MODULE) {
			km = KVM_MODULE
		} else {
			km = KVM_MODULE_BUILDIN
		}
	} else {
		if ModprobeKvmModule(KVM_MODULE_INTEL, false, nil) {
			km = KVM_MODULE_INTEL
		} else if ModprobeKvmModule(KVM_MODULE_AMD, false, nil) {
			km = KVM_MODULE_AMD
		} else if ModprobeKvmModule(KVM_MODULE, false, nil) {
			if isDevKVMExists() {
				km = KVM_MODULE
			}
		}
		if km == KVM_MODULE_UNSUPPORT {
			if isDevKVMExists() {
				km = KVM_MODULE_BUILDIN
			}
		}
	}
	return km
}

func ModprobeKvmModule(name string, remove bool, nest *bool) bool {
	var params = []string{"modprobe"}
	if remove {
		params = append(params, "-r")
	}
	params = append(params, name)
	if nest != nil {
		if *nest {
			params = append(params, "nested=1")
		} else {
			params = append(params, "nested=0")
		}
	}

	if err := procutils.NewRemoteCommandAsFarAsPossible(params[0], params[1:]...).Run(); err != nil {
		log.Errorf("Modprobe kvm %v failed: %s", params, err)
		return false
	}
	return true
}

func DetectNestSupport(enableNest bool) string {
	moduleName := GetKVMModuleSupport()
	if moduleName == KVM_MODULE_UNSUPPORT {
		return HOST_NEST_UNSUPPORT
	}
	if !isNestSupport(moduleName) {
		return HOST_NEST_UNSUPPORT
	}
	if loadKvmModuleWithNest(moduleName, enableNest) && enableNest {
		return HOST_NEST_ENABLE
	}
	return HOST_NEST_SUPPORT
}

func isNestSupport(name string) bool {
	output, err := procutils.NewRemoteCommandAsFarAsPossible("modinfo", name).Output()
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

func loadKvmModuleWithNest(name string, enableNest bool) bool {
	var notload = true
	var enabledNest = false
	if IsKernelModuleLoaded(name) {
		nest := GetKernelModuleParameter(name, "nested")
		if strings.TrimSpace(nest) == "Y" {
			enabledNest = true
		}
		if enableNest == enabledNest {
			return true
		}
		notload = unloadKvmModule(name)
	}

	if notload {
		if ModprobeKvmModule(name, false, &enableNest) {
			return true
		}
	}
	return false
}

func unloadKvmModule(name string) bool {
	return ModprobeKvmModule(name, true, nil)
}

func GetKernelModuleParameter(name, moduel string) string {
	pa := path.Join("/sys/module/", strings.Replace(name, "-", "_", -1), "/parameters/", moduel)
	return GetSysConfig(pa)
}

func getSysConfig(pa string, quiet bool) string {
	if f, err := os.Stat(pa); err == nil {
		if f.IsDir() {
			return ""
		}
		cont, err := fileutils2.FileGetContents(pa)
		if err != nil {
			if !quiet {
				log.Errorln(err)
			}
			return ""
		}
		return strings.TrimSpace(cont)
	}
	return ""
}

func GetSysConfig(pa string) string {
	return getSysConfig(pa, false)
}

func GetSysConfigQuiet(pa string) string {
	return getSysConfig(pa, true)
}

func IsKernelModuleLoaded(name string) bool {
	output, err := procutils.NewRemoteCommandAsFarAsPossible("lsmod").Output()
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
		if strings.TrimSpace(string(oval)) != val {
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
