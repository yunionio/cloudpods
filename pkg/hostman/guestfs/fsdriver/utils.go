package fsdriver

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

// SFileInfo implements os.FileInfo interface
type SFileInfo struct {
	name  string
	size  int64
	mode  os.FileMode
	isDir bool
	stat  *syscall.Stat_t
}

func NewFileInfo(name string, size int64, mode os.FileMode, isDir bool, stat *syscall.Stat_t) *SFileInfo {
	return &SFileInfo{name, size, mode, isDir, stat}
}

func (info SFileInfo) Name() string {
	return info.name
}

func (info SFileInfo) Size() int64 {
	return info.size
}

func (info SFileInfo) Mode() os.FileMode {
	return info.mode
}

func (info SFileInfo) IsDir() bool {
	return info.isDir
}

func (info SFileInfo) ModTime() time.Time {
	// TODO: impl
	return time.Now()
}

func (info SFileInfo) Sys() interface{} {
	return info.stat
}

func ModeStr2Bin(mode string) (uint32, error) {
	table := []map[byte]uint32{
		{'-': syscall.S_IRUSR, 'd': syscall.S_IFDIR, 'l': syscall.S_IFLNK},
		{'r': syscall.S_IRUSR},
		{'w': syscall.S_IWUSR},
		{'x': syscall.S_IXUSR, 's': syscall.S_ISUID},
		{'r': syscall.S_IRGRP},
		{'w': syscall.S_IWGRP},
		{'x': syscall.S_IXGRP, 's': syscall.S_ISGID},
		{'r': syscall.S_IROTH},
		{'w': syscall.S_IWOTH},
		{'x': syscall.S_IXOTH},
	}
	if len(mode) != len(table) {
		return 0, fmt.Errorf("Invalid mod %q", mode)
	}
	var ret uint32 = 0
	for i := 0; i < len(table); i++ {
		ret |= table[i][mode[i]]
	}
	return ret, nil
}
