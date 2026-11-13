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

package benchmark

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGuideLLMv07OutputTokensPerSecond(t *testing.T) {
	metrics := parseMetricsCSV(t, ""+
		"Run,Server Throughput,Scheduler Metrics,Request Counts,Request Counts,Request Counts,Token Throughput\n"+
		"Run ID,Successful Requests/Sec,Request Time Avg,Total,Successful,Errored,Successful Output Tokens/Sec\n"+
		",Mean,Sec,,,,Mean\n"+
		"run-1,12.5,0.4,10,8,2,345.67\n")
	assertFloat(t, "output tokens/s", metrics.OutputTokensPerSecondMean, 345.67)
	assertFloat(t, "requests/s", metrics.RequestsPerSecondMean, 12.5)
}

func TestParseGuideLLMv07OutputTokensPerSecondMissing(t *testing.T) {
	metrics := parseMetricsCSV(t, ""+
		"Run,Server Throughput,Scheduler Metrics,Request Counts,Request Counts,Request Counts\n"+
		"Run ID,Successful Requests/Sec,Request Time Avg,Total,Successful,Errored\n"+
		",Mean,Sec,,,\n"+
		"run-1,12.5,0.4,10,8,2\n")
	if metrics.OutputTokensPerSecondMean != nil {
		t.Fatalf("output tokens/s = %v, want empty", *metrics.OutputTokensPerSecondMean)
	}
	assertFloat(t, "requests/s", metrics.RequestsPerSecondMean, 12.5)
	assertNilFloat(t, "scheduler request time is not end-to-end latency", metrics.RequestLatencyMeanSec)
	assertNilFloat(t, "input tokens/s", metrics.InputTokensPerSecondMean)
	assertNilFloat(t, "total tokens/s", metrics.TokensPerSecondMean)
	assertNilFloat(t, "ttft", metrics.TimeToFirstTokenMeanMs)
	assertNilFloat(t, "tpot", metrics.InterTokenLatencyMeanMs)
	assertNilFloat(t, "concurrency", metrics.RequestConcurrencyMean)
	if metrics.RequestIncomplete != 0 {
		t.Fatalf("incomplete = %d, want 0", metrics.RequestIncomplete)
	}
}

func TestParseGuideLLMv07BenchmarkMetrics(t *testing.T) {
	percentiles := "[0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 1.9, 2.1, 3.4]"
	ttft := "[10, 20, 30, 40, 50, 60, 70, 80, 200, 250, 300]"
	tpot := "[1, 2, 3, 4, 5, 6, 7, 8, 18, 22, 40]"
	concurrency := "[1, 1, 1, 2, 2, 3, 4, 5, 6, 7, 9]"
	metrics := parseMetricsCSV(t, ""+
		"Run,Request Latency,Request Latency,Time to First Token,Time to First Token,Inter Token Latency,Inter Token Latency,Token Throughput,Token Throughput,Token Throughput,Server Throughput,Server Throughput,Request Counts,Scheduler Metrics\n"+
		"Run ID,Successful Sec,Successful Sec,Successful ms,Successful ms,Successful ms,Successful ms,Successful Input Tokens/Sec,Successful Total Tokens/Sec,Successful Output Tokens/Sec,Successful Concurrency,Successful Concurrency,Incomplete,Request Time Avg\n"+
		",Mean,Percentiles,Mean,Percentiles,Mean,Percentiles,Mean,Mean,Mean,Mean,Percentiles,,Sec\n"+
		"run-1,1.25,\""+percentiles+"\",120,\""+ttft+"\",15,\""+tpot+"\",80.5,200.25,345.67,4.5,\""+concurrency+"\",3,9.9\n")
	assertFloat(t, "end-to-end latency", metrics.RequestLatencyMeanSec, 1.25)
	assertFloat(t, "latency p95", metrics.RequestLatencyP95Sec, 1.9)
	assertFloat(t, "latency p99", metrics.RequestLatencyP99Sec, 2.1)
	assertFloat(t, "ttft", metrics.TimeToFirstTokenMeanMs, 120)
	assertFloat(t, "ttft p95", metrics.TimeToFirstTokenP95Ms, 200)
	assertFloat(t, "ttft p99", metrics.TimeToFirstTokenP99Ms, 250)
	assertFloat(t, "tpot", metrics.InterTokenLatencyMeanMs, 15)
	assertFloat(t, "tpot p95", metrics.InterTokenLatencyP95Ms, 18)
	assertFloat(t, "tpot p99", metrics.InterTokenLatencyP99Ms, 22)
	assertFloat(t, "input tokens/s", metrics.InputTokensPerSecondMean, 80.5)
	assertFloat(t, "total tokens/s", metrics.TokensPerSecondMean, 200.25)
	assertFloat(t, "output tokens/s", metrics.OutputTokensPerSecondMean, 345.67)
	assertFloat(t, "concurrency", metrics.RequestConcurrencyMean, 4.5)
	assertFloat(t, "max concurrency", metrics.RequestConcurrencyMax, 9)
	if metrics.RequestIncomplete != 3 {
		t.Fatalf("incomplete = %d, want 3", metrics.RequestIncomplete)
	}
}

