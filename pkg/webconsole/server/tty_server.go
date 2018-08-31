package server

import (
	"os/exec"
	"strings"

	socketio "github.com/googollee/go-socket.io"
	"github.com/kr/pty"

	"yunion.io/x/jsonutils"
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

func fetchCommand(so socketio.Socket) (*exec.Cmd, error) {
	req := so.Request()
	query, err := jsonutils.ParseQueryString(req.URL.RawQuery)
	if err != nil {
		return nil, err
	}
	cmd, err := query.GetString(COMMAND_QUERY)
	if err != nil {
		return nil, err
	}
	args := []string{}
	argsStr, _ := query.GetString(ARGS_QUERY)
	if len(argsStr) != 0 {
		args = strings.Split(argsStr, ",")
	}
	return exec.Command(cmd, args...), nil
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
			n, err := p.Pty.Read(buf)
			if err != nil {
				log.Errorf("Failed to read from pty master: %v", err)
				cleanUp(so, p)
				return
			}
			so.Emit(OUTPUT_EVENT, string(buf[0:n]))
		}
	}()

	// handle write
	so.On(INPUT_EVENT, func(data string) {
		p.Pty.Write([]byte(data))
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
}
