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
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"yunion.io/x/log"
)

type sTemplateCache map[string]string

func newSTemplateCache() sTemplateCache {
	return make(map[string]string)
}

type sConfigCache map[string]string

func newSConfigCache() sConfigCache {
	return make(map[string]string)
}

type sSenderManager struct {
	workerChan  chan struct{}
	templateDir string
	client      *sdk.Client  // client to send sms
	clientLock  sync.RWMutex // lock to protect client

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
	files, err := ioutil.ReadDir(self.templateDir)
	if err != nil {
		log.Errorf("Incorrect template directory '%s': %s", self.templateDir, err.Error())
		return
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		path := filepath.Join(self.templateDir, file.Name())
		content, err := ioutil.ReadFile(path)
		if err != nil {
			log.Errorf("can't read content of such file '%s' because that %s", path, err.Error())
			return
		}
		templateName := strings.TrimSpace(string(content))
		//templateName := filepath.Base(file[0].Name())
		self.templateLock.Lock()
		self.templateCache[strings.ToLower(file.Name())] = templateName
		self.templateLock.Unlock()
	}
}

func (self *sSenderManager) initClient() {
	self.configLock.RLock()
	accessKeyID, ok := self.configCache[ACCESS_KEY_ID]
	if !ok {
		self.configLock.RUnlock()
		return
	}
	accessKeySecret, ok := self.configCache[ACCESS_KEY_SECRET]
	if !ok {
		self.configLock.RUnlock()
		return
	}
	self.configLock.RUnlock()
	// lock and update
	self.clientLock.Lock()
	defer self.clientLock.Unlock()
	client, err := sdk.NewClientWithAccessKey("default", accessKeyID, accessKeySecret)
	if err != nil {
		log.Errorf("client connect failed because that %s.", err.Error())
		return
	}
	self.client = client
	log.Printf("Total %d workers.", cap(self.workerChan))
}

func (self *sSenderManager) send(req *requests.CommonRequest, reply *SSendReply) {
	self.clientLock.RLock()
	client := self.client
	self.clientLock.RUnlock()
	reponse, err := client.ProcessCommonRequest(req)
	if err == nil {
		if reponse.IsSuccess() {
			log.Debugf("send message successfully")
			reply.Success = true
			return
		}
		log.Debugf(reponse.GetHttpContentString())
		reply.Success = false
		log.Errorf("Send message failed because that %s.", err.Error())
		//todo There may be detailed processing for different errors.
		return
	}
	//todo
	self.initClient()
	// try again
	self.clientLock.RLock()
	client = self.client
	self.clientLock.RUnlock()
	reponse, err = client.ProcessCommonRequest(req)
	if err != nil {
		//todo There may be detailed processing for different errors.
		dealError(reply, err)
		return
	}
	if reponse.IsSuccess() {
		reply.Success = true
	}
}

func dealError(reply *SSendReply, err error) {
	reply.Success = false
	reply.Msg = err.Error()
	log.Errorf("Send message failed because that %s.", err.Error())
}
