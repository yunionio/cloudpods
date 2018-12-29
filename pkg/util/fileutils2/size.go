package fileutils2

import "os"

func FileSize(name string) int64 {
	info, err := os.Stat(name)
	if err != nil {
		return -1
	}
	return info.Size()
}