func TestParseGuideLLMPercentilesStayEmpty(t *testing.T) {
	metrics := parseMetricsCSV(t, ""+
		"Run,Request Latency,Request Latency,Server Throughput\n"+
		"Run ID,Successful Sec,Successful Sec,Successful Concurrency\n"+
		",Mean,Percentiles,Percentiles\n"+
		"run-1,1.5,\"[0.1, 0.2]\",not-a-list\n")
	assertFloat(t, "end-to-end latency", metrics.RequestLatencyMeanSec, 1.5)
	assertNilFloat(t, "latency p95", metrics.RequestLatencyP95Sec)
	assertNilFloat(t, "latency p99", metrics.RequestLatencyP99Sec)
	assertNilFloat(t, "max concurrency", metrics.RequestConcurrencyMax)
}

func TestGuideLLMBackgroundCommand(t *testing.T) {
	command := []string{"sh", "-lc", "guidellm run --note it's"}
	got, err := GuideLLMBackgroundCommand(command, "/workdir/benchmarks.json", "/workdir/benchmarks.csv")
	if err != nil {
		t.Fatalf("background command: %v", err)
	}
	if len(got) != 3 || got[0] != "sh" || got[1] != "-lc" {
		t.Fatalf("command = %#v", got)
	}
	script := got[2]
	for _, want := range []string{
		"guidellm run",
		"/workdir/benchmarks.json.exit",
		"/workdir/benchmarks.json.pid",
		"/workdir/benchmarks.csv",
		"setsid",
		"nohup",
		"rm -f",
		`'"'"'`,
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("script missing %q:\n%s", want, script)
		}
	}
	if mkdirAt, pidAt := strings.Index(script, "mkdir -p"), strings.Index(script, "echo $!"); mkdirAt < 0 || pidAt < 0 || mkdirAt > pidAt {
		t.Fatalf("workdir must be created before writing the pid file:\n%s", script)
	}

	formal, err := GuideLLMBackgroundCommand(GuideLLMRunCommand(), "/workdir/benchmarks.json", "/workdir/benchmarks.csv")
	if err != nil {
		t.Fatalf("formal command: %v", err)
	}
	if !strings.Contains(formal[2], "guidellm run --output kind=json,path=/workdir/benchmarks.json") {
		t.Fatalf("formal script = %s", formal[2])
	}
	if _, err := GuideLLMBackgroundCommand([]string{"guidellm", "run"}, "/workdir/benchmarks.json", "/workdir/benchmarks.csv"); err == nil {
		t.Fatal("expected error for a command that is not sh -lc")
	}
}

