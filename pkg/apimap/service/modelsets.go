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

package service

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/apihelper"
	"yunion.io/x/onecloud/pkg/apimap/options"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	agentmodels "yunion.io/x/onecloud/pkg/vpcagent/models"
)

// ModelSetsService keeps the model sets of the region in memory and serves
// them to the agents (vpcagent, and later lbagent and sdnagent), so that the
// agents do not have to rebuild them from the compute APIs each on their own.
//
// Two sets are kept apart:
//
//   - the staging set, held by the api helper, which is updated from the
//     region API on every sync;
//   - the published set, served to the agents, which is only replaced once a
//     sync came back complete and consistent.
//
// The published set comes with a timestamp, a version that is bumped whenever
// the published set changes.  Agents poll the version and only fetch the model
// sets when it moved, so an agent never has to guess whether what it holds is
// still current.
type ModelSetsService struct {
	apih *apihelper.APIHelper

	mu sync.RWMutex

	// startupTS is the version the published set starts at
	startupTS int64

	ready      bool
	timestamp  int64
	published  *agentmodels.ModelSets
	serialized []byte

	syncCount     int64
	lastSyncAt    time.Time
	lastElapsed   time.Duration
	lastChanged   bool
	lastCorrect   bool
	lastErr       string
	lastPublishAt time.Time
}

func NewModelSetsService(opts *options.SOptions) (*ModelSetsService, error) {
	svc := &ModelSetsService{
		startupTS: time.Now().Unix(),
	}
	apiOpts := &apihelper.Options{
		CommonOptions:        opts.CommonOptions,
		SyncIntervalSeconds:  opts.APISyncIntervalSeconds,
		RunDelayMilliseconds: opts.APIRunDelayMilliseconds,
		ListBatchSize:        opts.APIListBatchSize,

		// apimap only ever pulls from the region API
		FetchFromComputeService: true,
		IncludeDetails:          false,
		IncludeOtherCloudEnv:    false,
		OnSyncDone:              svc.onSyncDone,
	}
	apih, err := apihelper.NewAPIHelper(apiOpts, agentmodels.NewModelSets())
	if err != nil {
		return nil, errors.Wrap(err, "new api helper")
	}
	svc.apih = apih
	return svc, nil
}

// modelsetsPrefix is the path prefix of the model sets API.  The sync endpoint
// is registered under it by the api helper, which is why the very same prefix
// is handed to APIHelper.Start.
const modelsetsPrefix = "modelsets"

func (s *ModelSetsService) InitHandlers(app *appsrv.Application) {
	app.AddHandler2("GET", httputils.JoinPath("", modelsetsPrefix),
		auth.Authenticate(apihelper.RequireSystemAdmin(s.handleModelSets)), nil, "get_modelsets", nil)
	app.AddHandler2("GET", httputils.JoinPath(modelsetsPrefix, "timestamp"),
		auth.Authenticate(apihelper.RequireSystemAdmin(s.handleModelSetsTimestamp)), nil, "get_modelsets_timestamp", nil)
	app.AddHandler2("GET", httputils.JoinPath(modelsetsPrefix, "stats"),
		auth.Authenticate(apihelper.RequireSystemAdmin(s.handleModelSetsStats)), nil, "get_modelsets_stats", nil)
}

func (s *ModelSetsService) Start(ctx context.Context, app *appsrv.Application) {
	wg := ctx.Value("wg").(*sync.WaitGroup)
	defer func() {
		log.Infoln("apimap: model sets service bye")
		wg.Done()
	}()

	wg.Add(1)
	go s.apih.Start(ctx, app, modelsetsPrefix)

	for {
		select {
		case imss := <-s.apih.ModelSets():
			mss, ok := imss.(*agentmodels.ModelSets)
			if !ok {
				log.Errorf("apimap: unexpected model sets type %T", imss)
				continue
			}
			s.publish(mss)
		case <-ctx.Done():
			return
		}
	}
}

// onSyncDone is called by the api helper after every sync attempt.
func (s *ModelSetsService) onSyncDone(status apihelper.SyncStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.syncCount++
	s.lastSyncAt = status.At
	s.lastElapsed = status.Elapsed
	s.lastChanged = status.Changed
	s.lastCorrect = status.Correct
	if status.Err != nil {
		s.lastErr = status.Err.Error()
	} else {
		s.lastErr = ""
	}

	if status.Err != nil || !status.Correct {
		// the sync failed, or came back incomplete: keep serving what the
		// agents already have, and keep the version as it is, so that they
		// do not pick up a half baked set
		log.Warningf("apimap: keep the published model sets, sync error: %v", status.Err)
		return
	}
	if s.ready {
		return
	}
	// at least one sync came back complete, so what we have now is worth
	// serving: an empty platform is a valid answer, and the agents need one
	// to tell "nothing there" from "not ready yet"
	log.Infof("apimap: model sets ready")
	s.ready = true
	s.timestamp = s.startupTS
	// nothing is handed out yet: an empty platform gets an empty payload when
	// it is first asked for, while the model sets that come with the first
	// successful sync become the initial version rather than a change to it
}

