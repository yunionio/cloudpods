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

package apihelper

import (
	"context"
	"net/http"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/onecloud/pkg/apis/notify"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/consts"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	npk "yunion.io/x/onecloud/pkg/mcclient/modules/notify"
)

const (
	MinSyncIntervalSeconds  = 10
	MinRunDelayMilliseconds = 100
)

type APIHelper struct {
	opts        *Options
	modelSets   IModelSets
	modelSetsCh chan IModelSets

	mcclientSession *mcclient.ClientSession

	tick *time.Timer
}

func NewAPIHelper(opts *Options, modelSets IModelSets) (*APIHelper, error) {
	modelSetsCh := make(chan IModelSets)
	helper := &APIHelper{
		opts:        opts,
		modelSets:   modelSets,
		modelSetsCh: modelSetsCh,
	}
	return helper, nil
}

func (h *APIHelper) getSyncInterval() time.Duration {
	intv := h.opts.SyncIntervalSeconds
	if intv < MinSyncIntervalSeconds {
		intv = MinSyncIntervalSeconds
	}
	return time.Duration(intv) * time.Second
}

func (h *APIHelper) getRunDelay() time.Duration {
	delay := h.opts.RunDelayMilliseconds
	if delay < MinRunDelayMilliseconds {
		delay = MinRunDelayMilliseconds
	}
	return time.Duration(delay) * time.Millisecond
}

func (h *APIHelper) addSyncHandler(app *appsrv.Application, prefix string) {
	path := httputils.JoinPath(prefix, "sync")
	app.AddHandler("POST", path, h.handlerSync)
}

func (h *APIHelper) handlerSync(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	h.scheduleSync()
}

func (h *APIHelper) Start(ctx context.Context, app *appsrv.Application, prefix string) {
	defer func() {
		log.Infoln("apihelper: bye")
		wg := ctx.Value("wg").(*sync.WaitGroup)
		wg.Done()
	}()

	if app != nil {
		h.addSyncHandler(app, prefix)
	}

	h.run(ctx)

	tickDuration := h.getSyncInterval()
	h.tick = time.NewTimer(tickDuration)
	defer func() {
		tick := h.tick
		h.tick = nil
		tick.Stop()
	}()

	for {
		select {
		case <-h.tick.C:
			h.run(ctx)
			h.tick.Reset(tickDuration)
		case <-ctx.Done():
			return
		}
	}
}

func (h *APIHelper) scheduleSync() {
	if h.tick != nil {
		if !h.tick.Stop() {
			<-h.tick.C
		}
		h.tick.Reset(h.getRunDelay())
	}
}

func (h *APIHelper) ModelSets() <-chan IModelSets {
	return h.modelSetsCh
}

func (h *APIHelper) RunManually(ctx context.Context) {
	h.run(ctx)
}

func (h *APIHelper) run(ctx context.Context) {
	changed, err := h.doSync(ctx)
	if err != nil {
		log.Errorf("doSync error: %v", err)
	}
	if changed {
		mssCopy := h.modelSets.CopyJoined()
		select {
		case h.modelSetsCh <- mssCopy:
		case <-ctx.Done():
		}
	}
}

func (h *APIHelper) doSync(ctx context.Context) (changed bool, err error) {
	{
		stime := time.Now()
		defer func() {
			elapsed := time.Since(stime)
			log.Infof("sync data done, changed: %v, elapsed: %s", changed, elapsed.String())
		}()
	}

	s := h.adminClientSession(ctx)
	mss := h.modelSets.Copy()
	r, err := SyncModelSets(mss, s, h.opts)
	if err != nil {
		return false, errors.Wrap(err, "SyncModelSets")
	}
	h.modelSets = mss
	if !r.Correct {
		// 发送消息通知
		err := sendSyncErrNotify(s)
		if err != nil {
			log.Errorf("unable to EventNotify: %s", err)
		}
		return false, errors.Errorf("sync error")
	}
	changed = r.Changed
	return changed, nil
}

func (h *APIHelper) adminClientSession(ctx context.Context) *mcclient.ClientSession {
	s := h.mcclientSession
	if s != nil {
		token := s.GetToken()
		expires := token.GetExpires()
		if time.Now().Add(time.Hour).Before(expires) {
			return s
		}
	}

	region := h.opts.CommonOptions.Region
	h.mcclientSession = auth.GetAdminSession(ctx, region)
	return h.mcclientSession
}

func sendSyncErrNotify(s *mcclient.ClientSession) error {
	params := api.NotificationManagerEventNotifyInput{}
	params.Event = api.Event.WithAction(api.ActionNetOutOfSync).WithResourceType(api.TOPIC_RESOURCE_NET).String()
	params.AdvanceDays = 0
	message := &jsonutils.JSONDict{}
	message.Add(jsonutils.NewString(consts.GetServiceType()), "service_name")
	params.ResourceDetails = message
	_, err := npk.Notification.PerformClassAction(s, "event-notify", jsonutils.Marshal(params))
	return err
}