func TestBuildGuideLLMSpecKeepsFractionalRateAndWarmup(t *testing.T) {
	spec := BuildGuideLLMSpec(GuideLLMSpecInput{
		TargetURL:           "http://llm:8000",
		RequestFormat:       "/v1/chat/completions",
		RequestRate:         0.5,
		WarmupRequests:      2,
		TotalRequests:       10,
		MaxDurationSeconds:  600,
		MaxErrors:           10,
		DatasetInputTokens:  1024,
		DatasetOutputTokens: 128,
	})
	if spec.Profile["rate"] != 0.5 {
		t.Fatalf("rate = %#v", spec.Profile["rate"])
	}
	warmup, _ := spec.Profile["warmup"].(map[string]interface{})
	if warmup["value"] != 2 || warmup["mode"] != "requests" {
		t.Fatalf("warmup = %#v", spec.Profile["warmup"])
	}
	if count := constraintCount(t, spec, "max_requests"); count != 10 {
		t.Fatalf("stored max_requests = %v", count)
	}
	measured := WithoutGuideLLMWarmup(spec)
	if _, ok := measured.Profile["warmup"]; ok {
		t.Fatal("measured run still has a warmup window")
	}
	if count := constraintCount(t, measured, "max_requests"); count != 10 {
		t.Fatalf("measured max_requests = %v", count)
	}
	warmupSpec, ok := WarmupGuideLLMSpec(spec, 2)
	if !ok {
		t.Fatal("expected a warmup run")
	}
	if _, exists := warmupSpec.Profile["warmup"]; exists {
		t.Fatal("warmup run applies another warmup window")
	}
	if count := constraintCount(t, warmupSpec, "max_requests"); count != 2 {
		t.Fatalf("warmup max_requests = %v", count)
	}
	if _, ok := WarmupGuideLLMSpec(spec, 0); ok {
		t.Fatal("zero warmup requests still produced a warmup run")
	}
	preflight, err := BuildGuideLLMPreflightSpec(GuideLLMSpec{
		Data: []map[string]interface{}{{"kind": "json_file", "path": "/data/set.jsonl"}},
	}, 60)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := preflight.Profile["warmup"]; ok {
		t.Fatal("dataset preflight must not inherit warmup")
	}
}

func TestParseAutoTuneRampSelectsPeakAndSlo(t *testing.T) {
	dir := t.TempDir()
	peakPath := filepath.Join(dir, "peak.json")
	writeRamp(t, peakPath, `{
		"target":"saturation",
		"points":[
			{"knob":1,"total_tps":10,"output_tps":4,"achieved_rate":0.5,"ttft_ms":100,"itl_ms":20,"latency_ms":2000,"success":1},
			{"knob":4,"total_tps":30,"output_tps":12,"achieved_rate":2,"ttft_ms":180,"itl_ms":25,"latency_ms":4000,"success":0.98}
		]
	}`)
	peak, err := ParseAutoTuneRamp(peakPath)
	if err != nil {
		t.Fatal(err)
	}
	assertFloat(t, "peak total tok/s", peak.TokensPerSecondMean, 30)
	assertFloat(t, "peak output tok/s", peak.OutputTokensPerSecondMean, 12)
	assertFloat(t, "peak latency sec", peak.RequestLatencyMeanSec, 4)

	sloPath := filepath.Join(dir, "slo.json")
	writeRamp(t, sloPath, `{
		"target":"slo",
		"slo_bracket":[2, 4],
		"points":[
			{"knob":1,"total_tps":10,"output_tps":4,"success":1},
			{"knob":2,"total_tps":18,"output_tps":7,"ttft_ms":400,"success":0.97},
			{"knob":4,"total_tps":22,"output_tps":8,"success":0.9}
		]
	}`)
	slo, err := ParseAutoTuneRamp(sloPath)
	if err != nil {
		t.Fatal(err)
	}
	assertFloat(t, "slo total tok/s", slo.TokensPerSecondMean, 18)
	assertFloat(t, "slo ttft", slo.TimeToFirstTokenMeanMs, 400)
}

