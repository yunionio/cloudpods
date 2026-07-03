package benchmark

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
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
	RequestRate         int
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
	RequestsPerSecondMean *float64
	RequestLatencyMeanSec *float64
	RequestTotal          int
	RequestSuccessful     int
	RequestErrored        int
	ErrorRate             *float64
}

type RunnerPodInput struct {
	Name                   string
	Image                  string
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
	return GuideLLMSpec{
		Backend: backend,
		Profile: map[string]interface{}{
			"kind": "constant",
			"rate": input.RequestRate,
		},
		Tokenizer: tokenizer,
		Constraints: []map[string]interface{}{
			{"kind": "max_requests", "count": input.TotalRequests},
			{"kind": "max_duration", "seconds": input.MaxDurationSeconds},
			{"kind": "max_errors", "count": input.MaxErrors},
		},
		Data: data,
	}
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

func GuideLLMRunCommand() []string {
	return []string{
		"sh",
		"-lc",
		"mkdir -p /workdir && guidellm run --output kind=json,path=/workdir/benchmarks.json --output kind=csv,path=/workdir/benchmarks.csv > /workdir/guidellm.log 2>&1",
	}
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

	parts := []string{"mkdir -p /workdir"}
	for _, key := range keys {
		parts = append(parts, "export "+key+"="+shellQuote(envs[key]))
	}
	parts = append(parts, "guidellm run --output kind=json,path=/workdir/dataset-preflight.json --output kind=csv,path=/workdir/dataset-preflight.csv > /workdir/dataset-preflight.log 2>&1")
	return []string{"sh", "-lc", strings.Join(parts, " && ")}, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func BuildRunnerPodInput(input RunnerPodInput) *computeapi.ServerCreateInput {
	root := int64(0)
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
						ContainerSpec: commonapi.ContainerSpec{
							Image:         input.Image,
							Command:       []string{"sh", "-lc", "while true; do sleep 3600; done"},
							AlwaysRestart: false,
							SecurityContext: &commonapi.ContainerSecurityContext{
								RunAsUser:  &root,
								RunAsGroup: &root,
							},
						},
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
	ret.PreferHost = input.HostId
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
		RequestsPerSecondMean: parseFloatPtr(values["requests_per_second_mean"]),
		RequestLatencyMeanSec: parseFloatPtr(values["request_latency_mean_sec"]),
		RequestTotal:          parseInt(values["request_total"]),
		RequestSuccessful:     parseInt(values["request_successful"]),
		RequestErrored:        parseInt(values["request_errored"]),
		ErrorRate:             parseFloatPtr(values["error_rate"]),
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
	ret := &LLMBenchmarkMetrics{
		RequestsPerSecondMean: parseFloatPtr(value("Server Throughput", "Successful Requests/Sec", "Mean")),
		RequestLatencyMeanSec: parseFloatPtr(value("Scheduler Metrics", "Request Time Avg", "Sec")),
		RequestTotal:          parseInt(value("Request Counts", "Total", "")),
		RequestSuccessful:     parseInt(value("Request Counts", "Successful", "")),
		RequestErrored:        parseInt(value("Request Counts", "Errored", "")),
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
