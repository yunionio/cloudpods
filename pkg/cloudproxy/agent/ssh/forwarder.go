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
	"io"
	"net"
	"time"

	"yunion.io/x/log"
)

type TickFunc func(context.Context)

type LocalForwardReq struct {
	LocalAddr  string
	LocalPort  int
	RemoteAddr string
	RemotePort int

	Tick   time.Duration
	TickCb TickFunc
}

type RemoteForwardReq struct {
	// LocalAddr is the address the forward will forward to
	LocalAddr string
	// LocalPort is the port the forward will forward to
	LocalPort int

	// RemoteAddr is the address on the remote to listen on
	RemoteAddr string
	// RemotePort is the address on the remote to listen on
	RemotePort int

	Tick   time.Duration
	TickCb TickFunc
}

type dialFunc func(n, addr string) (net.Conn, error)
type doneFunc func(laddr string, lport int)

type forwarder struct {
	listener net.Listener

	dial     dialFunc
	dialAddr string
	dialPort int

	done     doneFunc
	doneAddr string
	donePort int

	tick   time.Duration
	tickCb TickFunc
}

func (fwd *forwarder) Stop(ctx context.Context) {
	fwd.listener.Close()
}

func (fwd *forwarder) Start(
	ctx context.Context,
) {
	var (
		listener = fwd.listener
		dial     = fwd.dial
		dialAddr = fwd.dialAddr
		dialPort = fwd.dialPort
		done     = fwd.done
		doneAddr = fwd.doneAddr
		donePort = fwd.donePort
		tick     = fwd.tick
		tickCb   = fwd.tickCb
	)

	ctx, cancelFunc := context.WithCancel(ctx)

	if done != nil {
		defer done(doneAddr, donePort)
	}

	defer listener.Close()

	go func() { // accept local/remote connection
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Warningf("local forward: accept: %v", err)
				cancelFunc()
				break
			}
			go func(local net.Conn) {
				defer local.Close()

				// dial remote/local
				addr := net.JoinHostPort(dialAddr, fmt.Sprintf("%d", dialPort))
				remote, err := dial("tcp", addr)
				if err != nil {
					log.Warningf("local forward: dial remote: %v", err)
					return
				}
				defer remote.Close()

				// forward
				go io.Copy(local, remote)
				go io.Copy(remote, local)
				<-ctx.Done()
			}(conn)
		}
	}()

	if tick > 0 && tickCb != nil {
		go func() {
			ticker := time.NewTicker(tick)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					tickCb(ctx)
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	for {
		select {
		case <-ctx.Done():
			return
		}
	}
}
