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

package pod

import "yunion.io/x/onecloud/pkg/apis"

// https://github.com/kubernetes/kubernetes/blob/release-1.26/pkg/securitycontext/util.go#L213-L236
var (
	// These *must* be kept in sync with moby/moby.
	// https://github.com/moby/moby/blob/master/oci/defaults.go#L116-L134
	// @jessfraz will watch changes to those files upstream.
	defaultMaskedPaths = []string{
		"/proc/acpi",
		"/proc/kcore",
		"/proc/keys",
		"/proc/latency_stats",
		"/proc/timer_list",
		"/proc/timer_stats",
		"/proc/sched_debug",
		"/proc/scsi",
		"/sys/firmware",
	}
	defaultReadonlyPaths = []string{
		"/proc/asound",
		"/proc/bus",
		"/proc/fs",
		"/proc/irq",
		"/proc/sys",
		"/proc/sysrq-trigger",
	}
)

func GetDefaultMaskedPaths(unmasks apis.ContainerProcMountType) []string {
	if unmasks == apis.ContainerUnmaskedProcMount {
		return []string{}
	}
	return defaultMaskedPaths
}

func GetReadonlyPaths(unmasks apis.ContainerProcMountType) []string {
	if unmasks == apis.ContainerUnmaskedProcMount {
		return []string{}
	}
	return defaultReadonlyPaths
}
