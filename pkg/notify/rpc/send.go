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

package rpc

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
	_interface "yunion.io/x/onecloud/pkg/notify/interface"
	"yunion.io/x/onecloud/pkg/notify/models"
	"yunion.io/x/onecloud/pkg/notify/rpc/apis"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

const (
	// ErrSendServiceNotFound means SRpcService's SendSerivces hasn't this Send Service.
	ErrSendServiceNotFound = errors.Error("No such send service")
	ErrSendServiceNotInit  = errors.Error("Send service hasn't been init")
)

// SRpcService provide rpc service about sending message for notify module and manage these services.
// SendServices storage all send service.
type SRpcService struct {
	SendServices  *ServiceMap
	socketFileDir string
	configStore   _interface.IServiceConfigStore
	templateStore _interface.ITemplateStore
}

// NewSRpcService create a SRpcService
func NewSRpcService(socketFileDir string, configStore _interface.IServiceConfigStore,
	tempalteStore _interface.ITemplateStore) *SRpcService {
	return &SRpcService{
		SendServices:  NewServiceMap(),
		socketFileDir: socketFileDir,
		configStore:   configStore,
		templateStore: tempalteStore,
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
	ctx := context.Background()
	for _, file := range files {
		filename := file.Name()
		if !file.IsDir() && strings.Contains(filename, ".sock") {
			serviceName := filename[:len(filename)-5]
			self.startNewService(ctx, serviceName)
		}
	}
	if self.SendServices.Len() == 0 {
		log.Errorf("No available send service.")
	} else {
		log.Infof("Total %d send service init successful", self.SendServices.Len())
	}
	return nil
}

// UpdateServices will detect the self.sockFileDir, add new service and
// delete disappeared one from self.SendServices.
func (self *SRpcService) UpdateServices(ctx context.Context, usreCred mcclient.TokenCredential, isStart bool) {
	err := self.updateService(ctx)
	if err != nil {
		log.Errorf("update services failed because that %s.", err.Error())
	}
}

// StopAll stop all send service in self.SenderServices normally which can delete the socket file.
func (self *SRpcService) StopAll() {
	f := func(client *apis.SendNotificationClient) {
		client.Conn.Close()
	}
	self.SendServices.Map(f)
}

// Send call the corresponding rpc server to send messager.
func (self *SRpcService) Send(ctx context.Context, contactType, contact, topic, msg, priority string) error {

	args, err := self.templateStore.NotifyFilter(contactType, topic, msg)
	if err != nil {
		return errors.Wrap(err, "templateStore.NotifyFilter")
	}

	args.Contact = contact
	args.Priority = priority

	f := func(service *apis.SendNotificationClient) (interface{}, error) {
		log.Debugf("send one")
		return service.Send(ctx, &args)
	}

	_, err = self.execute(ctx, f, contactType)
	if err != nil {
		return errors.Wrapf(err, "contactType: %s", contactType)
	}
	return nil
}

// RestartService can restart remote rpc server and pass config info.
// This function should be call immediately after init notify server firstly
// This function should be call immediately after accept the request about changing config.
func (self *SRpcService) RestartService(ctx context.Context, config _interface.SConfig, serviceName string) {
	_, err := self.restartWithConfig(ctx, serviceName, config)
	if err != nil {
		log.Debugf("restart service failed: %s", err)
	}
}

func (self *SRpcService) ContactByMobile(ctx context.Context, mobile, serviceName string) (string, error) {

	args := apis.UseridByMobileParams{}
	args.Mobile = mobile

	f := func(service *apis.SendNotificationClient) (interface{}, error) {
		return service.UseridByMobile(ctx, &args)
	}

	ret, err := self.execute(ctx, f, serviceName)
	if err != nil {
		return "", err
	}

	reply := ret.(*apis.UseridByMobileReply)
	return reply.Userid, nil
}

// Wrap function to execute function call rpc server
func (self *SRpcService) execute(ctx context.Context, f func(client *apis.SendNotificationClient) (interface{}, error),
	serviceName string) (interface{}, error) {

	sendService, ok := self.SendServices.Get(serviceName)

	log.Debugf("get service %s", serviceName)
	var err error
	if !ok {
		log.Debugf("get service first time failed")
		sendService, err = self.startNewService(ctx, serviceName)

		if err != nil {
			return nil, errors.Wrap(err, "start new service failed")
		}
	}

	ret, err := f(sendService)

	if err != nil {
		// hander error
		st := status.Convert(err)
		if st.Code() == codes.Unavailable {
			// sock is bad
			self.closeService(ctx, serviceName)
			return nil, ErrSendServiceNotFound
		}

		if st.Message() != ErrSendServiceNotInit.Error() {
			return nil, errors.Error(st.Message())
		}

		// if NOINIT, try to restart server and send again
		sendService, err = self.restartService(ctx, serviceName)
		if err != nil {
			return nil, errors.Wrapf(err, "restart service %s failed", serviceName)
		}

		_, err := f(sendService)
		if err != nil {
			st := status.Convert(err)
			if st.Code() == codes.Unavailable {
				// sock is bad
				self.closeService(ctx, serviceName)

				return nil, errors.Wrap(ErrSendServiceNotFound, serviceName)
			}
			return nil, errors.Error(st.Message())
		}
	}
	return ret, nil
}

// restartSrevice fetch config from IServiceConfigStore and Call rpc.UpdateConfig
func (self *SRpcService) restartService(ctx context.Context, serviceName string) (*apis.SendNotificationClient, error) {

	config, err := self.configStore.GetConfig(serviceName)
	if err != nil {
		log.Debugf("getConfig of serveice %s from database error", serviceName)
		return nil, models.ErrGetConfig
	}
	return self.restartWithConfig(ctx, serviceName, config)
}

func (self *SRpcService) restartWithConfig(ctx context.Context, serviceName string,
	config map[string]string) (*apis.SendNotificationClient, error) {

	var (
		sendService *apis.SendNotificationClient
		err         error
	)

	sendService, ok := self.SendServices.Get(serviceName)

	if !ok {
		return nil, fmt.Errorf("no such service, please start new service")
	}

	args := apis.UpdateConfigParams{}
	args.Configs = config
	_, err = sendService.UpdateConfig(ctx, &args)
	if err != nil {
		st := status.Convert(err)
		return nil, errors.Error(st.Message())
	}

	return sendService, nil
}

// startNewService try to start a new rpc service named serviceName
func (self *SRpcService) startNewService(ctx context.Context, serviceName string) (*apis.SendNotificationClient, error) {

	var (
		sendService *apis.SendNotificationClient
		err         error
	)

	filename := filepath.Join(self.socketFileDir, serviceName+".sock")
	if !fileutils2.Exists(filename) {
		return nil, errors.Error(fmt.Sprintf("no such socket file '%s'", filename))
	}

	grpcConn, err := grpcDialWithUnixSocket(ctx, filename)
	if err != nil {
		return nil, err
	}
	sendService = apis.NewSendNotificationClient(grpcConn)

	self.SendServices.Set(sendService, serviceName)

	// get config
	config, err := self.configStore.GetConfig(serviceName)
	if err != nil {
		log.Errorf("getConfig of serveice %s from database error", serviceName)
		return nil, models.ErrGetConfig
	}

	// update config for service
	args := apis.UpdateConfigParams{}
	args.Configs = config
	_, err = sendService.UpdateConfig(ctx, &args)
	if err != nil {
		st := status.Convert(err)
		if st.Code() == codes.Unavailable {
			// no such rpc serve
			os.Remove(filename)
			return nil, fmt.Errorf("no such rpc serve")
		}
		return nil, fmt.Errorf(st.Message())
	}

	return sendService, nil
}

// closeService will remove service record from self.SendServices and try to remove sock file
func (self *SRpcService) closeService(ctx context.Context, serviceName string) {
	filename := filepath.Join(self.socketFileDir, serviceName+".sock")
	self.SendServices.Remove(serviceName)
	os.Remove(filename)
}

func (self *SRpcService) updateService(ctx context.Context) error {
	files, err := ioutil.ReadDir(self.socketFileDir)
	if err != nil {
		return errors.Wrapf(err, "read dir %s failed", self.socketFileDir)
	}

	serviceNames := self.SendServices.ServiceNames()
	serviceNameSet := make(map[string]struct{})
	for _, name := range serviceNames {
		serviceNameSet[name] = struct{}{}
	}

	for _, file := range files {
		filename := file.Name()
		if !file.IsDir() && strings.Contains(filename, ".sock") {
			serviceName := filename[:len(filename)-5]
			if self.SendServices.IsExist(serviceName) {
				delete(serviceNameSet, serviceName)
				continue
			}
			self.startNewService(ctx, serviceName)
		}
	}

	serviceNames = serviceNames[:0]
	for serviceName := range serviceNameSet {
		serviceNames = append(serviceNames, serviceName)
	}

	self.SendServices.BatchRemove(serviceNames)
	return nil
}

func grpcDialWithUnixSocket(ctx context.Context, socketPath string) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, socketPath, grpc.WithInsecure(), grpc.WithTimeout(time.Second*5), grpc.WithDialer(
		func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)
}
