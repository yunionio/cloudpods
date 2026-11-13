package benchmark

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
)

type GuideLLMSpecInput struct {
	TargetURL           string
	RequestFormat       string
	Model               string
	Tokenizer           string
	RequestRate         float64
	WarmupRequests      int
	TotalRequests       int
	MaxDurationSeconds  int
	MaxErrors           int
	DatasetInputTokens  int
	DatasetOutputTokens int
	DatasetPath         string
}

type GuideLLMSpec struct {
	Backend     map[string]interface{}   `json:"backend"`
	Profile     map[string]interface{}   `json:"profile"`
	Tokenizer   map[string]interface{}   `json:"tokenizer,omitempty"`
	Constraints []map[string]interface{} `json:"constraints"`
	Data        []map[string]interface{} `json:"data"`
}

type LLMBenchmarkMetrics struct {
	RequestsPerSecondMean     *float64
	InputTokensPerSecondMean  *float64
	OutputTokensPerSecondMean *float64
	TokensPerSecondMean       *float64
	RequestLatencyMeanSec     *float64
	RequestLatencyP95Sec      *float64
	RequestLatencyP99Sec      *float64
	TimeToFirstTokenMeanMs    *float64
	TimeToFirstTokenP95Ms     *float64
	TimeToFirstTokenP99Ms     *float64
	InterTokenLatencyMeanMs   *float64
	InterTokenLatencyP95Ms    *float64
	InterTokenLatencyP99Ms    *float64
	RequestConcurrencyMean    *float64
	RequestConcurrencyMax     *float64
	RequestTotal              int
	RequestSuccessful         int
	RequestErrored            int
	RequestIncomplete         int
	ErrorRate                 *float64
}

type RunnerPodInput struct {
	Name                   string
	Image                  string
	ContainerImageId       string
	NetworkId              string
	HostId                 string
	CPU                    int
	MemoryMB               int
	PackageImageId         string
	PackageSizeMB          int
	PackageMountBase       string
	ModelImageId           string
	ModelSizeMB            int
	ModelMountBase         string
	ModelMountSubdirectory string
}

const DatasetPreflightSamples = 10

func BuildGuideLLMSpec(input GuideLLMSpecInput) GuideLLMSpec {
	backend := map[string]interface{}{
		"kind":           "openai_http",
		"target":         input.TargetURL,
		"request_format": input.RequestFormat,
	}
	if input.Model != "" {
		backend["model"] = input.Model
	}
	tokenizer := map[string]interface{}{}
	if strings.TrimSpace(input.DatasetPath) != "" && input.Model != "" {
		tokenizer = map[string]interface{}{
			"kind":  "hf_auto",
			"model": tokenizerModel(input.Model),
		}
	} else if strings.TrimSpace(input.Tokenizer) != "" {
		tokenizer = GuideLLMRemoteTokenizer(input.Tokenizer)
	}
	data := []map[string]interface{}{
		{
			"kind":          "synthetic_text",
			"prompt_tokens": input.DatasetInputTokens,
			"output_tokens": input.DatasetOutputTokens,
		},
	}
	if strings.TrimSpace(input.DatasetPath) != "" {
		data = []map[string]interface{}{
			{
				"kind": "json_file",
				"path": input.DatasetPath,
				"load_kwargs": map[string]interface{}{
					"split": "train",
				},
			},
		}
	}
	profile := map[string]interface{}{
		"kind": "constant",
		"rate": input.RequestRate,
	}
	if input.WarmupRequests > 0 {
		profile["warmup"] = map[string]interface{}{
			"value": input.WarmupRequests,
			"mode":  "requests",
		}
	}
	return GuideLLMSpec{
		Backend:   backend,
		Profile:   profile,
		Tokenizer: tokenizer,
		Constraints: []map[string]interface{}{
			{"kind": "max_requests", "count": input.TotalRequests},
			{"kind": "max_duration", "seconds": input.MaxDurationSeconds},
			{"kind": "max_errors", "count": input.MaxErrors},
		},
		Data: data,
	}
}

