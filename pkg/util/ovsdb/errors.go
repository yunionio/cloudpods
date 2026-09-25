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

package ovsdb

import (
	"yunion.io/x/pkg/errors"
)

const (
	// ErrAddress is for addresses that cannot be made sense of
	ErrAddress = errors.Error("bad ovsdb address")
	// ErrProtocol is for messages that do not conform to JSON-RPC 1.0 or RFC7047
	ErrProtocol = errors.Error("ovsdb protocol error")
	// ErrClosed is returned when the connection is already gone
	ErrClosed = errors.Error("ovsdb connection closed")
	// ErrRemote carries the error object sent back by the remote
	ErrRemote = errors.Error("ovsdb remote error")
	// ErrOperation is for the per-operation errors of RFC7047 5.2
	ErrOperation = errors.Error("ovsdb operation error")
	// ErrValue is for values that do not fit the column they are destined for
	ErrValue = errors.Error("bad ovsdb value")
	// ErrSchema is for schema lookups that come up empty
	ErrSchema = errors.Error("ovsdb schema error")
	// ErrMonitor is for monitor bookkeeping failures
	ErrMonitor = errors.Error("ovsdb monitor error")
)
