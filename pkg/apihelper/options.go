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
	"time"

	common_options "yunion.io/x/onecloud/pkg/cloudcommon/options"
)

// SyncStatus reports the outcome of one sync attempt.
type SyncStatus struct {
	// At is when the attempt started
	At time.Time
	// Elapsed is how long the attempt took
	Elapsed time.Duration
	// Changed tells whether the model sets changed in this attempt
	Changed bool
	// Correct tells whether the model sets are complete and consistent
	Correct bool
	// Err is the error of the attempt, nil on success
	Err error
}

// SyncDoneFunc is called after every sync attempt, successful or not.  It is
// how a service that does not consume the ModelSets() channel learns about the
// result of a sync.
type SyncDoneFunc func(status SyncStatus)

type Options struct {
	common_options.CommonOptions

	SyncIntervalSeconds     int
	RunDelayMilliseconds    int
	ListBatchSize           int
	IncludeDetails          bool
	IncludeOtherCloudEnv    bool
	FetchFromComputeService bool

	// OnSyncDone, when not nil, is called after every sync attempt with the
	// outcome of that attempt.
	OnSyncDone SyncDoneFunc
}