const (
	GuideLLMWarmupJSONPath = "/workdir/warmup.json"
	GuideLLMWarmupCSVPath  = "/workdir/warmup.csv"
)

// WithoutGuideLLMWarmup copies a spec for the measured run.
// GuideLLM's in-run warmup both consumes max_requests and then drops a
// timing-dependent number of early requests from the reported total. The
// measured run therefore has no warmup window, so Request Counts Total matches
// the configured request limit when the run is not cut short.
func WithoutGuideLLMWarmup(spec GuideLLMSpec) GuideLLMSpec {
	out := cloneGuideLLMSpec(spec)
	if out.Profile != nil {
		delete(out.Profile, "warmup")
	}
	return out
}

// WarmupGuideLLMSpec builds a discarded run that sends warmupRequests and
// does not apply another warmup window. ok is false when there is nothing to warm up.
func WarmupGuideLLMSpec(spec GuideLLMSpec, warmupRequests int) (GuideLLMSpec, bool) {
	if warmupRequests <= 0 {
		return GuideLLMSpec{}, false
	}
	out := WithoutGuideLLMWarmup(spec)
	replaced := false
	for i := range out.Constraints {
		if out.Constraints[i]["kind"] != "max_requests" {
			continue
		}
		out.Constraints[i]["count"] = warmupRequests
		replaced = true
	}
	if !replaced {
		out.Constraints = append(out.Constraints, map[string]interface{}{
			"kind":  "max_requests",
			"count": warmupRequests,
		})
	}
	return out, true
}

func cloneGuideLLMSpec(spec GuideLLMSpec) GuideLLMSpec {
	raw, err := json.Marshal(spec)
	if err != nil {
		return spec
	}
	var out GuideLLMSpec
	if err := json.Unmarshal(raw, &out); err != nil {
		return spec
	}
	return out
}

func GuideLLMLocalTokenizer(modelPath string) map[string]interface{} {
	return map[string]interface{}{
		"kind":  "hf_auto",
		"model": modelPath,
		"load_kwargs": map[string]interface{}{
			"local_files_only": true,
		},
	}
}

func GuideLLMRemoteTokenizer(name string) map[string]interface{} {
	return map[string]interface{}{
		"kind":  "hf_auto",
		"model": strings.TrimSpace(name),
	}
}

func ApplyGuideLLMAPIKey(spec *GuideLLMSpec, apiKey string) {
	if spec == nil || strings.TrimSpace(apiKey) == "" {
		return
	}
	if spec.Backend == nil {
		spec.Backend = map[string]interface{}{}
	}
	spec.Backend["api_key"] = strings.TrimSpace(apiKey)
}

// ApplyGuideLLMModelsHealthCheck points GuideLLM's startup probe at /v1/models.
// OpenAI-compatible gateways such as GPUStack do not serve /health.
func ApplyGuideLLMModelsHealthCheck(spec *GuideLLMSpec) {
	if spec == nil {
		return
	}
	if spec.Backend == nil {
		spec.Backend = map[string]interface{}{}
	}
	spec.Backend["api_routes"] = map[string]string{
		"/health": "v1/models",
	}
}

const GuideLLMConnectTimeoutSeconds = 30

// ApplyGuideLLMConnectTimeout gives the runner longer than GuideLLM's 5 second
// default to open a TCP connection to an external target.
func ApplyGuideLLMConnectTimeout(spec *GuideLLMSpec) {
	if spec == nil {
		return
	}
	if spec.Backend == nil {
		spec.Backend = map[string]interface{}{}
	}
	spec.Backend["timeout_connect"] = GuideLLMConnectTimeoutSeconds
}

func NormalizeBenchmarkTargetURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("target_url must be an http or https URL")
	}
	path := strings.TrimRight(parsed.Path, "/")
	if strings.EqualFold(path, "/v1") {
		path = ""
	}
	parsed.Path = path
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}

