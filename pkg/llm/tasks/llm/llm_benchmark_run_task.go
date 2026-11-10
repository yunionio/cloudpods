package llm

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	bench "yunion.io/x/onecloud/pkg/llm/benchmark"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/llm/options"
	llmutils "yunion.io/x/onecloud/pkg/llm/utils"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	computemod "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type LLMBenchmarkRunTask struct {
	taskman.STask
}

const (
	preflightRemoteJSON = "/workdir/dataset-preflight.json"
	preflightRemoteCSV  = "/workdir/dataset-preflight.csv"
	preflightRemoteLog  = "/workdir/dataset-preflight.log"
)

const (
	evaluationDatasetRemote = "/workdir/evaluation-dataset.jsonl"
	evaluationDatasetLocal  = "evaluation-dataset.jsonl"
)

func init() {
	taskman.RegisterTask(LLMBenchmarkRunTask{})
}

func applyBenchmarkRuntimeTokenizer(spec *bench.GuideLLMSpec, mount *models.LLMBenchmarkTokenizerMount) {
	if spec == nil || mount == nil {
		return
	}
	spec.Tokenizer = bench.GuideLLMLocalTokenizer(mount.ModelPath)
}

func (task *LLMBenchmarkRunTask) OnInit(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	b := obj.(*models.SLLMBenchmark)
	if stoppedBenchmark(b) {
		_, _ = b.FinishRun(ctx, nil)
		task.SetStageComplete(ctx, nil)
		return
	}
	_ = b.SetState(ctx, task.UserCred, api.LLMBenchmarkStateQueued, "")
	task.SetStage("OnRunComplete", nil)
	taskman.LocalTaskRun(task, func() (jsonutils.JSONObject, error) {
		defer cleanupRunner(ctx, task.UserCred, b)
		err := task.run(ctx, b)
		persistErr := task.persistBenchmarkArtifacts(ctx, b)
		if err == nil {
			err = persistErr
		}
		return nil, err
	})
}

