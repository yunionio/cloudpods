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
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"yunion.io/x/log"
)

const (
	fileHourLayout   = "20060102-15"
	fileMinuteLayout = "20060102-1504"
)

type Options struct {
	Enabled               bool
	LocalDir              string
	UploadEnabled         bool
	UploadIntervalSeconds int
	SegmentMinutes        int
	MinioEndpoint         string
	MinioAccessKey        string
	MinioSecretKey        string
	MinioBucket           string
	MinioSecure           bool
	MinioPrefix           string
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
}

type ReadResult struct {
	Logs      []Record `json:"logs"`
	Truncated bool     `json:"truncated"`
}

type Writer struct {
	opts Options
	mu   sync.Mutex
}

var defaultWriter = NewWriter(Options{})

func NewWriter(opts Options) *Writer {
	if opts.LocalDir == "" {
		opts.LocalDir = "/var/log/yunion/aiproxy/chat"
	}
	if opts.UploadIntervalSeconds <= 0 {
		opts.UploadIntervalSeconds = 300
	}
	if opts.SegmentMinutes <= 0 || opts.SegmentMinutes > 60 {
		opts.SegmentMinutes = 60
	}
	if opts.Instance == "" {
		opts.Instance, _ = os.Hostname()
	}
	return &Writer{opts: opts}
}

func Configure(opts Options) {
	defaultWriter = NewWriter(opts)
}

func Write(rec *Record) {
	defaultWriter.Write(rec)
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

func (w *Writer) Write(rec *Record) {
	if w == nil || rec == nil || !w.opts.Enabled {
		return
	}
	if rec.Timestamp.IsZero() {
		rec.Timestamp = time.Now()
	}
	data, err := json.Marshal(rec)
	if err != nil {
		log.Errorf("marshal aiproxy chat log: %v", err)
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := os.MkdirAll(w.opts.LocalDir, 0750); err != nil {
		log.Errorf("mkdir aiproxy chat log dir: %v", err)
		return
	}
	path := filepath.Join(w.opts.LocalDir, logFileName(rec.Timestamp, w.opts.SegmentMinutes))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		log.Errorf("open aiproxy chat log: %v", err)
		return
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		log.Errorf("write aiproxy chat log: %v", err)
	}
}

func FillUsageFromJSON(rec *Record, data []byte) bool {
	var wrap struct {
		Usage *struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if rec == nil || json.Unmarshal(data, &wrap) != nil || wrap.Usage == nil {
		if rec != nil {
			rec.UsageMissing = true
		}
		return false
	}
	rec.PromptTokens = wrap.Usage.PromptTokens
	rec.CompletionTokens = wrap.Usage.CompletionTokens
	rec.TotalTokens = wrap.Usage.TotalTokens
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
	if w.opts.MinioEndpoint == "" || w.opts.MinioBucket == "" || w.opts.MinioAccessKey == "" || w.opts.MinioSecretKey == "" {
		return nil, errors.New("missing MinIO/S3 config")
	}
	endpoint := strings.TrimSpace(w.opts.MinioEndpoint)
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		if w.opts.MinioSecure {
			endpoint = "https://" + endpoint
		} else {
			endpoint = "http://" + endpoint
		}
	}
	return s3.NewFromConfig(aws.Config{
		Region:       "us-east-1",
		Credentials:  credentials.NewStaticCredentialsProvider(w.opts.MinioAccessKey, w.opts.MinioSecretKey, ""),
		BaseEndpoint: aws.String(endpoint),
	}, func(o *s3.Options) {
		o.UsePathStyle = true
	}), nil
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

func markUploaded(dir, name string) error {
	return os.WriteFile(filepath.Join(dir, filepath.Base(name)+".uploaded"), []byte(time.Now().Format(time.RFC3339)), 0600)
}

func finishUploaded(path string) error {
	if err := os.Remove(path); err != nil {
		return markUploaded(filepath.Dir(path), filepath.Base(path))
	}
	return nil
}

func uploaded(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, filepath.Base(name)+".uploaded"))
	return err == nil
}

func fileSegmentStart(name string) (time.Time, error) {
	return fileSegmentStartInLocation(name, time.Local)
}

func fileSegmentStartInLocation(name string, loc *time.Location) (time.Time, error) {
	part := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(name), "chat-"), ".jsonl")
	if len(part) == len("20060102-1504") {
		return time.ParseInLocation(fileMinuteLayout, part, loc)
	}
	return time.ParseInLocation(fileHourLayout, part, loc)
}

func closedSegmentFiles(dir string, now time.Time, segmentMinutes int) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	current := segmentStart(now, segmentMinutes)
	files := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, "chat-") || !strings.HasSuffix(name, ".jsonl") || uploaded(dir, name) {
			continue
		}
		start, err := fileSegmentStartInLocation(name, now.Location())
		if err != nil || !start.Before(current) {
			continue
		}
		files = append(files, filepath.Join(dir, name))
	}
	sort.Strings(files)
	return files, nil
}

func StartUploader(ctx context.Context) {
	if defaultWriter == nil || !defaultWriter.opts.Enabled || !defaultWriter.opts.UploadEnabled {
		return
	}
	go defaultWriter.uploadLoop(ctx)
}

func (w *Writer) uploadLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(w.opts.UploadIntervalSeconds) * time.Second)
	defer ticker.Stop()
	w.uploadClosedSegments(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.uploadClosedSegments(ctx)
		}
	}
}

func (w *Writer) uploadClosedSegments(ctx context.Context) {
	files, err := closedSegmentFiles(w.opts.LocalDir, time.Now(), w.opts.SegmentMinutes)
	if err != nil {
		log.Errorf("list aiproxy chat logs for upload: %v", err)
		return
	}
	for _, path := range files {
		if err := w.uploadFile(ctx, path); err != nil {
			log.Errorf("upload aiproxy chat log %s: %v", path, err)
			continue
		}
		if err := finishUploaded(path); err != nil {
			log.Errorf("finish uploaded aiproxy chat log %s: %v", path, err)
		}
	}
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
	for _, prefix := range hourPrefixes(w.opts.MinioPrefix, opts.Start, opts.End) {
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
		Bucket: aws.String(w.opts.MinioBucket),
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
		Bucket: aws.String(w.opts.MinioBucket),
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
		if len(ret.Logs) >= opts.Limit {
			ret.Truncated = true
			return nil
		}
		ret.Logs = append(ret.Logs, rec)
	}
	return scanner.Err()
}

func (w *Writer) uploadFile(ctx context.Context, path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	ts, err := fileSegmentStart(filepath.Base(path))
	if err != nil {
		return err
	}
	client, err := w.s3Client()
	if err != nil {
		return err
	}
	if err := ensureBucket(ctx, client, w.opts.MinioBucket); err != nil {
		return err
	}
	key := UploadKey(w.opts.MinioPrefix, ts, path, w.opts.Instance)
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(w.opts.MinioBucket),
		Key:    aws.String(key),
		Body:   f,
	})
	return err
}

func ensureBucket(ctx context.Context, client *s3.Client, bucket string) error {
	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
	if err == nil {
		return nil
	}
	_, err = client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})
	return err
}
