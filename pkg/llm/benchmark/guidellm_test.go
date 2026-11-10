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

func assertFloat(t *testing.T, name string, got *float64, want float64) {
	t.Helper()
	if got == nil {
		t.Fatalf("%s is empty, want %v", name, want)
	}
	if *got != want {
		t.Fatalf("%s = %v, want %v", name, *got, want)
	}
}
