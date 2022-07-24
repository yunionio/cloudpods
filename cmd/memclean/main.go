package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"syscall"
	"time"

	"github.com/sevlyar/go-daemon"
	"golang.org/x/sys/unix"

	"yunion.io/x/log"
	"yunion.io/x/log/hooks"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/signalutils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/structarg"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

const (
	MEM_BACKEND_FD = "/memfd:memory-backend-memfd (deleted)"
	MMAP_SIZE      = 2 * 1024 * 1024 * 1024
)

type Options struct {
	Pid        int    `help:"qemu process pid" required:"true"`
	MemSize    int64  `help:"backend memory size" required:"true"`
	Foreground bool   `help:"run in foreground"`
	LogDir     string `help:"log dir" required:"true"`
}

var opt = &Options{}

func main() {
	procDir := fmt.Sprintf("/proc/%d", opt.Pid)
	if !fileutils2.IsDir(procDir) {
		log.Fatalf("Process %d not found", opt.Pid)
	}

	fdDir := fmt.Sprintf("%s/fd", procDir)
	memBackendFd, err := findMemBackendFd(fdDir)
	if err != nil {
		log.Fatalf("findMemBackendFd: %s", err)
	}
	log.Infof("found mem backend fd: %s", memBackendFd)

	f, err := os.OpenFile(memBackendFd, os.O_RDWR, 0644)
	if err != nil {
		log.Fatalf("failed open memfd: %s", err)
	}
	defer f.Close()

	if !opt.Foreground {
		cntxt := &daemon.Context{
			WorkDir: "./",
			Umask:   027,
		}

		d, err := cntxt.Reborn()
		if err != nil {
			log.Fatalf("Unable to run in background: %s", err)
		}
		if d != nil {
			return
		}
		defer cntxt.Release()
	}

	log.Infof("start watch proc exit")
	err = procutils.NewCommand("tail", fmt.Sprintf("--pid=%d", opt.Pid), "-f", "/dev/null").Run()
	if err != nil {
		log.Fatalf("failed watch process: %s", err)
	}

	log.Infof("watch proc exited, go to clean memory")
	var (
		start       = time.Now()
		size  int64 = 0
	)
	for size < opt.MemSize {
		mmapSize := MMAP_SIZE
		if opt.MemSize-size < MMAP_SIZE {
			mmapSize = int(opt.MemSize - size)
		}

		b, err := syscall.Mmap(
			int(f.Fd()), size, mmapSize,
			syscall.PROT_WRITE|syscall.PROT_READ,
			syscall.MAP_SHARED,
		)
		if err != nil {
			log.Fatalf("failed mmap mem backend fd: %s", err)
		}
		log.Infof("mmap memory offset %d, size %d", size, len(b))
		size += int64(len(b))

		// memsetRepeat(b, 0)
		for i := 0; i < len(b); i++ {
			b[i] = 0
		}
		unix.Msync(b, unix.MS_SYNC)
		syscall.Munmap(b)
	}

	log.Infof(
		"mem clean for process %d success, mem clean took %s",
		opt.Pid, time.Since(start),
	)
}

func findMemBackendFd(fdDir string) (string, error) {
	files, err := ioutil.ReadDir(fdDir)
	if err != nil {
		return "", errors.Wrapf(err, "read dir %s", fdDir)
	}

	for _, f := range files {
		p1 := path.Join(fdDir, f.Name())
		p2, e := os.Readlink(p1)
		if e != nil {
			log.Errorf("os.readlink %s: %s", p1, e)
			continue
		}
		log.Infof("os.readlink path %s", p2)
		if p2 == MEM_BACKEND_FD {
			return p1, nil
		}
	}

	return "", errors.Errorf("no mem backend fd found")
}

func memsetRepeat(a []byte, v byte) {
	if len(a) == 0 {
		return
	}
	a[0] = v
	for bp := 1; bp < len(a); bp *= 2 {
		copy(a[bp:], a[:bp])
	}
}

func init() {
	signalutils.RegisterSignal(func() {
		utils.DumpAllGoroutineStack(log.Logger().Out)
	}, syscall.SIGUSR1)
	signalutils.StartTrap()

	parser, err := structarg.NewArgumentParser(opt, "", "", "")
	if err != nil {
		log.Fatalf("Error define argument parser: %v", err)
	}
	err = parser.ParseArgs2(os.Args[1:], true, true)
	if err != nil {
		log.Fatalf("Failed parse args %s", err)
	}

	logFileHook := hooks.LogFileRotateHook{
		LogFileHook: hooks.LogFileHook{
			FileDir:  opt.LogDir,
			FileName: "memclean.log",
		},
		RotateNum:  10,
		RotateSize: 100 * 1024 * 1024,
	}
	logFileHook.Init()
	log.Logger().AddHook(&logFileHook)
}
