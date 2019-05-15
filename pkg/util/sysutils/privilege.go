package sysutils

import "os"

func IsRootPermission() bool {
	return os.Geteuid() == 0
}