func BuildGuideLLMPreflightSpec(formal GuideLLMSpec, maxDurationSeconds int) (GuideLLMSpec, error) {
	if len(formal.Data) != 1 || formal.Data[0]["kind"] != "json_file" {
		return GuideLLMSpec{}, fmt.Errorf("dataset preflight requires one json_file data source")
	}
	path, _ := formal.Data[0]["path"].(string)
	if strings.TrimSpace(path) == "" {
		return GuideLLMSpec{}, fmt.Errorf("dataset preflight path is empty")
	}
	data := map[string]interface{}{
		"kind": "json_file",
		"path": path,
		"load_kwargs": map[string]interface{}{
			"split": "train[:10]",
		},
	}
	return GuideLLMSpec{
		Backend:   formal.Backend,
		Tokenizer: formal.Tokenizer,
		Profile: map[string]interface{}{
			"kind": "constant",
			"rate": 1,
		},
		Constraints: []map[string]interface{}{
			{"kind": "max_requests", "count": DatasetPreflightSamples},
			{"kind": "max_duration", "seconds": maxDurationSeconds},
			{"kind": "max_errors", "count": DatasetPreflightSamples},
		},
		Data: []map[string]interface{}{data},
	}, nil
}

// GuideLLM 0.7 writes the backend config into the CSV with json.dumps.
// An external API key is a pydantic SecretStr, which that call cannot encode.
func guideLLMSecretJSONPrelude() string {
	return `mkdir -p /workdir && cat > /workdir/sitecustomize.py << 'PY'
import json
try:
    from pydantic import SecretStr
except Exception:
    SecretStr = ()
_default = json.JSONEncoder.default
def _default_with_secret(self, o):
    if SecretStr and isinstance(o, SecretStr):
        return "**********"
    return _default(self, o)
json.JSONEncoder.default = _default_with_secret
PY
export PYTHONPATH=/workdir${PYTHONPATH:+:$PYTHONPATH}`
}

func GuideLLMRunCommand() []string {
	return []string{
		"sh",
		"-lc",
		guideLLMSecretJSONPrelude() + "\nguidellm run --output kind=json,path=/workdir/benchmarks.json --output kind=csv,path=/workdir/benchmarks.csv > /workdir/guidellm.log 2>&1",
	}
}

func GuideLLMWarmupCommand() []string {
	return []string{
		"sh",
		"-lc",
		guideLLMSecretJSONPrelude() + "\nguidellm run --output kind=json,path=" + GuideLLMWarmupJSONPath + " --output kind=csv,path=" + GuideLLMWarmupCSVPath + " > /workdir/warmup.log 2>&1",
	}
}

func ExportGuideLLMEnv(command []string, envs map[string]string) ([]string, error) {
	script, err := guideLLMCommandScript(command)
	if err != nil {
		return nil, err
	}
	if len(envs) == 0 {
		return command, nil
	}
	keys := make([]string, 0, len(envs))
	for key := range envs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys)+1)
	for _, key := range keys {
		parts = append(parts, "export "+key+"="+shellQuote(envs[key]))
	}
	parts = append(parts, script)
	return []string{"sh", "-lc", strings.Join(parts, "; ")}, nil
}

const (
	AutoTuneRampPath  = "/workdir/benchmarks__ramp.json"
	autoTuneMaxPoints = 12
)

type AutoTuneCommandInput struct {
	TargetURL          string
	Model              string
	ProcessorPath      string
	LowerBound         float64
	UpperBound         float64
	MaxDurationSeconds int
	InputTokens        int
	OutputTokens       int
	SloAvgTtftMs       float64
	SloAvgTpotMs       float64
	SloAvgLatencyMs    float64
	LocalProcessor     bool
	APIKey             string
	ModelsHealthCheck  bool
}