func (task *LLMBenchmarkRunTask) OnRunComplete(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	b := obj.(*models.SLLMBenchmark)
	if _, err := b.FinishRun(ctx, nil); err != nil {
		task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	task.SetStageComplete(ctx, nil)
}

func (task *LLMBenchmarkRunTask) OnRunCompleteFailed(ctx context.Context, obj db.IStandaloneModel, body jsonutils.JSONObject) {
	b := obj.(*models.SLLMBenchmark)
	reason, _ := body.GetString("__reason__")
	if reason == "" {
		reason = body.String()
	}
	state, err := b.FinishRun(ctx, errors.Error(reason))
	if err != nil {
		task.SetStageFailed(ctx, jsonutils.NewString(err.Error()))
		return
	}
	if state == api.LLMBenchmarkStateStopped {
		task.SetStageComplete(ctx, nil)
		return
	}
	task.SetStageFailed(ctx, jsonutils.NewString(reason))
}

func (task *LLMBenchmarkRunTask) run(ctx context.Context, b *models.SLLMBenchmark) error {
	if err := os.MkdirAll(b.WorkDir, 0755); err != nil {
		return errors.Wrap(err, "MkdirAll")
	}
	if err := os.WriteFile(filepath.Join(b.WorkDir, "target_snapshot.json"), []byte(b.TargetSnapshot), 0644); err != nil {
		return errors.Wrap(err, "write target_snapshot")
	}
	llmObj, err := models.GetLLMManager().FetchById(b.LLMId)
	if err != nil {
		return errors.Wrap(err, "fetch LLM")
	}
	llm := llmObj.(*models.SLLM)
	targetServer, err := llm.GetServer(ctx)
	if err != nil {
		return errors.Wrap(err, "GetServer")
	}
	imageObj, err := models.GetLLMImageManager().FetchById(b.LLMImageId)
	if err != nil {
		return errors.Wrap(err, "FetchById benchmark image")
	}
	image := imageObj.(*models.SLLMImage)

	var spec bench.GuideLLMSpec
	if err := json.Unmarshal([]byte(b.GuideLLMSpec), &spec); err != nil {
		return errors.Wrap(err, "unmarshal GuideLLMSpec")
	}
	var tokenizerMount *models.LLMBenchmarkTokenizerMount
	if b.DatasetName == api.LLMBenchmarkDatasetSyntheticText {
		tokenizerMount, err = b.ResolveTokenizerMount()
		if err != nil {
			return errors.Wrap(err, "resolve offline benchmark tokenizer")
		}
		applyBenchmarkRuntimeTokenizer(&spec, tokenizerMount)
	}
	runtimeSpec, err := json.Marshal(spec)
	if err != nil {
		return errors.Wrap(err, "marshal runtime GuideLLMSpec")
	}
	if err := os.WriteFile(filepath.Join(b.WorkDir, "spec.json"), runtimeSpec, 0644); err != nil {
		return errors.Wrap(err, "write spec")
	}
	envs, err := bench.GuideLLMEnvs(spec)
	if err != nil {
		return errors.Wrap(err, "GuideLLMEnvs")
	}
	podInput, err := b.RunnerPodInput(image, targetServer, tokenizerMount)
	if err != nil {
		return err
	}
	podInput.Pod.Containers[0].Envs = append(podInput.Pod.Containers[0].Envs, envMapToKVs(envs)...)
	if stoppedBenchmark(b) {
		return errors.Error("benchmark stopped")
	}

	s := auth.GetSession(ctx, task.UserCred, options.Options.Region)
	resp, err := computemod.Servers.Create(s, jsonutils.Marshal(podInput))
	if err != nil {
		return errors.Wrap(err, "create runner pod")
	}
	runnerServerId, err := resp.GetString("id")
	if err != nil {
		return errors.Wrap(err, "runner server id")
	}
	_ = b.SetRunner(ctx, runnerServerId, "")
	if stoppedBenchmark(b) {
		return errors.Error("benchmark stopped")
	}

	server, err := llmutils.WaitServerStatus(ctx, runnerServerId, []string{computeapi.VM_RUNNING}, 600)
	if err != nil {
		return errors.Wrap(err, "wait runner server")
	}
	if len(server.Containers) == 0 {
		return errors.Error("runner pod has no containers")
	}
	containerId := server.Containers[0].Id
	_ = b.SetRunner(ctx, runnerServerId, containerId)
	if stoppedBenchmark(b) {
		return errors.Error("benchmark stopped")
	}
	if b.DatasetName == api.LLMBenchmarkDatasetPackage {
		if err := task.runDatasetPreflight(ctx, s, containerId, b, spec); err != nil {
			return err
		}
	}
	if stoppedBenchmark(b) {
		return errors.Error("benchmark stopped")
	}
	if err := b.SetState(ctx, task.UserCred, api.LLMBenchmarkStateRunning, ""); err != nil {
		return err
	}

	phaseErr := executeGuideLLMPhase(
		ctx,
		s,
		containerId,
		b,
		bench.GuideLLMRunCommand(),
		"/workdir/benchmarks.json",
		"/workdir/benchmarks.csv",
	)
	if phaseErr != nil {
		_ = copyBenchmarkArtifacts(s, containerId, b)
		return phaseErr
	}
	if err := copyBenchmarkArtifacts(s, containerId, b); err != nil {
		return err
	}

	metrics, err := bench.ParseMetricsCSV(b.ResultCsv)
	if err != nil {
		_ = b.SetState(ctx, task.UserCred, api.LLMBenchmarkStateRunning, "parse metrics: "+err.Error())
		if b.DatasetName == api.LLMBenchmarkDatasetPackage {
			return task.saveDatasetEvaluationFailure(ctx, b, "", 0, errors.Wrap(err, "parse formal metrics"))
		}
		return nil
	}
	if err := b.UpdateMetrics(ctx, task.UserCred, metrics); err != nil {
		return err
	}
	if b.DatasetName == api.LLMBenchmarkDatasetPackage {
		return task.runDatasetEvaluation(ctx, s, containerId, b, metrics.RequestTotal)
	}
	return nil
}

func evaluationDatasetExportCommand(datasetPath string, requestTotal int) []string {
	command := fmt.Sprintf(
		"head -n %d %s > %s",
		requestTotal,
		shellPath(datasetPath),
		evaluationDatasetRemote,
	)
	return []string{"sh", "-lc", command}
}

func datasetEvaluationFailure(
	workDir, answerColumn string,
	requestTotal int,
	cause error,
) (bench.EvaluationResult, error) {
	ret := bench.EvaluationResult{
		Summary: api.LLMBenchmarkDatasetEvaluation{
			State:        api.LLMBenchmarkEvaluationStateError,
			AnswerColumn: answerColumn,
			RequestTotal: requestTotal,
			Message:      cause.Error(),
		},
		LogPath: filepath.Join(workDir, "evaluation.log"),
	}
	err := os.WriteFile(
		ret.LogPath,
		[]byte("evaluation error: "+cause.Error()+"\n"),
		0644,
	)
	return ret, err
}

func appendArtifactStorageFallback(logPath string, cause error) {
	if logPath == "" || cause == nil {
		return
	}
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = fmt.Fprintf(file, "artifact storage fallback: %s\n", cause)
}

func exportEvaluationDataset(
	s *mcclient.ClientSession,
	containerID, datasetPath string,
	requestTotal int,
) error {
	if requestTotal <= 0 {
		return errors.Error("formal benchmark executed zero requests")
	}
	input := &computeapi.ContainerExecSyncInput{
		Command: evaluationDatasetExportCommand(datasetPath, requestTotal),
		Timeout: 60,
	}
	response, err := computemod.Containers.PerformAction(
		s,
		containerID,
		"exec-sync",
		jsonutils.Marshal(input),
	)
	if err != nil {
		return errors.Wrap(err, "export evaluation dataset")
	}
	if code, _ := response.Int("exit_code"); code != 0 {
		stderr, _ := response.GetString("stderr")
		return fmt.Errorf("export evaluation dataset exited %d: %s", code, stderr)
	}
	return nil
}

func (task *LLMBenchmarkRunTask) runDatasetEvaluation(
	ctx context.Context,
	s *mcclient.ClientSession,
	containerID string,
	b *models.SLLMBenchmark,
	requestTotal int,
) error {
	evaluating := api.LLMBenchmarkDatasetEvaluation{
		State:        api.LLMBenchmarkEvaluationStateEvaluating,
		RequestTotal: requestTotal,
	}
	if err := b.UpdateDatasetEvaluation(ctx, task.UserCred, evaluating, "", "", ""); err != nil {
		return err
	}

	pkgObj, err := models.GetLLMBenchmarkPackageManager().FetchById(b.BenchmarkPackageId)
	if err != nil {
		return task.saveDatasetEvaluationFailure(ctx, b, "", requestTotal, errors.Wrap(err, "fetch benchmark package"))
	}
	pkg := pkgObj.(*models.SLLMBenchmarkPackage)
	if err := exportEvaluationDataset(s, containerID, b.DatasetPath, requestTotal); err != nil {
		return task.saveDatasetEvaluationFailure(ctx, b, pkg.AnswerColumn, requestTotal, err)
	}
	localDataset := filepath.Join(b.WorkDir, evaluationDatasetLocal)
	copied, err := copyArtifacts(s, containerID, map[string]string{
		evaluationDatasetRemote: localDataset,
	})
	if err != nil || copied[evaluationDatasetRemote] == "" {
		if err == nil {
			err = errors.Error("evaluation dataset artifact not copied")
		}
		return task.saveDatasetEvaluationFailure(ctx, b, pkg.AnswerColumn, requestTotal, err)
	}
	defer os.Remove(localDataset)

	result, evaluationErr := bench.EvaluateDataset(ctx, bench.EvaluationInput{
		DatasetPath:   localDataset,
		BenchmarkPath: b.ResultJson,
		AnswerColumn:  pkg.AnswerColumn,
		OutputDir:     b.WorkDir,
	})
	if evaluationErr != nil && result.Summary.State == "" {
		return task.saveDatasetEvaluationFailure(ctx, b, pkg.AnswerColumn, requestTotal, evaluationErr)
	}
	if result.Summary.State != api.LLMBenchmarkEvaluationStateCompleted {
		result.Summary.RequestTotal = requestTotal
	}
	if err := b.UpdateDatasetEvaluation(
		ctx,
		task.UserCred,
		result.Summary,
		result.ResultJSON,
		result.ResultCSV,
		result.LogPath,
	); err != nil {
		return err
	}
	return nil
}

func (task *LLMBenchmarkRunTask) saveDatasetEvaluationFailure(
	ctx context.Context,
	b *models.SLLMBenchmark,
	answerColumn string,
	requestTotal int,
	cause error,
) error {
	result, writeErr := datasetEvaluationFailure(b.WorkDir, answerColumn, requestTotal, cause)
	if writeErr != nil {
		result.Summary.Message += ": write evaluation log: " + writeErr.Error()
		result.LogPath = ""
	}
	return b.UpdateDatasetEvaluation(
		ctx,
		task.UserCred,
		result.Summary,
		"",
		"",
		result.LogPath,
	)
}

func (task *LLMBenchmarkRunTask) persistBenchmarkArtifacts(
	ctx context.Context,
	b *models.SLLMBenchmark,
) error {
	local := b.ArtifactLocations()
	store := bench.DefaultArtifactStore()
	stored, storage, storageErr := store.Persist(ctx, b.ProjectId, b.Id, local)
	message := ""
	if storageErr != nil {
		message = storageErr.Error()
		appendArtifactStorageFallback(local["evaluation-log"], storageErr)
	}
	if err := b.UpdateArtifactLocations(ctx, task.UserCred, stored, storage, message); err != nil {
		if storage == api.LLMBenchmarkArtifactStorageMinio {
			_ = store.DeleteBenchmark(ctx, b.ProjectId, b.Id)
		}
		return err
	}
	if storage == api.LLMBenchmarkArtifactStorageMinio {
		if err := store.RemoveLocal(local); err != nil {
			log.Warningf("remove uploaded benchmark %s local artifacts: %s", b.Id, err)
		}
	}
	return nil
}

func evaluateDatasetPreflight(metrics *bench.LLMBenchmarkMetrics, phaseErr error) (api.LLMBenchmarkDatasetPreflight, error) {
	ret := api.LLMBenchmarkDatasetPreflight{
		State:           api.LLMBenchmarkStateCompleted,
		ExpectedSamples: bench.DatasetPreflightSamples,
	}
	if metrics == nil {
		ret.State = api.LLMBenchmarkStateError
		if phaseErr == nil {
			phaseErr = errors.Error("dataset preflight metrics not ready")
		}
		ret.Message = phaseErr.Error()
		return ret, phaseErr
	}
	ret.ActualSamples = metrics.RequestTotal
	ret.Successful = metrics.RequestSuccessful
	ret.Errored = metrics.RequestErrored
	if metrics.ErrorRate != nil {
		ret.ErrorRate = *metrics.ErrorRate
	} else if metrics.RequestTotal > 0 {
		ret.ErrorRate = float64(metrics.RequestErrored) / float64(metrics.RequestTotal)
	}
	if metrics.RequestLatencyMeanSec != nil {
		ret.LatencyMeanSeconds = *metrics.RequestLatencyMeanSec
	}
	if metrics.RequestTotal == 0 {
		ret.State = api.LLMBenchmarkStateError
		ret.Message = "dataset preflight executed zero requests"
		return ret, errors.Error(ret.Message)
	}
	if metrics.RequestSuccessful == 0 {
		ret.State = api.LLMBenchmarkStateError
		ret.Message = fmt.Sprintf("dataset preflight failed all %d requests", metrics.RequestTotal)
		if phaseErr != nil {
			ret.Message += ": " + phaseErr.Error()
		}
		return ret, errors.Error(ret.Message)
	}
	if metrics.RequestErrored > 0 || phaseErr != nil {
		ret.Message = fmt.Sprintf("dataset preflight completed with %d successful and %d errored requests", metrics.RequestSuccessful, metrics.RequestErrored)
		if phaseErr != nil {
			ret.Message += ": " + phaseErr.Error()
		}
	}
	return ret, nil
}

func (task *LLMBenchmarkRunTask) runDatasetPreflight(
	ctx context.Context,
	s *mcclient.ClientSession,
	containerId string,
	b *models.SLLMBenchmark,
	formal bench.GuideLLMSpec,
) error {
	preflight, err := bench.BuildGuideLLMPreflightSpec(formal, b.MaxDurationSeconds)
	if err != nil {
		return err
	}
	command, err := bench.GuideLLMPreflightRunCommand(preflight)
	if err != nil {
		return err
	}
	if err := b.SetState(ctx, task.UserCred, api.LLMBenchmarkStateValidating, ""); err != nil {
		return err
	}
	validating := api.LLMBenchmarkDatasetPreflight{
		State:           api.LLMBenchmarkStateValidating,
		ExpectedSamples: bench.DatasetPreflightSamples,
	}
	if err := b.UpdateDatasetPreflight(ctx, task.UserCred, validating, "", ""); err != nil {
		return errors.Wrap(err, "start dataset preflight")
	}

	phaseErr := executeGuideLLMPhase(ctx, s, containerId, b, command, preflightRemoteJSON, preflightRemoteCSV)
	paths := map[string]string{
		preflightRemoteJSON: filepath.Join(b.WorkDir, "dataset-preflight.json"),
		preflightRemoteCSV:  filepath.Join(b.WorkDir, "dataset-preflight.csv"),
		preflightRemoteLog:  filepath.Join(b.WorkDir, "dataset-preflight.log"),
	}
	copied, copyErr := copyArtifacts(s, containerId, paths)
	if phaseErr == nil {
		phaseErr = copyErr
	}

	if stoppedBenchmark(b) {
		summary := api.LLMBenchmarkDatasetPreflight{
			State:           api.LLMBenchmarkStateStopped,
			ExpectedSamples: bench.DatasetPreflightSamples,
			Message:         "benchmark stopped during dataset preflight",
		}
		_ = b.UpdateDatasetPreflight(ctx, task.UserCred, summary, copied[preflightRemoteJSON], copied[preflightRemoteLog])
		return errors.Error(summary.Message)
	}

	var metrics *bench.LLMBenchmarkMetrics
	if csvPath := copied[preflightRemoteCSV]; csvPath != "" {
		metrics, err = bench.ParseMetricsCSV(csvPath)
		if phaseErr == nil {
			phaseErr = err
		}
	}
	if metrics == nil || metrics.RequestSuccessful == 0 {
		phaseErr = datasetPreflightPhaseError(phaseErr, copied[preflightRemoteLog])
	}
	summary, decisionErr := evaluateDatasetPreflight(metrics, phaseErr)
	if err := b.UpdateDatasetPreflight(ctx, task.UserCred, summary, copied[preflightRemoteJSON], copied[preflightRemoteLog]); err != nil {
		return errors.Wrap(err, "save dataset preflight")
	}
	return decisionErr
}

func datasetPreflightPhaseError(phaseErr error, logPath string) error {
	f, err := os.Open(logPath)
	if err != nil {
		return phaseErr
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	last := ""
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			last = line
		}
	}
	if last == "" {
		return phaseErr
	}
	if phaseErr == nil {
		return errors.Error(last)
	}
	if strings.Contains(phaseErr.Error(), last) {
		return phaseErr
	}
	return errors.Wrap(phaseErr, last)
}

