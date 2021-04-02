// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ssh

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	ssh_util "yunion.io/x/onecloud/pkg/util/ssh"
)

type addrMap map[string]interface{}
type portMap map[int]addrMap

func (pm portMap) contains(port int, addr string) bool {
	am, ok := pm[port]
	if !ok {
		return false
	}
	return am.contains(addr)
}

func (pm portMap) get(port int, addr string) interface{} {
	am, ok := pm[port]
	if ok {
		return am.get(addr)
	}
	return nil
}

func (pm portMap) set(port int, addr string, v interface{}) {
	am, ok := pm[port]
	if !ok {
		am = addrMap{}
		pm[port] = am
	}
	am.set(addr, v)
}

func (pm portMap) delete(port int, addr string) {
	if am, ok := pm[port]; ok {
		am.delete(addr)
	}
}

func (am addrMap) contains(addr string) bool {
	const (
		ip4wild = "0.0.0.0"
		ip6wild = "::"
	)
	_, ok := am[addr]
	if ok {
		return true
	}
	if _, ok := am[ip4wild]; ok {
		return true
	}
	if _, ok := am[ip6wild]; ok {
		return true
	}
	return false
}

func (am addrMap) get(addr string) interface{} {
	return am[addr]
}

func (am addrMap) set(addr string, v interface{}) {
	am[addr] = v
}

func (am addrMap) delete(addr string) {
	delete(am, addr)
}

type Client struct {
	cc *ssh_util.ClientConfig
	c  *ssh.Client

	stopc   chan sets.Empty
	stopcEx *sync.Mutex
	stopcc  bool

	wakec chan sets.Empty
	lfc   chan LocalForwardReq
	rfc   chan RemoteForwardReq

	lfclosec chan LocalForwardReq
	rfclosec chan RemoteForwardReq

	localForwards  portMap
	remoteForwards portMap
}

func NewClient(cc *ssh_util.ClientConfig) *Client {
	c := &Client{
		cc: cc,

		stopc:   make(chan sets.Empty),
		stopcEx: &sync.Mutex{},

		wakec: make(chan sets.Empty),
		lfc:   make(chan LocalForwardReq),
		rfc:   make(chan RemoteForwardReq),

		lfclosec: make(chan LocalForwardReq),
		rfclosec: make(chan RemoteForwardReq),

		localForwards:  portMap{},
		remoteForwards: portMap{},
	}
	return c
}

func (c *Client) Stop(ctx context.Context) {
	c.stopcEx.Lock()
	defer c.stopcEx.Unlock()
	if !c.stopcc {
		close(c.stopc)
		c.stopcc = true
	}
}

func (c *Client) Start(ctx context.Context) {
	pingT := time.NewTimer(17 * time.Second)
	pingFailCount := 0
	const pingMaxFail = 3

	const (
		stateInit = iota
		stateOK
	)
	state := stateInit
	stateC := make(chan int)
	stateInitRetryInterval := 7 * time.Second
	stateInitRetryT := time.NewTicker(stateInitRetryInterval)
	for {
		switch state {
		case stateOK:
			// check forwards and start
		case stateInit:
			if sshc, err := c.connect(ctx); err != nil {
				log.Errorf("ssh connect: %v", err)
			} else {
				c.c = sshc
				state = stateOK
				go func() {
					defer c.c.Conn.Close()

					err := c.c.Conn.Wait()
					if err != nil {
						log.Errorf("ssh client conn: %v", err)
					}
					select {
					case stateC <- stateInit:
					case <-ctx.Done():
					}
				}()
			}
		}

		select {
		case req := <-c.lfc:
			if c.c != nil {
				c.localForward(ctx, req)
			}
		case req := <-c.rfc:
			if c.c != nil {
				c.remoteForward(ctx, req)
			}
		case req := <-c.lfclosec:
			c.localForwardClose(ctx, req)
		case req := <-c.rfclosec:
			c.remoteForwardClose(ctx, req)
		case <-c.wakec:
			break
		case <-pingT.C:
			//TODO ping check
			//ping fail
			if pingFailCount > pingMaxFail {
				state = stateInit
			}
		case newState := <-stateC:
			state = newState
		case <-stateInitRetryT.C:
		case <-c.stopc:
			if c.c != nil {
				c.c.Conn.Close()
			}
			return
		case <-ctx.Done():
			return
		}
	}
}

