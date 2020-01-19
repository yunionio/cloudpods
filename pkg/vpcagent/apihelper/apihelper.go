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
	"sync"
	"time"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/vpcagent/options"
)

const (
	ErrSync = errors.Error("sync error")
)

type APIHelper struct {
	opts        *options.Options
	modelSets   IModelSets
	modelSetsCh chan IModelSets

	mcclientSession *mcclient.ClientSession
}

func NewAPIHelper(opts *options.Options, modelSets IModelSets) (*APIHelper, error) {
	modelSetsCh := make(chan IModelSets)
	helper := &APIHelper{
		opts:        opts,
		modelSets:   modelSets,
		modelSetsCh: modelSetsCh,
	}
	return helper, nil
}

func (h *APIHelper) Start(ctx context.Context) {
	defer func() {
		log.Infoln("apihelper: bye")
		wg := ctx.Value("wg").(*sync.WaitGroup)
		wg.Done()
	}()

	h.run(ctx)

	tickDuration := time.Duration(h.opts.APISyncInterval) * time.Second
	tick := time.NewTimer(tickDuration)
	defer tick.Stop()

	for {
		select {
		case <-tick.C:
			h.run(ctx)
			tick.Reset(tickDuration)
		case <-ctx.Done():
			return
		}
	}
}

func (h *APIHelper) ModelSets() <-chan IModelSets {
	return h.modelSetsCh
}

func (h *APIHelper) run(ctx context.Context) {
	changed, err := h.doSync(ctx)
	if err != nil {
		log.Errorln(err)
	}
	if changed {
		mssCopy := h.modelSets.Copy()
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
	r, err := SyncModelSets(h.modelSets, s, h.opts.APIListBatchSize)
	if err != nil {
		return false, err
	}
	if !r.Correct {
		return false, errors.Wrap(ErrSync, "incorrect")
	}
	changed = r.Changed
	return changed, nil
}

func (h *APIHelper) adminClientSession(ctx context.Context) *mcclient.ClientSession {
	s := h.mcclientSession
	if s != nil {
		token := s.GetToken()
		expires := token.GetExpires()
		if time.Now().Add(time.Hour).After(expires) {
			return s
		}
	}

	region := h.opts.CommonOptions.Region
	apiVersion := "v2"
	h.mcclientSession = auth.GetAdminSession(ctx, region, apiVersion)
	return h.mcclientSession
}
