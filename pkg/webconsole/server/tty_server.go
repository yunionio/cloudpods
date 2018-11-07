package server

import (
	"os/exec"
	"strconv"
	"strings"
	"time"

	socketio "github.com/googollee/go-socket.io"
	"github.com/kr/pty"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/webconsole/session"
)

const (
	ON_CONNECTION    = "connection"
	ON_DISCONNECTION = "disconnection"
	ON_ERROR         = "error"

	OUTPUT_EVENT = "output"
	INPUT_EVENT  = "input"
	RESIZE_EVENT = "resize"

	COMMAND_QUERY = "command"
	ARGS_QUERY    = "args"
)

type TTYServer struct {
	*socketio.Server
}

func NewTTYServer(s *session.SSession) (*TTYServer, error) {
	socketioServer, err := socketio.NewServer(nil)
	if err != nil {
		return nil, err
	}
	server := &TTYServer{
		Server: socketioServer,
	}
	server.initEventHandler(s)
	return server, nil
}

func (server *TTYServer) initEventHandler(s *session.SSession) {
	server.On(ON_CONNECTION, func(so socketio.Socket) error {
		log.Infof("[%q] On connection", so.Id())
		p, err := session.NewPty(s)
		if err != nil {
			log.Errorf("Create Pty error: %v", err)
			return err
		}
		initSocketHandler(so, p)
		return nil
	})
}

func initSocketHandler(so socketio.Socket, p *session.Pty) {
	// handle read
	go func() {
		buf := make([]byte, 1024)
		for {
			if p.IsOk {
				if p.Cmd == nil || p.Cmd.Process == nil {
					p.IsOk = false
				} else if p.Pty == nil {
					p.IsOk = false
				} else if n, err := p.Pty.Read(buf); err != nil {
					p.IsOk = false
				} else {
					so.Emit(OUTPUT_EVENT, string(buf[0:n]))
				}
				if !p.IsOk {
					p.Stop()
					if info := p.Session.ShowInfo(); len(info) > 0 {
						so.Emit(OUTPUT_EVENT, info)
					}
				}
			} else if p.Exit {
				return
			} else {
				//避免goroutine死循环导致主进程卡死
				time.Sleep(time.Microsecond * 50)
			}
		}
	}()

	// handle write
	so.On(INPUT_EVENT, func(data string) {
		if !p.IsOk {
			if data == "\r" {
				p.Show, p.Output, p.Command = p.Session.GetData(p.Buffer)
				so.Emit(OUTPUT_EVENT, "\r\n")
				if len(p.Output) > 0 {
					so.Emit(OUTPUT_EVENT, p.Output)
				}
				if len(p.Command) > 0 {
					log.Infof("exec: %s", p.Command)
					args := strings.Split(p.Command, " ")
					cmd := exec.Command(args[0], args[1:]...)
					cmd.Env = append(cmd.Env, "TERM=screen-256color")
					if _pty, err := pty.Start(cmd); err != nil {
						so.Emit(OUTPUT_EVENT, err.Error()+"\r\n")
						log.Errorf("exec error: %v", err)
					} else {
						p.Pty, p.Cmd, p.IsOk = _pty, cmd, true
						if p.OriginSize != nil {
							p.Resize(p.OriginSize)
						}
					}
				}
				p.Buffer, data = "", ""
			} else if data == "\u007f" {
				//退格处理
				if len(p.Buffer) > 0 {
					p.Buffer = p.Buffer[:len(p.Buffer)-1]
					data = "\b \b"
				}
			} else if strconv.IsPrint([]rune(data)[0]) {
				p.Buffer += data
			}
			if p.Show && len(data) > 0 {
				so.Emit(OUTPUT_EVENT, data)
			}
		} else {
			p.Pty.Write([]byte(data))
		}
	})

	// handle resize
	so.On(RESIZE_EVENT, func(colRow []uint16) {
		if len(colRow) != 2 {
			log.Errorf("Invalid window size: %v", colRow)
			cleanUp(so, p)
			return
		}
		//size, err := pty.GetsizeFull(p.Pty)
		//if err != nil {
		//log.Errorf("Get pty window size error: %v", err)
		//return
		//}
		newSize := pty.Winsize{
			Cols: colRow[0],
			Rows: colRow[1],
		}
		p.Resize(&newSize)
	})

	// handle disconnection
	so.On(ON_DISCONNECTION, func(msg string) {
		log.Infof("[%s] closed: %s", so.Id(), msg)
		cleanUp(so, p)
	})

	// handle error
	so.On(ON_ERROR, func(err error) {
		log.Errorf("[%s] on error: %v", so.Id(), err)
		cleanUp(so, p)
	})
}

func cleanUp(so socketio.Socket, p *session.Pty) {
	so.Disconnect()
	p.Stop()
	p.Exit = true
}