func (c *Client) connect(ctx context.Context) (*ssh.Client, error) {
	sshc, err := c.cc.ConnectContext(ctx)
	return sshc, err
}

func (c *Client) LocalForward(ctx context.Context, req LocalForwardReq) {
	select {
	case c.lfc <- req:
	case <-ctx.Done():
	}
}

func (c *Client) localForward(ctx context.Context, req LocalForwardReq) {
	if err := c.localForward_(ctx, req); err != nil {
		log.Errorf("local forward: %v", err)
	}
}

func (c *Client) localForward_(ctx context.Context, req LocalForwardReq) error {
	// check LocalAddr/LocalPort existence
	if c.localForwards.contains(req.LocalPort, req.LocalAddr) {
		return errors.Errorf("local addr occupied: %s:%d", req.LocalAddr, req.LocalPort)
	}

	addr := net.JoinHostPort(req.LocalAddr, fmt.Sprintf("%d", req.LocalPort))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return errors.Wrapf(err, "tcp listen %s", addr)
	}
	fwd := &forwarder{
		listener: listener,

		dial:     c.c.Dial,
		dialAddr: req.RemoteAddr,
		dialPort: req.RemotePort,

		done:     c.localForwardDone,
		doneAddr: req.LocalAddr,
		donePort: req.LocalPort,

		tick:   req.Tick,
		tickCb: req.TickCb,
	}

	c.localForwards.set(req.LocalPort, req.LocalAddr, fwd)
	go fwd.Start(ctx)
	return nil
}

func (c *Client) localForwardDone(laddr string, lport int) {
	c.localForwards.delete(lport, laddr)
}

func (c *Client) RemoteForward(ctx context.Context, req RemoteForwardReq) {
	select {
	case c.rfc <- req:
	case <-ctx.Done():
	}
}

func (c *Client) remoteForward(ctx context.Context, req RemoteForwardReq) {
	if err := c.remoteForward_(ctx, req); err != nil {
		log.Errorf("remote forward: %v", err)
	}
}

func (c *Client) remoteForward_(ctx context.Context, req RemoteForwardReq) error {
	// check RemoteAddr/RemotePort existence
	if c.remoteForwards.contains(req.RemotePort, req.RemoteAddr) {
		return errors.Errorf("remote addr occupied: %s:%d", req.RemoteAddr, req.RemotePort)
	}

	addr := net.JoinHostPort(req.RemoteAddr, fmt.Sprintf("%d", req.RemotePort))
	listener, err := c.c.Listen("tcp", addr)
	if err != nil {
		return errors.Wrapf(err, "ssh listen %s", addr)
	}

	fwd := &forwarder{
		listener: listener,

		dial:     net.Dial,
		dialAddr: req.LocalAddr,
		dialPort: req.LocalPort,

		done:     c.remoteForwardDone,
		doneAddr: req.RemoteAddr,
		donePort: req.RemotePort,

		tick:   req.Tick,
		tickCb: req.TickCb,
	}
	c.remoteForwards.set(req.RemotePort, req.RemoteAddr, fwd)
	go fwd.Start(ctx)
	return nil
}

func (c *Client) remoteForwardDone(raddr string, rport int) {
	c.remoteForwards.delete(rport, raddr)
}

func (c *Client) LocalForwardClose(ctx context.Context, req LocalForwardReq) {
	select {
	case c.lfclosec <- req:
	case <-ctx.Done():
	}
}

func (c *Client) localForwardClose(ctx context.Context, req LocalForwardReq) {
	v := c.localForwards.get(req.LocalPort, req.LocalAddr)
	if v != nil {
		fwd := v.(*forwarder)
		fwd.Stop(ctx)
	}
}

func (c *Client) RemoteForwardClose(ctx context.Context, req RemoteForwardReq) {
	select {
	case c.rfclosec <- req:
	case <-ctx.Done():
	}
}

func (c *Client) remoteForwardClose(ctx context.Context, req RemoteForwardReq) {
	v := c.remoteForwards.get(req.RemotePort, req.RemoteAddr)
	if v != nil {
		fwd := v.(*forwarder)
		fwd.Stop(ctx)
	}
}
