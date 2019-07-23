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
	"strings"
	"sync"
	"text/template"

	"github.com/hugozhu/godingtalk"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

var senderManager *sSenderManager

type sTemplateCache map[string]*template.Template

func newSTemplateCache() sTemplateCache {
	return make(map[string]*template.Template)
}

type sConfigCache map[string]string

func newSConfigCache() sConfigCache {
	return make(map[string]string)
}

type sSendFunc func(*sSenderManager, string) error

type sSenderManager struct {
	workerChan  chan struct{}
	templateDir string
	client      *godingtalk.DingTalkClient // client to send sms
	clientLock  sync.RWMutex               // lock to protect client

	configCache   sConfigCache   // config cache
	configLock    sync.RWMutex   // lock to protect config cache
	templateCache sTemplateCache // template cache
	templateLock  sync.RWMutex   // lock to protect template cache
}

func newSSenderManager(config *SRegularConfig) *sSenderManager {
	return &sSenderManager{
		workerChan:  make(chan struct{}, config.SenderNum),
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

func (self *sSenderManager) getSendFunc(args *SSendArgs) (sSendFunc, error) {
	title, err := self.getContent("title", args.Topic, args.Message)
	if err != nil {
		return nil, err
	}
	content, err := self.getContent("content", args.Topic, args.Message)
	if err != nil {
		return nil, err
	}
	if title == args.Topic && content == args.Message {
		return func(manager *sSenderManager, agentID string) error {
			manager.clientLock.RLock()
			client := manager.client
			manager.clientLock.RUnlock()
			return client.SendAppMessage(agentID, args.Contact, content)
		}, nil
	}
	message := godingtalk.OAMessage{}
	message.Head.Text = args.Topic
	message.Body.Title = title
	message.Body.Content = content
	return func(manager *sSenderManager, agentID string) error {
		manager.clientLock.RLock()
		client := manager.client
		manager.clientLock.RUnlock()
		return client.SendAppOAMessage(agentID, args.Contact, message)
	}, nil
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

func (self *sSenderManager) initClient() {
	self.configLock.RLock()
	appKey, ok := self.configCache[APP_KEY]
	if !ok {
		self.configLock.RUnlock()
		return
	}
	appSecret, ok := self.configCache[APP_SECRET]
	if !ok {
		self.configLock.RUnlock()
		return
	}
	self.configLock.RUnlock()
	// lock and update
	self.clientLock.Lock()
	defer self.clientLock.Unlock()
	oldClient := self.client
	client := godingtalk.NewDingTalkClient(appKey, appSecret)
	err := client.RefreshAccessToken()
	if err != nil {
		self.client = oldClient
		return
	}
	self.client = client
}

func (self *sSenderManager) send(reply *SSendReply, sendFunc sSendFunc) {
	// get agentID
	self.configLock.RLock()
	agentID, ok := self.configCache[AGENT_ID]
	self.configLock.RUnlock()
	if !ok {
		reply.Success = false
		reply.Msg = "AgentID has not been init"
		return
	}

	// send message
	err := sendFunc(self, agentID)
	if err == nil {
		log.Debugf("send message successfully.")
		reply.Success = true
		return
	}

	// access_token maybe be expired
	if strings.Contains(err.Error(), "access_token") || strings.Contains(err.Error(), "accessToken") {
		self.initClient()
		// try again
		err = sendFunc(self, agentID)
		if err != nil {
			dealError(reply, err)
			return
		}
		reply.Success = true
		log.Debugf("send message successfully.")
		return
	}
	dealError(reply, err)
	return
}

func dealError(reply *SSendReply, err error) {
	reply.Success = false
	reply.Msg = err.Error()
	log.Errorf("Send message failed because that %s.", err.Error())
}
