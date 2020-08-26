package client

import (
	"bytes"
	"context"
	"io"
	"net"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"yunion.io/x/executor/apis"
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
		Executor: exec,
		Path:     path,
		Args:     args,
		wg:       new(sync.WaitGroup),
		stdoutCh: make(chan struct{}),
		stderrCh: make(chan struct{}),
	}
}

func CommandContext(ctx context.Context, path string, args ...string) *Cmd {
	if exec == nil {
		panic("executor not init ???")
	}
	return &Cmd{
		Executor: exec,
		Path:     path,
		Args:     args,
		wg:       new(sync.WaitGroup),
		stdoutCh: make(chan struct{}),
		stderrCh: make(chan struct{}),
		ctx:      ctx,
	}
}

type Cmd struct {
	*Executor

	ctx context.Context

	Path string
	Args []string
	Env  []string
	Dir  string

	conn   *grpc.ClientConn
	client apis.ExecutorClient

	sn     *apis.Sn
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	closeAfterWait  []io.Closer
	closeAfterAfter []io.Closer
	goroutine       []func() error
	errch           chan error

	waitDone chan struct{}
	stdoutCh chan struct{}
	stderrCh chan struct{}

	fetchError   chan error
	streamStdin  error
	streamStdout error
	streamStderr error

	wg             *sync.WaitGroup
	combinedOutput chan struct{}
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

func (c *Cmd) Connect(ctx context.Context, opts ...grpc.CallOption,
) error {
	var err error
	c.conn, err = grcpDialWithUnixSocket(ctx, c.socketPath)
	if err != nil {
		return errors.Wrap(err, "grpc dial error")
	}
	c.client = apis.NewExecutorClient(c.conn)
	return nil
}

func (c *Cmd) Run() error {
	if err := c.Start(); err != nil {
		return err
	}
	return c.Wait()
}

func (c *Cmd) CombinedOutput() ([]byte, error) {
	if c.Stdout != nil {
		return nil, errors.New("exec: Stdout already set")
	}
	if c.Stderr != nil {
		return nil, errors.New("exec: Stderr already set")
	}

	var b bytes.Buffer
	c.Stdout = &b
	c.Stderr = &b
	err := c.Run()
	return b.Bytes(), err
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
	if c.conn != nil {
		return errors.New("cmd executing")
	}
	if err := c.Connect(context.Background()); err != nil {
		return err
	}

	sn, err := c.client.ExecCommand(context.Background(), &apis.Command{
		Path: []byte(c.Path),
		Args: strArrayToBytesArray(c.Args),
		Env:  strArrayToBytesArray(c.Env),
		Dir:  []byte(c.Dir),
	})
	if err != nil {
		c.closeDescriptors()
		return errors.Wrap(err, "grcp exec command")
	}
	c.sn = sn

	if c.ctx != nil {
		select {
		case <-c.ctx.Done():
			c.closeDescriptors()
			return c.ctx.Err()
		default:
		}
	}

	var procIO = [3]*os.File{}
	type F func(*Cmd) (*os.File, error)
	for i, setupFd := range [3]F{(*Cmd).stdin, (*Cmd).stdout, (*Cmd).stderr} {
		if i == 2 && c.Stderr != nil && interfaceEqual(c.Stderr, c.Stdout) {
			procIO[2] = procIO[1]
			c.combinedOutput = make(chan struct{}, 2)
			continue
		}
		fd, err := setupFd(c)
		if err != nil {
			c.closeDescriptors()
			return errors.Wrap(err, "setup fd")
		}
		procIO[i] = fd
	}

	input := &apis.StartInput{
		Sn:        c.sn.Sn,
		HasStdin:  procIO[0] != nil,
		HasStdout: procIO[1] != nil,
		HasStderr: procIO[2] != nil,
	}

	res, err := c.client.Start(context.Background(), input)
	if err != nil {
		c.closeDescriptors()
		return errors.Wrap(err, "grpc start cmd")
	}

	if !res.Success {
		c.closeDescriptors()
		return errors.New(string(res.Error))
	}

	if procIO[0] != nil {
		go c.sendStdin(procIO[0])
	}
	if procIO[1] != nil {
		go c.fetchStdout(procIO[1])
		<-c.stdoutCh
	}
	if procIO[2] != nil {
		go c.fetchStderr(procIO[2])
		<-c.stderrCh
	}
	if c.combinedOutput != nil {
		go func(wc io.WriteCloser) {
			var closed bool
			for {
				select {
				case <-c.combinedOutput:
					if closed {
						wc.Close()
						return
					} else {
						closed = true
					}
				}
			}
		}(procIO[1])
	}

	c.errch = make(chan error, len(c.goroutine))
	for _, fn := range c.goroutine {
		go func(fn func() error) {
			c.errch <- fn()
		}(fn)
	}

	if c.ctx != nil {
		c.waitDone = make(chan struct{})
		go func() {
			select {
			case <-c.ctx.Done():
				c.Kill()
			case <-c.waitDone:
			}
		}()
	}

	return nil
}

func (c *Cmd) streamError() error {
	if c.streamStdin != nil {
		return c.streamStdin
	}
	if c.streamStdout != nil {
		return c.streamStdout
	}
	if c.streamStderr != nil {
		return c.streamStderr
	}
	return nil
}

func (c *Cmd) Kill() error {
	e, err := c.client.Kill(context.Background(), c.sn)
	if err != nil {
		return errors.Wrap(err, "grpc send kill")
	}
	if len(e.Error) > 0 {
		return errors.Errorf("kill process %s", e.Error)
	}
	return nil
}

func (c *Cmd) Wait() error {
	if c.conn == nil {
		return errors.New("cmd not executing")
	}

	res, err := c.client.Wait(context.Background(), c.sn)
	if err != nil {
		c.closeDescriptors()
		return errors.Wrap(err, "grpc wait proc")
	}

	if c.waitDone != nil {
		close(c.waitDone)
	}

	if err := c.streamError(); err != nil {
		c.closeDescriptors()
		return err
	}

	c.wg.Wait()

	if len(res.ErrContent) > 0 {
		return errors.New(string(res.ErrContent))
	}

	var copyError error
	for range c.goroutine {
		if err := <-c.errch; err != nil && copyError == nil {
			copyError = err
		}
	}

	c.closeDescriptors()

	if res.ExitStatus == 0 {
		if copyError != nil {
			return copyError
		}
		return nil
	} else {
		return &ExitError{ExitStatus: newWaitStatus(res.ExitStatus)}
	}
}

func (c *Cmd) closeDescriptors() {
	for _, fd := range c.closeAfterWait {
		fd.Close()
	}
	for _, fd := range c.closeAfterAfter {
		fd.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Cmd) StdinPipe() (io.WriteCloser, error) {
	if c.Stdin != nil {
		return nil, errors.New("exec: Stdin already set")
	}
	if c.conn != nil {
		return nil, errors.New("exec: StdinPipe after process started")
	}
	// do not use io.Pipe, block forever
	// https://stackoverflow.com/questions/47486128
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, errors.Wrap(err, "open stdinpipe")
	}
	c.Stdin = pr
	c.closeAfterWait = append(c.closeAfterWait, pr)
	wc := &CloseOnce{File: pw}
	c.closeAfterAfter = append(c.closeAfterAfter, wc)
	return wc, nil
}

func (c *Cmd) stdin() (*os.File, error) {
	if c.Stdin == nil {
		return nil, nil
	}

	if f, ok := c.Stdin.(*os.File); ok {
		return f, nil
	}

	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	c.closeAfterWait = append(c.closeAfterWait, pr)
	c.goroutine = append(c.goroutine, func() error {
		_, err := io.Copy(pw, c.Stdin)
		if err1 := pw.Close(); err == nil {
			err = err1
		}
		return err
	})
	return pr, nil
}

func (c *Cmd) stdout() (f *os.File, err error) {
	return c.writerDescriptor(c.Stdout)
}

func (c *Cmd) stderr() (f *os.File, err error) {
	return c.writerDescriptor(c.Stderr)
}

// interfaceEqual protects against panics from doing equality tests on
// two interfaces with non-comparable underlying types.
func interfaceEqual(a, b interface{}) bool {
	defer func() {
		recover()
	}()
	return a == b
}

func (c *Cmd) writerDescriptor(w io.Writer) (*os.File, error) {
	if w == nil {
		return nil, nil
	}

	if f, ok := w.(*os.File); ok {
		return f, nil
	}

	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	c.closeAfterWait = append(c.closeAfterWait, pw)
	c.goroutine = append(c.goroutine, func() error {
		_, err := io.Copy(w, pr)
		pr.Close() // in case io.Copy stopped due to write error
		return err
	})
	return pw, nil
}

func (c *Cmd) StdoutPipe() (io.ReadCloser, error) {
	if c.Stdout != nil {
		return nil, errors.New("exec: Stdout already set")
	}
	if c.conn != nil {
		return nil, errors.New("exec: StdoutPipe after process started")
	}
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, errors.Wrap(err, "open stdoutpipe")
	}
	c.Stdout = pw
	c.closeAfterWait = append(c.closeAfterWait, pw)
	wc := &CloseOnce{File: pr}
	c.closeAfterAfter = append(c.closeAfterAfter, wc)
	return wc, nil
}

