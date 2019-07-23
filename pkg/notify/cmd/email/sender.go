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

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"gopkg.in/gomail.v2"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

type sTemplateCache map[string]*template.Template

func newSTemplateCache() sTemplateCache {
	return make(map[string]*template.Template)
}

type sConfigCache map[string]string

func newSConfigCache() sConfigCache {
	return make(map[string]string)
}

type sSenderManager struct {
	msgChan     chan *sSendUnit
	senders     []sSender
	senderNum   int
	chanelSize  int
	templateDir string

	configCache   sConfigCache
	configLock    sync.RWMutex
	templateCache sTemplateCache
	templateLock  sync.RWMutex
}

func newSSenderManager(config *SRegularConfig) *sSenderManager {
	return &sSenderManager{
		senders:     make([]sSender, config.SenderNum),
		senderNum:   config.SenderNum,
		chanelSize:  config.ChannelSize,
		templateDir: config.TemplateDir,

		configCache:   newSConfigCache(),
		templateCache: newSTemplateCache(),
	}
}

func (self *sSenderManager) updateTemplateCache() {
	self.updateTemplate("title")
	self.updateTemplate("content")
}

func (self *sSenderManager) updateTemplate(torc string) {
	templateDir := filepath.Join(self.templateDir, torc)
	files, err := ioutil.ReadDir(templateDir)
	if err != nil {
		log.Errorf("Incorrect template directory '%s': %s", self.templateDir, err.Error())
		return
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		templatePath := filepath.Join(templateDir, file.Name())
		tem, err := template.ParseFiles(templatePath)
		if err != nil {
			log.Errorf("Parse file '%s' to template failed.", templatePath)
			continue
		}
		key := strings.ToLower(file.Name()) + "." + torc
		self.templateLock.Lock()
		self.templateCache[key] = tem
		self.templateLock.Unlock()
	}
}

func (self *sSenderManager) send(args *SSendArgs, reply *SSendReply) {
	gmsg := gomail.NewMessage()
	username := senderManager.configCache[USERNAME]
	gmsg.SetHeader("From", username)
	gmsg.SetHeader("To", args.Contact)
	gmsg.SetHeader("Subject", args.Topic)
	title, err := self.getContent("title", args.Topic, args.Message)
	if err != nil {
		reply.Success = false
		reply.Msg = err.Error()
		return
	}
	gmsg.SetHeader("Subject", title)
	content, err := self.getContent("content", args.Topic, args.Message)
	if err != nil {
		reply.Success = false
		reply.Msg = err.Error()
		return
	}
	gmsg.SetBody("text/html", content)
	ret := make(chan bool)
	senderManager.msgChan <- &sSendUnit{gmsg, ret}
	reply.Success = <-ret
	// try again
	if !reply.Success {
		senderManager.msgChan <- &sSendUnit{gmsg, ret}
		reply.Success = <-ret
		reply.Msg = "send failed."
	}
}

func (self *sSenderManager) getContent(torc, topic, msg string) (string, error) {
	key := topic + "." + msg
	self.templateLock.RLock()
	tem, ok := self.templateCache[key]
	self.templateLock.RUnlock()
	if !ok {
		if torc == "title" {
			return topic, nil
		}
		return msg, nil
	}
	var content string
	if torc == "title" {
		content = topic
	} else {
		content = msg
	}

	tmpMap := make(map[string]interface{})
	err := json.Unmarshal([]byte(content), &tmpMap)
	if err != nil {
		return "", errors.Error("Message should be a canonical JSON")
	}
	buffer := new(bytes.Buffer)
	err = tem.Execute(buffer, tmpMap)
	if err != nil {
		return "", errors.Error("Message content and template do not match")
	}
	return buffer.String(), nil
}

func (self *sSenderManager) restartSender() {
	for _, sender := range self.senders {
		sender.stop()
	}
	self.initSender()
}

func (self *sSenderManager) initSender() {
	self.configLock.RLock()
	hostName, ok := self.configCache[HOSTNAME]
	if !ok {
		self.configLock.RUnlock()
		return
	}
	password, ok := self.configCache[PASSWORD]
	if !ok {
		self.configLock.RUnlock()
		return
	}
	userName, ok := self.configCache[USERNAME]
	if !ok {
		self.configLock.RUnlock()
		return
	}
	hostPortStr, ok := self.configCache[HOSTPORT]
	if !ok {
		self.configLock.RUnlock()
		return
	}
	self.configLock.RUnlock()
	hostPort, _ := strconv.Atoi(hostPortStr)
	dialer := gomail.NewDialer(hostName, hostPort, userName, password)
	// Configs are obtained successfully, it's time to init msgChan.
	if self.msgChan == nil {
		self.msgChan = make(chan *sSendUnit, self.chanelSize)
	}
	for i := 0; i < self.senderNum; i++ {
		sender := sSender{
			number: i + 1,
			dialer: dialer,
			sender: nil,
			open:   false,
			stopC:  make(chan struct{}),
		}
		self.senders[i] = sender
		go sender.Run()
	}

	log.Infof("Total %d senders.", self.senderNum)
}

type sSender struct {
	number int
	dialer *gomail.Dialer
	sender gomail.SendCloser
	open   bool
	stopC  chan struct{}
}

func (self *sSender) Run() {
	var err error
Loop:
	for {
		select {
		case msg, ok := <-senderManager.msgChan:
			if !ok {
				break Loop
			}
			if !self.open {
				if self.sender, err = self.dialer.Dial(); err != nil {
					log.Errorf("No.%d sender connect to email serve failed because that %s.", self.number, err.Error())
					msg.result <- false
					break Loop
				}
				self.open = true
				if err := gomail.Send(self.sender, msg.message); err != nil {
					log.Errorf("No.%d sender send email failed because that %s.", self.number, err.Error())
					self.open = false
				}
				log.Debugf("No.%d sender send email successfully.", self.number)
				msg.result <- true
			}
		case <-self.stopC:
			break Loop
		case <-time.After(30 * time.Second):
			if self.open {
				if err = self.sender.Close(); err != nil {
					log.Errorf("No.%d sender has be idle for 30 seconds and closed failed because that %s.", self.number, err.Error())
					break Loop
				}
				self.open = false
				log.Infof("No.%d sender has be idle for 30 seconds so that closed temporarily.", self.number)
			}
		}
	}
}

func (self *sSender) stop() {
	// First restart
	if self.stopC == nil {
		return
	}
	close(self.stopC)
}

type sSendUnit struct {
	message *gomail.Message
	result  chan<- bool
}
