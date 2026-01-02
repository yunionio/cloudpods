//go:build linux
// +build linux

package fileutils2

import (
	"syscall"
	"time"
)

func GetAtim(stat *syscall.Stat_t) time.Time {
	return time.Unix(stat.Atim.Sec, stat.Atim.Nsec)
}