func (c *Cmd) StderrPipe() (io.ReadCloser, error) {
	if c.Stderr != nil {
		return nil, errors.New("exec: Stderr already set")
	}
	if c.conn != nil {
		return nil, errors.New("exec: StderrPipe after process started")
	}
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, errors.Wrap(err, "open stderrpipe")
	}
	c.Stderr = pw
	c.closeAfterWait = append(c.closeAfterWait, pw)
	wc := &CloseOnce{File: pr}
	c.closeAfterAfter = append(c.closeAfterAfter, wc)
	return wc, nil
}

func (c *Cmd) sendStdin(r io.Reader) {
	stream, err := c.client.SendInput(context.Background())
	if err != nil {
		c.streamStdin = errors.Wrap(err, "grpc send input")
		return
	}

	var data = make([]byte, 4096)
	for {
		n, err := r.Read(data)
		if err == io.EOF {
			e, err := stream.CloseAndRecv()
			if err != nil {
				c.streamStdin = errors.Wrap(err, "grpc send stdin on close and recv")
				return
			}
			if len(e.Error) > 0 {
				c.streamStdin = errors.New(string(e.Error))
				return
			}
			return
		} else if err != nil {
			c.streamStdin = errors.Wrap(err, "read from stdin")
			return
		}
		err = stream.Send(&apis.Input{
			Sn:    c.sn.Sn,
			Input: data[:n],
		})
		if err != nil {
			c.streamStdin = errors.Wrap(err, "grpc send stdin")
			return
		}
	}
}