func AutoTuneRunCommand(input AutoTuneCommandInput) []string {
	args := []string{
		"if ! command -v benchmark-runner >/dev/null 2>&1; then echo 'benchmark-runner is not installed in this image' >&2; exit 127; fi;",
		"benchmark-runner benchmark run",
		"--target", shellQuote(input.TargetURL),
		"--auto-tune",
		"--axis", "rate",
		"--lower-bound", formatRate(input.LowerBound),
		"--upper-bound", formatRate(input.UpperBound),
		"--max-points", strconv.Itoa(autoTuneMaxPoints),
		"--max-total-seconds", strconv.Itoa(input.MaxDurationSeconds),
		"--data", shellQuote(fmt.Sprintf("prompt_tokens=%d,output_tokens=%d", input.InputTokens, input.OutputTokens)),
		"--output-dir", "/workdir",
		"--outputs", "benchmarks.dual_json",
		"--disable-console-outputs",
	}
	if input.Model != "" {
		args = append(args, "--model", shellQuote(input.Model))
	}
	if input.APIKey != "" || input.ModelsHealthCheck {
		payload := map[string]interface{}{}
		if input.APIKey != "" {
			payload["api_key"] = input.APIKey
		}
		if input.ModelsHealthCheck {
			payload["api_routes"] = map[string]string{"/health": "v1/models"}
			payload["timeout_connect"] = GuideLLMConnectTimeoutSeconds
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			return []string{"sh", "-lc", "echo 'invalid backend kwargs' >&2; exit 1"}
		}
		args = append(args, "--backend-kwargs", shellQuote(string(encoded)))
	}
	if input.ProcessorPath != "" {
		args = append(args, "--processor", shellQuote(input.ProcessorPath))
		if input.LocalProcessor {
			args = append(args, "--processor-args", shellQuote(`{"local_files_only":true}`))
		}
	}
	if input.SloAvgTtftMs > 0 {
		args = append(args, "--slo-avg-ttft-ms", formatRate(input.SloAvgTtftMs))
	}
	if input.SloAvgTpotMs > 0 {
		args = append(args, "--slo-avg-tpot-ms", formatRate(input.SloAvgTpotMs))
	}
	if input.SloAvgLatencyMs > 0 {
		args = append(args, "--slo-avg-latency-ms", formatRate(input.SloAvgLatencyMs))
	}
	args = append(args, "> /workdir/guidellm.log 2>&1")
	return []string{"sh", "-lc", guideLLMSecretJSONPrelude() + "\n" + strings.Join(args, " ")}
}

