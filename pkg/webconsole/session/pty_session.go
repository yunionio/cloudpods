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

package session

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/creack/pty"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type Pty struct {
	Session    *SSession
	Cmd        *exec.Cmd
	Pty        *os.File
	sizeCh     chan os.Signal
	size       *pty.Winsize
	OriginSize *pty.Winsize
	Exit       bool
}

func NewPty(session *SSession) (p *Pty, err error) {
	cmd := session.GetCommand()
	p = &Pty{
		Session: session,
		Cmd:     cmd,
		Exit:    false,
		Pty:     nil,
	}
	log.Debugf("[session %s] Start command: %#v", session.Id, cmd)
	if cmd != nil {
		p.Pty, err = pty.Start(p.Cmd)
		if err != nil {
			log.Errorf("start cmd error: %v", err)
			p.Cmd = nil
			return
		}
	}
	p.sizeCh = make(chan os.Signal, 1)
	p.size = &pty.Winsize{}
	p.startResizeMonitor()
	signal.Notify(p.sizeCh, syscall.SIGWINCH) // Initail resize
	return
}

func (p *Pty) IsInShellMode() bool {
	if p.Cmd == nil || p.Cmd.Process == nil {
		return false
	}
	return true
}

func (p *Pty) Read() ([]byte, error) {
	if !p.IsInShellMode() {
		return nil, errors.Error("not in shell mode")
	}
	buf := make([]byte, 1024)
	n, err := p.Pty.Read(buf)
	if err != nil {
		return nil, errors.Wrap(err, "Pty.Read")
	}
	return buf[0:n], nil
}

func (p *Pty) startResizeMonitor() {
	go func() {
		for range p.sizeCh {
			if p.Pty != nil {
				if err := pty.Setsize(p.Pty, p.size); err != nil {
					log.Errorf("Resize pty error: %v", err)
				} else {
					log.Debugf("Resize pty to %#v, cmd: %#v", p.size, p.Cmd)
				}
			}
			p.OriginSize = p.size
		}
	}()
}

func (p *Pty) Resize(size *pty.Winsize) {
	p.size = size
	p.sizeCh <- syscall.SIGWINCH
}

func (p *Pty) Stop() (err error) {
	var errs []error

	defer func() {
		p.Cmd, p.Pty = nil, nil
	}()

	defer func() {
		err = errors.NewAggregate(errs)
	}()
	// LOCK required
	defer func() {
		if err := p.Session.Close(); err != nil {
			errs = append(errs, err)
		}
	}()
	defer func() {
		if p.Cmd != nil && p.Cmd.Process != nil {
			log.Debugf("[%s] stop cmd: %s", p.Session.Id, p.Cmd.String())
			err := p.Cmd.Process.Signal(os.Kill)
			if err != nil {
				errs = append(errs, err)
				log.Errorf("Kill command process error: %v", err)
				return
			}
			err = p.Cmd.Wait()
			if err != nil {
				errs = append(errs, err)
				log.Errorf("Wait command error: %v", err)
				return
			}
		}
	}()
	defer func() {
		if p.Pty != nil {
			err := p.Pty.Close()
			if err != nil {
				log.Errorf("Close PTY error: %v", err)
				errs = append(errs, err)
				return
			}
		}
	}()

	return
}
