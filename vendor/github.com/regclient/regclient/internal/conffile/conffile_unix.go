//go:build !windows
// +build !windows

package conffile

import (
	"io/fs"
	"syscall"
)

const homeEnv = "HOME"

func getFileOwner(stat fs.FileInfo) (int, int, error) {
	var uid, gid int
	if sysstat, ok := stat.Sys().(*syscall.Stat_t); ok {
		uid = int(sysstat.Uid)
		gid = int(sysstat.Gid)
	}
	return uid, gid, nil
}
