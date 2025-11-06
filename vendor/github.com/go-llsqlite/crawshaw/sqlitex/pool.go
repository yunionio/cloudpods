// Copyright (c) 2018 David Crawshaw <david@zentus.com>
//
// Permission to use, copy, modify, and distribute this software for any
// purpose with or without fee is hereby granted, provided that the above
// copyright notice and this permission notice appear in all copies.
//
// THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
// WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
// ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
// WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
// ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
// OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.

package sqlitex

import (
	"context"
	"fmt"
	"runtime/trace"
	"sync"
	"time"

	"github.com/go-llsqlite/crawshaw"
)

// Pool is a pool of SQLite connections.
//
// It is safe for use by multiple goroutines concurrently.
//
// Typically, a goroutine that needs to use an SQLite *Conn
// Gets it from the pool and defers its return:
//
//	conn := dbpool.Get(nil)
//	defer dbpool.Put(conn)
//
// As Get may block, a context can be used to return if a task
// is cancelled. In this case the Conn returned will be nil:
//
//	conn := dbpool.Get(ctx)
//	if conn == nil {
//		return context.Canceled
//	}
//	defer dbpool.Put(conn)
type Pool struct {
	// If checkReset, the Put method checks all of the connection's
	// prepared statements and ensures they were correctly cleaned up.
	// If they were not, Put will panic with details.
	//
	// TODO: export this? Is it enough of a performance concern?
	checkReset bool

	free   chan *sqlite.Conn
	closed chan struct{}

	all map[*sqlite.Conn]context.CancelFunc

	mu sync.RWMutex
}

// Open opens a fixed-size pool of SQLite connections.
//
// A flags value of 0 defaults to:
//
//	SQLITE_OPEN_READWRITE
//	SQLITE_OPEN_CREATE
//	SQLITE_OPEN_WAL
//	SQLITE_OPEN_URI
//	SQLITE_OPEN_NOMUTEX
func Open(uri string, flags sqlite.OpenFlags, poolSize int) (pool *Pool, err error) {
	return OpenInit(nil, uri, flags, poolSize, "")
}

// OpenInit opens a fixed-size pool of SQLite connections, each initialized
// with initScript.
//
// A flags value of 0 defaults to:
//
//	SQLITE_OPEN_READWRITE
//	SQLITE_OPEN_CREATE
//	SQLITE_OPEN_WAL
//	SQLITE_OPEN_URI
//	SQLITE_OPEN_NOMUTEX
//
// Each initScript is run an all Conns in the Pool. This is intended for PRAGMA
// or CREATE TEMP VIEW which need to be run on all connections.
//
// WARNING: Ensure all queries in initScript are completely idempotent, meaning
// that running it multiple times is the same as running it once. For example
// do not run INSERT in any of the initScripts or else it may create duplicate
// data unintentionally or fail.
func OpenInit(ctx context.Context, uri string, flags sqlite.OpenFlags, poolSize int, initScript string) (pool *Pool, err error) {
	if uri == ":memory:" {
		return nil, strerror{msg: `sqlite: ":memory:" does not work with multiple connections, use "file::memory:?mode=memory"`}
	}

	p := &Pool{
		checkReset: true,
		free:       make(chan *sqlite.Conn, poolSize),
		closed:     make(chan struct{}),
	}
	defer func() {
		// If an error occurred, call Close outside the lock so this doesn't deadlock.
		if err != nil {
			p.Close()
		}
	}()

	if flags == 0 {
		flags = sqlite.SQLITE_OPEN_READWRITE |
			sqlite.SQLITE_OPEN_CREATE |
			sqlite.SQLITE_OPEN_WAL |
			sqlite.SQLITE_OPEN_URI |
			sqlite.SQLITE_OPEN_NOMUTEX
	}

	// sqlitex_pool is also defined in package sqlite
	const sqlitex_pool = sqlite.OpenFlags(0x01000000)
	flags |= sqlitex_pool

	p.all = make(map[*sqlite.Conn]context.CancelFunc)
	for i := 0; i < poolSize; i++ {
		conn, err := sqlite.OpenConn(uri, flags)
		if err != nil {
			return nil, err
		}
		p.free <- conn
		p.all[conn] = func() {}

		if initScript != "" {
			conn.SetInterrupt(ctx.Done())
			if err := ExecScript(conn, initScript); err != nil {
				return nil, err
			}
			conn.SetInterrupt(nil)
		}
	}

	return p, nil
}