func writeRamp(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestGuideLLMWaitSeconds(t *testing.T) {
	// Default 600s cap plus 180s startup is 780s, which cuts off 100 in-flight requests.
	got := GuideLLMWaitSeconds(600, 100, 1)
	if got <= 780 {
		t.Fatalf("wait = %d, want more than 780", got)
	}
	if want := 180 + 600 + 100*30; got != want {
		t.Fatalf("wait = %d, want %d", got, want)
	}
	if got := GuideLLMWaitSeconds(600, 10, 1); got != 180+600+600 {
		t.Fatalf("short run wait = %d", got)
	}
	if got := GuideLLMWaitSeconds(600, 100, 0); got != GuideLLMWaitSeconds(600, 100, 1) {
		t.Fatalf("rate 0 = %d", got)
	}
	if got := GuideLLMWaitSeconds(10, 100000, 1); got != 180+100000+3600 {
		t.Fatalf("capped drain wait = %d", got)
	}
	if got := GuideLLMWaitSeconds(5, 10, 0.5); got != 180+20+300 {
		t.Fatalf("fractional rate wait = %d", got)
	}
}

func TestParseLegacyOutputTokensPerSecond(t *testing.T) {
	metrics := parseMetricsCSV(t, ""+
		"requests_per_second_mean,output_tokens_per_second_mean,request_latency_mean_sec,request_total,request_successful,request_errored,error_rate\n"+
		"3.5,88.25,1.2,4,4,0,0\n")
	assertFloat(t, "output tokens/s", metrics.OutputTokensPerSecondMean, 88.25)
	assertFloat(t, "requests/s", metrics.RequestsPerSecondMean, 3.5)
}

func parseMetricsCSV(t *testing.T, content string) *LLMBenchmarkMetrics {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "benchmarks.csv")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write csv: %v", err)
	}
	metrics, err := ParseMetricsCSV(path)
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	return metrics
}

func assertNilFloat(t *testing.T, name string, got *float64) {
	t.Helper()
	if got != nil {
		t.Fatalf("%s = %v, want empty", name, *got)
	}
}

func constraintCount(t *testing.T, spec GuideLLMSpec, kind string) int {
	t.Helper()
	for _, constraint := range spec.Constraints {
		if constraint["kind"] != kind {
			continue
		}
		switch count := constraint["count"].(type) {
		case int:
			return count
		case float64:
			return int(count)
		default:
			t.Fatalf("%s count has type %T", kind, constraint["count"])
		}
	}
	t.Fatalf("missing %s constraint", kind)
	return 0
}

func assertFloat(t *testing.T, name string, got *float64, want float64) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s is empty, want %v", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %v, want %v", name, *got, want)
	}
}

func TestNormalizeBenchmarkTargetURLStripsV1(t *testing.T) {
	got, err := NormalizeBenchmarkTargetURL("https://gpu.example.com/v1/")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://gpu.example.com" {
		t.Fatalf("got %s", got)
	}
	if _, err := NormalizeBenchmarkTargetURL("ftp://gpu.example.com"); err == nil {
		t.Fatal("expected scheme error")
	}
}

func TestExternalSpecKeepsTokenizerNameAndOmitsAPIKey(t *testing.T) {
	spec := BuildGuideLLMSpec(GuideLLMSpecInput{
		TargetURL:           "https://gpu.example.com",
		RequestFormat:       "/v1/chat/completions",
		Model:               "qwen",
		Tokenizer:           "Qwen/Qwen2.5-7B",
		RequestRate:         1,
		TotalRequests:       10,
		MaxDurationSeconds:  60,
		DatasetInputTokens:  8,
		DatasetOutputTokens: 4,
	})
	if _, ok := spec.Backend["api_key"]; ok {
		t.Fatal("stored spec includes api_key")
	}
	if spec.Tokenizer["model"] != "Qwen/Qwen2.5-7B" {
		t.Fatalf("tokenizer = %#v", spec.Tokenizer)
	}
	if _, ok := spec.Tokenizer["load_kwargs"]; ok {
		t.Fatal("external tokenizer set local_files_only")
	}
	ApplyGuideLLMAPIKey(&spec, "secret")
	if spec.Backend["api_key"] != "secret" {
		t.Fatalf("runtime api_key = %#v", spec.Backend["api_key"])
	}
	if _, ok := spec.Backend["api_routes"]; ok {
		t.Fatal("stored spec remaps the health check")
	}
	ApplyGuideLLMModelsHealthCheck(&spec)
	ApplyGuideLLMConnectTimeout(&spec)
	routes, _ := spec.Backend["api_routes"].(map[string]string)
	if routes["/health"] != "v1/models" {
		t.Fatalf("api_routes = %#v", spec.Backend["api_routes"])
	}
	if spec.Backend["timeout_connect"] != GuideLLMConnectTimeoutSeconds {
		t.Fatalf("timeout_connect = %#v", spec.Backend["timeout_connect"])
	}
}

