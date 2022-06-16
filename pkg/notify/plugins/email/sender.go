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

package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"strconv"
	"time"

	"gopkg.in/gomail.v2"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/notify/models"
)

type sMessage struct {
	message *gomail.Message
	err     chan<- error
}

type sDialer struct {
	*gomail.Dialer
	open     bool
	sendAddr string
	err      chan error
	message  chan *sMessage
}

func (self *sDialer) validate() error {
	errChan := make(chan error, 1)
	go func() {
		sender, err := self.Dialer.Dial()
		if err != nil {
			errChan <- err
			return
		}
		defer sender.Close()
		errChan <- nil
	}()
	ticker := time.Tick(10 * time.Second)
	select {
	case <-ticker:
		return errors.Error("timeout")
	case err := <-errChan:
		return err
	}
}

func (self *sDialer) GetSenderAddress() string {
	return self.sendAddr
}

func newDialer(config api.SNotifyConfigContent) *sDialer {
	port, _ := strconv.Atoi(config.Hostport)
	dialer := gomail.NewDialer(config.Hostname, port, config.Username, config.Password)
	dialer.SSL = true
	if !config.SslGlobal {
		dialer.SSL = false
		dialer.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	ret := &sDialer{
		Dialer:   dialer,
		open:     false,
		sendAddr: config.SenderAddress,
		message:  make(chan *sMessage),
	}
	if len(ret.sendAddr) == 0 {
		ret.sendAddr = config.Username
	}
	return ret
}

func (self *sDialer) Run() {
	go func() {
		var s gomail.SendCloser
		var err error
		for {
			select {
			case m, ok := <-self.message:
				if !ok {
					return
				}
				if !self.open {
					s, err = self.Dialer.Dial()
					if err != nil {
						m.err <- errors.Wrapf(err, "Dialer.Dial")
						continue
					}
					self.open = true
				}
				err = gomail.Send(s, m.message)
				if err != nil {
					m.err <- errors.Wrapf(err, "gomail.Send")
				}
			// Close the connection to the SMTP server if no email was sent in
			// the last 30 seconds.
			case <-time.After(30 * time.Second):
				if self.open {
					err = s.Close()
					if err != nil {
						log.Errorf("failed close dialer")
						continue
					}
					self.open = false
				}
			}
		}
	}()
}

func (self *sDialer) Stop() {
	close(self.message)
}

type SEmailSender struct {
	dialers map[string]*sDialer
}

func (self *SEmailSender) GetSenderType() string {
	return api.EMAIL
}

func (self *SEmailSender) getSender(domainId string) (*sDialer, error) {
	dia, ok := self.dialers[domainId]
	if ok {
		return dia, nil
	}
	return nil, fmt.Errorf("no available email config")
}

func (self *SEmailSender) Send(ctx context.Context, args api.SendParams) error {
	if !self.IsValid() {
		return fmt.Errorf("no available email config")
	}
	for i := range args.Receivers {
		go func(i int) {
			sender, err := self.getSender(args.Receivers[i].DomainId)
			if err != nil {
				args.Receivers[i].Callback(errors.Wrapf(err, "getSender"))
				return
			}
			msg := gomail.NewMessage()
			msg.SetHeader("Subject", args.Topic)
			msg.SetHeader("Subject", args.Title)
			msg.SetBody("text/html", args.Message)
			msg.SetHeader("From", sender.GetSenderAddress())
			msg.SetHeader("To", args.Receivers[i].Contact)

			errMsg := make(chan error, 1)

			sender.message <- &sMessage{message: msg, err: errMsg}
			timer := time.NewTimer(1 * time.Minute)
			defer timer.Stop()
			select {
			case err := <-errMsg:
				args.Receivers[i].Callback(err)
			case <-timer.C:
				args.Receivers[i].Callback(errors.Wrapf(cloudprovider.ErrTimeout, "timeout for sending"))
			}
		}(i)
	}
	return nil
}

func (self *SEmailSender) ValidateConfig(ctx context.Context, config api.NotifyConfig) (string, error) {
	dialer := newDialer(config.SNotifyConfigContent)
	return "", dialer.validate()
}

func (self *SEmailSender) UpdateConfig(ctx context.Context, config api.NotifyConfig) error {
	sender, _ := self.getSender(config.GetDomainId())
	if sender != nil {
		sender.Stop()
	}
	dialer := newDialer(config.SNotifyConfigContent)
	dialer.Run()
	self.dialers[config.GetDomainId()] = dialer
	return nil
}

func (self *SEmailSender) AddConfig(ctx context.Context, config api.NotifyConfig) error {
	dialer := newDialer(config.SNotifyConfigContent)
	dialer.Run()
	self.dialers[config.GetDomainId()] = dialer
	return nil
}

func (self *SEmailSender) DeleteConfig(ctx context.Context, config api.NotifyConfig) error {
	sender, _ := self.getSender(config.GetDomainId())
	if sender != nil {
		sender.Stop()
	}
	return nil
}

func (self *SEmailSender) ContactByMobile(ctx context.Context, mobile, domainId string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SEmailSender) IsPersonal() bool {
	return true
}

func (self *SEmailSender) IsRobot() bool {
	return true
}

func (self *SEmailSender) IsValid() bool {
	return len(self.dialers) > 0
}

func (self *SEmailSender) IsPullType() bool {
	return true
}

func (self *SEmailSender) IsSystemConfigContactType() bool {
	return true
}

func init() {
	models.Register(&SEmailSender{
		dialers: map[string]*sDialer{},
	})
}