// Get returns an SQLite connection from the Pool.
//
// If no Conn is available, Get will block until one is, or until either the
// Pool is closed or the context expires. If no Conn can be obtained, nil is
// returned.
//
// The provided context is used to control the execution lifetime of the
// connection. See Conn.SetInterrupt for details.
//
// Applications must ensure that all non-nil Conns returned from Get are
// returned to the same Pool with Put.
func (p *Pool) Get(ctx context.Context) *sqlite.Conn {
	var tr sqlite.Tracer
	if ctx != nil {
		tr = &tracer{ctx: ctx}
	} else {
		ctx = context.Background()
	}
	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

outer:
	select {
	case conn := <-p.free:
		p.mu.Lock()
		defer p.mu.Unlock()

		select {
		case <-p.closed:
			p.free <- conn
			break outer
		default:
		}

		conn.SetTracer(tr)
		conn.SetInterrupt(ctx.Done())

		p.all[conn] = cancel

		return conn
	case <-ctx.Done():
	case <-p.closed:
	}
	cancel()
	return nil
}

// Put puts an SQLite connection back into the Pool.
//
// Put will panic if conn is nil or if the conn was not originally created by
// p.
//
// Applications must ensure that all non-nil Conns returned from Get are
// returned to the same Pool with Put.
func (p *Pool) Put(conn *sqlite.Conn) {
	if conn == nil {
		panic("attempted to Put a nil Conn into Pool")
	}
	if p.checkReset {
		query := conn.CheckReset()
		if query != "" {
			panic(fmt.Sprintf(
				"connection returned to pool has active statement: %q",
				query))
		}
	}

	p.mu.RLock()
	cancel, found := p.all[conn]
	p.mu.RUnlock()

	if !found {
		panic("sqlite.Pool.Put: connection not created by this pool")
	}

	cancel()
	p.free <- conn
}

// PoolCloseTimeout is the maximum time for Pool.Close to wait for all Conns to
// be returned to the Pool.
//
// Do not modify this concurrently with calling Pool.Close.
var PoolCloseTimeout = 5 * time.Second

// Close interrupts and closes all the connections in the Pool.
//
// Close blocks until all connections are returned to the Pool.
//
// Close will panic if not all connections are returned before
// PoolCloseTimeout.
func (p *Pool) Close() (err error) {
	close(p.closed)

	p.mu.RLock()
	for _, cancel := range p.all {
		cancel()
	}
	p.mu.RUnlock()

	timeout := time.After(PoolCloseTimeout)
	for closed := 0; closed < len(p.all); closed++ {
		select {
		case conn := <-p.free:
			err2 := conn.Close()
			if err == nil {
				err = err2
			}
		case <-timeout:
			panic("not all connections returned to Pool before timeout")
		}
	}
	return
}

type strerror struct {
	msg string
}

func (err strerror) Error() string { return err.msg }

type tracer struct {
	ctx       context.Context
	ctxStack  []context.Context
	taskStack []*trace.Task
}

func (t *tracer) pctx() context.Context {
	if len(t.ctxStack) != 0 {
		return t.ctxStack[len(t.ctxStack)-1]
	}
	return t.ctx
}

func (t *tracer) Push(name string) {
	ctx, task := trace.NewTask(t.pctx(), name)
	t.ctxStack = append(t.ctxStack, ctx)
	t.taskStack = append(t.taskStack, task)
}

func (t *tracer) Pop() {
	t.taskStack[len(t.taskStack)-1].End()
	t.taskStack = t.taskStack[:len(t.taskStack)-1]
	t.ctxStack = t.ctxStack[:len(t.ctxStack)-1]
}

func (t *tracer) NewTask(name string) sqlite.TracerTask {
	ctx, task := trace.NewTask(t.pctx(), name)
	return &tracerTask{
		ctx:  ctx,
		task: task,
	}
}

type tracerTask struct {
	ctx    context.Context
	task   *trace.Task
	region *trace.Region
}

func (t *tracerTask) StartRegion(regionType string) {
	if t.region != nil {
		panic("sqlitex.tracerTask.StartRegion: already in region")
	}
	t.region = trace.StartRegion(t.ctx, regionType)
}

func (t *tracerTask) EndRegion() {
	t.region.End()
	t.region = nil
}

func (t *tracerTask) End() {
	t.task.End()
}
