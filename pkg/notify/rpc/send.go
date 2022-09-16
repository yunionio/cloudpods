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

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/mcclient"
	notifyv2 "yunion.io/x/onecloud/pkg/notify"
	"yunion.io/x/onecloud/pkg/notify/rpc/apis"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

const (
	// ErrSendServiceNotFound means SRpcService's SendSerivces hasn't this Send Service.
	ErrSendServiceNotFound = errors.Error("No such send service")
	// ErrSendServiceNotInit  = errors.Error("Send service hasn't been init")
)

// SRpcService provide rpc service about sending message for notify module and manage these services.
// SendServices storage all send service.
type SRpcService struct {
	SendServices  *ServiceMap
	socketFileDir string
	configStore   notifyv2.IServiceConfigStore
	templateStore notifyv2.ITemplateStore
}

// NewSRpcService create a SRpcService
func NewSRpcService(socketFileDir string, configStore notifyv2.IServiceConfigStore,
	tempalteStore notifyv2.ITemplateStore) *SRpcService {
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
	files, err := os.ReadDir(self.socketFileDir)
	if err != nil {
		return errors.Wrapf(err, "read dir %s failed", self.socketFileDir)
	}
	ctx := context.Background()
	for _, file := range files {
		filename := file.Name()
		if !file.IsDir() && strings.Contains(filename, ".sock") {
			serviceName := filename[:len(filename)-5]
			self.startNewService(ctx, serviceName, true)
		}
	}
	if self.SendServices.Len() == 0 {
		log.Infof("No available send service.")
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
func (self *SRpcService) Send(ctx context.Context, contactType string, args apis.SendParams) error {
	// Stop sending that must fail early
	if len(args.RemoteTemplate) == 0 && contactType == api.MOBILE {
		return fmt.Errorf("empty remote template for mobile type notification")
	}
	var err error
	f := func(service *apis.SendNotificationClient) (interface{}, error) {
		log.Debugf("send one")
		return service.Send(ctx, &args)
	}

	_, err = self.execute(ctx, f, contactType)
	if err != nil {
		s, ok := status.FromError(err)
		if !ok {
			return err
		}
		return errors.Error(s.Message())
	}
	return nil
}

func (self *SRpcService) BatchSend(ctx context.Context, contactType string, args apis.BatchSendParams) ([]*apis.FailedRecord, error) {
	// Stop sending that must fail early
	if len(args.RemoteTemplate) == 0 && contactType == api.MOBILE {
		return nil, fmt.Errorf("empty remote template for mobile type notification")
	}
	ret := make([]*apis.FailedRecord, 0)
	f := func(service *apis.SendNotificationClient) (interface{}, error) {
		return service.BatchSend(ctx, &args)
	}

	i, err := self.execute(ctx, f, contactType)
	if err != nil {
		s, ok := status.FromError(err)
		if !ok {
			return nil, err
		}
		return nil, errors.Error(s.Message())
	}
	reply := i.(*apis.BatchSendReply)
	return append(ret, reply.FailedRecords...), nil
}

// UpdateConfig can update config for rpc service with domainId
func (self *SRpcService) UpdateConfig(ctx context.Context, service string, config notifyv2.SConfig) error {
	var (
		sendService *apis.SendNotificationClient
		err         error
	)

	sendService, ok := self.SendServices.Get(service)
	if !ok {
		return fmt.Errorf("no such service %s", service)
	}

	args := apis.UpdateConfigInput{
		Configs:  config.Config,
		DomainId: config.DomainId,
	}
	_, err = sendService.UpdateConfig(ctx, &args)
	if err != nil {
		st := status.Convert(err)
		if st.Code() != codes.NotFound {
			return errors.Error(st.Message())
		}
		_, err = sendService.AddConfig(ctx, &apis.AddConfigInput{
			Configs:  config.Config,
			DomainId: config.DomainId,
		})
		if err != nil {
			return errors.Wrap(err, "try to add config but failed")
		}
	}
	return nil
}

func (self *SRpcService) ContactByMobile(ctx context.Context, mobile, serviceName string, domainId string) (string, error) {

	iMobile := api.ParseInternationalMobile(mobile)
	// compatible
	if iMobile.AreaCode == "86" {
		mobile = iMobile.Mobile
	}
	args := apis.UseridByMobileParams{
		Mobile:   mobile,
		DomainId: domainId,
	}

	f := func(service *apis.SendNotificationClient) (interface{}, error) {
		return service.UseridByMobile(ctx, &args)
	}

	ret, err := self.execute(ctx, f, serviceName)
	if err == nil {
		reply := ret.(*apis.UseridByMobileReply)
		return reply.Userid, nil
	}
	s, ok := status.FromError(err)
	if !ok {
		return "", err
	}
	if s.Code() == codes.NotFound {
		return "", errors.Wrap(notifyv2.ErrNoSuchMobile, s.Message())
	}
	if s.Code() == codes.FailedPrecondition {
		return "", errors.Wrap(notifyv2.ErrIncompleteConfig, s.Message())
	}
	return "", err
}

// Wrap function to execute function call rpc server
func (self *SRpcService) execute(ctx context.Context, f func(client *apis.SendNotificationClient) (interface{}, error),
	serviceName string) (interface{}, error) {

	sendService, ok := self.SendServices.Get(serviceName)

	log.Debugf("get service %s", serviceName)
	var err error
	if !ok {
		log.Debugf("get service first time failed")
		sendService, err = self.startNewService(ctx, serviceName, true)

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
		return nil, err
	}
	return ret, nil
}

var ErrGetConfig = errors.Error("Get Config Failed")

func (self *SRpcService) completeConfig(ctx context.Context, serviceName string, sendService *apis.SendNotificationClient) error {
	// get config
	configs, err := self.configStore.GetConfigs(serviceName)
	if err != nil {
		log.Errorf("getConfig of serveice %s from database error", serviceName)
		return ErrGetConfig
	}

	// update config for service
	configInput := make([]*apis.AddConfigInput, len(configs))
	for i := range configInput {
		configInput[i] = &apis.AddConfigInput{
			Configs:  configs[i].Config,
			DomainId: configs[i].DomainId,
		}
	}
	_, err = sendService.CompleteConfig(ctx, &apis.CompleteConfigInput{
		ConfigInput: configInput,
	})
	if err != nil {
		st := status.Convert(err)
		if st.Code() == codes.FailedPrecondition {
			// no such rpc serve
			err = fmt.Errorf(st.Message())
		}
		if st.Code() == codes.Unavailable {
			err = fmt.Errorf("service is unavailable for now: %s", st.Message())
		}
		return errors.Wrap(err, "UpdateConfig")
	}
	return nil
}

func (self *SRpcService) restartService(ctx context.Context, service string) (*apis.SendNotificationClient, error) {
	sendService, ok := self.SendServices.Get(service)
	if !ok {
		return nil, fmt.Errorf("no such service, please start new service")
	}
	return sendService, self.completeConfig(ctx, service, sendService)
}

// startNewService try to start a new rpc service named serviceName
// passConfig means if pass config to send service
func (self *SRpcService) startNewService(ctx context.Context, serviceName string, passConfig bool) (*apis.SendNotificationClient, error) {

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

	if !passConfig {
		return sendService, nil
	}

	return sendService, self.completeConfig(ctx, serviceName, sendService)
}

// closeService will remove service record from self.SendServices and try to remove sock file
func (self *SRpcService) closeService(ctx context.Context, serviceName string) {
	filename := filepath.Join(self.socketFileDir, serviceName+".sock")
	self.SendServices.Remove(serviceName)
	os.Remove(filename)
}

func (self *SRpcService) updateService(ctx context.Context) error {
	files, err := os.ReadDir(self.socketFileDir)
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
			self.startNewService(ctx, serviceName, true)
		}
	}

	serviceNames = serviceNames[:0]
	for serviceName := range serviceNameSet {
		serviceNames = append(serviceNames, serviceName)
	}

	self.SendServices.BatchRemove(serviceNames)
	return nil
}