func formatRate(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

type autoTunePoint struct {
	Knob         float64 `json:"knob"`
	OutputTps    float64 `json:"output_tps"`
	TotalTps     float64 `json:"total_tps"`
	TtftMs       float64 `json:"ttft_ms"`
	TtftP95Ms    float64 `json:"ttft_p95_ms"`
	TtftP99Ms    float64 `json:"ttft_p99_ms"`
	ItlMs        float64 `json:"itl_ms"`
	ItlP95Ms     float64 `json:"itl_p95_ms"`
	ItlP99Ms     float64 `json:"itl_p99_ms"`
	LatencyMs    float64 `json:"latency_ms"`
	LatencyP95Ms float64 `json:"latency_p95_ms"`
	LatencyP99Ms float64 `json:"latency_p99_ms"`
	AchievedRate float64 `json:"achieved_rate"`
	Success      float64 `json:"success"`
}

type autoTuneRamp struct {
	Target     string          `json:"target"`
	Points     []autoTunePoint `json:"points"`
	SloBracket []*float64      `json:"slo_bracket"`
}

func ParseAutoTuneRamp(path string) (*LLMBenchmarkMetrics, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ramp autoTuneRamp
	if err := json.Unmarshal(raw, &ramp); err != nil {
		return nil, err
	}
	point, err := selectAutoTunePoint(ramp)
	if err != nil {
		return nil, err
	}
	return metricsFromAutoTunePoint(point), nil
}

func selectAutoTunePoint(ramp autoTuneRamp) (autoTunePoint, error) {
	if len(ramp.Points) == 0 {
		return autoTunePoint{}, fmt.Errorf("auto-tune ramp has no measured points")
	}
	if ramp.Target == "slo" {
		if len(ramp.SloBracket) == 0 || ramp.SloBracket[0] == nil {
			return autoTunePoint{}, fmt.Errorf("auto-tune found no load that meets the SLO")
		}
		want := *ramp.SloBracket[0]
		best := ramp.Points[0]
		for _, point := range ramp.Points[1:] {
			if math.Abs(point.Knob-want) < math.Abs(best.Knob-want) {
				best = point
			}
		}
		return best, nil
	}
	best := ramp.Points[0]
	for _, point := range ramp.Points[1:] {
		if point.TotalTps > best.TotalTps || (point.TotalTps == best.TotalTps && point.Knob < best.Knob) {
			best = point
		}
	}
	return best, nil
}

func metricsFromAutoTunePoint(point autoTunePoint) *LLMBenchmarkMetrics {
	errorRate := 1 - point.Success
	return &LLMBenchmarkMetrics{
		RequestsPerSecondMean:     floatPtr(point.AchievedRate),
		OutputTokensPerSecondMean: floatPtr(point.OutputTps),
		TokensPerSecondMean:       floatPtr(point.TotalTps),
		RequestLatencyMeanSec:     floatPtr(point.LatencyMs / 1000),
		RequestLatencyP95Sec:      floatPtr(point.LatencyP95Ms / 1000),
		RequestLatencyP99Sec:      floatPtr(point.LatencyP99Ms / 1000),
		TimeToFirstTokenMeanMs:    floatPtr(point.TtftMs),
		TimeToFirstTokenP95Ms:     floatPtr(point.TtftP95Ms),
		TimeToFirstTokenP99Ms:     floatPtr(point.TtftP99Ms),
		InterTokenLatencyMeanMs:   floatPtr(point.ItlMs),
		InterTokenLatencyP95Ms:    floatPtr(point.ItlP95Ms),
		InterTokenLatencyP99Ms:    floatPtr(point.ItlP99Ms),
		ErrorRate:                 floatPtr(errorRate),
	}
}

func floatPtr(value float64) *float64 {
	return &value
}

func GuideLLMPreflightRunCommand(spec GuideLLMSpec) ([]string, error) {
	envs, err := GuideLLMEnvs(spec)
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(envs))
	for key := range envs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := []string{guideLLMSecretJSONPrelude(), "mkdir -p /workdir"}
	for _, key := range keys {
		parts = append(parts, "export "+key+"="+shellQuote(envs[key]))
	}
	parts = append(parts, "guidellm run --output kind=json,path=/workdir/dataset-preflight.json --output kind=csv,path=/workdir/dataset-preflight.csv > /workdir/dataset-preflight.log 2>&1")
	return []string{"sh", "-lc", strings.Join(parts, " && ")}, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func GuideLLMResultExitPath(jsonPath string) string {
	return jsonPath + ".exit"
}

func GuideLLMResultPidPath(jsonPath string) string {
	return jsonPath + ".pid"
}

const (
	GuideLLMStartupSlackSeconds    = 180
	guideLLMPerRequestDrainSeconds = 30
	guideLLMMaxDrainSeconds        = 3600
)

// GuideLLMWaitSeconds is how long to wait for a detached run to write its
// exit file. max_duration only stops new requests; ones already sent keep
// running, and a 100-request run does not fit in max_duration plus startup.
func GuideLLMWaitSeconds(maxDuration, totalRequests int, requestRate float64) int {
	if maxDuration < 1 {
		maxDuration = 1
	}
	if requestRate <= 0 {
		requestRate = 1
	}
	if totalRequests < 1 {
		totalRequests = 1
	}
	send := int(math.Ceil(float64(totalRequests) / requestRate))
	window := maxDuration
	if send > window {
		window = send
	}
	drain := window
	if tail := totalRequests * guideLLMPerRequestDrainSeconds; tail > drain {
		drain = tail
	}
	if drain > guideLLMMaxDrainSeconds {
		drain = guideLLMMaxDrainSeconds
	}
	return GuideLLMStartupSlackSeconds + window + drain
}

// GuideLLMBackgroundCommand starts the original sh -lc script in its own
// session and records the shell pid and exit code. The caller can return
// immediately instead of holding exec-sync open for the whole run.
func GuideLLMBackgroundCommand(command []string, jsonPath, csvPath string) ([]string, error) {
	script, err := guideLLMCommandScript(command)
	if err != nil {
		return nil, err
	}
	exitPath := GuideLLMResultExitPath(jsonPath)
	pidPath := GuideLLMResultPidPath(jsonPath)
	body := strings.TrimSpace(script)
	if !strings.HasSuffix(body, ";") {
		body += ";"
	}
	inner := "{ " + body + " }; echo $? > " + shellQuote(exitPath)
	quotedInner := shellQuote(inner)
	start := "setsid sh -c " + quotedInner + " </dev/null >/dev/null 2>&1 & echo $! > " + shellQuote(pidPath)
	fallback := "nohup sh -c " + quotedInner + " </dev/null >/dev/null 2>&1 & echo $! > " + shellQuote(pidPath)
	dirs := shellQuote(path.Dir(jsonPath))
	removes := shellQuote(exitPath) + " " + shellQuote(pidPath) + " " + shellQuote(jsonPath)
	if csvPath != "" {
		dirs += " " + shellQuote(path.Dir(csvPath))
		removes += " " + shellQuote(csvPath)
	}
	launcher := strings.Join([]string{
		"mkdir -p " + dirs,
		"rm -f " + removes,
		"if command -v setsid >/dev/null 2>&1; then " + start + "; else " + fallback + "; fi",
		"test -s " + shellQuote(pidPath),
	}, "; ")
	return []string{"sh", "-lc", launcher}, nil
}

func guideLLMCommandScript(command []string) (string, error) {
	if len(command) == 3 && command[0] == "sh" && (command[1] == "-lc" || command[1] == "-c") && strings.TrimSpace(command[2]) != "" {
		return command[2], nil
	}
	return "", fmt.Errorf("guidellm command must be sh -lc")
}

func BuildRunnerPodInput(input RunnerPodInput) *computeapi.ServerCreateInput {
	root := int64(0)
	ctrSpec := commonapi.ContainerSpec{
		Command:       []string{"sh", "-lc", "mkdir -p /workdir && touch /workdir/guidellm.log /workdir/warmup.log /workdir/dataset-preflight.log && exec tail -F /workdir/guidellm.log /workdir/warmup.log /workdir/dataset-preflight.log"},
		AlwaysRestart: false,
		SecurityContext: &commonapi.ContainerSecurityContext{
			RunAsUser:  &root,
			RunAsGroup: &root,
		},
	}
	if input.ContainerImageId != "" {
		ctrSpec.ContainerImageId = input.ContainerImageId
	} else {
		ctrSpec.Image = input.Image
	}
	ret := &computeapi.ServerCreateInput{
		ServerConfigs: computeapi.NewServerConfigs(),
		VcpuCount:     input.CPU,
		VmemSize:      input.MemoryMB,
		AutoStart:     true,
		Pod: &computeapi.PodCreateInput{
			Containers: []*computeapi.PodContainerCreateInput{
				{
					Name: "guidellm",
					ContainerSpec: computeapi.ContainerSpec{
						ContainerSpec: ctrSpec,
					},
				},
			},
		},
	}
	if input.PackageImageId != "" {
		if input.PackageSizeMB <= 0 {
			input.PackageSizeMB = 1024
		}
		if input.PackageMountBase == "" {
			input.PackageMountBase = "/data/benchmark-packages"
		}
		diskIndex := len(ret.Disks)
		ret.Disks = append(ret.Disks, &computeapi.DiskConfig{
			DiskType: "data",
			Format:   "raw",
			Fs:       "ext4",
			SizeMb:   input.PackageSizeMB,
			Index:    diskIndex,
		})
		ret.Pod.Containers[0].VolumeMounts = append(ret.Pod.Containers[0].VolumeMounts, &commonapi.ContainerVolumeMount{
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: input.PackageMountBase,
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        &diskIndex,
				SubDirectory: strings.TrimPrefix(input.PackageMountBase, "/"),
				PostOverlay: []*commonapi.ContainerVolumeMountDiskPostOverlay{
					{
						Image: &commonapi.ContainerVolumeMountDiskPostImageOverlay{
							Id: input.PackageImageId,
						},
					},
				},
			},
			Propagation: commonapi.MOUNTPROPAGATION_PROPAGATION_HOST_TO_CONTAINER,
		})
	}
	if input.ModelImageId != "" {
		if input.ModelSizeMB <= 0 {
			input.ModelSizeMB = 1024
		}
		if input.ModelMountBase == "" {
			input.ModelMountBase = "/data/models"
		}
		diskIndex := len(ret.Disks)
		ret.Disks = append(ret.Disks, &computeapi.DiskConfig{
			DiskType: "data",
			Format:   "raw",
			Fs:       "ext4",
			SizeMb:   input.ModelSizeMB,
			Index:    diskIndex,
		})
		ret.Pod.Containers[0].VolumeMounts = append(ret.Pod.Containers[0].VolumeMounts, &commonapi.ContainerVolumeMount{
			Type:      commonapi.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath: input.ModelMountBase,
			ReadOnly:  true,
			Disk: &commonapi.ContainerVolumeMountDisk{
				Index:        &diskIndex,
				SubDirectory: input.ModelMountSubdirectory,
				PostOverlay: []*commonapi.ContainerVolumeMountDiskPostOverlay{
					{
						Image: &commonapi.ContainerVolumeMountDiskPostImageOverlay{
							Id: input.ModelImageId,
						},
					},
				},
			},
			Propagation: commonapi.MOUNTPROPAGATION_PROPAGATION_HOST_TO_CONTAINER,
		})
	}
	ret.Hypervisor = computeapi.HYPERVISOR_POD
	ret.Name = input.Name
	if input.HostId != "" {
		ret.PreferHost = input.HostId
	}
	ret.Count = 1
	ret.Networks = []*computeapi.NetworkConfig{
		{
			Network: input.NetworkId,
		},
	}
	return ret
}

