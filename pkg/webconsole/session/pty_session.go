package session

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/kr/pty"

	"yunion.io/x/log"
)

type Pty struct {
	Session *SSession
	Cmd     *exec.Cmd
	Pty     *os.File
	sizeCh  chan os.Signal
	size    *pty.Winsize
	Show    bool
	IsOk    bool
	Buffer  string
	Output  string
	Command string
	Exit    bool
}

func NewPty(session *SSession) (p *Pty, err error) {
	cmd := session.GetCommand()
	p = &Pty{
		Session: session,
		Cmd:     cmd,
		Show:    true,
		IsOk:    true,
		Exit:    false,
		Pty:     nil,
	}
	log.Debugf("[session %s] Start command: %#v", session.Id, cmd)
	if cmd != nil {
		p.Pty, err = pty.Start(p.Cmd)
		if err != nil {
			return
		}
	}
	p.sizeCh = make(chan os.Signal, 1)
	p.size = &pty.Winsize{}
	p.startResizeMonitor()
	signal.Notify(p.sizeCh, syscall.SIGWINCH) // Initail resize
	return
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
		}
	}()
}

func (p *Pty) Resize(size *pty.Winsize) {
	p.size = size
	p.sizeCh <- syscall.SIGWINCH
}

func (p *Pty) Stop() {
	if p.Pty != nil {
		if err := p.Pty.Close(); err != nil {
			log.Errorf("Close PTY error: %v", err)
		}
	}
	if p.Cmd != nil && p.Cmd.Process != nil {
		if err := p.Cmd.Process.Signal(os.Kill); err != nil {
			log.Errorf("Kill command process error: %v", err)
		} else if err := p.Cmd.Wait(); err != nil {
			log.Errorf("Wait command error: %v", err)
		}
	}
	p.Session.Close()
}
