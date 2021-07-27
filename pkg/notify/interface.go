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

package notify

import (
	"context"

	"yunion.io/x/pkg/errors"

	notify_apis "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/notify/rpc/apis"
)

type INotifyService interface {
	InitAll() error
	StopAll()
	UpdateServices(ctx context.Context, userCred mcclient.TokenCredential, isStart bool)
	UpdateConfig(ctx context.Context, service string, config SConfig) error
	Send(ctx context.Context, contactType string, args apis.SendParams) error
	ContactByMobile(ctx context.Context, mobile, serviceName, domainId string) (string, error)
	BatchSend(ctx context.Context, contactType string, args apis.BatchSendParams) ([]*apis.FailedRecord, error)
	SendRobotMessage(ctx context.Context, rType string, receivers []*apis.SReceiver, title string, message string) ([]*apis.FailedRecord, error)
	AddConfig(ctx context.Context, service string, config SConfig) error
	DeleteConfig(ctx context.Context, service, domainId string) error
	ValidateConfig(ctx context.Context, cType string, configs map[string]string) (isValid bool, message string, err error)
}

type SSendParams struct {
	ContactType string
	Contact     string
	Topic       string
	Message     string
	Priority    string
	Lang        string
}

type SBatchSendParams struct {
	ContactType string
	Contacts    []string
	Topic       string
	Message     string
	Priority    string
	Lang        string
}

type IServiceConfigStore interface {
	GetConfigs(service string) ([]SConfig, error)
	GetConfig(service, domainId string) (SConfig, error)
	SetConfig(service string, config SConfig) error
	HasSystemConfig(service string) (bool, error)
	BatchCheckConfig(service string, domainIds []string) ([]bool, error)
}

type SNotification struct {
	ContactType string
	Topic       string
	Message     string
	Event       notify_apis.SNotifyEvent
	AdvanceDays int
}

type ITemplateStore interface {
	// NotifyFilter(contactType, topic, msg, lang string) (params apis.SendParams, err error)
	FillWithTemplate(ctx context.Context, lang string, notification SNotification) (params apis.SendParams, err error)
}

type SConfig struct {
	Config   map[string]string
	DomainId string
}

var (
	ErrNoSuchMobile     = errors.Error("no such mobile")
	ErrIncompleteConfig = errors.Error("incomplete config")
)
