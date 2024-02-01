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

package db

import (
	"context"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient"
)

var (
	updateNotifyHook        updateNotifyHookFunc
	statusChangedNotifyHook statusChangedNotifyHookFunc
	customizeNotifyHook     customizeNotifyHookFunc
)

type updateNotifyHookFunc func(ctx context.Context, userCred mcclient.TokenCredential, obj IModel)
type statusChangedNotifyHookFunc func(ctx context.Context, userCred mcclient.TokenCredential, oldStatus, status string, obj IModel)
type customizeNotifyHookFunc func(ctx context.Context, userCred mcclient.TokenCredential, action string, obj IModel, moreDetails jsonutils.JSONObject)

func SetUpdateNotifyHook(f updateNotifyHookFunc) {
	if updateNotifyHook != nil {
		panic("updateNotifyHook already set")
	}
	updateNotifyHook = f
}

func CallUpdateNotifyHook(ctx context.Context, userCred mcclient.TokenCredential, obj IModel) {
	if updateNotifyHook == nil {
		return
	}
	updateNotifyHook(ctx, userCred, obj)
}

func SetCustomizeNotifyHook(f customizeNotifyHookFunc) {
	if customizeNotifyHook != nil {
		panic("updateNotifyHook already set")
	}
	customizeNotifyHook = f
}

func CallCustomizeNotifyHook(ctx context.Context, userCred mcclient.TokenCredential, action string, obj IModel, customizeDetails jsonutils.JSONObject) {
	if customizeNotifyHook == nil {
		return
	}
	customizeNotifyHook(ctx, userCred, action, obj, customizeDetails)
}

func SetStatusChangedNotifyHook(f statusChangedNotifyHookFunc) {
	if statusChangedNotifyHook != nil {
		panic("updateNotifyHook already set")
	}
	statusChangedNotifyHook = f
}

func CallStatusChanegdNotifyHook(ctx context.Context, userCred mcclient.TokenCredential, oldStatis, newStatus string, obj IModel) {
	if statusChangedNotifyHook == nil {
		return
	}
	statusChangedNotifyHook(ctx, userCred, oldStatis, newStatus, obj)
}
