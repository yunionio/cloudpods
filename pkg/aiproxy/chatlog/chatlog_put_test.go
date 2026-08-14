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

package chatlog

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

type memStore struct {
	mu sync.Mutex
	m  map[string][]byte
}

func newMemStore() *memStore {
	return &memStore{m: map[string][]byte{}}
}

func (s *memStore) Get(_ context.Context, key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	body, ok := s.m[key]
	if !ok {
		return nil, errObjectNotFound
	}
	return append([]byte(nil), body...), nil
}

func (s *memStore) Put(_ context.Context, key string, body []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[key] = append([]byte(nil), body...)
	return nil
}

func newTestWriter(store objectStore) *Writer {
	w := NewWriter(Options{
		Enabled:        true,
		UploadEnabled:  true,
		Instance:       "node-1",
		SegmentMinutes: 60,
	})
	w.store = store
	return w
}

func TestWriterSameHourOverwriteJSONL(t *testing.T) {
	store := newMemStore()
	w := newTestWriter(store)
	ts := time.Date(2026, 8, 13, 10, 15, 0, 0, time.UTC)
	w.Write(&Record{RequestID: "a", Timestamp: ts})
	w.Write(&Record{RequestID: "b", Timestamp: ts.Add(time.Minute)})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := w.Flush(ctx); err != nil {
		t.Fatalf("flush: %v", err)
	}
	body, err := store.Get(ctx, w.objectKey(ts))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines: %s", len(lines), body)
	}
	if !strings.Contains(lines[0], `"request_id":"a"`) || !strings.Contains(lines[1], `"request_id":"b"`) {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestWriterSplitsObjectByHour(t *testing.T) {
	store := newMemStore()
	w := newTestWriter(store)
	h1 := time.Date(2026, 8, 13, 10, 50, 0, 0, time.UTC)
	h2 := time.Date(2026, 8, 13, 11, 5, 0, 0, time.UTC)
	w.Write(&Record{RequestID: "h1", Timestamp: h1})
	w.Write(&Record{RequestID: "h2", Timestamp: h2})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := w.Flush(ctx); err != nil {
		t.Fatalf("flush: %v", err)
	}
	b1, err := store.Get(ctx, w.objectKey(h1))
	if err != nil {
		t.Fatalf("get h1: %v", err)
	}
	b2, err := store.Get(ctx, w.objectKey(h2))
	if err != nil {
		t.Fatalf("get h2: %v", err)
	}
	if w.objectKey(h1) == w.objectKey(h2) {
		t.Fatal("expected different hour keys")
	}
	if bytes.Contains(b1, []byte(`"request_id":"h2"`)) || bytes.Contains(b2, []byte(`"request_id":"h1"`)) {
		t.Fatalf("hours mixed: h1=%s h2=%s", b1, b2)
	}
	if !bytes.Contains(b1, []byte(`"request_id":"h1"`)) || !bytes.Contains(b2, []byte(`"request_id":"h2"`)) {
		t.Fatalf("missing records: h1=%s h2=%s", b1, b2)
	}
}

func TestWriterReloadAfterRestart(t *testing.T) {
	store := newMemStore()
	ts := time.Date(2026, 8, 13, 10, 20, 0, 0, time.UTC)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	w1 := newTestWriter(store)
	w1.Write(&Record{RequestID: "old", Timestamp: ts})
	if err := w1.Flush(ctx); err != nil {
		t.Fatalf("flush w1: %v", err)
	}

	w2 := newTestWriter(store)
	w2.Write(&Record{RequestID: "new", Timestamp: ts.Add(time.Minute)})
	if err := w2.Flush(ctx); err != nil {
		t.Fatalf("flush w2: %v", err)
	}
	body, err := store.Get(ctx, w2.objectKey(ts))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !bytes.Contains(body, []byte(`"request_id":"old"`)) || !bytes.Contains(body, []byte(`"request_id":"new"`)) {
		t.Fatalf("restart lost lines: %s", body)
	}
	if bytes.Count(body, []byte("\n")) != 2 {
		t.Fatalf("expected 2 lines, got %q", body)
	}
}

func TestWriterFlushUploadsPending(t *testing.T) {
	store := newMemStore()
	w := newTestWriter(store)
	ts := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	w.Write(&Record{RequestID: "x", Timestamp: ts})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := w.Flush(ctx); err != nil {
		t.Fatalf("flush: %v", err)
	}
	body, err := store.Get(ctx, w.objectKey(ts))
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !bytes.Contains(body, []byte(`"request_id":"x"`)) {
		t.Fatalf("flush did not upload: %s", body)
	}
	w.mu.Lock()
	dirty, pending := w.dirty, len(w.pending)
	w.mu.Unlock()
	if dirty || pending != 0 {
		t.Fatalf("buffer still dirty after flush: dirty=%v pending=%d", dirty, pending)
	}
}
