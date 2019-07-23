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

package models

import (
	"fmt"
	"io/ioutil"
	"net/rpc"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
)

const (
	// ErrSendServiceNotFound means SRpcService's SendSerivces hasn't this Send Service.
	ErrSendServiceNotFound = errors.Error("Send Service Not Found")
	NOTINIT                = "Send service hasn't been init"
)

// RpcService is a single case of SRpcService
var RpcService *SRpcService

// SRpcService provide rpc service about sending message for notify module and manage these services.
// SendServices storage all send service and its name.
// lock protect the SendServices.
type SRpcService struct {
	SendServices  map[string]*rpc.Client
	socketFileDir string
	lock          sync.RWMutex
}

// NewSRpcService create a SRpcService
func NewSRpcService(socketFileDir string) *SRpcService {
	return &SRpcService{
		SendServices:  make(map[string]*rpc.Client),
		socketFileDir: socketFileDir,
	}
}

// InitAll init all Send Services, the init process is that:
// find all socket file in directory 'self.socketFileDir', if wrong return error;
// the name of file is the service's name; then try to dial to this rpc service
// through corresponding socket file, if failed, only print log but not return error.
func (self *SRpcService) InitAll() error {
	files, err := ioutil.ReadDir(self.socketFileDir)
	if err != nil {
		return errors.Wrapf(err, "read dir %s failed", self.socketFileDir)
	}
	for _, file := range files {
		filename := file.Name()
		if !file.IsDir() && strings.Contains(filename, ".sock") {
			serviceName := filename[:len(filename)-5]
			self.checkAndAddOne(serviceName)
		}
	}
	if len(self.SendServices) == 0 {
		log.Errorf("No available send service.")
	} else {
		log.Infof("Total %d send service init successful", len(self.SendServices))
	}
	return nil
}

// UpdateServices will detect the self.sockFileDir every delay seconds.
// Add new service and delete disappeared one from self.SendServices.
func (self *SRpcService) UpdateServices(delay int) {
	for {
		select {
		case <-time.After(time.Duration(delay) * time.Second):
			err := self.updateService()
			if err != nil {
				log.Errorf("update services failed because that %s.", err.Error())
			}
		}
	}
}

// StopAll stop all send service in self.SenderServices normally which can delete the socket file.
func (self *SRpcService) StopAll() {
	for _, service := range self.SendServices {
		service.Close()
	}
}

// Send call the corresponding rpc server.Send to send messager.
func (self *SRpcService) Send(contactType, contact, topic, msg, priority string) error {
	self.lock.RLock()
	sendService, ok := self.SendServices[contactType]
	self.lock.RUnlock()
	var err error
	if !ok {
		sendService, err = self.checkAndAddOne(contactType)
		if err == ErrDial {
			return ErrSendServiceNotFound
		}
		if err != nil {
			return errors.Wrap(err, "Check or Add connection failed")
		}
	}
	args := SSendArgs{
		Contact:  contact,
		Topic:    topic,
		Message:  msg,
		Priority: priority,
	}
	reply := SSendReply{}
	err = sendService.Call("Server.Send", &args, &reply)
	if err != nil {
		// should check and send again.
		// Possible situation: notify always keep connection but remote guy have restarted
		// so that connection valid.
		sendService, err = self.checkAndAddOne(contactType)
		if err != nil {
			return errors.Wrap(err, "Check or Add connection failed")
		}
		err = sendService.Call("Server.Send", &args, &reply)
		if err != nil {
			return errors.Wrap(err, "Send message failed.")
		}
		if !reply.Success {
			return errors.Error(fmt.Sprintf("Send message failed because that %s.", reply.Msg))
		}
	}
	if !reply.Success {
		if reply.Msg != NOTINIT {
			return errors.Error(fmt.Sprintf("Send message failed because that %s.", reply.Msg))
		}
		// should check and send again
		sendService, err = self.checkAndAddOne(contactType)
		if err != nil {
			return errors.Wrap(err, "Check or Add connection failed")
		}
		err = sendService.Call("Server.Send", &args, &reply)
		if err != nil {
			return errors.Wrap(err, "Send message failed.")
		}
		if !reply.Success {
			return errors.Error(fmt.Sprintf("Send message failed because that %s.", reply.Msg))
		}
	}
	return nil
}

// RestartService can restart remote rpc server and pass config info.
// When first init notify Server, must Call this function.
// When accept the request about changing config, must Call this function.
func (self *SRpcService) RestartService(config map[string]string, serviceName string) {
	self.lock.RLock()
	sendService, ok := self.SendServices[serviceName]
	self.lock.RUnlock()
	var err error
	if !ok {
		sendService, err = self.checkAndAddOne(serviceName)
		if err != nil {
			log.Debugf("Restart Failed: %s", err.Error())
			return
		}
	}
	args := SRestartArgs{Config: config}

	reply := SSendReply{}
	err = sendService.Call("Server.UpdateConfig", &args, &reply)
	if err != nil || !reply.Success {
		log.Errorf("Restart rpc serve whose name is %s failed.", serviceName)
		return
	}
}

// CheckAndAddOne check the status of service 'serviceName'
// If fail to dial to service, delete and remove sock file.
// if dial successfully, try to restart the service.
func (self *SRpcService) checkAndAddOne(serviceName string) (*rpc.Client, error) {
	// Try to connect again
	filename := filepath.Join(self.socketFileDir, serviceName+".sock")
	rpcService, err := rpc.Dial("unix", filename)
	if err != nil {
		log.Debugf("Try to dial to service failed which unix socket file name is %s.", filename)
		// This file maybe left behind inadvertently, so we should try to delete it
		os.Remove(filename)
		self.lock.Lock()
		delete(self.SendServices, serviceName)
		self.lock.Unlock()
		return nil, ErrDial
	}

	// GetKeyValue to config rpc Service
	config, err := ConfigManager.GetVauleByType(serviceName)
	if err != nil {
		log.Debugf("Init service error which unix socket file name is %s because that get config about this failed", filename)
		return nil, ErrGetConfig
	}
	args := SRestartArgs{config}
	reply := SSendReply{}
	rpcService.Call("Server.UpdateConfig", &args, &reply)
	if !reply.Success {
		log.Debugf("Init service error which unix socket file name is %s because that %s.", filename, reply.Msg)
		return nil, ErrUpdateConfig
	}
	self.lock.Lock()
	self.SendServices[serviceName] = rpcService
	self.lock.Unlock()
	return rpcService, nil
}

func (self *SRpcService) updateService() error {
	files, err := ioutil.ReadDir(self.socketFileDir)
	if err != nil {
		return errors.Wrapf(err, "read dir %s failed", self.socketFileDir)
	}
	original := make(map[string]*rpc.Client)
	self.lock.RLock()
	for serviceName, client := range self.SendServices {
		original[serviceName] = client
	}
	self.lock.RUnlock()
	for _, file := range files {
		filename := file.Name()
		if !file.IsDir() && strings.Contains(filename, ".sock") {
			serviceName := filename[:len(filename)-5]
			if _, ok := self.SendServices[serviceName]; ok {
				delete(original, serviceName)
				continue
			}
			self.checkAndAddOne(serviceName)
		}
	}
	self.lock.Lock()
	for serviceName := range original {
		delete(self.SendServices, serviceName)
	}
	self.lock.Unlock()
	for _, client := range original {
		client.Close()
	}
	return nil
}

type SSendArgs struct {
	Contact  string
	Topic    string
	Message  string
	Priority string
}

type SRestartArgs struct {
	Config map[string]string
}

type SSendReply struct {
	Success bool
	Msg     string
}