func GuideLLMEnvs(spec GuideLLMSpec) (map[string]string, error) {
	backend, err := json.Marshal(spec.Backend)
	if err != nil {
		return nil, err
	}
	profile, err := json.Marshal(spec.Profile)
	if err != nil {
		return nil, err
	}
	tokenizer, err := json.Marshal(spec.Tokenizer)
	if err != nil {
		return nil, err
	}
	constraints, err := json.Marshal(spec.Constraints)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(spec.Data)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"GUIDELLM__SPEC__BACKEND":     string(backend),
		"GUIDELLM__SPEC__PROFILE":     string(profile),
		"GUIDELLM__SPEC__TOKENIZER":   string(tokenizer),
		"GUIDELLM__SPEC__CONSTRAINTS": string(constraints),
		"GUIDELLM__SPEC__DATA":        string(data),
	}, nil
}

func tokenizerModel(model string) string {
	if i := strings.LastIndex(model, ":"); i > 0 {
		return model[:i]
	}
	return model
}

func ParseMetricsCSV(path string) (*LLMBenchmarkMetrics, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) >= 4 && len(rows[1]) > 0 && rows[1][0] == "Run ID" {
		return parseGuideLLMv07MetricsCSV(rows), nil
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("metrics csv has %d rows", len(rows))
	}
	values := map[string]string{}
	for i, key := range rows[0] {
		if i < len(rows[1]) {
			values[key] = rows[1][i]
		}
	}
	ret := &LLMBenchmarkMetrics{
		RequestsPerSecondMean:     parseFloatPtr(values["requests_per_second_mean"]),
		OutputTokensPerSecondMean: parseFloatPtr(values["output_tokens_per_second_mean"]),
		RequestLatencyMeanSec:     parseFloatPtr(values["request_latency_mean_sec"]),
		RequestTotal:              parseInt(values["request_total"]),
		RequestSuccessful:         parseInt(values["request_successful"]),
		RequestErrored:            parseInt(values["request_errored"]),
		ErrorRate:                 parseFloatPtr(values["error_rate"]),
	}
	return ret, nil
}

