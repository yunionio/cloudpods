package ansible

import (
	"context"
)

// Session is a container for execution of playbook
type Session struct {
	// Ctx is the context under which the playbook will run
	Ctx context.Context
	// Playbook is the ansible playbook to be run
	Playbook *Playbook

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

// Add adds a Playbook to the manager keyed with the specified id
func (sm SessionManager) Add(id string, pb *Playbook) *Session {
	ctx, cancelFunc := context.WithCancel(context.Background())
	session := &Session{
		Playbook:   pb,
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
	return s.Playbook.Run(s.Ctx)
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
