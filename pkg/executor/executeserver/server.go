package executeserver

import (
	"flag"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	"yunion.io/x/log"
	"yunion.io/x/pkg/util/signalutils"
	"yunion.io/x/pkg/utils"

	execapi "yunion.io/x/onecloud/pkg/executor/apis"
)

var socketPath string

type Commander struct {
	stream execapi.Executor_ExecCommandServer
	cmd    *execapi.Command

	wg *sync.WaitGroup

	waitDone    chan struct{}
	killProcess chan struct{}
	streamErr   chan error
	runErr      chan error
}

func NewCommander(stream execapi.Executor_ExecCommandServer, cmd *execapi.Command) *Commander {
	return &Commander{
		stream:      stream,
		cmd:         cmd,
		wg:          new(sync.WaitGroup),
		waitDone:    make(chan struct{}, 1),
		killProcess: make(chan struct{}, 1),
		streamErr:   make(chan error, 3),
		runErr:      make(chan error, 1),
	}
}

// watch process already started

type Executor struct{}

func (e *Executor) ExecCommand(stream execapi.Executor_ExecCommandServer) error {
	cmd, err := stream.Recv()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	return NewCommander(stream, cmd).Start()
}

func bytesArrayToStrArray(ba [][]byte) []string {
	if len(ba) == 0 {
		return nil
	}
	res := make([]string, len(ba))
	for i := 0; i < len(ba); i++ {
		res[i] = string(ba[i])
	}
	return res
}

func (m *Commander) Start() error {
	log.Infof("Recv Command -> %s", m.cmd)
	c := exec.Command(string(m.cmd.Path), bytesArrayToStrArray(m.cmd.Args)...)
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if len(m.cmd.Env) > 0 {
		c.Env = bytesArrayToStrArray(m.cmd.Env)
	}
	if len(m.cmd.Dir) > 0 {
		c.Dir = string(m.cmd.Dir)
	}

	stdin, err := c.StdinPipe()
	if err != nil {
		return errors.Wrap(err, "stdin pipe")
	}
	defer stdin.Close()

	stdout, err := c.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "stdout pipe")
	}
	defer stdout.Close()

	stderr, err := c.StderrPipe()
	if err != nil {
		return errors.Wrap(err, "stderr pipe")
	}
	defer stderr.Close()

	if err := c.Start(); err != nil {
		m.runErr <- err
	} else {
		m.streamStdin(stdin)
		m.streamStdout(stdout)
		m.streamStderr(stderr)
		m.waitProcess(c)
	}

	var (
		exitCode   int32
		errContent string
	)
	select {
	case err := <-m.streamErr:
		return errors.Wrap(err, "stream data")
	case err := <-m.runErr:
		if err != nil {
			if exiterr, ok := err.(*exec.ExitError); ok {
				// The program has exited with an exit code != 0
				// This works on both Unix and Windows. Although package
				// syscall is generally platform dependent, WaitStatus is
				// defined for both Unix and Windows and in both cases has
				// an ExitStatus() method with the same signature.
				status := exiterr.Sys().(syscall.WaitStatus)
				exitCode = int32(status.ExitStatus())
			} else {
				// command not found or io problem
				errContent = err.Error()
			}
		} else {
			exitCode = 0
		}
	}

	m.wg.Wait()
	return m.stream.Send(&execapi.Response{
		ExitCode:   exitCode,
		IsExit:     true,
		ErrContent: []byte(errContent),
	})
}

func (m *Commander) waitProcess(c *exec.Cmd) {
	go func() {
		select {
		case <-m.killProcess:
			c.Process.Kill()
		case <-m.waitDone:
		}
	}()
	go func() {
		if err := c.Wait(); err != nil {
			m.runErr <- err
		} else {
			m.runErr <- nil
		}
		close(m.waitDone)
	}()
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

func (m *Commander) streamStdin(w io.Writer) {
	go func() {
		for {
			c, e := m.stream.Recv()
			if e == io.EOF {
				break
			}
			if e != nil {
				m.streamErr <- e
				break
			}
			if len(c.Input) > 0 {
				e = writeTo(c.Input, w)
				if e != nil {
					m.streamErr <- errors.Wrap(e, "write to process")
					break
				}
			} else if c.KillProcess {
				close(m.killProcess)
			} else {
				log.Warningf("stream stdin receive cmd not input: %s", c)
			}
		}
	}()
}

func (m *Commander) streamStdout(r io.Reader) {
	m.streamData(r, "stdout")
}

func (m *Commander) streamStderr(r io.Reader) {
	m.streamData(r, "stderr")
}

func (m *Commander) streamData(r io.Reader, kind string) {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		var data = make([]byte, 4069)
		for {
			n, err := r.Read(data)
			if err == io.EOF {
				break
			} else if err != nil {
				// os/exec.Command.Wait will close pipe
				if pe, ok := err.(*os.PathError); ok && pe.Err == os.ErrClosed {
					break
				} else {
					m.streamErr <- errors.Wrap(err, "read stream data")
					break
				}
			}
			err = m.stream.Send(&execapi.Response{Stdout: data[:n]})
			if err != nil {
				m.streamErr <- err
				break
			}
		}
	}()
}

type SExecuteService struct {
}

func NewExecuteService() *SExecuteService {
	return &SExecuteService{}
}

func (s *SExecuteService) fixPathEnv() error {
	var paths = []string{
		"/usr/local/sbin",
		"/usr/local/bin",
		"/sbin",
		"/bin",
		"/usr/sbin",
		"/usr/bin",
	}
	return os.Setenv("PATH", strings.Join(paths, ":"))
}

func (s *SExecuteService) prepareEnv() error {
	if err := s.fixPathEnv(); err != nil {
		return err
	}
	return nil
}

func (s *SExecuteService) runService() {
	grpcServer := grpc.NewServer()
	execapi.RegisterExecutorServer(grpcServer, &Executor{})
	if _, err := os.Stat(socketPath); !os.IsNotExist(err) {
		// socket file already exist, remove first
		if err := os.Remove(socketPath); err != nil {
			log.Fatalln(err)
		}
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer listener.Close()
	log.Infof("Init net listener on %s succ", socketPath)
	grpcServer.Serve(listener)
}

func (s *SExecuteService) initService() {
	if len(socketPath) == 0 {
		log.Fatalf("missing socket path")
	}
	if err := s.prepareEnv(); err != nil {
		log.Fatalln(err)
	}

	signalutils.RegisterSignal(func() {
		utils.DumpAllGoroutineStack(log.Logger().Out)
	}, syscall.SIGUSR1)
	signalutils.StartTrap()
}

func (s *SExecuteService) Run() {
	s.initService()
	s.runService()
}

func init() {
	flag.StringVar(&socketPath, "socket-path", "", "execute service listen socket path")
	flag.Parse()
}