// publish replaces the published model sets.  It is called for every change
// the api helper reports, which is what the version counts.
func (s *ModelSetsService) publish(mss *agentmodels.ModelSets) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.ready {
		// the api helper only reports a change once a sync came back
		// complete, so this is only here to keep the two in step
		s.ready = true
		s.timestamp = s.startupTS
	}
	if s.published != nil {
		// a version has already been handed out, so these model sets are a
		// change to it
		s.timestamp++
	}
	s.published = mss
	s.lastPublishAt = time.Now()
	s.buildPayloadLocked()
	log.Infof("apimap: model sets published, timestamp: %d, stats: %s", s.timestamp, mss.Stats().Dump())
}

// buildPayloadLocked serializes the published model sets, so that a request
// only has to compress and send, rather than marshal the whole set again.
func (s *ModelSetsService) buildPayloadLocked() {
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewInt(s.timestamp), "timestamp")
	body.Add(jsonutils.Marshal(s.published), "models")
	s.serialized = []byte(body.String())
}

// materializeLocked puts something in place to be handed out when the service
// is ready but no sync has published anything yet: the platform is then empty,
// and the agents have to be able to tell that from "not ready".  Handing that
// empty payload out is also what pins the initial version to it, so that the
// model sets arriving later count as a change.
func (s *ModelSetsService) materializeLocked() {
	if s.serialized == nil {
		if s.published == nil {
			s.published = agentmodels.NewModelSets()
		}
		s.buildPayloadLocked()
	}
}

// Payload returns the serialized published model sets and their version.  The
// second return value is false when the service is not ready yet.
func (s *ModelSetsService) Payload() ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ready {
		return nil, false
	}
	s.materializeLocked()
	return s.serialized, true
}

// Timestamp returns the version of the published model sets.  The second
// return value is false when the service is not ready yet.
func (s *ModelSetsService) Timestamp() (int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.ready {
		return 0, false
	}
	s.materializeLocked()
	return s.timestamp, true
}

// ModelSetsStatus is what the stats endpoint reports.
type ModelSetsStatus struct {
	Ready           bool                        `json:"ready"`
	Timestamp       int64                       `json:"timestamp"`
	SyncCount       int64                       `json:"sync_count"`
	LastSyncAt      time.Time                   `json:"last_sync_at"`
	LastSyncElapsed string                      `json:"last_sync_elapsed"`
	LastSyncChanged bool                        `json:"last_sync_changed"`
	LastSyncCorrect bool                        `json:"last_sync_correct"`
	LastSyncError   string                      `json:"last_sync_error"`
	LastPublishAt   time.Time                   `json:"last_publish_at"`
	Counts          *agentmodels.ModelSetsStats `json:"counts"`
}

func (s *ModelSetsService) Status() *ModelSetsStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	st := &ModelSetsStatus{
		Ready:           s.ready,
		Timestamp:       s.timestamp,
		SyncCount:       s.syncCount,
		LastSyncAt:      s.lastSyncAt,
		LastSyncElapsed: s.lastElapsed.String(),
		LastSyncChanged: s.lastChanged,
		LastSyncCorrect: s.lastCorrect,
		LastSyncError:   s.lastErr,
		LastPublishAt:   s.lastPublishAt,
	}
	if s.published != nil {
		stats := s.published.Stats()
		st.Counts = &stats
	}
	return st
}

func (s *ModelSetsService) handleModelSets(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	payload, ok := s.Payload()
	if !ok {
		sendNotReady(ctx, w)
		return
	}
	sendPayload(w, r, payload)
}

func (s *ModelSetsService) handleModelSetsTimestamp(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	ts, ok := s.Timestamp()
	if !ok {
		sendNotReady(ctx, w)
		return
	}
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewInt(ts), "timestamp")
	appsrv.SendJSON(w, body)
}

func (s *ModelSetsService) handleModelSetsStats(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	appsrv.SendJSON(w, jsonutils.Marshal(s.Status()))
}

func sendNotReady(ctx context.Context, w http.ResponseWriter) {
	httperrors.JsonClientError(ctx, w, httputils.NewJsonClientError(
		http.StatusServiceUnavailable, "ServiceUnavailable", "model sets are not ready yet"))
}

// sendPayload writes the model sets out, compressing them when the client asks
// for it.  The payload is cached uncompressed, since compressing it again per
// request is much cheaper than marshalling it again.
func sendPayload(w http.ResponseWriter, r *http.Request, payload []byte) {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	w.Header().Set("Vary", "Accept-Encoding")
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		if compressed, err := compressPayload(payload); err != nil {
			log.Errorf("apimap: compress model sets: %v", err)
		} else {
			w.Header().Set("Content-Encoding", "gzip")
			payload = compressed
		}
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
	w.Write(payload)
}

func compressPayload(payload []byte) ([]byte, error) {
	buf := &bytes.Buffer{}
	zw := gzip.NewWriter(buf)
	// the payload is built once per version, so it is worth the extra bytes
	// to make sure everything is written before the stream is flushed
	if _, err := zw.Write(payload); err != nil {
		zw.Close()
		return nil, errors.Wrap(err, "write")
	}
	if err := zw.Close(); err != nil {
		return nil, errors.Wrap(err, "close")
	}
	return buf.Bytes(), nil
}
