package sysutils

import (
	"os"
	"os/exec"
)

func Start(closeFd bool, args ...string) (p *os.Process, err error) {
	if args[0], err = exec.LookPath(args[0]); err == nil {
		var procAttr os.ProcAttr
		if closeFd {
			procAttr.Files = []*os.File{nil, nil, nil}
		} else {
			procAttr.Files = []*os.File{os.Stdin,
				os.Stdout, os.Stderr}
		}
		p, err := os.StartProcess(args[0], args, &procAttr)
		if err == nil {
			return p, nil
		}
	}
	return nil, err
}