const (
	guideLLMStartTimeoutSeconds = 30
	guideLLMPollTimeoutSeconds  = 10
)

func executeGuideLLMPhase(
	ctx context.Context,
	s *mcclient.ClientSession,
	containerId string,
	b *models.SLLMBenchmark,
	command []string,
	jsonPath, csvPath string,
) error {
	// The region exec-sync client closes after 15s, before GuideLLM can write
	// its result files. Start the run detached, then poll the exit file.
	launch, err := bench.GuideLLMBackgroundCommand(command, jsonPath, csvPath)
	if err != nil {
		return err
	}
	execResp, err := execContainerSync(s, containerId, launch, guideLLMStartTimeoutSeconds)
	if err != nil {
		return errors.Wrap(err, "guidellm start")
	}
	if code, _ := execResp.Int("exit_code"); code != 0 {
		stderr, _ := execResp.GetString("stderr")
		return fmt.Errorf("guidellm start exited %d: %s", code, stderr)
	}
	return waitGuideLLMPhase(ctx, s, containerId, b, jsonPath, csvPath, bench.GuideLLMWaitSeconds(b.MaxDurationSeconds, b.TotalRequests, b.RequestRate))
}

func waitGuideLLMPhase(
	ctx context.Context,
	s *mcclient.ClientSession,
	containerId string,
	b *models.SLLMBenchmark,
	jsonPath, csvPath string,
	seconds int,
) error {
	if seconds <= 0 {
		seconds = bench.GuideLLMStartupSlackSeconds
	}
	deadline := time.Now().Add(time.Duration(seconds) * time.Second)
	statusCommand := guideLLMStatusCommand(jsonPath, csvPath)
	for {
		if stoppedBenchmark(b) {
			_, _ = execContainerSync(s, containerId, stopGuideLLMCommand(jsonPath), guideLLMPollTimeoutSeconds)
			return errors.Error("benchmark stopped")
		}
		resp, err := execContainerSync(s, containerId, statusCommand, guideLLMPollTimeoutSeconds)
		if err == nil {
			if code, _ := resp.Int("exit_code"); code == 0 {
				stdout, _ := resp.GetString("stdout")
				switch state, exitCode := classifyGuideLLMStatus(stdout); state {
				case "ready":
					return nil
				case "missing":
					return errors.Error("guidellm exited without result files")
				case "exit":
					return fmt.Errorf("guidellm exited %d", exitCode)
				}
			}
		}
		if !time.Now().Before(deadline) {
			_, _ = execContainerSync(s, containerId, stopGuideLLMCommand(jsonPath), guideLLMPollTimeoutSeconds)
			return fmt.Errorf("guidellm did not finish within %d seconds", seconds)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func execContainerSync(s *mcclient.ClientSession, containerId string, command []string, timeout int64) (jsonutils.JSONObject, error) {
	input := &computeapi.ContainerExecSyncInput{
		Command: command,
		Timeout: timeout,
	}
	return computemod.Containers.PerformAction(s, containerId, "exec-sync", jsonutils.Marshal(input))
}

func guideLLMStatusCommand(jsonPath, csvPath string) []string {
	exitPath := shellPath(bench.GuideLLMResultExitPath(jsonPath))
	script := fmt.Sprintf(
		"if [ ! -s %s ]; then printf 'pending\\n'; exit 0; fi; "+
			"code=$(tr -d '[:space:]' < %s); "+
			"printf 'exit %%s\\n' \"$code\"; "+
			"if [ \"$code\" = 0 ]; then if [ -s %s ] && [ -s %s ]; then printf 'ready\\n'; else printf 'missing\\n'; fi; fi",
		exitPath,
		exitPath,
		shellPath(jsonPath),
		shellPath(csvPath),
	)
	return []string{"sh", "-lc", script}
}

func stopGuideLLMCommand(jsonPath string) []string {
	pidPath := shellPath(bench.GuideLLMResultPidPath(jsonPath))
	script := fmt.Sprintf(
		"if [ -s %s ]; then pid=$(tr -d '[:space:]' < %s); kill -TERM \"$pid\" 2>/dev/null || true; kill -TERM -\"$pid\" 2>/dev/null || true; fi",
		pidPath,
		pidPath,
	)
	return []string{"sh", "-lc", script}
}

func classifyGuideLLMStatus(stdout string) (string, int) {
	state := "pending"
	code := 0
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		switch line {
		case "ready":
			return "ready", 0
		case "missing":
			return "missing", 0
		}
		if !strings.HasPrefix(line, "exit ") {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "exit ")))
		if err != nil {
			continue
		}
		state = "exit"
		code = n
	}
	if state == "exit" && code == 0 {
		return "missing", 0
	}
	if state == "exit" {
		return "exit", code
	}
	return "pending", 0
}

