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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"

	"yunion.io/x/log"
)

const (
	fileHourLayout   = "20060102-15"
	fileMinuteLayout = "20060102-1504"
)

var errObjectNotFound = errors.New("object not found")

type Options struct {
	Enabled               bool
	LocalDir              string
	UploadEnabled         bool
	UploadIntervalSeconds int
	SegmentMinutes        int
	S3Endpoint            string
	S3AccessKey           string
	S3SecretKey           string
	S3Bucket              string
	S3Secure              bool
	S3Prefix              string
	Instance              string
}

type Record struct {
	RequestID      string      `json:"request_id,omitempty"`
	Timestamp      time.Time   `json:"timestamp"`
	Path           string      `json:"path,omitempty"`
	Stream         bool        `json:"stream"`
	Client         string      `json:"client,omitempty"`
	Metadata       interface{} `json:"metadata,omitempty"`
	VirtualKey     string      `json:"virtual_key,omitempty"`
	ProjectID      string      `json:"project_id,omitempty"`
	DomainID       string      `json:"domain_id,omitempty"`
	AiKey          string      `json:"ai_key,omitempty"`
	ModelRequested string      `json:"model_requested,omitempty"`
	ModelFinal     string      `json:"model_final,omitempty"`
	Provider       string      `json:"provider,omitempty"`
	AiProviderId   string      `json:"provider_id,omitempty"`
	Success        bool        `json:"success"`
	StatusCode     int         `json:"status_code,omitempty"`
	ErrorCode      string      `json:"error_code,omitempty"`
	ErrorMessage   string      `json:"error_message,omitempty"`
	LatencyMs      int64       `json:"latency_ms,omitempty"`

	PromptTokens     int  `json:"prompt_tokens,omitempty"`
	CompletionTokens int  `json:"completion_tokens,omitempty"`
	TotalTokens      int  `json:"total_tokens,omitempty"`
	UsageMissing     bool `json:"usage_missing,omitempty"`

	RoutingEnabled       bool                   `json:"routing_enabled"`
	RoutingCandidates    []string               `json:"routing_candidates,omitempty"`
	RoutingSelectedModel string                 `json:"routing_selected_model,omitempty"`
	RoutingMethod        string                 `json:"routing_method,omitempty"`
	RoutingScores        map[string]interface{} `json:"routing_scores,omitempty"`
	RoutingConfidence    *float64               `json:"routing_confidence,omitempty"`
	RoutingReason        string                 `json:"routing_reason,omitempty"`
	RoutingLatencyMs     int64                  `json:"routing_latency_ms,omitempty"`
	RoutingError         string                 `json:"routing_error,omitempty"`

	ToolCallEnabled      bool            `json:"tool_call_enabled"`
	ToolCallCount        int             `json:"tool_call_count,omitempty"`
	ToolCallSuccessCount int             `json:"tool_call_success_count,omitempty"`
	ToolCallErrorCount   int             `json:"tool_call_error_count,omitempty"`
	ToolCalls            json.RawMessage `json:"tool_calls,omitempty"`
}

type ReadOptions struct {
	Start     time.Time
	End       time.Time
	Limit     int
	RequestID string
	Instance  string
	Filter    func(Record) bool
}

type ReadResult struct {
	Logs      []Record `json:"logs"`
	Truncated bool     `json:"truncated"`
}

type objectStore interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Put(ctx context.Context, key string, body []byte) error
}

type flushItem struct {
	key  string
	body []byte
}

type Writer struct {
	opts  Options
	store objectStore
	now   func() time.Time

	mu       sync.Mutex
	buf      []byte
	key      string
	dirty    bool
	loaded   bool
	flushing bool
	pending  []flushItem
	flushErr error
}

var defaultWriter = NewWriter(Options{})

func NewWriter(opts Options) *Writer {
	if opts.SegmentMinutes <= 0 || opts.SegmentMinutes > 60 {
		opts.SegmentMinutes = 60
	}
	if opts.Instance == "" {
		opts.Instance, _ = os.Hostname()
	}
	return &Writer{opts: opts, now: time.Now}
}

func Configure(opts Options) {
	defaultWriter = NewWriter(opts)
}

func Write(rec *Record) {
	defaultWriter.Write(rec)
}

func Flush() error {
	return defaultWriter.Flush(context.Background())
}

func segmentStart(ts time.Time, minutes int) time.Time {
	if minutes <= 0 || minutes > 60 {
		minutes = 60
	}
	return ts.Truncate(time.Duration(minutes) * time.Minute)
}

func logFileName(ts time.Time, minutes int) string {
	start := segmentStart(ts, minutes)
	if minutes >= 60 {
		return "chat-" + start.Format(fileHourLayout) + ".jsonl"
	}
	return "chat-" + start.Format(fileMinuteLayout) + ".jsonl"
}