func (c *Cmd) closeWithCombined() {
	c.combinedOutput <- struct{}{}
}

func (c *Cmd) fetchStdout(w io.WriteCloser) {
	if c.combinedOutput != nil {
		defer c.closeWithCombined()
	} else {
		defer w.Close()
	}

	c.wg.Add(1)
	defer c.wg.Done()
	stream, err := c.client.FetchStdout(context.Background(), c.sn)
	if err != nil {
		close(c.stdoutCh)
		c.streamStdout = errors.Wrap(err, "grpc fetch stdout")
		return
	}

	data, err := stream.Recv()
	close(c.stdoutCh)
	if err != nil {
		c.streamStdout = errors.Wrap(err, "stream stdout")
		return
	}
	if !data.Start {
		c.streamStdout = errors.Wrap(err, "stream stdout not start")
		return
	}

	for {
		data, err := stream.Recv()
		if err == io.EOF {
			close(c.stdoutCh)
			return
		} else if err != nil {
			close(c.stdoutCh)
			c.streamStdout = errors.Wrap(err, "grpc stdout recv")
			return
		}
		if data.Closed {
			return
		} else if len(data.RuntimeError) > 0 {
			c.streamStdout = errors.New(string(data.RuntimeError))
			return
		} else {
			err := writeTo(data.Stdout, w)
			if err != nil {
				c.streamStdout = errors.Wrap(err, "write to stdout")
				return
			}
		}
	}
}

func (c *Cmd) fetchStderr(w io.WriteCloser) {
	if c.combinedOutput != nil {
		defer c.closeWithCombined()
	} else {
		defer w.Close()
	}

	c.wg.Add(1)
	defer c.wg.Done()
	stream, err := c.client.FetchStderr(context.Background(), c.sn)
	if err != nil {
		close(c.stderrCh)
		c.streamStderr = errors.Wrap(err, "grpc fetch stderr")
		return
	}

	data, err := stream.Recv()
	close(c.stderrCh)
	if err != nil {
		c.streamStderr = errors.Wrap(err, "stream stderr")
		return
	}
	if !data.Start {
		c.streamStderr = errors.Wrap(err, "stream stderr not start")
		return
	}

	for {
		data, err := stream.Recv()
		if err == io.EOF {
			return
		} else if err != nil {
			c.streamStderr = errors.Wrap(err, "grpc stderr recv")
			return
		}
		if data.Closed {
			return
		} else if len(data.RuntimeError) > 0 {
			c.streamStderr = errors.New(string(data.RuntimeError))
			return
		} else {
			err := writeTo(data.Stderr, w)
			if err != nil {
				c.streamStderr = errors.Wrap(err, "write to stderr")
				return
			}
		}
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

func (e *ExitError) Sys() interface{} {
	return e.ExitStatus
}

func (e *ExitError) Error() string {
	return exitStatusToString(e.ExitStatus)
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

func writeTo(data []byte, w io.Writer) error {
	var n = 0
	var length = len(data)
	for n < length {
		r, e := w.Write(data[n:])
		if e != nil {
			return e
		}
		n += r
	}
	return nil
}

type CloseOnce struct {
	*os.File

	once sync.Once
	err  error
}

func (c *CloseOnce) Close() error {
	c.once.Do(c.close)
	return c.err
}

func (c *CloseOnce) close() {
	c.err = c.File.Close()
}