func shellPath(path string) string {
	return "'" + strings.ReplaceAll(path, "'", "'\"'\"'") + "'"
}

func copyBenchmarkArtifacts(s *mcclient.ClientSession, containerId string, b *models.SLLMBenchmark) error {
	paths := map[string]string{
		"/workdir/guidellm.log":    filepath.Join(b.WorkDir, "guidellm.log"),
		"/workdir/benchmarks.json": filepath.Join(b.WorkDir, "benchmarks.json"),
		"/workdir/benchmarks.csv":  filepath.Join(b.WorkDir, "benchmarks.csv"),
	}
	copied, firstErr := copyArtifacts(s, containerId, paths)
	_, err := db.Update(b, func() error {
		applyBenchmarkArtifactPaths(b, copied)
		return nil
	})
	if err != nil {
		return err
	}
	return firstErr
}

func applyBenchmarkArtifactPaths(b *models.SLLMBenchmark, copied map[string]string) {
	b.LogPath = copied["/workdir/guidellm.log"]
	b.ResultJson = copied["/workdir/benchmarks.json"]
	b.ResultCsv = copied["/workdir/benchmarks.csv"]
}

func copyArtifacts(s *mcclient.ClientSession, containerId string, paths map[string]string) (map[string]string, error) {
	copied := map[string]string{}
	var firstErr error
	for remote, local := range paths {
		out, err := os.Create(local)
		if err != nil {
			if firstErr == nil {
				firstErr = errors.Wrapf(err, "create %s", local)
			}
			continue
		}
		err = computemod.Containers.CopyFrom(s, containerId, remote, out)
		closeErr := out.Close()
		if err != nil {
			if firstErr == nil {
				firstErr = errors.Wrapf(err, "copy %s", remote)
			}
			continue
		}
		if closeErr != nil {
			if firstErr == nil {
				firstErr = errors.Wrap(closeErr, "close artifact")
			}
			continue
		}
		copied[remote] = local
	}
	return copied, firstErr
}

func cleanupRunner(ctx context.Context, userCred mcclient.TokenCredential, b *models.SLLMBenchmark) {
	if b.RunnerServerId == "" {
		return
	}
	_ = b.DeleteRunnerServer(ctx, userCred)
}

func stoppedBenchmark(b *models.SLLMBenchmark) bool {
	obj, err := models.GetLLMBenchmarkManager().FetchById(b.Id)
	if err != nil {
		return b.StopRequested
	}
	cur := obj.(*models.SLLMBenchmark)
	return cur.StopRequested || cur.State == api.LLMBenchmarkStateStopped
}

func envMapToKVs(envs map[string]string) []*commonapi.ContainerKeyValue {
	ret := make([]*commonapi.ContainerKeyValue, 0, len(envs))
	for k, v := range envs {
		ret = append(ret, &commonapi.ContainerKeyValue{Key: k, Value: v})
	}
	return ret
}
