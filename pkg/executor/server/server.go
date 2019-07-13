package server

import (
	"context"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/pkg/errors"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/executor/apis"
	"yunion.io/x/onecloud/pkg/executor/utils"
)

var sn int32

func GetSN() uint32 {
	return uint32(atomic.AddInt32(&sn, 1))
}

var socketPath string

func Init(socketPath string) {
	socketPath = socketPath
	cmds = &sync.Map{}
}

func Len(sm *sync.Map) int {
	lengh := 0
	f := func(key, value interface{}) bool {
		lengh++
		return true
	}
	sm.Range(f)
	return lengh
}

var cmds *sync.Map

type Executor struct{}

func (e *Executor) ExecCommand(ctx context.Context, req *apis.Command) (*apis.Sn, error) {
	cm := NewCommander(req)
	sn := GetSN()
	log.Infof("%d/%d Exec %s", sn, Len(cmds), req.String())
	cmds.Store(sn, cm)
	return &apis.Sn{Sn: sn}, nil
}

func (e *Executor) Start(ctx context.Context, req *apis.StartInput) (*apis.StartResponse, error) {
	icm, ok := cmds.Load(req.Sn)
	if !ok {
		return nil, errors.Errorf("unknown sn %d", req.Sn)
	}
	var (
		m   = icm.(*Commander)
		err error
	)
	if req.HasStdin {
		m.stdin, err = m.c.StdinPipe()
		if err != nil {
			return &apis.StartResponse{
				Success: false,
				Error:   []byte(err.Error()),
			}, nil
		}
	}
	if req.HasStdout {
		m.stdout, err = m.c.StdoutPipe()
		if err != nil {
			return &apis.StartResponse{
				Success: false,
				Error:   []byte(err.Error()),
			}, nil
		}
	}
	if req.HasStderr {
		m.stderr, err = m.c.StderrPipe()
		if err != nil {
			return &apis.StartResponse{
				Success: false,
				Error:   []byte(err.Error()),
			}, nil
		}
	}

	if err := m.c.Start(); err != nil {
		return &apis.StartResponse{
			Success: false,
			Error:   []byte(err.Error()),
		}, nil
	}

	return &apis.StartResponse{
		Success: true,
		Error:   nil,
	}, nil
}

func (e *Executor) Wait(ctx context.Context, in *apis.Sn) (*apis.WaitResponse, error) {
	icm, ok := cmds.Load(in.Sn)
	if !ok {
		return nil, errors.Errorf("unknown sn %d", in.Sn)
	}
	var (
		m   = icm.(*Commander)
		err error
	)

	if m.stdout != nil {
		<-m.stdoutCh
	}
	if m.stderr != nil {
		<-m.stderrCh
	}

	m.wg.Wait()
	err = m.c.Wait()
	var (
		exitStatus uint32
		errContent string
	)
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0
			// This works on both Unix and Windows. Although package
			// syscall is generally platform dependent, WaitStatus is
			// defined for both Unix and Windows and in both cases has
			// an ExitStatus() method with the same signature.
			exitStatus = uint32(exiterr.Sys().(syscall.WaitStatus))
		} else {
			// command not found or io problem
			errContent = err.Error()
		}
	} else {
		exitStatus = 0
	}
	cmds.Delete(in.Sn) // 其他地方也能也需要delete
	return &apis.WaitResponse{
		ExitStatus: exitStatus,
		ErrContent: []byte(errContent),
	}, nil
}

func (e *Executor) Kill(ctx context.Context, req *apis.Sn) (*apis.Error, error) {
	icm, ok := cmds.Load(req.Sn)
	if !ok {
		return nil, errors.Errorf("unknown sn %d", req.Sn)
	}

	m := icm.(*Commander)
	err := m.c.Process.Kill()
	if err != nil {
		return &apis.Error{Error: []byte(err.Error())}, nil
	}
	return &apis.Error{}, nil
}

func (e *Executor) SendInput(s apis.Executor_SendInputServer) error {
	var m *Commander
	for {
		input, err := s.Recv()
		if err == io.EOF {
			return s.SendAndClose(&apis.Error{})
		} else if err != nil {
			return s.SendAndClose(&apis.Error{
				Error: []byte(err.Error()),
			})
		}
		if m == nil {
			icm, ok := cmds.Load(input.Sn)
			if !ok {
				return errors.Errorf("unknown sn %d", input.Sn)
			}
			m = icm.(*Commander)
			if m.stdin == nil {
				return errors.New("Process stdin not init")
			}
		}
		_, err = m.stdin.Write(input.Input)
		if err != nil {
			return s.SendAndClose(&apis.Error{
				Error: []byte(err.Error()),
			})
		}
	}
}

func (e *Executor) FetchStdout(sn *apis.Sn, s apis.Executor_FetchStdoutServer) error {
	icm, ok := cmds.Load(sn.Sn)
	if !ok {
		return errors.Errorf("unknown sn %d", sn.Sn)
	}
	var (
		m    = icm.(*Commander)
		data = make([]byte, 4096)
		err  error
		n    int
	)

	if m.stdout == nil {
		return errors.New("Process stdout not init")
	}

	close(m.stdoutCh)
	m.wg.Add(1)
	defer m.wg.Done()
	s.Send(&apis.Stdout{Start: true})
	for {
		n, err = m.stdout.Read(data)
		if err == io.EOF {
			return s.Send(&apis.Stdout{Closed: true})
		} else if pe, ok := err.(*os.PathError); ok && pe.Err == os.ErrClosed {
			return s.Send(&apis.Stdout{Closed: true})
		} else if err != nil {
			return s.Send(&apis.Stdout{RuntimeError: []byte(err.Error())})
		}
		err = s.Send(&apis.Stdout{Stdout: data[:n]})
		if err != nil {
			return err
		}
	}
}

func (e *Executor) FetchStderr(sn *apis.Sn, s apis.Executor_FetchStderrServer) error {
	icm, ok := cmds.Load(sn.Sn)
	if !ok {
		return errors.Errorf("unknown sn %d", sn.Sn)
	}
	var (
		m    = icm.(*Commander)
		data = make([]byte, 4096)
		err  error
		n    int
	)

	if m.stderr == nil {
		return errors.New("Process stderr not init")
	}

	close(m.stderrCh)
	m.wg.Add(1)
	defer m.wg.Done()
	s.Send(&apis.Stderr{Start: true})
	for {
		n, err = m.stderr.Read(data)
		if err == io.EOF {
			return s.Send(&apis.Stderr{Closed: true})
		} else if pe, ok := err.(*os.PathError); ok && pe.Err == os.ErrClosed {
			return s.Send(&apis.Stderr{Closed: true})
		} else if err != nil {
			return s.Send(&apis.Stderr{RuntimeError: []byte(err.Error())})
		}
		err = s.Send(&apis.Stderr{Stderr: data[:n]})
		if err != nil {
			return err
		}
	}
}

type Commander struct {
	// stream apis.Executor_ExecCommandServer

	c      *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser

	wg       *sync.WaitGroup
	stdoutCh chan struct{}
	stderrCh chan struct{}
}

func NewCommander(in *apis.Command) *Commander {
	cmd := exec.Command(string(in.Path), utils.BytesArrayToStrArray(in.Args)...)
	if len(in.Env) > 0 {
		cmd.Env = utils.BytesArrayToStrArray(in.Env)
	}
	if len(in.Dir) > 0 {
		cmd.Dir = string(in.Dir)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return &Commander{
		c:        cmd,
		wg:       new(sync.WaitGroup),
		stdoutCh: make(chan struct{}),
		stderrCh: make(chan struct{}),
	}
}
