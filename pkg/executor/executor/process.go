package executor

import (
	"bytes"
	"io"

	"github.com/pkg/errors"
	"yunion.io/x/log"

	execapi "yunion.io/x/onecloud/pkg/executor/apis"
)

type Process struct {
	client execapi.Executor_ExecCommandClient

	streamError chan error
	ExitCode    int
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
			p.ExitCode = int(res.ExitCode)
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

func (p *Process) Wait() error {
	err := <-p.streamError
	if err != nil {
		return err
	}
	if p.ExitCode < 0 {
		return errors.New(p.ErrContent)
	}
	return nil
}
