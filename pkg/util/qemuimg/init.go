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

package qemuimg

import (
	"fmt"
	"regexp"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemutils"
	"yunion.io/x/onecloud/pkg/util/version"
)

const (
	qemuImgVersionPattern = `qemu-img version (?P<ver>\d+\.\d+(\.\d+)?)`
)

var (
	qemuImgVersionRegexp = regexp.MustCompile(qemuImgVersionPattern)

	qemuImgVersion string
)

func getQemuImgVersion() string {
	out, err := procutils.NewRemoteCommandAsFarAsPossible(qemutils.GetQemuImg(), "--version").Output()
	if err != nil {
		log.Errorf("check qemu-img version fail %s", out)
		return ""
	}
	matches := qemuImgVersionRegexp.FindStringSubmatch(string(out))
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func QemuImgInit() error {
	ver := getQemuImgVersion()
	if len(ver) == 0 {
		return fmt.Errorf("fail to find qemu-img")
	}
	qemuImgVersion = ver
	return nil
}

func qcow2SparseOptions() []string {
	if version.LE(qemuImgVersion, "1.1") {
		return []string{"preallocation=metadata", "cluster_size=2M"}
	} else if version.LE(qemuImgVersion, "1.7.1") {
		return []string{"preallocation=metadata", "lazy_refcounts=on"}
	} else if version.LE(qemuImgVersion, "2.2") {
		return []string{"preallocation=metadata", "lazy_refcounts=on", "cluster_size=2M"}
	} else {
		return []string{}
	}
}
