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

package panicutils

import (
	"context"
	"runtime/debug"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	npk "yunion.io/x/onecloud/pkg/mcclient/modules/notify"
)

const PANIC_ERROR = "PANIC_ERROR"

func SendPanicMessage(ctx context.Context, err error) {
	iNotes := jsonutils.Marshal(map[string]string{"stack": string(debug.Stack()), "error": err.Error()})
	notifyclient.SystemNotifyWithCtx(ctx, npk.NotifyPriorityCritical, PANIC_ERROR, iNotes)
}