func (w *Writer) objectKey(ts time.Time) string {
	start := segmentStart(ts, w.opts.SegmentMinutes)
	return UploadKey(w.opts.S3Prefix, start, logFileName(ts, w.opts.SegmentMinutes), w.opts.Instance)
}

func (w *Writer) Write(rec *Record) {
	if w == nil || rec == nil || !w.opts.Enabled {
		return
	}
	if rec.Timestamp.IsZero() {
		rec.Timestamp = w.now()
	}
	data, err := json.Marshal(rec)
	if err != nil {
		log.Errorf("marshal aiproxy chat log: %v", err)
		return
	}
	line := append(data, '\n')
	w.mu.Lock()
	defer w.mu.Unlock()
	w.rotateLocked(rec.Timestamp)
	w.buf = append(w.buf, line...)
	w.dirty = true
	w.kickFlushLocked()
}

func (w *Writer) rotateLocked(ts time.Time) {
	key := w.objectKey(ts)
	if w.key == key {
		w.ensureLoadedLocked()
		return
	}
	if w.key != "" && w.dirty {
		w.pending = append(w.pending, flushItem{key: w.key, body: append([]byte(nil), w.buf...)})
		w.dirty = false
	}
	w.key = key
	w.buf = nil
	w.dirty = false
	w.loaded = false
	w.ensureLoadedLocked()
}

func (w *Writer) ensureLoadedLocked() {
	if w.loaded || w.key == "" || !w.opts.UploadEnabled {
		return
	}
	store, err := w.getStoreLocked()
	if err != nil {
		log.Errorf("init aiproxy chat log store: %v", err)
		w.loaded = true
		return
	}
	body, err := store.Get(context.Background(), w.key)
	if err != nil {
		if !isObjectNotFound(err) {
			log.Errorf("load aiproxy chat log %s: %v", w.key, err)
		}
		w.loaded = true
		return
	}
	w.buf = append([]byte(nil), body...)
	w.loaded = true
}

func (w *Writer) getStoreLocked() (objectStore, error) {
	if w.store != nil {
		return w.store, nil
	}
	client, err := w.s3Client()
	if err != nil {
		return nil, err
	}
	w.store = &s3ObjectStore{
		client: client,
		bucket: w.opts.S3Bucket,
	}
	return w.store, nil
}

func (w *Writer) kickFlushLocked() {
	if !w.opts.UploadEnabled || w.flushing {
		return
	}
	w.flushing = true
	go w.flushLoop()
}

func (w *Writer) flushLoop() {
	for {
		w.mu.Lock()
		jobs := w.takeFlushJobsLocked()
		if len(jobs) == 0 {
			w.flushing = false
			w.mu.Unlock()
			return
		}
		store, err := w.getStoreLocked()
		w.mu.Unlock()
		if err != nil {
			log.Errorf("init aiproxy chat log store: %v", err)
			w.mu.Lock()
			w.requeueFlushJobsLocked(jobs)
			w.flushing = false
			w.flushErr = err
			w.mu.Unlock()
			return
		}
		for _, job := range jobs {
			if err := store.Put(context.Background(), job.key, job.body); err != nil {
				log.Errorf("put aiproxy chat log %s: %v", job.key, err)
				w.mu.Lock()
				w.requeueFlushJobsLocked([]flushItem{job})
				w.flushing = false
				w.flushErr = err
				w.mu.Unlock()
				return
			}
		}
		w.mu.Lock()
		w.flushErr = nil
		w.mu.Unlock()
	}
}

func (w *Writer) takeFlushJobsLocked() []flushItem {
	jobs := append([]flushItem(nil), w.pending...)
	w.pending = w.pending[:0]
	if w.dirty && w.key != "" {
		jobs = append(jobs, flushItem{key: w.key, body: append([]byte(nil), w.buf...)})
		w.dirty = false
	}
	return jobs
}

func (w *Writer) requeueFlushJobsLocked(jobs []flushItem) {
	for _, job := range jobs {
		if job.key == w.key {
			if !bytes.Equal(w.buf, job.body) {
				w.dirty = true
				continue
			}
			w.dirty = true
			continue
		}
		w.pending = append(w.pending, job)
	}
}

