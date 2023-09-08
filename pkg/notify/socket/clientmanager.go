package socket

import (
	"context"
	"fmt"

	socketio "github.com/googollee/go-socket.io"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type Client struct {
	Session string
	Name    string
	UserID  string
	Conn    socketio.Conn
}

type ClientManager struct {
	UserID2ClientMap        map[string][]Client
	SID2ClientMap           map[string]Client
	Server                  *socketio.Server
	ForceSingleSessionLogin bool
}

const (
	// BroadcastRoom  所有的在线 client 自动加入聊天室
	BroadcastRoom = "YunionBcast"
)

// IsEmpty 当前是否无人在线（浏览器）
func (manager *ClientManager) IsEmpty() bool {
	log.Debugf("[Clients] there are %d clients belonging to %d tenants.", len(manager.SID2ClientMap), len(manager.UserID2ClientMap))
	return len(manager.SID2ClientMap)+len(manager.UserID2ClientMap) == 0
}

// Register 浏览器user 登录时，注册 socketio 链接
func (manager *ClientManager) Register(ctx context.Context, s socketio.Conn) error {
	session, cred, err := getCred(ctx, s)
	if err != nil {
		log.Errorf("login FAILED: with id %s error: %v", s.ID(), err)
		return errors.Wrapf(err, "getCred")
	}
	log.Debugf("Login PASS ! ID: %s , user: %s(%s)", s.ID(), cred.GetUserName(), cred.GetUserId())
	UserID := cred.GetUserId()
	client := Client{
		Session: session,
		Name:    cred.GetUserName(),
		UserID:  UserID,
		Conn:    s,
	}

	manager.UserID2ClientMap[UserID] = append(manager.UserID2ClientMap[UserID], client)
	manager.SID2ClientMap[s.ID()] = client
	log.Debugf("registered successful for user %s(%s) with id %s", cred.GetUserName(), cred.GetUserId(), s.ID())
	s.Join(BroadcastRoom)
	s.SetContext("")
	return nil
}

// Gretting 组合一个显示用户 socketio 链接信息的子串
func (manager *ClientManager) Gretting(s socketio.Conn) string {
	client := manager.SID2ClientMap[s.ID()]
	return fmt.Sprintf("hello %s(%s), your socket io id: %s, your session: %s.", client.Name, client.UserID, s.ID(), client.Session)
}

// Unregister 浏览器 user 刷新或关闭页面，断开链接（自动重连）
func (manager *ClientManager) Unregister(s socketio.Conn, reason string) {
	delete(manager.SID2ClientMap, s.ID())
	log.Debugf("[ Unregister ] ID: %s; reason: %s", s.ID(), reason)
	s.Emit("bye", "")
	s.Leave(BroadcastRoom)
	s.Close()
}

// NotifyByUserID 按照用户 id，通知到用户的所有在线浏览器页面。支持用户多 session 登录，例如 sysadmin
func (manager *ClientManager) NotifyByUserID(message string, UserID string) error {
	count := 0
	name := ""
	msg := ""

	for _, c := range manager.UserID2ClientMap[UserID] {
		if c.UserID == UserID {
			c.Conn.Emit("message", string(message))
			name = manager.SID2ClientMap[c.Conn.ID()].Name
			count++
		}
	}

	if count == 0 {
		log.Warningf("UserID %s is not online!", UserID)
		return errors.Errorf(msg)
	}
	log.Debugf("[%d clients] NOTIFY OK to %s(@%s)  ", count, UserID, name)
	return nil
}

//Broadcasts 对所有在线用户发广播
func (manager *ClientManager) Broadcasts(message string) {
	// BroadcastToRoom(namespace string, room, event string, args ...interface{}) bool {
	if manager.IsEmpty() {
		log.Debugf("Ignore Broadcasting for empty room\n")
		return
	}
	if !manager.Server.BroadcastToRoom("", BroadcastRoom, "message", message) {
		log.Errorf("broadcasting error with msg %s\n", message)
		return
	}
	log.Infof("broadcasting OK with msg %s\n", message)
}
