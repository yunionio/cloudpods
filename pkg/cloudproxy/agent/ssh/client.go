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

	stopc   chan sets.Empty
	stopcEx *sync.Mutex
	stopcc  bool

	lfc chan LocalForwardReq
	rfc chan RemoteForwardReq

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

		lfc: make(chan LocalForwardReq),
		rfc: make(chan RemoteForwardReq),

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
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	pingT := time.NewTimer(17 * time.Second)
	pingFailCount := 0
	const pingMaxFail = 3

	sshClientC := make(chan *ssh.Client)
	var sshClient *ssh.Client
	go c.runClientState(ctx, sshClientC)
	for {
		select {
		case sshClient = <-sshClientC:
		case req := <-c.lfc:
			if sshClient != nil {
				c.localForward(ctx, sshClient, req)
			}
		case req := <-c.rfc:
			if sshClient != nil {
				c.remoteForward(ctx, sshClient, req)
			}
		case req := <-c.lfclosec:
			c.localForwardClose(ctx, req)
		case req := <-c.rfclosec:
			c.remoteForwardClose(ctx, req)
		case <-pingT.C:
			//TODO ping check
			//ping fail
			if pingFailCount > pingMaxFail {
			}
		case <-c.stopc:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (c *Client) runClientState(ctx context.Context, sshClientC chan<- *ssh.Client) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		cc := c.cc
		tmoCtx, _ := context.WithTimeout(ctx, 31*time.Second)
		sshc, err := cc.ConnectContext(tmoCtx)
		if err != nil {
			log.Errorf("ssh connect: %s@%s, port %d: %v", cc.Username, cc.Host, cc.Port, err)
			waitTmo := time.NewTimer(13 * time.Second)
			select {
			case <-ctx.Done():
				return
			case <-waitTmo.C:
			}
			continue
		}

		func() {
			defer sshc.Conn.Close()

			closeC := make(chan struct{})
			go func() {
				defer close(closeC)

				err := sshc.Conn.Wait()
				if err != nil {
					log.Infof("ssh client conn: %v", err)
				}
			}()

			select {
			case sshClientC <- sshc:
			case <-closeC:
			case <-ctx.Done():
				return
			}

			select {
			case <-closeC:
			case <-ctx.Done():
				return
			}
		}()
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

func (c *Client) localForward(ctx context.Context, sshc *ssh.Client, req LocalForwardReq) {
	if err := c.localForward_(ctx, sshc, req); err != nil {
		log.Errorf("local forward: %v", err)
	}
}

func (c *Client) localForward_(ctx context.Context, sshc *ssh.Client, req LocalForwardReq) error {
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

		dial:     sshc.Dial,
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

func (c *Client) remoteForward(ctx context.Context, sshc *ssh.Client, req RemoteForwardReq) {
	if err := c.remoteForward_(ctx, sshc, req); err != nil {
		log.Errorf("remote forward: %v", err)
	}
}

func (c *Client) remoteForward_(ctx context.Context, sshc *ssh.Client, req RemoteForwardReq) error {
	// check RemoteAddr/RemotePort existence
	if c.remoteForwards.contains(req.RemotePort, req.RemoteAddr) {
		return errors.Errorf("remote addr occupied: %s:%d", req.RemoteAddr, req.RemotePort)
	}

	addr := net.JoinHostPort(req.RemoteAddr, fmt.Sprintf("%d", req.RemotePort))
	listener, err := sshc.Listen("tcp", addr)
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
