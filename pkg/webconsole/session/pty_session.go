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
}

func NewPty(session *SSession) (p *Pty, err error) {
	cmd := session.GetCommand()
	p = &Pty{
		Session: session,
		Cmd:     cmd,
	}
	log.Debugf("[session %s] Start command: %#v", session.Id, cmd)
	p.Pty, err = pty.Start(p.Cmd)
	if err != nil {
		return
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
			if err := pty.Setsize(p.Pty, p.size); err != nil {
				log.Errorf("Resize pty error: %v", err)
			} else {
				log.Debugf("Resize pty to %#v, cmd: %#v", p.size, p.Cmd)
			}
		}
	}()
}

func (p *Pty) Resize(size *pty.Winsize) {
	p.size = size
	p.sizeCh <- syscall.SIGWINCH
}

func (p *Pty) Stop() {
	var err error
	err = p.Pty.Close()
	if err != nil {
		log.Errorf("Close PTY error: %v", err)
	}
	err = p.Cmd.Process.Signal(os.Kill)
	if err != nil {
		log.Errorf("Kill command process error: %v", err)
	}
	err = p.Cmd.Wait()
	if err != nil {
		log.Errorf("Wait command error: %v", err)
	}
	p.Session.Close()
}
