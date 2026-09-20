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
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
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

	// apiMapTS is the version of the model sets that modelSets reflects, as
	// served by the apimap service; hasAPIMapTS tells whether that version
	// is known at all.  Both are only touched from the sync loop.
	apiMapTS    int64
	hasAPIMapTS bool
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
	app.AddHandler("POST", path, auth.Authenticate(RequireSystemAdmin(h.handlerSync)))
}

func (h *APIHelper) handlerSync(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	h.scheduleSync()
}

// RequireSystemAdmin wraps a handler so that only a caller with system scope
// privilege reaches it.
func RequireSystemAdmin(h func(context.Context, http.ResponseWriter, *http.Request)) func(context.Context, http.ResponseWriter, *http.Request) {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
		if userCred == nil || !userCred.HasSystemAdminPrivilege() {
			httperrors.ForbiddenError(ctx, w, "only system admin is allowed to access this resource")
			return
		}
		h(ctx, w, r)
	}
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
	status := h.doSync(ctx)
	if status.Err != nil {
		log.Errorf("doSync error: %v", status.Err)
	}
	if status.Changed {
		mssCopy := h.modelSets.CopyJoined()
		select {
		case h.modelSetsCh <- mssCopy:
		case <-ctx.Done():
		}
	}
}

func (h *APIHelper) doSync(ctx context.Context) (status SyncStatus) {
	status.At = time.Now()
	defer func() {
		status.Elapsed = time.Since(status.At)
		log.Infof("sync data done, changed: %v, elapsed: %s, err: %v", status.Changed, status.Elapsed.String(), status.Err)
		if h.opts.OnSyncDone != nil {
			h.opts.OnSyncDone(status)
		}
	}()

	s := h.adminClientSession(ctx)
	var (
		mss IModelSets
		r   ModelSetsUpdateResult
		err error
	)
	if h.opts.FetchFromComputeService {
		mss = h.modelSets.Copy()
		r, err = SyncModelSets(mss, s, h.opts)
		if err != nil {
			status.Err = errors.Wrap(err, "SyncModelSets")
			return
		}
	} else {
		mss, r, err = h.syncModelSetsFromAPIMap(s)
		if err != nil {
			status.Err = err
			return
		}
	}
	h.modelSets = mss
	if !r.Correct {
		// 发送消息通知
		err := sendSyncErrNotify(s)
		if err != nil {
			log.Errorf("unable to EventNotify: %s", err)
		}
		status.Err = errors.Errorf("sync error")
		return
	}
	status.Changed = r.Changed
	status.Correct = true
	return
}

// syncModelSetsFromAPIMap refreshes the model sets from the apimap service.
//
// The apimap service publishes a version with the model sets, so a round that
// finds the version unchanged is skipped without copying anything.  The
// version returned along with the payload is the authoritative one: the model
// sets may have moved on between the version query and the fetch.
func (h *APIHelper) syncModelSetsFromAPIMap(s *mcclient.ClientSession) (mss IModelSets, r ModelSetsUpdateResult, err error) {
	apims, ok := h.modelSets.(IAPIMapModelSets)
	if !ok {
		return nil, r, errors.Errorf("model sets %T cannot be fetched from apimap", h.modelSets)
	}
	curTS, err := apims.APIMapTimestamp(s)
	if err != nil {
		return nil, r, errors.Wrap(err, "APIMapTimestamp")
	}
	if h.hasAPIMapTS && curTS == h.apiMapTS {
		// nothing new since the last round
		return h.modelSets, ModelSetsUpdateResult{Correct: true, Changed: false}, nil
	}
	mssNews, newTS, err := apims.FetchFromAPIMap(s)
	if err != nil {
		return nil, r, errors.Wrap(err, "FetchFromAPIMap")
	}
	mss = h.modelSets.Copy()
	r = mss.ApplyUpdates(mssNews)
	h.apiMapTS = newTS
	h.hasAPIMapTS = true
	return mss, r, nil
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
