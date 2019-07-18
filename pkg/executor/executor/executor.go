package executor

import (
	"bytes"
	"context"
	"io"
	"net"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	execapi "yunion.io/x/onecloud/pkg/executor/apis"
)

var exec *Executor

type Executor struct {
	socketPath string
}

func Init(socketPath string) {
	exec = &Executor{socketPath}
}

func Command(path string, args ...string) *Cmd {
	if exec == nil {
		panic("executor not init ???")
	}
	return &Cmd{
		Path:     path,
		Args:     args,
		executor: exec,
		err:      make(chan error),
	}
}

func CommandContext(ctx context.Context, path string, args ...string) *Cmd {
	if ctx == nil {
		panic("nil Context")
	}
	cmd := Command(path, args...)
	cmd.ctx = ctx
	return cmd
}

func NewExecutorCommand(socketPath, path string, args ...string) *Cmd {
	return &Cmd{
		Path:     path,
		Args:     args,
		executor: &Executor{socketPath},
		err:      make(chan error),
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

// convert exit code to error string
// source code in exec posix
func exitCodeToString(exitCode int) string {
	status := syscall.WaitStatus(exitCode)
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
	ExitCode int
	Stderr   []byte
}

func (e *ExitError) Error() string {
	return exitCodeToString(e.ExitCode)
}

type Cmd struct {
	Path string
	Args []string
	Env  []string
	Dir  string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	ctx      context.Context
	executor *Executor
	conn     *grpc.ClientConn

	Proc     *Process
	finished bool
	err      chan error
	waitDone chan struct{}
}

func grcpDialWithUnixSocket(ctx context.Context, socketPath string) (*grpc.ClientConn, error) {
	return grpc.DialContext(
		ctx, socketPath,
		grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Second*3),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)
}

func (c *Cmd) initializeClient(ctx context.Context, opts ...grpc.CallOption,
) (*grpc.ClientConn, execapi.Executor_ExecCommandClient, error) {
	conn, err := grcpDialWithUnixSocket(ctx, c.executor.socketPath)
	if err != nil {
		return nil, nil, errors.Wrap(err, "grpc dial error")
	}
	client, err := execapi.NewExecutorClient(conn).ExecCommand(ctx, opts...)
	if err != nil {
		return conn, nil, errors.Wrap(err, "grpc new client error")
	}
	return conn, client, nil
}

func (c *Cmd) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

func (c *Cmd) Output() ([]byte, error) {
	if c.Stdout != nil {
		return nil, errors.New("exec: Stdout already set")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	// function run return err its mean grpc stream transport error
	// cmd execute error indicate by exit code
	if err := c.Run(); err != nil {
		if e, ok := err.(*ExitError); ok {
			e.Stderr = stderr.Bytes()
		}
		return nil, err
	}
	return stdout.Bytes(), nil
}

func (c *Cmd) Start() error {
	conn, client, err := c.initializeClient(context.Background())
	if err != nil {
		if conn != nil {
			conn.Close()
		}
		return err
	} else {
		c.conn = conn
	}

	if c.ctx != nil {
		select {
		case <-c.ctx.Done():
			c.conn.Close()
			return c.ctx.Err()
		default:
		}
	}

	c.Proc, err = startProcess(
		client, c.Path, c.Args, c.Env, c.Dir,
		c.Stdin, c.Stdout, c.Stderr,
	)
	if err != nil {
		c.conn.Close()
		return err
	}

	if c.ctx != nil {
		c.waitDone = make(chan struct{})
		go func() {
			select {
			case <-c.ctx.Done():
				c.Proc.Kill()
			case <-c.waitDone:
			}
		}()
	}
	return nil
}

func (c *Cmd) Wait() error {
	if c.conn == nil {
		return errors.New("conn: not connect")
	}
	if c.finished {
		return errors.New("exec: Wait was already called")
	}
	err := c.Proc.Wait()
	c.finished = true
	if c.waitDone != nil {
		close(c.waitDone)
	}
	c.conn.Close()
	if err != nil {
		return err
	}
	if c.Proc.ExitCode > 0 {
		return &ExitError{ExitCode: c.Proc.ExitCode}
	}

	if c.Proc.ExitCode != 0 {
		return &ExitError{
			ExitCode: c.Proc.ExitCode,
		}
	} else {
		return nil
	}
}
