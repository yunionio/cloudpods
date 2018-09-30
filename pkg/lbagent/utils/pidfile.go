package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func ReadPidFile(pidFile string) *os.Process {
	data, err := ioutil.ReadFile(pidFile)
	if err != nil {
		return nil
	}
	s := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return nil
	}
	return proc
}

func WritePidFile(pid int, pidFile string) error {
	data := fmt.Sprintf("%d\n", pid)
	err := ioutil.WriteFile(pidFile, []byte(data), FileModeFile)
	return err
}
