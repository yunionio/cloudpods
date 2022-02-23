// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.
//

package memory

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/jaypipes/ghw/pkg/linuxpath"
	"github.com/jaypipes/ghw/pkg/unitutil"
	"github.com/jaypipes/ghw/pkg/util"
)

const (
	_WARN_CANNOT_DETERMINE_PHYSICAL_MEMORY = `
Could not determine total physical bytes of memory. This may
be due to the host being a virtual machine or container with no
/var/log/syslog file or /sys/devices/system/memory directory, or
the current user may not have necessary privileges to read the syslog.
We are falling back to setting the total physical amount of memory to
the total usable amount of memory
`
)

var (
	// System log lines will look similar to the following:
	// ... kernel: [0.000000] Memory: 24633272K/25155024K ...
	_REGEX_SYSLOG_MEMLINE = regexp.MustCompile(`Memory:\s+\d+K\/(\d+)K`)
)

func (i *Info) load() error {
	paths := linuxpath.New(i.ctx)
	tub := memTotalUsableBytes(paths)
	if tub < 1 {
		return fmt.Errorf("Could not determine total usable bytes of memory")
	}
	i.TotalUsableBytes = tub
	tpb := memTotalPhysicalBytes(paths)
	i.TotalPhysicalBytes = tpb
	if tpb < 1 {
		i.ctx.Warn(_WARN_CANNOT_DETERMINE_PHYSICAL_MEMORY)
		i.TotalPhysicalBytes = tub
	}
	i.SupportedPageSizes = memSupportedPageSizes(paths)
	return nil
}

func memTotalPhysicalBytes(paths *linuxpath.Paths) (total int64) {
	defer func() {
		// fallback to the syslog file approach in case of error
		if total < 0 {
			total = memTotalPhysicalBytesFromSyslog(paths)
		}
	}()

	// detect physical memory from /sys/devices/system/memory
	dir := paths.SysDevicesSystemMemory

	// get the memory block size in byte in hexadecimal notation
	blockSize := filepath.Join(dir, "block_size_bytes")

	d, err := ioutil.ReadFile(blockSize)
	if err != nil {
		return -1
	}
	blockSizeBytes, err := strconv.ParseUint(strings.TrimSpace(string(d)), 16, 64)
	if err != nil {
		return -1
	}

	// iterate over memory's block /sys/devices/system/memory/memory*,
	// if the memory block state is 'online' we increment the total
	// with the memory block size to determine the amount of physical
	// memory available on this system
	sysMemory, err := filepath.Glob(filepath.Join(dir, "memory*"))
	if err != nil {
		return -1
	} else if sysMemory == nil {
		return -1
	}

	for _, path := range sysMemory {
		s, err := ioutil.ReadFile(filepath.Join(path, "state"))
		if err != nil {
			return -1
		}
		if strings.TrimSpace(string(s)) != "online" {
			continue
		}
		total += int64(blockSizeBytes)
	}

	return total
}

func memTotalPhysicalBytesFromSyslog(paths *linuxpath.Paths) int64 {
	// In Linux, the total physical memory can be determined by looking at the
	// output of dmidecode, however dmidecode requires root privileges to run,
	// so instead we examine the system logs for startup information containing
	// total physical memory and cache the results of this.
	findPhysicalKb := func(line string) int64 {
		matches := _REGEX_SYSLOG_MEMLINE.FindStringSubmatch(line)
		if len(matches) == 2 {
			i, err := strconv.Atoi(matches[1])
			if err != nil {
				return -1
			}
			return int64(i * 1024)
		}
		return -1
	}

	// /var/log will contain a file called syslog and 0 or more files called
	// syslog.$NUMBER or syslog.$NUMBER.gz containing system log records. We
	// search each, stopping when we match a system log record line that
	// contains physical memory information.
	logDir := paths.VarLog
	logFiles, err := ioutil.ReadDir(logDir)
	if err != nil {
		return -1
	}
	for _, file := range logFiles {
		if strings.HasPrefix(file.Name(), "syslog") {
			fullPath := filepath.Join(logDir, file.Name())
			unzip := strings.HasSuffix(file.Name(), ".gz")
			var r io.ReadCloser
			r, err = os.Open(fullPath)
			if err != nil {
				return -1
			}
			defer util.SafeClose(r)
			if unzip {
				r, err = gzip.NewReader(r)
				if err != nil {
					return -1
				}
			}

			scanner := bufio.NewScanner(r)
			for scanner.Scan() {
				line := scanner.Text()
				size := findPhysicalKb(line)
				if size > 0 {
					return size
				}
			}
		}
	}
	return -1
}

func memTotalUsableBytes(paths *linuxpath.Paths) int64 {
	// In Linux, /proc/meminfo contains a set of memory-related amounts, with
	// lines looking like the following:
	//
	// $ cat /proc/meminfo
	// MemTotal:       24677596 kB
	// MemFree:        21244356 kB
	// MemAvailable:   22085432 kB
	// ...
	// HugePages_Total:       0
	// HugePages_Free:        0
	// HugePages_Rsvd:        0
	// HugePages_Surp:        0
	// ...
	//
	// It's worth noting that /proc/meminfo returns exact information, not
	// "theoretical" information. For instance, on the above system, I have
	// 24GB of RAM but MemTotal is indicating only around 23GB. This is because
	// MemTotal contains the exact amount of *usable* memory after accounting
	// for the kernel's resident memory size and a few reserved bits. For more
	// information, see:
	//
	//  https://www.kernel.org/doc/Documentation/filesystems/proc.txt
	filePath := paths.ProcMeminfo
	r, err := os.Open(filePath)
	if err != nil {
		return -1
	}
	defer util.SafeClose(r)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		key := strings.Trim(parts[0], ": \t")
		if key != "MemTotal" {
			continue
		}
		value, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return -1
		}
		inKb := (len(parts) == 3 && strings.TrimSpace(parts[2]) == "kB")
		if inKb {
			value = value * int(unitutil.KB)
		}
		return int64(value)
	}
	return -1
}

func memSupportedPageSizes(paths *linuxpath.Paths) []uint64 {
	// In Linux, /sys/kernel/mm/hugepages contains a directory per page size
	// supported by the kernel. The directory name corresponds to the pattern
	// 'hugepages-{pagesize}kb'
	dir := paths.SysKernelMMHugepages
	out := make([]uint64, 0)

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, file := range files {
		parts := strings.Split(file.Name(), "-")
		sizeStr := parts[1]
		// Cut off the 'kb'
		sizeStr = sizeStr[0 : len(sizeStr)-2]
		size, err := strconv.Atoi(sizeStr)
		if err != nil {
			return out
		}
		out = append(out, uint64(size*int(unitutil.KB)))
	}
	return out
}
