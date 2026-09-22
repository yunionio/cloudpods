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

package apimap

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	APIMap APIMapManager
)

func init() {
	APIMap = APIMapManager{
		ResourceManager: modules.NewAPIMapManager("", "", nil, nil),
	}
}

type APIMapManager struct {
	modulebase.ResourceManager
}

// GetModelSets returns the model sets served by the apimap service, along with
// the version they correspond to.
func (m APIMapManager) GetModelSets(s *mcclient.ClientSession) (jsonutils.JSONObject, int64, error) {
	_, ret, err := modulebase.JsonRequest(m.ResourceManager, s, "GET", "/modelsets", nil, nil)
	if err != nil {
		return nil, 0, err
	}
	ts, err := ret.Int("timestamp")
	if err != nil {
		return nil, 0, errors.Wrap(err, "timestamp")
	}
	return ret, ts, nil
}

// GetModelSetsTimestamp returns the version of the model sets currently served
// by the apimap service, without fetching the model sets themselves.
func (m APIMapManager) GetModelSetsTimestamp(s *mcclient.ClientSession) (int64, error) {
	_, ret, err := modulebase.JsonRequest(m.ResourceManager, s, "GET", "/modelsets/timestamp", nil, nil)
	if err != nil {
		return 0, err
	}
	return ret.Int("timestamp")
}

// GetModelSetsStats returns the sync status and the model set sizes of the
// apimap service.
func (m APIMapManager) GetModelSetsStats(s *mcclient.ClientSession) (jsonutils.JSONObject, error) {
	_, ret, err := modulebase.JsonRequest(m.ResourceManager, s, "GET", "/modelsets/stats", nil, nil)
	return ret, err
}

// SyncModelSets asks the apimap service to sync its model sets.  The sync runs
// asynchronously on the apimap side: the call returns as soon as the sync has
// been scheduled, which is why a caller that needs the new data to be there
// uses TriggerSyncAndWait instead.
func (m APIMapManager) SyncModelSets(s *mcclient.ClientSession) error {
	_, _, err := modulebase.JsonRequest(m.ResourceManager, s, "POST", "/modelsets/sync", nil, nil)
	return err
}

const (
	// syncPollInterval is how often the model sets version is polled while
	// waiting for a sync to complete
	syncPollInterval = time.Second
	// syncWaitTimeout is how long to wait for the model sets to change
	syncWaitTimeout = 10 * time.Second
)

// SyncOutcome is what TriggerSyncAndWait made of the apimap service.
type SyncOutcome int

const (
	// SyncOutcomeUnavailable means the apimap service could not be reached at
	// all, typically because it is not deployed.  A caller then carries on
	// the way it did before apimap existed.
	SyncOutcomeUnavailable SyncOutcome = iota
	// SyncOutcomeChanged means apimap published a new version of the model
	// sets: the agents that consume them have something to fetch.
	SyncOutcomeChanged
	// SyncOutcomeUnchanged means apimap is there but published nothing new
	// within the timeout.  "The data did not change" and "the sync is still
	// running" look the same from here, so this is a normal outcome rather
	// than an error.
	SyncOutcomeUnchanged
)

// ShouldTriggerAgent reports whether the caller should go on and trigger the
// agent that consumes the model sets (vpcagent, and later lbagent and
// sdnagent).  There is nothing to fetch when apimap is there and published
// nothing new, and the caller stops there.
func (o SyncOutcome) ShouldTriggerAgent() bool {
	return o != SyncOutcomeUnchanged
}

// TriggerSyncAndWait triggers a model set sync of the apimap service and waits
// until the model sets change, so that a caller that goes on to trigger a
// downstream agent knows that the new data is there to be fetched.
//
// It is best effort on purpose, because the apimap service is optional and a
// caller must never fail a task such as starting a server because of it: an
// endpoint that does not resolve, a service that is not ready yet or a failed
// call are logged and reported as SyncOutcomeUnavailable, which leaves the
// caller doing what it did before apimap existed.
func TriggerSyncAndWait(s *mcclient.ClientSession) SyncOutcome {
	t0, err := APIMap.GetModelSetsTimestamp(s)
	if err != nil {
		log.Warningf("apimap: cannot read model sets timestamp: %v", err)
		return SyncOutcomeUnavailable
	}
	err = APIMap.SyncModelSets(s)
	if err != nil {
		log.Warningf("apimap: cannot trigger model sets sync: %v", err)
		return SyncOutcomeUnavailable
	}
	deadline := time.Now().Add(syncWaitTimeout)
	for time.Now().Before(deadline) {
		time.Sleep(syncPollInterval)
		ts, err := APIMap.GetModelSetsTimestamp(s)
		if err != nil {
			log.Warningf("apimap: cannot read model sets timestamp: %v", err)
			return SyncOutcomeUnavailable
		}
		if ts != t0 {
			return SyncOutcomeChanged
		}
	}
	log.Infof("apimap: model sets did not change within %s", syncWaitTimeout)
	return SyncOutcomeUnchanged
}
