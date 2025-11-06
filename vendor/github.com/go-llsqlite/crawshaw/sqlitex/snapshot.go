package sqlitex

import (
	"context"
	"runtime"

	"github.com/go-llsqlite/crawshaw"
)

// GetSnapshot returns a Snapshot that should remain available for reads until
// it is garbage collected.
//
// This sets aside a Conn from the Pool with an open read transaction until the
// Snapshot is garbage collected or the Pool is closed.  Thus, until the
// returned Snapshot is garbage collected, the Pool will have one fewer Conn,
// and it should not be possible for the WAL to be checkpointed beyond the
// point of the Snapshot.
//
// See sqlite.Conn.GetSnapshot and sqlite.Snapshot for more details.
func (p *Pool) GetSnapshot(ctx context.Context, schema string) (*sqlite.Snapshot, error) {
	conn := p.Get(ctx)
	if conn == nil {
		return nil, context.Canceled
	}
	conn.SetInterrupt(nil)
	s, release, err := conn.GetSnapshot(schema)
	if err != nil {
		return nil, err
	}

	snapshotGCd := make(chan struct{})
	runtime.SetFinalizer(s, nil)
	runtime.SetFinalizer(s, func(s *sqlite.Snapshot) {
		// Free the C resources associated with the Snapshot.
		s.Free()
		close(snapshotGCd)
	})

	go func() {
		select {
		case <-p.closed:
		case <-snapshotGCd:
		}
		// Allow the WAL to be checkpointed past the point of
		// the Snapshot.
		release()
		// Return the conn to the Pool for reuse.
		p.Put(conn)
	}()

	return s, nil
}