func parseGuideLLMv07MetricsCSV(rows [][]string) *LLMBenchmarkMetrics {
	value := func(group, name, unit string) string {
		for i := range rows[1] {
			if i < len(rows[0]) && i < len(rows[2]) && i < len(rows[3]) &&
				rows[0][i] == group && rows[1][i] == name && (unit == "" || rows[2][i] == unit) {
				return rows[3][i]
			}
		}
		return ""
	}
	latencyPercentiles := value("Request Latency", "Successful Sec", "Percentiles")
	ttftPercentiles := value("Time to First Token", "Successful ms", "Percentiles")
	tpotPercentiles := value("Inter Token Latency", "Successful ms", "Percentiles")
	concurrencyPercentiles := value("Server Throughput", "Successful Concurrency", "Percentiles")
	ret := &LLMBenchmarkMetrics{
		RequestsPerSecondMean:     parseFloatPtr(value("Server Throughput", "Successful Requests/Sec", "Mean")),
		InputTokensPerSecondMean:  parseFloatPtr(value("Token Throughput", "Successful Input Tokens/Sec", "Mean")),
		OutputTokensPerSecondMean: parseFloatPtr(value("Token Throughput", "Successful Output Tokens/Sec", "Mean")),
		TokensPerSecondMean:       parseFloatPtr(value("Token Throughput", "Successful Total Tokens/Sec", "Mean")),
		RequestLatencyMeanSec:     parseFloatPtr(value("Request Latency", "Successful Sec", "Mean")),
		RequestLatencyP95Sec:      percentile(latencyPercentiles, 8),
		RequestLatencyP99Sec:      percentile(latencyPercentiles, 9),
		TimeToFirstTokenMeanMs:    parseFloatPtr(value("Time to First Token", "Successful ms", "Mean")),
		TimeToFirstTokenP95Ms:     percentile(ttftPercentiles, 8),
		TimeToFirstTokenP99Ms:     percentile(ttftPercentiles, 9),
		InterTokenLatencyMeanMs:   parseFloatPtr(value("Inter Token Latency", "Successful ms", "Mean")),
		InterTokenLatencyP95Ms:    percentile(tpotPercentiles, 8),
		InterTokenLatencyP99Ms:    percentile(tpotPercentiles, 9),
		RequestConcurrencyMean:    parseFloatPtr(value("Server Throughput", "Successful Concurrency", "Mean")),
		RequestConcurrencyMax:     percentile(concurrencyPercentiles, 10),
		RequestTotal:              parseInt(value("Request Counts", "Total", "")),
		RequestSuccessful:         parseInt(value("Request Counts", "Successful", "")),
		RequestErrored:            parseInt(value("Request Counts", "Errored", "")),
		RequestIncomplete:         parseInt(value("Request Counts", "Incomplete", "")),
	}
	if ret.RequestsPerSecondMean == nil {
		if duration, err := strconv.ParseFloat(value("Timings", "Duration", "Sec"), 64); err == nil && duration > 0 {
			v := float64(ret.RequestSuccessful) / duration
			ret.RequestsPerSecondMean = &v
		}
	}
	if ret.RequestTotal > 0 {
		v := float64(ret.RequestErrored) / float64(ret.RequestTotal)
		ret.ErrorRate = &v
	}
	return ret
}

func percentile(raw string, index int) *float64 {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "[")
	raw = strings.TrimSuffix(raw, "]")
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	if index < 0 || index >= len(parts) {
		return nil
	}
	return parseFloatPtr(strings.TrimSpace(parts[index]))
}

func parseFloatPtr(s string) *float64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

func parseInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}