func (self *SRpcService) AddConfig(ctx context.Context, service string, config notifyv2.SConfig) error {
	var (
		sendService *apis.SendNotificationClient
		err         error
	)

	sendService, ok := self.SendServices.Get(service)
	if !ok {
		return fmt.Errorf("no such service %s", service)
	}
	args := apis.AddConfigInput{
		DomainId: config.DomainId,
		Configs:  config.Config,
	}
	_, err = sendService.AddConfig(ctx, &args)
	if err != nil {
		return err
	}
	return nil
}

func (self *SRpcService) DeleteConfig(ctx context.Context, service, domainId string) error {
	var (
		sendService *apis.SendNotificationClient
		err         error
	)

	sendService, ok := self.SendServices.Get(service)
	if !ok {
		return fmt.Errorf("no such service %s", service)
	}
	args := apis.DeleteConfigInput{
		DomainId: domainId,
	}
	_, err = sendService.DeleteConfig(ctx, &args)
	if err != nil {
		return err
	}
	return nil
}

func (self *SRpcService) ValidateConfig(ctx context.Context, cType string, configs map[string]string) (isValid bool, message string, err error) {

	sendService, ok := self.SendServices.Get(cType)

	log.Debugf("get service %s", cType)
	if !ok {
		log.Debugf("get service first time failed")
		sendService, err = self.startNewService(ctx, cType, false)

		if err != nil {
			err = errors.Wrap(err, "start new service failed")
			return
		}
	}
	param := apis.ValidateConfigInput{
		Configs: configs,
	}
	rep, err := sendService.ValidateConfig(ctx, &param)
	if err != nil {
		st := status.Convert(err)
		if st.Code() == codes.Unimplemented {
			err = errors.ErrNotImplemented
			return
		}
		err = fmt.Errorf(st.Message())
		return
	}
	return rep.IsValid, rep.Msg, nil
}

func robotType2ContactType(rType string) string {
	switch rType {
	case api.ROBOT_TYPE_FEISHU:
		return api.FEISHU_ROBOT
	case api.ROBOT_TYPE_DINGTALK:
		return api.DINGTALK_ROBOT
	case api.ROBOT_TYPE_WORKWX:
		return api.WORKWX_ROBOT
	case api.ROBOT_TYPE_WEBHOOK:
		return api.WEBHOOK
	}
	return rType
}

func (self *SRpcService) SendRobotMessage(ctx context.Context, rType string, receivers []*apis.SReceiver, title string, message string) ([]*apis.FailedRecord, error) {
	log.Infof("rType: %s", rType)
	contactType := robotType2ContactType(rType)
	args := apis.BatchSendParams{
		Receivers: receivers,
		Title:     title,
		Message:   message,
	}
	f := func(service *apis.SendNotificationClient) (interface{}, error) {
		return service.BatchSend(ctx, &args)
	}

	ret, err := self.execute(ctx, f, contactType)
	if err != nil {
		s, ok := status.FromError(err)
		if !ok {
			return nil, err
		}
		return nil, errors.Error(s.Message())
	}
	reply := ret.(*apis.BatchSendReply)
	return reply.FailedRecords, nil
}

func grpcDialWithUnixSocket(ctx context.Context, socketPath string) (*grpc.ClientConn, error) {
	return grpc.DialContext(ctx, socketPath, grpc.WithInsecure(), grpc.WithTimeout(time.Second*5), grpc.WithDialer(
		func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)
}