func TestExternalAutoTuneUsesTokenizerName(t *testing.T) {
	command := strings.Join(AutoTuneRunCommand(AutoTuneCommandInput{
		TargetURL:          "https://gpu.example.com",
		Model:              "qwen",
		ProcessorPath:      "Qwen/Qwen2.5-7B",
		APIKey:             "secret",
		ModelsHealthCheck:  true,
		LowerBound:         1,
		UpperBound:         4,
		InputTokens:        8,
		OutputTokens:       4,
		MaxDurationSeconds: 60,
	}), " ")
	if strings.Contains(command, "local_files_only") {
		t.Fatal("external processor pinned local files")
	}
	if !strings.Contains(command, "--processor 'Qwen/Qwen2.5-7B'") {
		t.Fatalf("command = %s", command)
	}
	if !strings.Contains(command, `"api_key":"secret"`) {
		t.Fatal("api key was not passed to the runtime command")
	}
	if !strings.Contains(command, `"/health":"v1/models"`) {
		t.Fatal("external health check was not remapped to /v1/models")
	}
	if !strings.Contains(command, `"timeout_connect":30`) {
		t.Fatal("external connect timeout was not raised")
	}
	if patchAt, runAt := strings.Index(command, "sitecustomize.py"), strings.Index(command, "benchmark-runner"); patchAt < 0 || runAt < 0 || patchAt > runAt {
		t.Fatal("SecretStr JSON patch is missing before benchmark-runner")
	}
	local := strings.Join(AutoTuneRunCommand(AutoTuneCommandInput{
		TargetURL:          "http://10.0.0.8:8000",
		ProcessorPath:      "/data/models/qwen",
		LocalProcessor:     true,
		LowerBound:         1,
		UpperBound:         2,
		MaxDurationSeconds: 60,
	}), " ")
	if !strings.Contains(local, "local_files_only") {
		t.Fatal("local processor lost local_files_only")
	}
}

func TestGuideLLMCommandsAreValidShell(t *testing.T) {
	preflight, err := GuideLLMPreflightRunCommand(GuideLLMSpec{
		Backend: map[string]interface{}{"target": "http://127.0.0.1"},
		Data: []map[string]interface{}{{
			"kind": "json_file",
			"path": "/workdir/data.json",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	commands := [][]string{
		GuideLLMRunCommand(),
		GuideLLMWarmupCommand(),
		preflight,
		AutoTuneRunCommand(AutoTuneCommandInput{
			TargetURL:          "http://127.0.0.1",
			LowerBound:         1,
			UpperBound:         2,
			MaxDurationSeconds: 30,
		}),
	}
	for _, command := range commands {
		if len(command) != 3 {
			t.Fatalf("command = %#v", command)
		}
		out, err := exec.Command("sh", "-n", "-c", command[2]).CombinedOutput()
		if err != nil {
			t.Fatalf("shell syntax: %v\n%s\n%s", err, out, command[2])
		}
	}
}

func TestExportGuideLLMEnvPrefixesTheCommand(t *testing.T) {
	command, err := ExportGuideLLMEnv(GuideLLMRunCommand(), map[string]string{
		"GUIDELLM__SPEC__BACKEND": `{"target":"http://10.0.0.8:8000"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	script := command[2]
	if !strings.Contains(script, "export GUIDELLM__SPEC__BACKEND=") {
		t.Fatalf("script = %s", script)
	}
	if !strings.Contains(script, "guidellm run") {
		t.Fatalf("script = %s", script)
	}
	if strings.Index(script, "export ") > strings.Index(script, "guidellm run") {
		t.Fatal("env export is after guidellm")
	}
	if strings.Index(script, "sitecustomize.py") > strings.Index(script, "guidellm run") {
		t.Fatal("SecretStr JSON patch is installed after guidellm")
	}
}
