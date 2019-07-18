package executor

import (
	"bytes"
	"io"
	"syscall"

	"github.com/pkg/errors"
	"yunion.io/x/log"

	execapi "yunion.io/x/onecloud/pkg/executor/apis"
)

type Process struct {
	client execapi.Executor_ExecCommandClient

	streamError chan error
	ExitStatus  syscall.WaitStatus
	ErrContent  string

	stdin  io.Reader
	stderr io.Writer
	stdout io.Writer
}

func strArrayToBytesArray(sa []string) [][]byte {
	if len(sa) == 0 {
		return nil
	}
	res := make([][]byte, len(sa))
	for i := 0; i < len(sa); i++ {
		res[i] = []byte(sa[i])
	}
	return res
}

func startProcess(
	client execapi.Executor_ExecCommandClient,
	path string,
	argv, env []string,
	dir string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) (*Process, error) {
	c := &execapi.Command{
		Path: []byte(path),
		Args: strArrayToBytesArray(argv),
		Env:  strArrayToBytesArray(env),
		Dir:  []byte(dir),
	}

	if stdout == nil {
		stdout = &bytes.Buffer{}
	}
	if stderr == nil {
		stderr = &bytes.Buffer{}
	}

	p := &Process{
		client:      client,
		streamError: make(chan error, 2),
		stdin:       stdin,
		stdout:      stdout,
		stderr:      stderr,
	}
	err := p.client.Send(c)
	if err != nil {
		return nil, errors.Wrap(err, "grpc send cmd error")
	}
	if p.stdin != nil {
		go p.streamInput()
	}
	go p.fetchOutput()
	return p, nil
}

func (p *Process) streamInput() {
	if p.stdin == nil {
		return
	}
	var data = make([]byte, 4096)
	for {
		n, err := p.stdin.Read(data)
		if err == io.EOF {
			break
		}
		if err != nil {
			p.streamError <- errors.Wrap(err, "process read from stdin")
			break
		}
		if n > 0 {
			c := &execapi.Command{
				Input: data[:n],
			}
			if err := p.client.Send(c); err != nil {
				p.streamError <- errors.Wrap(err, "grpc send input")
				break
			}
		}
	}
}

func (p *Process) fetchOutput() {
	for {
		res, err := p.client.Recv()
		if err == io.EOF {
			p.streamError <- nil
			return
		}
		if err != nil {
			p.streamError <- errors.Wrap(err, "grpc recv error")
			return
		}
		switch {
		case res.IsExit:
			p.ExitStatus = syscall.WaitStatus(res.ExitStatus)
			p.streamError <- nil
			p.ErrContent = string(res.ErrContent)
			return
		case len(res.Stderr) > 0:
			io.Copy(p.stderr, bytes.NewBuffer(res.Stderr))
		case len(res.Stdout) > 0:
			io.Copy(p.stdout, bytes.NewBuffer(res.Stdout))
		default:
			log.Warningf("grpc recv empty message ?? %s", res)
		}
	}
}

func (p *Process) Kill() error {
	c := &execapi.Command{KillProcess: true}
	if err := p.client.Send(c); err != nil {
		err = errors.Wrap(err, "grpc send kill process")
		p.streamError <- err
		return err
	}
	return nil
}

func (p *Process) Wait() error {
	err := <-p.streamError
	if err != nil {
		return err
	}
	if len(p.ErrContent) > 0 {
		return errors.New(p.ErrContent)
	}
	if p.ExitStatus.ExitStatus() == 0 {
		return nil
	} else {
		return &ExitError{ExitStatus: p.ExitStatus}
	}
}

// Convert integer to decimal string
func itoa(val int) string {
	if val < 0 {
		return "-" + uitoa(uint(-val))
	}
	return uitoa(uint(val))
}

// Convert unsigned integer to decimal string
func uitoa(val uint) string {
	if val == 0 { // avoid string allocation
		return "0"
	}
	var buf [20]byte // big enough for 64bit value base 10
	i := len(buf) - 1
	for val >= 10 {
		q := val / 10
		buf[i] = byte('0' + val - q*10)
		i--
		val = q
	}
	// val < 10
	buf[i] = byte('0' + val)
	return string(buf[i:])
}

// Convert exit status to error string
// Source code in exec posix
func exitStatusToString(status syscall.WaitStatus) string {
	res := ""
	switch {
	case status.Exited():
		res = "exit status " + itoa(status.ExitStatus())
	case status.Signaled():
		res = "signal: " + status.Signal().String()
	case status.Stopped():
		res = "stop signal: " + status.StopSignal().String()
		if status.StopSignal() == syscall.SIGTRAP && status.TrapCause() != 0 {
			res += " (trap " + itoa(status.TrapCause()) + ")"
		}
	case status.Continued():
		res = "continued"
	}
	if status.CoreDump() {
		res += " (core dumped)"
	}
	return res
}

type ExitError struct {
	ExitStatus syscall.WaitStatus
	Stderr     []byte
}

func (e *ExitError) Error() string {
	return exitStatusToString(e.ExitStatus)
}
