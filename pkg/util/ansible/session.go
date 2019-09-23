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

package ansible

import (
	"context"
)

type Runnable interface {
	Run(context.Context) error
}

// Session is a container for execution of playbook
type Session struct {
	// Ctx is the context under which the playbook will run
	Ctx context.Context
	// Runnable is the task to be run
	Runnable Runnable

	// cancelFunc can be called to cancel the running playbook
	cancelFunc context.CancelFunc
}

// SessionManager manages a collection of keyed sessions
type SessionManager map[string]*Session

// Has returns true if a session with the specified id exists in the manager
func (sm SessionManager) Has(id string) bool {
	_, ok := sm[id]
	return ok
}

// Add adds a Runnable to the manager keyed with the specified id
func (sm SessionManager) Add(id string, runnable Runnable) *Session {
	ctx, cancelFunc := context.WithCancel(context.Background())
	session := &Session{
		Runnable:   runnable,
		Ctx:        ctx,
		cancelFunc: cancelFunc,
	}
	sm[id] = session
	return session
}

// Remove stops (if appliable) and removes the playbook keyed with the
// specified id from the manager
func (sm SessionManager) Remove(id string) {
	sm.Stop(id)
	delete(sm, id)
}

// Run runs the playbook keyed with specified id
func (sm SessionManager) Run(id string) error {
	s, ok := sm[id]
	if !ok {
		return nil
	}
	return s.Runnable.Run(s.Ctx)
}

// Stop stops the playbook keyed with specified id
func (sm SessionManager) Stop(id string) {
	s, ok := sm[id]
	if !ok {
		return
	}
	s.cancelFunc()
}

// Err returns possible error from sm.Ctx.Err()
func (sm SessionManager) Err(id string) error {
	s, ok := sm[id]
	if !ok {
		return nil
	}
	return s.Ctx.Err()
}
