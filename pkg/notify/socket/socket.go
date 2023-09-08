package socket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	socketio "github.com/googollee/go-socket.io"
	engineio "github.com/googollee/go-socket.io/engineio"
	"github.com/googollee/go-socket.io/engineio/transport"
	"github.com/googollee/go-socket.io/engineio/transport/polling"
	"github.com/googollee/go-socket.io/engineio/transport/websocket"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

type Message struct {
	Sender    string `json:"sender,omitempty"`
	Recipient string `json:"recipient,omitempty"`
	Content   string `json:"content,omitempty"`
}

var (
	SocketServer     *socketio.Server
	DefaultSocketMan = ClientManager{
		UserID2ClientMap:        make(map[string][]Client),
		SID2ClientMap:           make(map[string]Client),
		Server:                  nil,
		ForceSingleSessionLogin: false,
	}
)

func init() {
	SocketServer = NewSocketServer()
	if SocketServer == nil {
		log.Fatalln("got nil socketio server !")
		os.Exit(1)
	}
	DefaultSocketMan.Server = SocketServer
}

func NewSocketServer() *socketio.Server {

	pt := polling.Default

	wt := websocket.Default
	wt.CheckOrigin = func(req *http.Request) bool {
		return true
	}
	server := socketio.NewServer(&engineio.Options{
		Transports: []transport.Transport{
			pt,
			wt,
		},
	})

	server.OnConnect("/", func(s socketio.Conn) error {
		ctx := context.WithValue(context.Background(), "", "")
		err := DefaultSocketMan.Register(ctx, s)
		if err != nil {
			return errors.Wrapf(err, "Register")
		}
		s.SetContext("")
		return nil
	})

	server.OnEvent("/", "bye", func(s socketio.Conn) string {
		last := s.Context().(string)
		DefaultSocketMan.Unregister(s, "bye")
		return last
	})

	server.OnError("/", func(s socketio.Conn, err error) {
		log.Errorf("meet error: %v", err)
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		DefaultSocketMan.Unregister(s, reason)
	})

	return server
}

// SocketHandler 将 http 的上下文传递给  socketio
func SocketHandler(ctx context.Context, w http.ResponseWriter, req *http.Request) {
	req = req.Clone(ctx)
	SocketServer.ServeHTTP(w, req)
}

func SocketMiddleware(f func(context.Context, http.ResponseWriter, *http.Request)) func(context.Context, http.ResponseWriter, *http.Request) {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		f(ctx, w, r)
	}
}

/*
新建websocket handle。默认1周超时，100个worker.
*/
func NewSocketIOHandler(method, prefix string, handler func(context.Context, http.ResponseWriter, *http.Request)) *appsrv.SHandlerInfo {
	log.Debugf("%s - %s", method, prefix)
	hi := appsrv.SHandlerInfo{}
	hi.SetMethod(method)
	hi.SetPath(prefix)
	hi.SetHandler(handler)
	hi.SetProcessTimeout(12 * time.Hour) // session timeout
	hi.SetWorkerManager(GetSocketWorker())
	return &hi
}

// AddSocketHandlers  路径分发
func AddSocketHandlers(prefix string, app *appsrv.Application) {
	getSocketIO := NewSocketIOHandler("GET", fmt.Sprintf("%s/socket.io/", prefix), SocketMiddleware(SocketHandler))
	postSocketIO := NewSocketIOHandler("POST", fmt.Sprintf("%s/socket.io/", prefix), SocketMiddleware(SocketHandler))
	postWebSocket := NewSocketIOHandler("POST", fmt.Sprintf("%s/websockets", prefix), auth.Authenticate(notification))
	app.AddHandler3(getSocketIO)
	app.AddHandler3(postSocketIO)
	app.AddHandler3(postWebSocket)
}

// notification 外部(http)通知接口，用于向单个用户或全部用户发消息
func notification(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var body SMsg

	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		log.Fatalf("[notification] decode body [%v] error: %s", r.Body, err)
		return
	}
	message, err := json.Marshal(body.WebSocket)
	if err != nil {
		log.Fatalf("Marshal body error: %s", err)
		return
	}

	if body.WebSocket.Broadcast {
		DefaultSocketMan.Broadcasts(string(message))
		log.Infof("message: %+v", message)
	} else {
		DefaultSocketMan.NotifyByUserID(string(message), body.WebSocket.UserId)
		log.Errorf("message: %+v", message)
	}
}