func (w *Writer) Flush(ctx context.Context) error {
	if w == nil || !w.opts.Enabled {
		return nil
	}
	if !w.opts.UploadEnabled {
		return nil
	}
	w.mu.Lock()
	w.kickFlushLocked()
	w.mu.Unlock()
	for {
		w.mu.Lock()
		done := !w.flushing && !w.dirty && len(w.pending) == 0
		stuck := !w.flushing && (w.dirty || len(w.pending) > 0)
		err := w.flushErr
		if stuck && err == nil {
			w.kickFlushLocked()
		}
		w.mu.Unlock()
		if done {
			return err
		}
		if stuck && err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func FillUsageFromJSON(rec *Record, data []byte) bool {
	var wrap struct {
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
			InputTokens      int `json:"input_tokens"`
			OutputTokens     int `json:"output_tokens"`
		} `json:"usage"`
	}
	if rec == nil || json.Unmarshal(data, &wrap) != nil || wrap.Usage == nil {
		if rec != nil {
			rec.UsageMissing = true
		}
		return false
	}
	u := wrap.Usage
	prompt := u.PromptTokens
	completion := u.CompletionTokens
	total := u.TotalTokens
	if prompt == 0 {
		prompt = u.InputTokens
	}
	if completion == 0 {
		completion = u.OutputTokens
	}
	if total == 0 {
		total = prompt + completion
	}
	if prompt == 0 && completion == 0 && total == 0 {
		rec.UsageMissing = true
		return false
	}
	rec.PromptTokens = prompt
	rec.CompletionTokens = completion
	rec.TotalTokens = total
	rec.UsageMissing = false
	return true
}

func FillToolCallsFromJSON(rec *Record, data []byte) bool {
	var wrap struct {
		Choices []struct {
			Message struct {
				ToolCalls json.RawMessage `json:"tool_calls"`
			} `json:"message"`
			Delta struct {
				ToolCalls json.RawMessage `json:"tool_calls"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if rec == nil || json.Unmarshal(data, &wrap) != nil {
		return false
	}
	var calls []json.RawMessage
	for _, c := range wrap.Choices {
		for _, raw := range []json.RawMessage{c.Message.ToolCalls, c.Delta.ToolCalls} {
			if len(raw) == 0 || string(raw) == "null" {
				continue
			}
			var arr []json.RawMessage
			if json.Unmarshal(raw, &arr) == nil {
				calls = append(calls, arr...)
			}
		}
	}
	if len(calls) == 0 {
		return false
	}
	rec.ToolCallCount += len(calls)
	rec.ToolCallSuccessCount += len(calls)
	rec.ToolCalls, _ = json.Marshal(calls)
	return true
}

func UploadKey(prefix string, ts time.Time, filename string, instance string) string {
	prefix = strings.Trim(strings.TrimSpace(prefix), "/")
	base := strings.TrimSuffix(filepath.Base(filename), ".jsonl")
	if instance != "" {
		base += "-" + strings.NewReplacer("/", "_", "\\", "_").Replace(instance)
	}
	key := "date=" + ts.Format("2006-01-02") + "/hour=" + ts.Format("15") + "/" + base + ".jsonl"
	if prefix == "" {
		return key
	}
	return prefix + "/" + key
}

func (w *Writer) s3Client() (*s3.Client, error) {
	if w.opts.S3Endpoint == "" || w.opts.S3Bucket == "" || w.opts.S3AccessKey == "" || w.opts.S3SecretKey == "" {
		return nil, errors.New("missing S3 config")
	}
	endpoint := strings.TrimSpace(w.opts.S3Endpoint)
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		if w.opts.S3Secure {
			endpoint = "https://" + endpoint
		} else {
			endpoint = "http://" + endpoint
		}
	}
	return s3.NewFromConfig(aws.Config{
		Region:       "us-east-1",
		Credentials:  credentials.NewStaticCredentialsProvider(w.opts.S3AccessKey, w.opts.S3SecretKey, ""),
		BaseEndpoint: aws.String(endpoint),
	}, func(o *s3.Options) {
		o.UsePathStyle = true
	}), nil
}

type s3ObjectStore struct {
	client      *s3.Client
	bucket      string
	ensureOnce  sync.Once
	ensureError error
}

func (s *s3ObjectStore) Get(ctx context.Context, key string) ([]byte, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		if isObjectNotFound(err) {
			return nil, errObjectNotFound
		}
		return nil, err
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

func (s *s3ObjectStore) Put(ctx context.Context, key string, body []byte) error {
	s.ensureOnce.Do(func() {
		s.ensureError = ensureBucket(ctx, s.client, s.bucket)
	})
	if s.ensureError != nil {
		return s.ensureError
	}
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(body),
	})
	return err
}

func isObjectNotFound(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errObjectNotFound) {
		return true
	}
	var nsk *types.NoSuchKey
	if errors.As(err, &nsk) {
		return true
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NoSuchKey", "NotFound", "NoSuchBucket":
			return true
		}
	}
	return false
}

func hourObjectPrefix(prefix string, ts time.Time) string {
	prefix = strings.Trim(strings.TrimSpace(prefix), "/")
	key := "date=" + ts.Format("2006-01-02") + "/hour=" + ts.Format("15") + "/"
	if prefix == "" {
		return key
	}
	return prefix + "/" + key
}

func hourPrefixes(prefix string, start, end time.Time) []string {
	start = start.Truncate(time.Hour)
	end = end.Truncate(time.Hour)
	ret := make([]string, 0)
	for ts := start; !ts.After(end); ts = ts.Add(time.Hour) {
		ret = append(ret, hourObjectPrefix(prefix, ts))
	}
	return ret
}

func instanceSuffix(instance string) string {
	if instance == "" {
		return ""
	}
	return "-" + strings.NewReplacer("/", "_", "\\", "_").Replace(instance) + ".jsonl"
}

func objectMatchesInstance(key string, instance string) bool {
	base := filepath.Base(key)
	if !strings.HasPrefix(base, "chat-") || !strings.HasSuffix(base, ".jsonl") {
		return false
	}
	suffix := instanceSuffix(instance)
	return suffix == "" || strings.HasSuffix(base, suffix)
}

func StartUploader(ctx context.Context) {
	if defaultWriter == nil || !defaultWriter.opts.Enabled || !defaultWriter.opts.UploadEnabled {
		return
	}
	defaultWriter.mu.Lock()
	defaultWriter.rotateLocked(defaultWriter.now())
	defaultWriter.mu.Unlock()
	go func() {
		<-ctx.Done()
		if err := defaultWriter.Flush(context.Background()); err != nil {
			log.Errorf("flush aiproxy chat log on stop: %v", err)
		}
	}()
}

func Read(ctx context.Context, opts ReadOptions) (*ReadResult, error) {
	return defaultWriter.Read(ctx, opts)
}

func (w *Writer) Read(ctx context.Context, opts ReadOptions) (*ReadResult, error) {
	if w == nil || !w.opts.Enabled {
		return &ReadResult{}, nil
	}
	now := time.Now()
	if opts.End.IsZero() {
		opts.End = now
	}
	if opts.Start.IsZero() {
		opts.Start = opts.End.Add(-time.Hour)
	}
	if opts.End.Before(opts.Start) {
		return nil, errors.New("end must be after start")
	}
	if opts.Limit <= 0 {
		opts.Limit = 1000
	}
	if opts.Limit > 10000 {
		opts.Limit = 10000
	}
	if opts.Instance == "" {
		opts.Instance = w.opts.Instance
	}
	client, err := w.s3Client()
	if err != nil {
		return nil, err
	}
	ret := &ReadResult{Logs: make([]Record, 0)}
	for _, prefix := range hourPrefixes(w.opts.S3Prefix, opts.Start, opts.End) {
		if err := w.readPrefix(ctx, client, prefix, opts, ret); err != nil {
			return nil, err
		}
		if ret.Truncated {
			break
		}
	}
	return ret, nil
}

func (w *Writer) readPrefix(ctx context.Context, client *s3.Client, prefix string, opts ReadOptions, ret *ReadResult) error {
	in := &s3.ListObjectsV2Input{
		Bucket: aws.String(w.opts.S3Bucket),
		Prefix: aws.String(prefix),
	}
	for {
		out, err := client.ListObjectsV2(ctx, in)
		if err != nil {
			return err
		}
		for _, obj := range out.Contents {
			key := aws.ToString(obj.Key)
			if !objectMatchesInstance(key, opts.Instance) {
				continue
			}
			if err := w.readObject(ctx, client, key, opts, ret); err != nil {
				return err
			}
			if ret.Truncated {
				return nil
			}
		}
		if !aws.ToBool(out.IsTruncated) || out.NextContinuationToken == nil {
			return nil
		}
		in.ContinuationToken = out.NextContinuationToken
	}
}

func (w *Writer) readObject(ctx context.Context, client *s3.Client, key string, opts ReadOptions, ret *ReadResult) error {
	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(w.opts.S3Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}
	defer out.Body.Close()
	return readJSONLines(out.Body, opts, ret)
}

func readJSONLines(r io.Reader, opts ReadOptions, ret *ReadResult) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		var rec Record
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}
		if rec.Timestamp.Before(opts.Start) || rec.Timestamp.After(opts.End) {
			continue
		}
		if opts.RequestID != "" && rec.RequestID != opts.RequestID {
			continue
		}
		if opts.Filter != nil && !opts.Filter(rec) {
			continue
		}
		ret.Logs = append(ret.Logs, rec)
		if opts.Limit > 0 && len(ret.Logs) >= opts.Limit {
			ret.Truncated = true
			return nil
		}
	}
	return scanner.Err()
}

func ensureBucket(ctx context.Context, client *s3.Client, bucket string) error {
	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
	if err == nil {
		return nil
	}
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	return err
}
