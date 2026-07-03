package models

import (
	"context"
	"database/sql"
	stderrors "errors"
	"os"
	"path/filepath"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	bench "yunion.io/x/onecloud/pkg/llm/benchmark"
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

func init() {
	GetLLMBenchmarkManager()
}

var llmBenchmarkManager *SLLMBenchmarkManager

func GetLLMBenchmarkManager() *SLLMBenchmarkManager {
	if llmBenchmarkManager != nil {
		return llmBenchmarkManager
	}
	llmBenchmarkManager = &SLLMBenchmarkManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SLLMBenchmark{},
			"llm_benchmarks_tbl",
			"llm_benchmark",
			"llm_benchmarks",
		),
	}
	llmBenchmarkManager.SetVirtualObject(llmBenchmarkManager)
	return llmBenchmarkManager
}

type SLLMBenchmarkManager struct {
	db.SVirtualResourceBaseManager
}

type LLMBenchmarkTokenizerMount struct {
	ImageId           string
	ModelPath         string
	SizeMB            int
	MountBase         string
	MountSubdirectory string
}

func selectBenchmarkTokenizerModel(llmType, model string, mounted []*SInstantModel) (*SInstantModel, error) {
	matched := make([]*SInstantModel, 0, 1)
	for _, candidate := range mounted {
		if upstreamModelKeyFromInstantModel(llmType, candidate) == strings.TrimSpace(model) {
			matched = append(matched, candidate)
		}
	}
	if len(matched) == 1 {
		return matched[0], nil
	}
	if len(matched) > 1 {
		return nil, httperrors.NewInputParameterError("multiple mounted models match %s", model)
	}
	if len(mounted) == 1 {
		return mounted[0], nil
	}
	return nil, httperrors.NewInputParameterError("cannot uniquely resolve tokenizer model %s", model)
}

func buildBenchmarkTokenizerMount(llmType string, model *SInstantModel) (*LLMBenchmarkTokenizerMount, error) {
	if model == nil || model.ImageId == "" || model.Status != imageapi.IMAGE_STATUS_ACTIVE || model.GetActualSizeMb() <= 0 {
		return nil, httperrors.NewInvalidStatusError("benchmark tokenizer model image is not active")
	}
	subdirectory := ""
	switch strings.ToLower(strings.TrimSpace(llmType)) {
	case string(api.LLM_CONTAINER_VLLM):
		subdirectory = api.LLM_VLLM
	case string(api.LLM_CONTAINER_SGLANG):
		subdirectory = api.LLM_SGLANG
	default:
		return nil, httperrors.NewInputParameterError("offline synthetic tokenizer is unsupported for %s", llmType)
	}
	modelPath := ""
	for _, mount := range model.Mounts {
		clean := filepath.Clean(mount)
		rel, err := filepath.Rel(api.LLM_VLLM_BASE_PATH, clean)
		if err == nil && rel != "." && rel != ".." && !filepath.IsAbs(rel) && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			modelPath = clean
			break
		}
	}
	if modelPath == "" {
		return nil, httperrors.NewInvalidStatusError("benchmark tokenizer model has no mount below %s", api.LLM_VLLM_BASE_PATH)
	}
	return &LLMBenchmarkTokenizerMount{
		ImageId:           model.ImageId,
		ModelPath:         modelPath,
		SizeMB:            int(model.GetActualSizeMb()) + 512,
		MountBase:         api.LLM_VLLM_BASE_PATH,
		MountSubdirectory: subdirectory,
	}, nil
}

func resolveBenchmarkTokenizerMount(llm *SLLM, model string) (*LLMBenchmarkTokenizerMount, error) {
	boolTrue := true
	relations, err := llm.FetchModels(nil, &boolTrue, nil)
	if err != nil {
		return nil, errors.Wrap(err, "fetch mounted models")
	}
	mounted := make([]*SInstantModel, 0, len(relations))
	for i := range relations {
		instant, err := GetInstantModelManager().GetInstantModelById(relations[i].InstantModelId)
		if err != nil {
			return nil, errors.Wrap(err, "fetch mounted instant model")
		}
		mounted = append(mounted, instant)
	}
	llmType := string(llm.GetLLMContainerDriver().GetType())
	selected, err := selectBenchmarkTokenizerModel(llmType, model, mounted)
	if err != nil {
		return nil, err
	}
	return buildBenchmarkTokenizerMount(llmType, selected)
}

