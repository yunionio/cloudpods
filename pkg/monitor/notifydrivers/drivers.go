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

package notifydrivers

import (
	"context"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/monitor"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const (
	ErrUnsupportedNotificationType = errors.Error("Unsupported notification type")
)

// Notifier is responsible for sending alert notifications.
type Notifier interface {
	GetType() string

	GetNotifierId() string
	// GetIsDefault() bool
	GetSendReminder() bool
	GetDisableResolveMessage() bool
	GetFrequency() time.Duration
}

type NotificationConfig struct {
	Ctx                   context.Context
	Id                    string
	Name                  string
	Type                  string
	SendReminder          bool
	DisableResolveMessage bool
	Frequency             time.Duration
	Settings              jsonutils.JSONObject
}

type NotifierFactory func(notification NotificationConfig) (Notifier, error)

var notifierFactories = make(map[string]*NotifierPlugin)

type NotifierPlugin struct {
	Type               string
	Factory            NotifierFactory
	ValidateCreateData func(cred mcclient.IIdentityProvider, input monitor.NotificationCreateInput) (monitor.NotificationCreateInput, error)
}

func RegisterNotifier(plugin *NotifierPlugin) {
	notifierFactories[plugin.Type] = plugin
}

func GetNotifiers() []*NotifierPlugin {
	list := make([]*NotifierPlugin, 0)

	for _, value := range notifierFactories {
		list = append(list, value)
	}

	return list
}

func GetPlugin(typ string) (*NotifierPlugin, error) {
	plugin, found := notifierFactories[typ]
	if !found {
		return nil, errors.Wrapf(ErrUnsupportedNotificationType, "type %s", typ)
	}
	return plugin, nil
}

// InitNotifier instantiate a new notifier based on the model
func InitNotifier(config NotificationConfig) (Notifier, error) {
	plugin, err := GetPlugin(config.Type)
	if err != nil {
		return nil, err
	}
	return plugin.Factory(config)
}
