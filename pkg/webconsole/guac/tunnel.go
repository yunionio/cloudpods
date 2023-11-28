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

package guac

import (
	"fmt"
	"io"
	"net"

	"yunion.io/x/pkg/errors"
)

type GuacamoleTunnel struct {
	opts *GuacOptions
	conn net.Conn

	instructs chan *Instruction
	err       chan error

	stopChan chan bool

	isStart bool
}

func (self *GuacamoleTunnel) Write(data []byte) (int, error) {
	return self.conn.Write(data)
}

func (self *GuacamoleTunnel) ReadOne() (*Instruction, error) {
	for {
		select {
		case instruct := <-self.instructs:
			return instruct, nil
		default:
		}
	}
}

func (self *GuacamoleTunnel) Start() error {
	return self.start()
}

func (self *GuacamoleTunnel) start() error {
	if self.isStart {
		return nil
	}
	go func() {
		self.isStart = true
		self.instructs = make(chan *Instruction, 10)
		self.stopChan = make(chan bool)
		self.err = make(chan error)
		var buf []byte = make([]byte, 4*1024*1024)
		var left []byte = []byte{}
		for {
			select {
			case <-self.stopChan:
				close(self.instructs)
				close(self.err)
				return
			default:
				n, err := self.conn.Read(buf)
				if err != nil && err != io.EOF {
					self.err <- err
					return
				}
				instructions, _left, err := parse(append(left, buf[:n]...))
				if err != nil {
					self.err <- err
					return
				}
				left = _left
				for i := range instructions {
					self.instructs <- instructions[i]
				}
			}
		}
	}()
	return nil
}

func (self *GuacamoleTunnel) Stop() {
	self.stopChan <- true
}

func (self *GuacamoleTunnel) Wait() error {
	err := <-self.err
	return err
}

func (self *GuacamoleTunnel) Handshake() error {
	selectArg := self.opts.Protocol
	if len(self.opts.ConnectionId) > 0 {
		selectArg = self.opts.ConnectionId
	}

	err := self.start()
	if err != nil {
		return err
	}

	_, err = self.Write([]byte(NewInstruction("select", selectArg).String()))
	if err != nil {
		return errors.Wrapf(err, "select")
	}

	args, err := self.ReadOne()
	if err != nil {
		return errors.Wrapf(err, "ReadOne")
	}

	if args.Opcode != "args" {
		return errors.Errorf("Invalid instruct %s", args.String())
	}

	for i, arg := range args.Args {
		args.Args[i] = self.opts.Parameters[arg]
	}

	_, err = self.Write([]byte(NewInstruction("size",
		fmt.Sprintf("%v", self.opts.OptimalScreenWidth),
		fmt.Sprintf("%v", self.opts.OptimalScreenHeight),
		fmt.Sprintf("%v", self.opts.OptimalResolution),
	).String()))
	if err != nil {
		return errors.Wrapf(err, "set size")
	}

	_, err = self.Write([]byte(NewInstruction("audio", self.opts.AudioMimetypes...).String()))
	if err != nil {
		return errors.Wrapf(err, "set audio")
	}
	_, err = self.Write([]byte(NewInstruction("video", self.opts.VideoMimetypes...).String()))
	if err != nil {
		return errors.Wrapf(err, "set video")
	}
	_, err = self.Write([]byte(NewInstruction("image", self.opts.ImageMimetypes...).String()))
	if err != nil {
		return errors.Wrapf(err, "set image")
	}
	_, err = self.Write([]byte(NewInstruction("connect", args.Args...).String()))
	if err != nil {
		return errors.Wrapf(err, "connect %s", args.Args)
	}

	ready, err := self.ReadOne()
	if err != nil {
		return errors.Wrapf(err, "read ready")
	}

	if ready.Opcode != "ready" {
		return fmt.Errorf("invalid ready instruction %s", ready.String())
	}

	if len(ready.Args) == 0 {
		return fmt.Errorf("no connection id received")
	}

	self.opts.ConnectionId = ready.Args[0]
	return nil
}
