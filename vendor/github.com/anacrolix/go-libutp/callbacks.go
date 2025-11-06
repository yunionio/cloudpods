package utp

/*
#include "utp.h"
*/
import "C"

import (
	"net"
	"reflect"
	"strings"
	"sync/atomic"
	"unsafe"

	"github.com/anacrolix/log"
)

type utpCallbackArguments C.utp_callback_arguments

func (a *utpCallbackArguments) goContext() *utpContext {
	return (*utpContext)(a.context)
}

func (a *utpCallbackArguments) bufBytes() []byte {
	return *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		uintptr(unsafe.Pointer(a.buf)),
		int(a.len),
		int(a.len),
	}))
}

func (a *utpCallbackArguments) state() C.int {
	return *(*C.int)(unsafe.Pointer(&a.anon0))
}

func (a *utpCallbackArguments) error_code() C.int {
	return *(*C.int)(unsafe.Pointer(&a.anon0))
}

func (a *utpCallbackArguments) address() *C.struct_sockaddr {
	return *(**C.struct_sockaddr)(unsafe.Pointer(&a.anon0[0]))
}

func (a *utpCallbackArguments) addressLen() C.socklen_t {
	return *(*C.socklen_t)(unsafe.Pointer(&a.anon1[0]))
}

var sends int64

//export sendtoCallback
func sendtoCallback(a *utpCallbackArguments) (ret C.uint64) {
	s := getSocketForLibContext(a.goContext())
	b := a.bufBytes()
	var sendToUdpAddr net.UDPAddr
	if err := structSockaddrToUDPAddr(a.address(), &sendToUdpAddr); err != nil {
		panic(err)
	}
	newSends := atomic.AddInt64(&sends, 1)
	if logCallbacks {
		s.logger.Printf("sending %d bytes, %d packets", len(b), newSends)
	}
	expMap.Add("socket PacketConn writes", 1)
	n, err := s.pc.WriteTo(b, &sendToUdpAddr)
	c := s.conns[a.socket]
	if err != nil {
		expMap.Add("socket PacketConn write errors", 1)
		if c != nil && c.userOnError != nil {
			go c.userOnError(err)
		} else if c != nil &&
			(strings.Contains(err.Error(), "can't assign requested address") ||
				strings.Contains(err.Error(), "invalid argument")) {
			// Should be an bad argument or network configuration problem we
			// can't recover from.
			c.onError(err)
		} else if c != nil && strings.Contains(err.Error(), "operation not permitted") {
			// Rate-limited. Probably Linux. The implementation might try
			// again later.
		} else {
			s.logger.Levelf(log.Debug, "error sending packet: %v", err)
		}
		return
	}
	if n != len(b) {
		expMap.Add("socket PacketConn short writes", 1)
		s.logger.Printf("expected to send %d bytes but only sent %d", len(b), n)
	}
	return
}

//export errorCallback
func errorCallback(a *utpCallbackArguments) C.uint64 {
	s := getSocketForLibContext(a.goContext())
	err := errorForCode(a.error_code())
	if logCallbacks {
		s.logger.Printf("error callback: socket %p: %s", a.socket, err)
	}
	libContextToSocket[a.goContext()].conns[a.socket].onError(err)
	return 0
}

//export logCallback
func logCallback(a *utpCallbackArguments) C.uint64 {
	s := getSocketForLibContext(a.goContext())
	s.logger.Printf("libutp: %s", C.GoString((*C.char)(unsafe.Pointer(a.buf))))
	return 0
}

//export stateChangeCallback
func stateChangeCallback(a *utpCallbackArguments) C.uint64 {
	s := libContextToSocket[a.goContext()]
	c := s.conns[a.socket]
	if logCallbacks {
		s.logger.Printf("state changed: conn %p: %s", c, libStateName(a.state()))
	}
	switch a.state() {
	case C.UTP_STATE_CONNECT:
		c.setConnected()
		// A dialled connection will not tell the remote it's ready until it
		// writes. If the dialer has no intention of writing, this will stall
		// everything. We do an empty write to get things rolling again. This
		// circumstance occurs when c1 in the RacyRead nettest is the dialer.
		C.utp_write(a.socket, nil, 0)
	case C.UTP_STATE_WRITABLE:
		c.cond.Broadcast()
	case C.UTP_STATE_EOF:
		c.setGotEOF()
	case C.UTP_STATE_DESTROYING:
		c.onDestroyed()
		s.onLibSocketDestroyed(a.socket)
	default:
		panic(a.state)
	}
	return 0
}

//export readCallback
func readCallback(a *utpCallbackArguments) C.uint64 {
	s := libContextToSocket[a.goContext()]
	c := s.conns[a.socket]
	b := a.bufBytes()
	if logCallbacks {
		s.logger.Printf("read callback: conn %p: %d bytes", c, len(b))
	}
	if len(b) == 0 {
		panic("that will break the read drain invariant")
	}
	c.readBuf.Write(b)
	c.cond.Broadcast()
	return 0
}

//export acceptCallback
func acceptCallback(a *utpCallbackArguments) C.uint64 {
	s := getSocketForLibContext(a.goContext())
	if logCallbacks {
		s.logger.Printf("accept callback: %#v", *a)
	}
	c := s.newConn(a.socket)
	c.setRemoteAddr()
	s.pushBacklog(c)
	return 0
}

//export getReadBufferSizeCallback
func getReadBufferSizeCallback(a *utpCallbackArguments) (ret C.uint64) {
	s := libContextToSocket[a.goContext()]
	c := s.conns[a.socket]
	if c == nil {
		// socket hasn't been added to the Socket.conns yet. The read buffer
		// starts out empty, and the default implementation for this callback
		// returns 0, so we'll return that.
		return 0
	}
	ret = C.uint64(c.readBuf.Len())
	return
}

//export firewallCallback
func firewallCallback(a *utpCallbackArguments) C.uint64 {
	s := getSocketForLibContext(a.goContext())
	if s.syncFirewallCallback != nil {
		var addr net.UDPAddr
		structSockaddrToUDPAddr(a.address(), &addr)
		if s.syncFirewallCallback(&addr) {
			return 1
		}
	} else if s.asyncBlock {
		return 1
	}
	return 0
}
