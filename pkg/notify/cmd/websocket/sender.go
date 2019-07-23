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
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
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
	region      string
	clientLock  sync.RWMutex // lock to protect client
	session     *mcclient.ClientSession

	configCache   sConfigCache   // config cache
	configLock    sync.RWMutex   // lock to protect config cache
	templateCache sTemplateCache // template cache
	templateLock  sync.RWMutex   // lock to protect template cache
}

func newSSenderManager(config *SRegularConfig) *sSenderManager {
	return &sSenderManager{
		workerChan:  make(chan struct{}, config.SenderNum),
		templateDir: config.TemplateDir,
		region:      config.Region,

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
	authUri, ok := self.configCache[AUTH_URI]
	if !ok {
		self.configLock.RUnlock()
		return
	}
	adminUser, ok := self.configCache[ADMIN_USER]
	if !ok {
		self.configLock.RUnlock()
		return
	}
	adminPassword, ok := self.configCache[ADMIN_PASSWORD]
	if !ok {
		self.configLock.RUnlock()
		return
	}
	adminTenantName, ok := self.configCache[ADMIN_TENANT_NAME]
	self.configLock.RUnlock()

	self.clientLock.Lock()
	defer self.clientLock.Unlock()
	a := auth.NewAuthInfo(authUri, "", adminUser, adminPassword, adminTenantName)
	auth.Init(a, false, true, "", "")
	self.session = auth.GetAdminSession(self.region, "")
}

func (self *sSenderManager) send(args *SSendArgs, reply *SSendReply) {
	var title, content string
	title, err := self.getContent("title", args.Topic, args.Message)
	if err != nil {
		dealError(reply, err)
		return
	}
	content, err = self.getContent("content", args.Topic, args.Message)
	if err != nil {
		dealError(reply, err)
		return
	}
	// component request body
	body := jsonutils.DeepCopy(params).(*jsonutils.JSONDict)
	body.Add(jsonutils.NewString(title), "action")
	body.Add(jsonutils.NewString(fmt.Sprintf("priority=%s; content=%s", args.Priority, content)), "notes")
	body.Add(jsonutils.NewString(args.Contact), "user_id")
	body.Add(jsonutils.NewString(args.Contact), "user")
	if len(args.Contact) == 0 {
		body.Add(jsonutils.JSONTrue, "broadcast")
	}
	self.clientLock.RLock()
	session := self.session
	self.clientLock.RUnlock()
	_, err = modules.Websockets.Create(session, body)
	if err != nil {
		// failed
		self.initClient()
		self.clientLock.RLock()
		session = self.session
		self.clientLock.RUnlock()
		_, err = modules.Websockets.Create(session, body)
		if err != nil {
			dealError(reply, err)
			return
		}
	}
	reply.Success = true
}

func dealError(reply *SSendReply, err error) {
	reply.Success = false
	reply.Msg = err.Error()
	log.Errorf("Send message failed because that %s.", err.Error())
}
