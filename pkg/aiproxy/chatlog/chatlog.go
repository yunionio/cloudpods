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
	"context"
	"encoding/json"
	"errors"
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

const fileTimeLayout = "20060102-15"

type Options struct {
	Enabled               bool
	LocalDir              string
	UploadEnabled         bool
	UploadIntervalSeconds int
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
	path := filepath.Join(w.opts.LocalDir, "chat-"+rec.Timestamp.Format(fileTimeLayout)+".jsonl")
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

func markUploaded(dir, name string) error {
	return os.WriteFile(filepath.Join(dir, filepath.Base(name)+".uploaded"), []byte(time.Now().Format(time.RFC3339)), 0600)
}

func uploaded(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, filepath.Base(name)+".uploaded"))
	return err == nil
}

func closedHourFiles(dir string, now time.Time) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	current := now.Format(fileTimeLayout)
	files := make([]string, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, "chat-") || !strings.HasSuffix(name, ".jsonl") || uploaded(dir, name) {
			continue
		}
		hour := strings.TrimSuffix(strings.TrimPrefix(name, "chat-"), ".jsonl")
		if hour == current {
			continue
		}
		if _, err := time.Parse(fileTimeLayout, hour); err != nil {
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
	w.uploadClosedHours(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.uploadClosedHours(ctx)
		}
	}
}

func (w *Writer) uploadClosedHours(ctx context.Context) {
	files, err := closedHourFiles(w.opts.LocalDir, time.Now())
	if err != nil {
		log.Errorf("list aiproxy chat logs for upload: %v", err)
		return
	}
	for _, path := range files {
		if err := w.uploadFile(ctx, path); err != nil {
			log.Errorf("upload aiproxy chat log %s: %v", path, err)
			continue
		}
		if err := markUploaded(w.opts.LocalDir, filepath.Base(path)); err != nil {
			log.Errorf("mark aiproxy chat log uploaded %s: %v", path, err)
		}
	}
}

func (w *Writer) uploadFile(ctx context.Context, path string) error {
	if w.opts.MinioEndpoint == "" || w.opts.MinioBucket == "" || w.opts.MinioAccessKey == "" || w.opts.MinioSecretKey == "" {
		return errors.New("missing MinIO/S3 upload config")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	ts, err := fileHour(filepath.Base(path))
	if err != nil {
		return err
	}
	endpoint := strings.TrimSpace(w.opts.MinioEndpoint)
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		if w.opts.MinioSecure {
			endpoint = "https://" + endpoint
		} else {
			endpoint = "http://" + endpoint
		}
	}
	client := s3.NewFromConfig(aws.Config{
		Region:       "us-east-1",
		Credentials:  credentials.NewStaticCredentialsProvider(w.opts.MinioAccessKey, w.opts.MinioSecretKey, ""),
		BaseEndpoint: aws.String(endpoint),
	}, func(o *s3.Options) {
		o.UsePathStyle = true
	})
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

func fileHour(name string) (time.Time, error) {
	hour := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(name), "chat-"), ".jsonl")
	return time.Parse(fileTimeLayout, hour)
}
