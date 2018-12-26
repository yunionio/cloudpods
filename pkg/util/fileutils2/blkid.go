package fileutils2

import (
	"regexp"
	"os/exec"
	"yunion.io/x/log"
)

const (
	blkidTypePattern = `TYPE="(?P<type>\w+)"`
)

var (
	blkidTypeRegexp = regexp.MustCompile(blkidTypePattern)
)

func GetBlkidType(filepath string) string {
	cmd := exec.Command("blkid", filepath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Errorf("blkid fail %s %s", filepath, err)
		return ""
	}
	matches := blkidTypeRegexp.FindStringSubmatch(string(out))
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