type SLLMBenchmark struct {
	db.SVirtualResourceBase

	LLMId              string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
	LLMDeploymentId    string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional" index:"true"`
	LLMSkuId           string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional" index:"true"`
	LLMImageId         string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"optional" index:"true"`
	BenchmarkPackageId string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional" index:"true"`

	Backend       string `width:"64" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	Model         string `width:"256" charset:"utf8" nullable:"true" list:"user" create:"optional" update:"user"`
	TargetUrl     string `charset:"utf8" length:"medium" nullable:"true" list:"user" create:"optional"`
	RequestFormat string `width:"64" charset:"ascii" nullable:"true" list:"user" create:"optional"`

	Profile             string `width:"32" charset:"ascii" nullable:"false" default:"constant" list:"user" create:"optional"`
	DatasetName         string `width:"64" charset:"ascii" nullable:"false" default:"synthetic_text" list:"user" create:"optional"`
	DatasetInputTokens  int    `nullable:"true" list:"user" create:"optional" update:"user"`
	DatasetOutputTokens int    `nullable:"true" list:"user" create:"optional" update:"user"`
	DatasetPath         string `charset:"utf8" length:"medium" nullable:"true" list:"user" create:"optional"`

	RequestRate        int `nullable:"true" list:"user" create:"optional" update:"user"`
	TotalRequests      int `nullable:"true" list:"user" create:"optional" update:"user"`
	MaxDurationSeconds int `nullable:"true" list:"user" create:"optional" update:"user"`
	MaxErrors          int `nullable:"true" list:"user" create:"optional" update:"user"`

	State         string `width:"32" charset:"ascii" nullable:"false" default:"pending" list:"user" create:"optional" update:"user" index:"true"`
	StateMessage  string `charset:"utf8" length:"medium" nullable:"true" list:"user" create:"optional" update:"user"`
	TaskId        string `width:"128" charset:"ascii" nullable:"true" list:"user" update:"user"`
	StopRequested bool   `nullable:"false" default:"false" list:"user" update:"user"`

	RunnerServerId    string `width:"128" charset:"ascii" nullable:"true" list:"user" update:"user"`
	RunnerContainerId string `width:"128" charset:"ascii" nullable:"true" list:"user" update:"user"`

	WorkDir    string `charset:"utf8" length:"medium" nullable:"true" list:"user" create:"optional"`
	LogPath    string `charset:"utf8" length:"medium" nullable:"true" list:"user" update:"user"`
	ResultJson string `charset:"utf8" length:"medium" nullable:"true" list:"user" update:"user"`
	ResultCsv  string `charset:"utf8" length:"medium" nullable:"true" list:"user" update:"user"`

	DatasetPreflight    *api.LLMBenchmarkDatasetPreflight `json:"dataset_preflight,omitempty" length:"long" nullable:"true" list:"user" update:"user"`
	PreflightLogPath    string                            `charset:"utf8" length:"medium" nullable:"true" update:"user"`
	PreflightResultJson string                            `charset:"utf8" length:"medium" nullable:"true" update:"user"`
	RawPreflightLog     string                            `charset:"utf8" length:"long" nullable:"true" update:"user"`
	RawPreflightResult  string                            `charset:"utf8" length:"long" nullable:"true" update:"user"`

	DatasetEvaluation    *api.LLMBenchmarkDatasetEvaluation `json:"dataset_evaluation,omitempty" length:"long" nullable:"true" list:"user" update:"user"`
	EvaluationResultJson string                             `charset:"utf8" length:"medium" nullable:"true" update:"user"`
	EvaluationResultCsv  string                             `charset:"utf8" length:"medium" nullable:"true" update:"user"`
	EvaluationLogPath    string                             `charset:"utf8" length:"medium" nullable:"true" update:"user"`

	ArtifactStorage        string `width:"32" charset:"ascii" nullable:"true" list:"user" update:"user"`
	ArtifactStorageMessage string `charset:"utf8" length:"medium" nullable:"true" list:"user" update:"user"`

	TargetSnapshot string `charset:"utf8" length:"long" nullable:"true" list:"user" create:"optional" update:"user"`
	GuideLLMSpec   string `charset:"utf8" length:"long" nullable:"true" list:"user" create:"optional" update:"user"`
	RawMetrics     string `charset:"utf8" length:"long" nullable:"true" list:"user" update:"user"`
	RawLog         string `charset:"utf8" length:"long" nullable:"true" update:"user"`
	RawCsv         string `charset:"utf8" length:"long" nullable:"true" update:"user"`

	RequestsPerSecondMean float64 `nullable:"true" list:"user" update:"user"`
	RequestLatencyMeanSec float64 `nullable:"true" list:"user" update:"user"`

	RequestTotal      int     `nullable:"true" list:"user" update:"user"`
	RequestSuccessful int     `nullable:"true" list:"user" update:"user"`
	RequestErrored    int     `nullable:"true" list:"user" update:"user"`
	ErrorRate         float64 `nullable:"true" list:"user" update:"user"`
}

func (man *SLLMBenchmarkManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, input *api.LLMBenchmarkCreateInput) (*api.LLMBenchmarkCreateInput, error) {
	var err error
	input.VirtualResourceCreateInput, err = man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.VirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate VirtualResourceCreateInput")
	}
	return prepareLLMBenchmarkCreateInput(ctx, userCred, ownerId, input)
}

func prepareLLMBenchmarkCreateInput(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.LLMBenchmarkCreateInput) (*api.LLMBenchmarkCreateInput, error) {
	llm, dep, err := resolveBenchmarkTarget(ctx, userCred, input)
	if err != nil {
		return input, err
	}
	sku, err := llm.GetLLMSku(llm.LLMSkuId)
	if err != nil {
		return input, errors.Wrap(err, "GetLLMSku")
	}
	image, err := resolveBenchmarkImage(ctx, userCred, input.BenchmarkImage)
	if err != nil {
		return input, err
	}
	pkg, err := resolveBenchmarkPackage(ctx, userCred, input.BenchmarkPackage)
	if err != nil {
		return input, err
	}
	if pkg != nil {
		benchmarkProjectID := ""
		if ownerId != nil {
			benchmarkProjectID = ownerId.GetProjectId()
		}
		if err := validateBenchmarkPackageProject(pkg.ProjectId, benchmarkProjectID); err != nil {
			return input, err
		}
	}
	access, err := llm.GetLLMAccessUrlInfo(ctx, userCred, jsonutils.NewDict())
	if err != nil {
		return input, errors.Wrap(err, "GetLLMAccessUrlInfo")
	}
	targetURL := ""
	if access != nil {
		targetURL = access.InternalUrl
		if targetURL == "" {
			targetURL = access.LoginUrl
		}
	}
	if targetURL == "" {
		return input, errors.Wrap(httperrors.ErrInvalidStatus, "target url is empty")
	}

	defaultLLMBenchmarkInput(input, pkg)
	if err := validateLLMBenchmarkInput(input, pkg != nil); err != nil {
		return input, err
	}
	if input.Model == "" {
		input.Model = resolveBenchmarkModel(ctx, targetURL, sku.LLMType, llm)
	}
	if pkg == nil {
		if _, err := resolveBenchmarkTokenizerMount(llm, input.Model); err != nil {
			return input, errors.Wrap(err, "resolve offline benchmark tokenizer")
		}
	}

	input.LLMId = llm.Id
	input.LLMSkuId = sku.Id
	input.LLMImageId = image.Id
	input.BenchmarkPackageId = ""
	if pkg != nil {
		input.BenchmarkPackageId = pkg.Id
	}
	input.LLMDeploymentId = llm.LLMDeploymentId
	if dep != nil {
		input.LLMDeploymentId = dep.Id
	}
	input.Backend = sku.LLMType
	input.TargetUrl = targetURL
	workDir := benchmarkWorkDirRoot()
	input.WorkDir = filepath.Join(workDir, input.Name)
	if !benchmarkWorkDirIsSafe(workDir, input.WorkDir) {
		return input, httperrors.NewInputParameterError("name produces unsafe benchmark workdir")
	}
	input.TargetSnapshot = jsonutils.Marshal(map[string]interface{}{
		"llm_id":               llm.Id,
		"llm_name":             llm.Name,
		"llm_status":           llm.Status,
		"llm_deployment_id":    input.LLMDeploymentId,
		"llm_sku_id":           sku.Id,
		"llm_sku_name":         sku.Name,
		"backend":              sku.LLMType,
		"llm_image_id":         image.Id,
		"llm_image_name":       image.Name,
		"benchmark_package_id": input.BenchmarkPackageId,
		"target_url":           targetURL,
		"request_format":       input.RequestFormat,
		"model":                input.Model,
		"request_rate":         input.RequestRate,
		"total_requests":       input.TotalRequests,
	}).String()
	spec := bench.BuildGuideLLMSpec(bench.GuideLLMSpecInput{
		TargetURL:           input.TargetUrl,
		RequestFormat:       input.RequestFormat,
		Model:               input.Model,
		RequestRate:         input.RequestRate,
		TotalRequests:       input.TotalRequests,
		MaxDurationSeconds:  input.MaxDurationSeconds,
		MaxErrors:           input.MaxErrors,
		DatasetInputTokens:  input.DatasetInputTokens,
		DatasetOutputTokens: input.DatasetOutputTokens,
		DatasetPath:         input.DatasetPath,
	})
	input.GuideLLMSpec = jsonutils.Marshal(spec).String()
	return input, nil
}

func benchmarkWorkDirRoot() string {
	if root := options.Options.LLMBenchmarkWorkDir; root != "" {
		return root
	}
	return "/opt/cloud/workspace/llm/benchmarks"
}

func benchmarkWorkDirIsSafe(root, path string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
	return err == nil && rel != "." && rel != ".." && !filepath.IsAbs(rel) &&
		!strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func resolveBenchmarkTarget(ctx context.Context, userCred mcclient.TokenCredential, input *api.LLMBenchmarkCreateInput) (*SLLM, *SLLMDeployment, error) {
	if input.LLMId != "" && input.LLMDeploymentId != "" {
		return nil, nil, errors.Wrap(httperrors.ErrInputParameter, "llm_id and llm_deployment_id are mutually exclusive")
	}
	if input.LLMDeploymentId != "" {
		depObj, err := GetLLMDeploymentManager().FetchByIdOrName(ctx, userCred, input.LLMDeploymentId)
		if err != nil {
			return nil, nil, errors.Wrap(err, "fetch LLMDeployment")
		}
		dep := depObj.(*SLLMDeployment)
		llm := &SLLM{}
		err = GetLLMManager().Query().
			Equals("llm_deployment_id", dep.Id).
			Equals("status", api.LLM_STATUS_RUNNING).
			Asc("created_at").
			First(llm)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, nil, errors.Wrapf(httperrors.ErrInvalidStatus, "deployment %s has no running llm", dep.Name)
			}
			return nil, nil, errors.Wrap(err, "query running deployment LLM")
		}
		llm.SetModelManager(GetLLMManager(), llm)
		return llm, dep, nil
	}
	if input.LLMId == "" {
		return nil, nil, errors.Wrap(httperrors.ErrMissingParameter, "llm_id or llm_deployment_id")
	}
	llmObj, err := GetLLMManager().FetchByIdOrName(ctx, userCred, input.LLMId)
	if err != nil {
		return nil, nil, errors.Wrap(err, "fetch LLM")
	}
	llm := llmObj.(*SLLM)
	if llm.Status != api.LLM_STATUS_RUNNING {
		return nil, nil, errors.Wrapf(httperrors.ErrInvalidStatus, "llm %s status is %s", llm.Name, llm.Status)
	}
	return llm, nil, nil
}

func resolveBenchmarkModel(ctx context.Context, targetURL string, llmType string, llm *SLLM) string {
	if providerType, ok := benchmarkProviderType(llmType); ok {
		out, err := GetLLMManager().performProviderModels(ctx, api.LLMProviderModelsInput{
			URL:          targetURL,
			ProviderType: providerType,
		})
		if err == nil {
			for _, model := range out.Models {
				if model = strings.TrimSpace(model); model != "" {
					return model
				}
			}
		}
	}
	if infos, err := llm.FetchMountedModelInfo(); err == nil && len(infos) > 0 {
		return upstreamModelKeyFromMountedInfo(llmType, &infos[0])
	}
	return ""
}

func benchmarkProviderType(llmType string) (api.LLMClientType, bool) {
	switch strings.ToLower(strings.TrimSpace(llmType)) {
	case string(api.LLM_CONTAINER_OLLAMA):
		return api.LLM_CLIENT_OLLAMA, true
	case string(api.LLM_CONTAINER_VLLM), string(api.LLM_CONTAINER_SGLANG):
		return api.LLM_CLIENT_OPENAI, true
	default:
		return "", false
	}
}

func defaultLLMBenchmarkInput(input *api.LLMBenchmarkCreateInput, pkg *SLLMBenchmarkPackage) {
	if input.RequestFormat == "" {
		input.RequestFormat = api.LLMBenchmarkDefaultRequestFormat
	}
	if input.Profile == "" {
		input.Profile = api.LLMBenchmarkProfileConstant
	}
	if pkg != nil {
		if input.DatasetName == "" || input.DatasetName == api.LLMBenchmarkDatasetSyntheticText {
			input.DatasetName = api.LLMBenchmarkDatasetPackage
		}
		if input.DatasetPath == "" {
			input.DatasetPath = pkg.DatasetPath
		}
	} else if input.DatasetName == "" {
		input.DatasetName = api.LLMBenchmarkDatasetSyntheticText
	}
	if input.RequestRate <= 0 {
		input.RequestRate = options.Options.LLMBenchmarkDefaultRequestRate
		if input.RequestRate <= 0 {
			input.RequestRate = 1
		}
	}
	if input.TotalRequests <= 0 {
		input.TotalRequests = options.Options.LLMBenchmarkDefaultTotalRequests
		if input.TotalRequests <= 0 {
			input.TotalRequests = 100
		}
	}
	if input.MaxDurationSeconds <= 0 {
		input.MaxDurationSeconds = 600
	}
	if input.MaxErrors <= 0 {
		input.MaxErrors = 10
	}
	if input.DatasetInputTokens <= 0 {
		input.DatasetInputTokens = options.Options.LLMBenchmarkDefaultInputTokens
		if input.DatasetInputTokens <= 0 {
			input.DatasetInputTokens = 1024
		}
	}
	if input.DatasetOutputTokens <= 0 {
		input.DatasetOutputTokens = options.Options.LLMBenchmarkDefaultOutputTokens
		if input.DatasetOutputTokens <= 0 {
			input.DatasetOutputTokens = 128
		}
	}
}

func validateLLMBenchmarkInput(input *api.LLMBenchmarkCreateInput, hasPackage bool) error {
	if input.RequestFormat != api.LLMBenchmarkDefaultRequestFormat {
		return errors.Wrap(httperrors.ErrInputParameter, "request_format only supports "+api.LLMBenchmarkDefaultRequestFormat)
	}
	if input.Profile != api.LLMBenchmarkProfileConstant {
		return errors.Wrap(httperrors.ErrInputParameter, "profile only supports constant")
	}
	if hasPackage {
		if input.DatasetName != api.LLMBenchmarkDatasetPackage {
			return errors.Wrap(httperrors.ErrInputParameter, "dataset_name must be benchmark_package when benchmark_package is set")
		}
		if strings.TrimSpace(input.DatasetPath) == "" {
			return httperrors.NewMissingParameterError("dataset_path")
		}
	} else if input.DatasetName != api.LLMBenchmarkDatasetSyntheticText {
		return errors.Wrap(httperrors.ErrInputParameter, "dataset_name only supports synthetic_text")
	}
	if max := options.Options.LLMBenchmarkMaxRequestRate; max > 0 && input.RequestRate > max {
		return errors.Wrapf(httperrors.ErrInputParameter, "request_rate must be <= %d", max)
	}
	if max := options.Options.LLMBenchmarkMaxTotalRequests; max > 0 && input.TotalRequests > max {
		return errors.Wrapf(httperrors.ErrInputParameter, "total_requests must be <= %d", max)
	}
	if max := options.Options.LLMBenchmarkMaxDurationSeconds; max > 0 && input.MaxDurationSeconds > max {
		return errors.Wrapf(httperrors.ErrInputParameter, "max_duration_seconds must be <= %d", max)
	}
	return nil
}

func validateBenchmarkMutableState(state string) error {
	if utils.IsInStringArray(state, []string{
		api.LLMBenchmarkStateCompleted,
		api.LLMBenchmarkStateStopped,
		api.LLMBenchmarkStateError,
	}) {
		return nil
	}
	return httperrors.NewInvalidStatusError("benchmark is %s", state)
}

func (b *SLLMBenchmark) ValidateUpdateCondition(ctx context.Context) error {
	return validateBenchmarkMutableState(b.State)
}

func positiveBenchmarkUpdate(name string, value *int) error {
	if value != nil && *value <= 0 {
		return httperrors.NewInputParameterError("%s must be greater than 0", name)
	}
	return nil
}

func (b *SLLMBenchmark) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.LLMBenchmarkUpdateInput) (api.LLMBenchmarkUpdateInput, error) {
	var err error
	input.VirtualResourceBaseUpdateInput, err = b.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, input.VirtualResourceBaseUpdateInput)
	if err != nil {
		return input, errors.Wrap(err, "ValidateUpdateData")
	}
	if err := validateBenchmarkMutableState(b.State); err != nil {
		return input, err
	}
	for name, value := range map[string]*int{
		"request_rate":          input.RequestRate,
		"total_requests":        input.TotalRequests,
		"max_duration_seconds":  input.MaxDurationSeconds,
		"max_errors":            input.MaxErrors,
		"dataset_input_tokens":  input.DatasetInputTokens,
		"dataset_output_tokens": input.DatasetOutputTokens,
	} {
		if err := positiveBenchmarkUpdate(name, value); err != nil {
			return input, err
		}
	}
	if b.BenchmarkPackageId != "" && (input.DatasetInputTokens != nil || input.DatasetOutputTokens != nil) {
		return input, httperrors.NewInputParameterError("dataset token fields only apply to synthetic_text")
	}
	candidate := benchmarkCreateInputFromModel(b)
	applyLLMBenchmarkUpdateInput(candidate, input)
	if err := validateLLMBenchmarkInput(candidate, b.BenchmarkPackageId != ""); err != nil {
		return input, err
	}
	if b.BenchmarkPackageId == "" && benchmarkUpdateChangesRunConfig(input) {
		obj, err := GetLLMManager().FetchById(b.LLMId)
		if err != nil {
			return input, errors.Wrap(err, "fetch benchmark LLM")
		}
		if _, err := resolveBenchmarkTokenizerMount(obj.(*SLLM), candidate.Model); err != nil {
			return input, errors.Wrap(err, "resolve offline benchmark tokenizer")
		}
	}
	return input, nil
}

func benchmarkCreateInputFromModel(b *SLLMBenchmark) *api.LLMBenchmarkCreateInput {
	ret := &api.LLMBenchmarkCreateInput{
		LLMId:               b.LLMId,
		LLMDeploymentId:     b.LLMDeploymentId,
		LLMImageId:          b.LLMImageId,
		BenchmarkPackageId:  b.BenchmarkPackageId,
		RequestFormat:       b.RequestFormat,
		Model:               b.Model,
		Profile:             b.Profile,
		RequestRate:         b.RequestRate,
		TotalRequests:       b.TotalRequests,
		MaxDurationSeconds:  b.MaxDurationSeconds,
		MaxErrors:           b.MaxErrors,
		DatasetName:         b.DatasetName,
		DatasetInputTokens:  b.DatasetInputTokens,
		DatasetOutputTokens: b.DatasetOutputTokens,
		DatasetPath:         b.DatasetPath,
		Backend:             b.Backend,
		TargetUrl:           b.TargetUrl,
		WorkDir:             b.WorkDir,
		TargetSnapshot:      b.TargetSnapshot,
		GuideLLMSpec:        b.GuideLLMSpec,
	}
	ret.Name = b.Name
	ret.Description = b.Description
	return ret
}

func applyLLMBenchmarkUpdateInput(candidate *api.LLMBenchmarkCreateInput, input api.LLMBenchmarkUpdateInput) {
	if input.Model != nil {
		candidate.Model = strings.TrimSpace(*input.Model)
	}
	if input.RequestRate != nil {
		candidate.RequestRate = *input.RequestRate
	}
	if input.TotalRequests != nil {
		candidate.TotalRequests = *input.TotalRequests
	}
	if input.MaxDurationSeconds != nil {
		candidate.MaxDurationSeconds = *input.MaxDurationSeconds
	}
	if input.MaxErrors != nil {
		candidate.MaxErrors = *input.MaxErrors
	}
	if input.DatasetInputTokens != nil {
		candidate.DatasetInputTokens = *input.DatasetInputTokens
	}
	if input.DatasetOutputTokens != nil {
		candidate.DatasetOutputTokens = *input.DatasetOutputTokens
	}
}

func benchmarkUpdateChangesRunConfig(input api.LLMBenchmarkUpdateInput) bool {
	return input.Model != nil ||
		input.RequestRate != nil ||
		input.TotalRequests != nil ||
		input.MaxDurationSeconds != nil ||
		input.MaxErrors != nil ||
		input.DatasetInputTokens != nil ||
		input.DatasetOutputTokens != nil
}

func buildLLMBenchmarkUpdateData(b *SLLMBenchmark, input api.LLMBenchmarkUpdateInput) *jsonutils.JSONDict {
	if !benchmarkUpdateChangesRunConfig(input) {
		return jsonutils.NewDict()
	}
	candidate := benchmarkCreateInputFromModel(b)
	applyLLMBenchmarkUpdateInput(candidate, input)

	spec := bench.BuildGuideLLMSpec(bench.GuideLLMSpecInput{
		TargetURL:           candidate.TargetUrl,
		RequestFormat:       candidate.RequestFormat,
		Model:               candidate.Model,
		RequestRate:         candidate.RequestRate,
		TotalRequests:       candidate.TotalRequests,
		MaxDurationSeconds:  candidate.MaxDurationSeconds,
		MaxErrors:           candidate.MaxErrors,
		DatasetInputTokens:  candidate.DatasetInputTokens,
		DatasetOutputTokens: candidate.DatasetOutputTokens,
		DatasetPath:         candidate.DatasetPath,
	})
	snapshot, _ := jsonutils.ParseString(b.TargetSnapshot)
	snapshotDict, ok := snapshot.(*jsonutils.JSONDict)
	if !ok {
		snapshotDict = jsonutils.NewDict()
	}
	snapshotDict.Set("model", jsonutils.NewString(candidate.Model))
	snapshotDict.Set("request_rate", jsonutils.NewInt(int64(candidate.RequestRate)))
	snapshotDict.Set("total_requests", jsonutils.NewInt(int64(candidate.TotalRequests)))

	data := jsonutils.NewDict()
	if input.Model != nil {
		data.Set("model", jsonutils.NewString(candidate.Model))
	}
	if input.RequestRate != nil {
		data.Set("request_rate", jsonutils.NewInt(int64(candidate.RequestRate)))
	}
	if input.TotalRequests != nil {
		data.Set("total_requests", jsonutils.NewInt(int64(candidate.TotalRequests)))
	}
	if input.MaxDurationSeconds != nil {
		data.Set("max_duration_seconds", jsonutils.NewInt(int64(candidate.MaxDurationSeconds)))
	}
	if input.MaxErrors != nil {
		data.Set("max_errors", jsonutils.NewInt(int64(candidate.MaxErrors)))
	}
	if input.DatasetInputTokens != nil {
		data.Set("dataset_input_tokens", jsonutils.NewInt(int64(candidate.DatasetInputTokens)))
	}
	if input.DatasetOutputTokens != nil {
		data.Set("dataset_output_tokens", jsonutils.NewInt(int64(candidate.DatasetOutputTokens)))
	}
	data.Set("guide_llm_spec", jsonutils.NewString(jsonutils.Marshal(spec).String()))
	data.Set("target_snapshot", jsonutils.NewString(snapshotDict.String()))
	data.Set("state", jsonutils.NewString(api.LLMBenchmarkStateStopped))
	for _, field := range []string{
		"state_message", "task_id", "runner_server_id", "runner_container_id",
		"log_path", "result_json", "result_csv", "raw_metrics", "raw_log", "raw_csv",
		"preflight_log_path", "preflight_result_json",
		"raw_preflight_log", "raw_preflight_result",
		"evaluation_result_json", "evaluation_result_csv", "evaluation_log_path",
		"artifact_storage", "artifact_storage_message",
	} {
		data.Set(field, jsonutils.NewString(""))
	}
	data.Set("dataset_preflight", jsonutils.JSONNull)
	data.Set("dataset_evaluation", jsonutils.JSONNull)
	data.Set("stop_requested", jsonutils.JSONFalse)
	for _, field := range []string{
		"requests_per_second_mean", "request_latency_mean_sec", "error_rate",
	} {
		data.Set(field, jsonutils.NewFloat64(0))
	}
	for _, field := range []string{
		"request_total", "request_successful", "request_errored",
	} {
		data.Set(field, jsonutils.NewInt(0))
	}
	return data
}

func sanitizeLLMBenchmarkUpdateData(data *jsonutils.JSONDict) {
	for _, field := range []string{
		"state", "state_message", "task_id", "stop_requested",
		"runner_server_id", "runner_container_id",
		"log_path", "result_json", "result_csv",
		"dataset_preflight", "preflight_log_path", "preflight_result_json",
		"raw_preflight_log", "raw_preflight_result",
		"dataset_evaluation", "evaluation_result_json", "evaluation_result_csv", "evaluation_log_path",
		"artifact_storage", "artifact_storage_message",
		"target_snapshot", "guide_llm_spec", "raw_metrics", "raw_log", "raw_csv",
		"requests_per_second_mean", "request_latency_mean_sec",
		"request_total", "request_successful", "request_errored", "error_rate",
	} {
		data.RemoveIgnoreCase(field)
	}
}

func (b *SLLMBenchmark) PreUpdate(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	dataDict := data.(*jsonutils.JSONDict)
	input := api.LLMBenchmarkUpdateInput{}
	_ = dataDict.Unmarshal(&input)
	if benchmarkUpdateChangesRunConfig(input) {
		if err := b.CleanupArtifacts(ctx); err != nil {
			log.Warningf("cleanup benchmark %s artifacts before update: %s", b.Id, err)
		}
	}
	sanitizeLLMBenchmarkUpdateData(dataDict)
	dataDict.Update(buildLLMBenchmarkUpdateData(b, input))
	b.SVirtualResourceBase.PreUpdate(ctx, userCred, query, data)
}

func resolveBenchmarkPackage(ctx context.Context, userCred mcclient.TokenCredential, packageName string) (*SLLMBenchmarkPackage, error) {
	if packageName == "" {
		return nil, nil
	}
	obj, err := GetLLMBenchmarkPackageManager().FetchByIdOrName(ctx, userCred, packageName)
	if err != nil {
		return nil, errors.Wrap(err, "fetch benchmark package")
	}
	pkg := obj.(*SLLMBenchmarkPackage)
	if pkg.ImageId == "" {
		return nil, httperrors.NewInvalidStatusError("benchmark package %s has no image", pkg.Name)
	}
	if pkg.Status != imageapi.IMAGE_STATUS_ACTIVE {
		return nil, httperrors.NewInvalidStatusError("benchmark package %s is %s", pkg.Name, pkg.Status)
	}
	return pkg, nil
}

func PrepareLLMBenchmarkCreateInput(pkg *SLLMBenchmarkPackage, spec *api.LLMBenchmarkCreateInput) (*api.LLMBenchmarkCreateInput, error) {
	if spec == nil {
		return nil, nil
	}
	if pkg == nil || pkg.Id == "" {
		return nil, errors.Wrap(httperrors.ErrInputParameter, "empty benchmark package")
	}
	if pkg.Status != imageapi.IMAGE_STATUS_ACTIVE || pkg.ImageId == "" {
		return nil, httperrors.NewInvalidStatusError("benchmark package %s is not active", pkg.Name)
	}

	input := *spec
	input.BenchmarkPackage = pkg.Id
	input.BenchmarkPackageId = pkg.Id
	input.DatasetName = api.LLMBenchmarkDatasetPackage
	input.DatasetPath = pkg.DatasetPath
	return &input, nil
}

func validateBenchmarkPackageUnusedCount(count int) error {
	if count > 0 {
		return errors.Wrap(httperrors.ErrInvalidStatus, "benchmark package is still used by benchmarks")
	}
	return nil
}

func validateBenchmarkPackageProject(packageProjectID, benchmarkProjectID string) error {
	if packageProjectID != benchmarkProjectID {
		return httperrors.NewInputParameterError("benchmark package and benchmark must belong to the same project")
	}
	return nil
}

func CountBenchmarkPackageReferences(packageID, excludeBenchmarkID string) (int, error) {
	if packageID == "" {
		return 0, nil
	}
	q := GetLLMBenchmarkManager().Query().
		Equals("benchmark_package_id", packageID).
		IsFalse("deleted")
	if excludeBenchmarkID != "" {
		q = q.NotEquals("id", excludeBenchmarkID)
	}
	count, err := q.CountWithError()
	if err != nil {
		return 0, errors.Wrap(err, "count benchmark package references")
	}
	return count, nil
}

func ValidateBenchmarkPackageUnused(packageID, excludeBenchmarkID string) error {
	count, err := CountBenchmarkPackageReferences(packageID, excludeBenchmarkID)
	if err != nil {
		return err
	}
	return validateBenchmarkPackageUnusedCount(count)
}

func buildLLMBenchmarkCopyCreateInput(source *SLLMBenchmark, input api.LLMBenchmarkCopyInput) (*api.LLMBenchmarkCreateInput, error) {
	if strings.TrimSpace(input.Name) == "" {
		return nil, httperrors.NewMissingParameterError("name")
	}
	if strings.TrimSpace(input.LLMDeploymentId) == "" {
		return nil, httperrors.NewMissingParameterError("llm_deployment_id")
	}
	if source.BenchmarkPackageId != "" && (input.DatasetInputTokens != nil || input.DatasetOutputTokens != nil) {
		return nil, httperrors.NewInputParameterError("dataset token fields only apply to synthetic_text")
	}
	ret := &api.LLMBenchmarkCreateInput{
		LLMDeploymentId:     input.LLMDeploymentId,
		BenchmarkImage:      source.LLMImageId,
		BenchmarkPackage:    source.BenchmarkPackageId,
		RequestFormat:       source.RequestFormat,
		Profile:             source.Profile,
		RequestRate:         source.RequestRate,
		TotalRequests:       source.TotalRequests,
		MaxDurationSeconds:  source.MaxDurationSeconds,
		MaxErrors:           source.MaxErrors,
		DatasetName:         source.DatasetName,
		DatasetInputTokens:  source.DatasetInputTokens,
		DatasetOutputTokens: source.DatasetOutputTokens,
		DatasetPath:         source.DatasetPath,
	}
	ret.Name = input.Name
	ret.Description = source.Description
	if input.Description != nil {
		ret.Description = *input.Description
	}
	if input.Model != nil {
		ret.Model = strings.TrimSpace(*input.Model)
	}
	if input.RequestRate != nil {
		ret.RequestRate = *input.RequestRate
	}
	if input.TotalRequests != nil {
		ret.TotalRequests = *input.TotalRequests
	}
	if input.MaxDurationSeconds != nil {
		ret.MaxDurationSeconds = *input.MaxDurationSeconds
	}
	if input.MaxErrors != nil {
		ret.MaxErrors = *input.MaxErrors
	}
	if input.DatasetInputTokens != nil {
		ret.DatasetInputTokens = *input.DatasetInputTokens
	}
	if input.DatasetOutputTokens != nil {
		ret.DatasetOutputTokens = *input.DatasetOutputTokens
	}
	return ret, nil
}

func resolveBenchmarkImage(ctx context.Context, userCred mcclient.TokenCredential, imageName string) (*SLLMImage, error) {
	var image *SLLMImage
	if imageName != "" {
		obj, err := GetLLMImageManager().FetchByIdOrName(ctx, userCred, imageName)
		if err != nil {
			return nil, errors.Wrap(err, "fetch benchmark image")
		}
		image = obj.(*SLLMImage)
	} else {
		defaultImage := options.Options.LLMBenchmarkDefaultImage
		if defaultImage == "" {
			defaultImage = api.LLMBenchmarkDefaultImage
		}
		name, label := parseImageRef(defaultImage)
		image = &SLLMImage{}
		err := GetLLMImageManager().Query().
			Equals("image_name", name).
			Equals("image_label", label).
			Equals("llm_type", string(api.LLM_IMAGE_TYPE_BENCHMARK)).
			First(image)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewNotFoundError("default benchmark llm_image %s not found, create it first", defaultImage)
			}
			return nil, errors.Wrap(err, "query default benchmark image")
		}
		image.SetModelManager(GetLLMImageManager(), image)
	}
	if image.LLMType != string(api.LLM_IMAGE_TYPE_BENCHMARK) {
		return nil, errors.Wrapf(httperrors.ErrInputParameter, "image %s is not benchmark type", image.Name)
	}
	return image, nil
}

func parseImageRef(ref string) (string, string) {
	idx := strings.LastIndex(ref, ":")
	if idx <= 0 {
		return ref, "latest"
	}
	return ref[:idx], ref[idx+1:]
}

func (man *SLLMBenchmarkManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.LLMBenchmarkListInput) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.VirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SVirtualResourceBaseManager.ListItemFilter")
	}
	if input.LLMId != "" {
		q = q.Equals("llm_id", input.LLMId)
	}
	if input.LLMDeploymentId != "" {
		depObj, err := GetLLMDeploymentManager().FetchByIdOrName(ctx, userCred, input.LLMDeploymentId)
		if err != nil {
			return nil, errors.Wrap(err, "fetch LLMDeployment")
		}
		q = q.Equals("llm_deployment_id", depObj.GetId())
	}
	if input.State != "" {
		q = q.Equals("state", input.State)
	}
	return q, nil
}

func (man *SLLMBenchmarkManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	if !isLLMBenchmarkDeploymentExtraField(field) {
		return q, httperrors.ErrNotFound
	}
	depQ := GetLLMDeploymentManager().Query("id", "name").Distinct().SubQuery()
	q.AppendField(depQ.Field("name", field))
	q = q.Join(depQ, sqlchemy.Equals(q.Field("llm_deployment_id"), depQ.Field("id")))
	q.GroupBy(depQ.Field("name"))
	return q, nil
}

func isLLMBenchmarkDeploymentExtraField(field string) bool {
	return field == "llm_deployment"
}

func (man *SLLMBenchmarkManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.LLMBenchmarkDetails {
	virtRows := man.SVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	benchmarks := make([]SLLMBenchmark, len(objs))
	jsonutils.Update(&benchmarks, objs)

	rows := make([]api.LLMBenchmarkDetails, len(objs))
	deploymentIds := make([]string, 0, len(objs))
	for i := range rows {
		rows[i].VirtualResourceDetails = virtRows[i]
		if benchmarks[i].LLMDeploymentId != "" {
			deploymentIds = append(deploymentIds, benchmarks[i].LLMDeploymentId)
		}
	}
	if len(deploymentIds) == 0 {
		return rows
	}

	deploymentNames, err := db.FetchIdNameMap2(GetLLMDeploymentManager(), deploymentIds)
	if err != nil {
		return rows
	}
	for i := range rows {
		rows[i].LLMDeployment = deploymentNames[benchmarks[i].LLMDeploymentId]
	}
	return rows
}

func (man *SLLMBenchmarkManager) CreateAndStart(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, input *api.LLMBenchmarkCreateInput) (*SLLMBenchmark, error) {
	data := jsonutils.Marshal(input)
	obj, err := db.DoCreate(man, ctx, userCred, nil, data, ownerId)
	if err != nil {
		return nil, errors.Wrap(err, "DoCreate benchmark")
	}
	benchmark := obj.(*SLLMBenchmark)
	func() {
		lockman.LockObject(ctx, benchmark)
		defer lockman.ReleaseObject(ctx, benchmark)
		benchmark.PostCreate(ctx, userCred, ownerId, nil, data)
		if err := man.GetExtraHook().AfterPostCreate(ctx, userCred, ownerId, benchmark, nil, data); err != nil {
			logclient.AddActionLogWithContext(ctx, benchmark, logclient.ACT_POST_CREATE_HOOK, err, userCred, false)
		}
	}()
	notes := benchmark.GetShortDesc(ctx)
	db.OpsLog.LogEvent(benchmark, db.ACT_CREATE, notes, userCred)
	logclient.AddActionLogWithContext(ctx, benchmark, logclient.ACT_CREATE, notes, userCred, true)
	man.OnCreateComplete(ctx, []db.IModel{benchmark}, userCred, ownerId, nil, []jsonutils.JSONObject{data})
	return benchmark, nil
}

func (b *SLLMBenchmark) PerformCopy(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.LLMBenchmarkCopyInput) (jsonutils.JSONObject, error) {
	createInput, err := buildLLMBenchmarkCopyCreateInput(b, input)
	if err != nil {
		return nil, err
	}
	benchmark, err := GetLLMBenchmarkManager().CreateAndStart(ctx, userCred, b.GetOwnerId(), createInput)
	if err != nil {
		return nil, err
	}
	ret := jsonutils.NewDict()
	ret.Set("benchmark_id", jsonutils.NewString(benchmark.Id))
	ret.Set("state", jsonutils.NewString(benchmark.State))
	return ret, nil
}

func (b *SLLMBenchmark) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	b.SVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	if err := b.StartRunTask(ctx, userCred, ""); err != nil {
		_ = b.SetState(ctx, userCred, api.LLMBenchmarkStateError, err.Error())
	}
}

func (b *SLLMBenchmark) SetState(ctx context.Context, userCred mcclient.TokenCredential, state string, message string) error {
	_, err := db.Update(b, func() error {
		b.State = state
		b.StateMessage = message
		return nil
	})
	return err
}

func (b *SLLMBenchmark) FinishRun(ctx context.Context, runErr error) (string, error) {
	lockman.LockObject(ctx, b)
	defer lockman.ReleaseObject(ctx, b)

	obj, err := GetLLMBenchmarkManager().FetchById(b.Id)
	if err != nil {
		return "", err
	}
	current := obj.(*SLLMBenchmark)
	state, message := benchmarkRunFinalState(current.State, current.StopRequested, runErr)
	_, err = db.Update(current, func() error {
		current.State = state
		current.StateMessage = message
		return nil
	})
	if err == nil {
		b.State = state
		b.StateMessage = message
		b.StopRequested = current.StopRequested
	}
	return state, err
}

func (b *SLLMBenchmark) SetRunner(ctx context.Context, serverId string, containerId string) error {
	_, err := db.Update(b, func() error {
		b.RunnerServerId = serverId
		b.RunnerContainerId = containerId
		return nil
	})
	return err
}

func resetLLMBenchmarkResultFields(b *SLLMBenchmark) {
	b.StateMessage = ""
	b.TaskId = ""
	b.StopRequested = false
	b.RunnerServerId = ""
	b.RunnerContainerId = ""
	b.LogPath = ""
	b.ResultJson = ""
	b.ResultCsv = ""
	b.RawMetrics = ""
	b.RawLog = ""
	b.RawCsv = ""
	b.DatasetPreflight = nil
	b.PreflightLogPath = ""
	b.PreflightResultJson = ""
	b.RawPreflightLog = ""
	b.RawPreflightResult = ""
	b.DatasetEvaluation = nil
	b.EvaluationResultJson = ""
	b.EvaluationResultCsv = ""
	b.EvaluationLogPath = ""
	b.ArtifactStorage = ""
	b.ArtifactStorageMessage = ""
	b.RequestsPerSecondMean = 0
	b.RequestLatencyMeanSec = 0
	b.RequestTotal = 0
	b.RequestSuccessful = 0
	b.RequestErrored = 0
	b.ErrorRate = 0
}

func (b *SLLMBenchmark) StartRunTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LLMBenchmarkRunTask", b, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	if _, err := db.Update(b, func() error {
		b.TaskId = task.GetId()
		b.State = api.LLMBenchmarkStatePending
		return nil
	}); err != nil {
		return err
	}
	return task.ScheduleRun(nil)
}

func (b *SLLMBenchmark) UpdateMetrics(ctx context.Context, userCred mcclient.TokenCredential, metrics *bench.LLMBenchmarkMetrics) error {
	_, err := db.Update(b, func() error {
		b.RequestsPerSecondMean = floatValue(metrics.RequestsPerSecondMean)
		b.RequestLatencyMeanSec = floatValue(metrics.RequestLatencyMeanSec)
		b.RequestTotal = metrics.RequestTotal
		b.RequestSuccessful = metrics.RequestSuccessful
		b.RequestErrored = metrics.RequestErrored
		b.ErrorRate = floatValue(metrics.ErrorRate)
		return nil
	})
	return err
}

func (b *SLLMBenchmark) UpdateDatasetPreflight(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	summary api.LLMBenchmarkDatasetPreflight,
	resultPath, logPath string,
) error {
	_, err := db.Update(b, func() error {
		b.DatasetPreflight = &summary
		b.PreflightResultJson = resultPath
		b.PreflightLogPath = logPath
		return nil
	})
	return err
}

func (b *SLLMBenchmark) UpdateDatasetEvaluation(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	summary api.LLMBenchmarkDatasetEvaluation,
	resultJSON, resultCSV, logPath string,
) error {
	_, err := db.Update(b, func() error {
		b.DatasetEvaluation = &summary
		b.EvaluationResultJson = resultJSON
		b.EvaluationResultCsv = resultCSV
		b.EvaluationLogPath = logPath
		return nil
	})
	return err
}

func floatValue(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func (b *SLLMBenchmark) ArtifactPath(kind string) (string, error) {
	switch kind {
	case "preflight":
		return b.PreflightResultJson, nil
	case "preflight-log":
		return b.PreflightLogPath, nil
	case "log":
		return b.LogPath, nil
	case "json":
		return b.ResultJson, nil
	case "csv":
		return b.ResultCsv, nil
	case "evaluation":
		return b.EvaluationResultJson, nil
	case "evaluation-csv":
		return b.EvaluationResultCsv, nil
	case "evaluation-log":
		return b.EvaluationLogPath, nil
	default:
		return "", httperrors.NewInputParameterError("unknown artifact type %s", kind)
	}
}

func (b *SLLMBenchmark) ArtifactLocations() map[string]string {
	return map[string]string{
		"preflight":      b.PreflightResultJson,
		"preflight-log":  b.PreflightLogPath,
		"log":            b.LogPath,
		"json":           b.ResultJson,
		"csv":            b.ResultCsv,
		"evaluation":     b.EvaluationResultJson,
		"evaluation-csv": b.EvaluationResultCsv,
		"evaluation-log": b.EvaluationLogPath,
	}
}

func applyLLMBenchmarkArtifactLocations(b *SLLMBenchmark, locations map[string]string) {
	b.PreflightResultJson = locations["preflight"]
	b.PreflightLogPath = locations["preflight-log"]
	b.LogPath = locations["log"]
	b.ResultJson = locations["json"]
	b.ResultCsv = locations["csv"]
	b.EvaluationResultJson = locations["evaluation"]
	b.EvaluationResultCsv = locations["evaluation-csv"]
	b.EvaluationLogPath = locations["evaluation-log"]
}

func (b *SLLMBenchmark) UpdateArtifactLocations(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	locations map[string]string,
	storage, message string,
) error {
	_, err := db.Update(b, func() error {
		applyLLMBenchmarkArtifactLocations(b, locations)
		b.ArtifactStorage = storage
		b.ArtifactStorageMessage = message
		return nil
	})
	return err
}

func (b *SLLMBenchmark) CleanupArtifacts(ctx context.Context) error {
	var firstErr error
	if err := bench.DefaultArtifactStore().DeleteBenchmark(ctx, b.ProjectId, b.Id); err != nil {
		firstErr = err
	}
	if b.WorkDir == "" {
		return firstErr
	}
	if !benchmarkWorkDirIsSafe(benchmarkWorkDirRoot(), b.WorkDir) {
		if firstErr != nil {
			return firstErr
		}
		return errors.Errorf("unsafe benchmark workdir %s", b.WorkDir)
	}
	if err := os.RemoveAll(b.WorkDir); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

func (b *SLLMBenchmark) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if utils.IsInStringArray(b.State, []string{api.LLMBenchmarkStatePending, api.LLMBenchmarkStateQueued, api.LLMBenchmarkStateValidating, api.LLMBenchmarkStateRunning}) {
		return httperrors.NewInvalidStatusError("benchmark is %s, stop it first", b.State)
	}
	return nil
}

func (b *SLLMBenchmark) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	return b.StartDeleteTask(ctx, userCred, "")
}

func (b *SLLMBenchmark) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, parentTaskId string) error {
	task, err := taskman.TaskManager.NewTask(ctx, "LLMBenchmarkDeleteTask", b, userCred, nil, parentTaskId, "", nil)
	if err != nil {
		return errors.Wrap(err, "NewTask LLMBenchmarkDeleteTask")
	}
	return task.ScheduleRun(nil)
}

func (b *SLLMBenchmark) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (b *SLLMBenchmark) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return b.SVirtualResourceBase.Delete(ctx, userCred)
}

func (b *SLLMBenchmark) PerformRetest(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input api.LLMBenchmarkRetestInput) (jsonutils.JSONObject, error) {
	if err := validateBenchmarkMutableState(b.State); err != nil {
		return nil, err
	}

	createInput := benchmarkCreateInputFromModel(b)
	createInput.BenchmarkImage = b.LLMImageId
	createInput.BenchmarkPackage = b.BenchmarkPackageId
	if b.LLMDeploymentId != "" {
		createInput.LLMId = ""
	}
	prepared, err := prepareLLMBenchmarkCreateInput(ctx, userCred, b.GetOwnerId(), createInput)
	if err != nil {
		return nil, err
	}

	if b.RunnerServerId != "" {
		if err := b.DeleteRunnerServer(ctx, userCred); err != nil {
			return nil, errors.Wrap(err, "delete old benchmark runner")
		}
	}
	if err := b.CleanupArtifacts(ctx); err != nil {
		log.Warningf("cleanup benchmark %s artifacts before retest: %s", b.Id, err)
	}

	_, err = db.Update(b, func() error {
		b.LLMId = prepared.LLMId
		b.LLMDeploymentId = prepared.LLMDeploymentId
		b.LLMSkuId = prepared.LLMSkuId
		b.LLMImageId = prepared.LLMImageId
		b.BenchmarkPackageId = prepared.BenchmarkPackageId
		b.Backend = prepared.Backend
		b.TargetUrl = prepared.TargetUrl
		b.Model = prepared.Model
		b.WorkDir = prepared.WorkDir
		b.TargetSnapshot = prepared.TargetSnapshot
		b.GuideLLMSpec = prepared.GuideLLMSpec
		resetLLMBenchmarkResultFields(b)
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "reset benchmark")
	}
	if err := b.StartRunTask(ctx, userCred, ""); err != nil {
		_ = b.SetState(ctx, userCred, api.LLMBenchmarkStateError, err.Error())
		return nil, err
	}
	return nil, nil
}

func requestLLMBenchmarkStop(b *SLLMBenchmark) {
	b.StopRequested = true
}

func (b *SLLMBenchmark) PerformStop(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	obj, err := GetLLMBenchmarkManager().FetchById(b.Id)
	if err != nil {
		return nil, err
	}
	current := obj.(*SLLMBenchmark)
	if !utils.IsInStringArray(current.State, []string{api.LLMBenchmarkStatePending, api.LLMBenchmarkStateQueued, api.LLMBenchmarkStateValidating, api.LLMBenchmarkStateRunning}) {
		return nil, httperrors.NewInvalidStatusError("benchmark is %s", current.State)
	}
	_, err = db.Update(current, func() error {
		requestLLMBenchmarkStop(current)
		return nil
	})
	if err != nil {
		return nil, err
	}
	stopLLMBenchmarkAsync(current.Id, userCred)
	return nil, nil
}

func stopLLMBenchmarkAsync(benchmarkId string, userCred mcclient.TokenCredential) {
	go func() {
		ctx := context.Background()
		obj, err := GetLLMBenchmarkManager().FetchById(benchmarkId)
		if err != nil {
			log.Warningf("fetch benchmark %s for stop: %s", benchmarkId, err)
			return
		}
		benchmark := obj.(*SLLMBenchmark)
		if err := benchmark.DeleteRunnerServer(ctx, userCred); err != nil {
			log.Warningf("delete benchmark %s runner: %s", benchmarkId, err)
		}
	}()
}

func (b *SLLMBenchmark) DeleteRunnerServer(ctx context.Context, userCred mcclient.TokenCredential) error {
	if b.RunnerServerId == "" {
		return nil
	}
	s := auth.GetSession(ctx, userCred, options.Options.Region)
	_, _ = compute.Servers.Update(s, b.RunnerServerId, jsonutils.Marshal(map[string]interface{}{
		"disable_delete": false,
	}))
	_, err := compute.Servers.DeleteWithParam(s, b.RunnerServerId, jsonutils.Marshal(map[string]interface{}{
		"override_pending_delete": true,
	}), nil)
	if err != nil && !isBenchmarkRunnerNotFound(err) {
		return err
	}
	_, err = db.Update(b, func() error {
		b.RunnerServerId = ""
		b.RunnerContainerId = ""
		return nil
	})
	return err
}

func isBenchmarkRunnerNotFound(err error) bool {
	var clientErr *httputils.JSONClientError
	return stderrors.As(err, &clientErr) && clientErr.Code == 404
}

func (b *SLLMBenchmark) ResolveTokenizerMount() (*LLMBenchmarkTokenizerMount, error) {
	obj, err := GetLLMManager().FetchById(b.LLMId)
	if err != nil {
		return nil, errors.Wrap(err, "fetch benchmark LLM")
	}
	return resolveBenchmarkTokenizerMount(obj.(*SLLM), b.Model)
}

func (b *SLLMBenchmark) RunnerPodInput(image string, server *computeapi.ServerDetails, tokenizer *LLMBenchmarkTokenizerMount) (*computeapi.ServerCreateInput, error) {
	if len(server.Nics) == 0 || server.Nics[0].NetworkId == "" {
		return nil, errors.Wrap(httperrors.ErrInvalidStatus, "target server network is empty")
	}
	cpu := options.Options.LLMBenchmarkRunnerCPU
	if cpu <= 0 {
		cpu = 1
	}
	mem := options.Options.LLMBenchmarkRunnerMemoryMB
	if mem <= 0 {
		mem = 2048
	}
	input := bench.RunnerPodInput{
		Name:      "llm-bench-" + b.Id,
		Image:     image,
		NetworkId: server.Nics[0].NetworkId,
		HostId:    server.HostId,
		CPU:       cpu,
		MemoryMB:  mem,
	}
	if b.BenchmarkPackageId != "" {
		obj, err := GetLLMBenchmarkPackageManager().FetchById(b.BenchmarkPackageId)
		if err != nil {
			return nil, errors.Wrap(err, "fetch benchmark package")
		}
		pkg := obj.(*SLLMBenchmarkPackage)
		input.PackageImageId = pkg.ImageId
		input.PackageMountBase = api.LLMBenchmarkPackageMountBase
		input.PackageSizeMB = int(pkg.ActualSizeMb) + 512
	}
	if tokenizer != nil {
		input.ModelImageId = tokenizer.ImageId
		input.ModelSizeMB = tokenizer.SizeMB
		input.ModelMountBase = tokenizer.MountBase
		input.ModelMountSubdirectory = tokenizer.MountSubdirectory
	}
	return bench.BuildRunnerPodInput(input), nil
}
