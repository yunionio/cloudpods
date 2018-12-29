package qemuimg

import (
	"fmt"
	"os/exec"
	"regexp"

	"yunion.io/x/log"

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
	cmd := exec.Command(qemutils.GetQemuImg(), "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Errorf("check qemu-img version fail %s", err)
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
