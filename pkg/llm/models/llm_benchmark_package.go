package models

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/sqlchemy"

	commonapis "yunion.io/x/onecloud/pkg/apis"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	imagemodules "yunion.io/x/onecloud/pkg/mcclient/modules/image"
)

func init() {
	GetLLMBenchmarkPackageManager()
}

var llmBenchmarkPackageManager *SLLMBenchmarkPackageManager

func GetLLMBenchmarkPackageManager() *SLLMBenchmarkPackageManager {
	if llmBenchmarkPackageManager != nil {
		return llmBenchmarkPackageManager
	}
	llmBenchmarkPackageManager = &SLLMBenchmarkPackageManager{
		SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
			SLLMBenchmarkPackage{},
			"llm_benchmark_packages_tbl",
			"llm_benchmark_package",
			"llm_benchmark_packages",
		),
	}
	llmBenchmarkPackageManager.SetVirtualObject(llmBenchmarkPackageManager)
	return llmBenchmarkPackageManager
}

type SLLMBenchmarkPackageManager struct {
	db.SSharableVirtualResourceBaseManager
}

type SLLMBenchmarkPackage struct {
	db.SSharableVirtualResourceBase

	Source       string `width:"64" charset:"ascii" nullable:"false" default:"huggingface" list:"user" create:"optional"`
	RepoId       string `width:"256" charset:"utf8" nullable:"true" list:"user" create:"optional"`
	Revision     string `width:"128" charset:"utf8" nullable:"true" list:"user" create:"optional"`
	FilePath     string `width:"512" charset:"utf8" nullable:"true" list:"user" create:"optional"`
	Format       string `width:"64" charset:"ascii" nullable:"false" default:"guidellm_jsonl" list:"user" create:"optional"`
	AnswerColumn string `width:"128" charset:"utf8" nullable:"true" list:"user" create:"optional"`

	ImageId      string `width:"128" charset:"ascii" nullable:"true" list:"user" create:"optional"`
	Size         int64  `nullable:"true" list:"user" create:"optional"`
	ActualSizeMb int32  `nullable:"true" list:"user" create:"optional"`

	MountPath   string `charset:"utf8" length:"medium" nullable:"true" list:"user" create:"optional"`
	DatasetPath string `charset:"utf8" length:"medium" nullable:"true" list:"user" create:"optional"`
	Manifest    string `charset:"utf8" length:"long" nullable:"true" list:"user" create:"optional"`
}

func (man *SLLMBenchmarkPackageManager) ValidateCreateData(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	ownerId mcclient.IIdentityProvider,
	query jsonutils.JSONObject,
	input api.LLMBenchmarkPackageCreateInput,
) (api.LLMBenchmarkPackageCreateInput, error) {
	var err error
	input.SharableVirtualResourceCreateInput, err = man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, input.SharableVirtualResourceCreateInput)
	if err != nil {
		return input, errors.Wrap(err, "validate SharableVirtualResourceCreateInput")
	}
	defaultLLMBenchmarkPackageInput(&input)
	if input.ImageId != "" {
		img, err := fetchImage(ctx, userCred, input.ImageId)
		if err != nil {
			return input, errors.Wrap(err, "fetch image")
		}
		if img.DiskFormat != imageapi.IMAGE_DISK_FORMAT_TGZ {
			return input, errors.Wrapf(httperrors.ErrInvalidFormat, "cannot use image of format %s", img.DiskFormat)
		}
		input.ImageId = img.Id
		input.Size = img.Size
		input.ActualSizeMb = img.MinDiskMB
		input.Status = img.Status
		return input, nil
	}
	if input.Source != api.LLMBenchmarkPackageSourceHuggingFace {
		return input, errors.Wrap(httperrors.ErrInputParameter, "source only supports huggingface")
	}
	if input.Format != api.LLMBenchmarkPackageFormatGuideLLMJSONL {
		return input, errors.Wrap(httperrors.ErrInputParameter, "format only supports guidellm_jsonl")
	}
	if strings.TrimSpace(input.RepoId) == "" {
		return input, httperrors.NewMissingParameterError("repo_id")
	}
	if strings.TrimSpace(input.FilePath) == "" {
		return input, httperrors.NewMissingParameterError("file_path")
	}
	input.Status = imageapi.IMAGE_STATUS_QUEUED
	return input, nil
}

func defaultLLMBenchmarkPackageInput(input *api.LLMBenchmarkPackageCreateInput) {
	input.AnswerColumn = strings.TrimSpace(input.AnswerColumn)
	if input.Source == "" {
		input.Source = api.LLMBenchmarkPackageSourceHuggingFace
	}
	if input.Revision == "" {
		input.Revision = "main"
	}
	if input.Format == "" {
		input.Format = api.LLMBenchmarkPackageFormatGuideLLMJSONL
	}
	if input.MountPath == "" {
		input.MountPath = path.Join(api.LLMBenchmarkPackageMountBase, benchmarkPackagePathName(input.Name))
	}
	if input.DatasetPath == "" {
		input.DatasetPath = path.Join(input.MountPath, "data.jsonl")
	}
}

func benchmarkPackagePathName(name string) string {
	name = strings.TrimSpace(name)
	var b strings.Builder
	lastDash := false
	for _, r := range name {
		keep := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-'
		if keep {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "package"
	}
	return out
}

func (man *SLLMBenchmarkPackageManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input api.LLMBenchmarkPackageListInput) (*sqlchemy.SQuery, error) {
	q, err := man.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "ListItemFilter")
	}
	if input.Source != "" {
		q = q.Equals("source", input.Source)
	}
	if input.RepoId != "" {
		q = q.Equals("repo_id", input.RepoId)
	}
	if input.Format != "" {
		q = q.Equals("format", input.Format)
	}
	return q, nil
}

func (man *SLLMBenchmarkPackageManager) PerformImport(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.LLMBenchmarkPackageImportInput,
) (*SLLMBenchmarkPackage, error) {
	input.ImageId = ""
	data := jsonutils.Marshal(input)
	obj, err := db.DoCreate(man, ctx, userCred, nil, data, userCred)
	if err != nil {
		return nil, errors.Wrap(err, "DoCreate")
	}
	pkg := obj.(*SLLMBenchmarkPackage)
	if err := data.Unmarshal(&input); err != nil {
		_ = pkg.SetStatus(ctx, userCred, imageapi.IMAGE_STATUS_KILLED, err.Error())
		return nil, errors.Wrap(err, "unmarshal validated import input")
	}
	if err := pkg.startImportTask(ctx, userCred, input, ""); err != nil {
		_ = pkg.SetStatus(ctx, userCred, imageapi.IMAGE_STATUS_KILLED, err.Error())
		return nil, err
	}
	return pkg, nil
}

func (pkg *SLLMBenchmarkPackage) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	pkg.SSharableVirtualResourceBase.PostCreate(ctx, userCred, ownerId, query, data)
	if pkg.ImageId != "" {
		return
	}
	input := api.LLMBenchmarkPackageImportInput{}
	if err := data.Unmarshal(&input); err != nil {
		_ = pkg.SetStatus(ctx, userCred, imageapi.IMAGE_STATUS_KILLED, err.Error())
		return
	}
	if err := pkg.startImportTask(ctx, userCred, input, ""); err != nil {
		_ = pkg.SetStatus(ctx, userCred, imageapi.IMAGE_STATUS_KILLED, err.Error())
	}
}

func (pkg *SLLMBenchmarkPackage) startImportTask(ctx context.Context, userCred mcclient.TokenCredential, input api.LLMBenchmarkPackageImportInput, parentTaskId string) error {
	params := jsonutils.NewDict()
	params.Add(jsonutils.Marshal(input), "import_input")
	task, err := taskman.TaskManager.NewTask(ctx, "LLMBenchmarkPackageImportTask", pkg, userCred, params, parentTaskId, "")
	if err != nil {
		return errors.Wrap(err, "NewTask")
	}
	return task.ScheduleRun(nil)
}

func (pkg *SLLMBenchmarkPackage) updateImportStatus(ctx context.Context, userCred mcclient.TokenCredential, status, reason string) error {
	if pkg.Status == status {
		return nil
	}
	_, err := db.Update(pkg, func() error {
		pkg.Status = status
		return nil
	})
	return errors.Wrap(err, reason)
}

func (pkg *SLLMBenchmarkPackage) DoImport(ctx context.Context, userCred mcclient.TokenCredential, s *mcclient.ClientSession, input api.LLMBenchmarkPackageCreateInput) (string, error) {
	_ = pkg.SetProgress(0)
	tmpDir, err := getLLMBenchmarkPackageImportWorkDir(options.Options.LLMWorkingDirectory, input)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return "", errors.Wrap(err, "mkdir import work dir")
	}
	rootDir := filepath.Join(tmpDir, "package")
	dataDir := filepath.Join(rootDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return tmpDir, errors.Wrap(err, "mkdir data dir")
	}
	if err := downloadHuggingFaceBenchmarkPackageFile(ctx, filepath.Join(dataDir, "data.jsonl"), input); err != nil {
		return tmpDir, errors.Wrap(err, "download benchmark package")
	}
	_ = pkg.SetProgress(90)

	manifest := map[string]string{
		"source":        input.Source,
		"repo_id":       input.RepoId,
		"revision":      input.Revision,
		"file_path":     input.FilePath,
		"format":        input.Format,
		"dataset_path":  pkg.DatasetPath,
		"answer_column": input.AnswerColumn,
	}
	manifestStr := jsonutils.Marshal(manifest).String()
	if err := os.WriteFile(filepath.Join(rootDir, "manifest.json"), []byte(manifestStr), 0644); err != nil {
		return tmpDir, errors.Wrap(err, "write manifest")
	}
	archivePath := filepath.Join(tmpDir, "package.tgz")
	if err := createTarGz(rootDir, archivePath); err != nil {
		return tmpDir, errors.Wrap(err, "createTarGz")
	}
	_ = pkg.SetProgress(95)

	if err := pkg.updateImportStatus(ctx, userCred, imageapi.IMAGE_STATUS_SAVING, "saving"); err != nil {
		return tmpDir, err
	}
	imageId, imgSize, err := pkg.uploadImage(ctx, userCred, s, input, archivePath, manifestStr)
	if err != nil {
		return tmpDir, errors.Wrap(err, "upload image")
	}
	_ = pkg.SetProgress(98)

	_, err = db.Update(pkg, func() error {
		pkg.ImageId = imageId
		pkg.Size = imgSize
		pkg.Manifest = manifestStr
		return nil
	})
	if err != nil {
		return tmpDir, errors.Wrap(err, "update benchmark package")
	}
	if _, err := pkg.WaitImageStatus(ctx, userCred, []string{imageapi.IMAGE_STATUS_ACTIVE}, 1800); err != nil {
		return tmpDir, errors.Wrap(err, "wait image")
	}
	if err := pkg.syncImageStatus(ctx, userCred); err != nil {
		return tmpDir, errors.Wrap(err, "sync image")
	}
	_ = pkg.SetProgress(100)
	return tmpDir, nil
}

func getLLMBenchmarkPackageImportWorkDir(root string, input api.LLMBenchmarkPackageCreateInput) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", errors.Error("LLMWorkingDirectory is empty")
	}
	key := strings.Join([]string{input.Source, input.RepoId, input.Revision, input.FilePath, input.Format, input.AnswerColumn}, "\x00")
	sum := sha256.Sum256([]byte(key))
	display := benchmarkPackagePathName(input.Name)
	return filepath.Join(root, "benchmark-package-import-cache", fmt.Sprintf("%s-%s", display, hex.EncodeToString(sum[:])[:16])), nil
}

func (pkg *SLLMBenchmarkPackage) uploadImage(ctx context.Context, userCred mcclient.TokenCredential, s *mcclient.ClientSession, input api.LLMBenchmarkPackageCreateInput, archivePath string, manifest string) (string, int64, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", 0, errors.Wrap(err, "open archive")
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return "", 0, errors.Wrap(err, "stat archive")
	}
	size := stat.Size()
	protected := false
	imgParams := imageapi.ImageCreateInput{}
	imgParams.GenerateName = pkg.Name
	imgParams.DiskFormat = imageapi.IMAGE_DISK_FORMAT_TGZ
	imgParams.Size = &size
	imgParams.Protected = &protected
	imgParams.Properties = map[string]string{
		"llm_benchmark_package":             "true",
		"source":                            input.Source,
		"source_repo_id":                    input.RepoId,
		"source_requested_revision":         input.Revision,
		"source_file_path":                  input.FilePath,
		"format":                            input.Format,
		"answer_column":                     input.AnswerColumn,
		"manifest":                          manifest,
		imageapi.IMAGE_INTERNAL_PATH_MAP:    jsonutils.Marshal(map[string]string{"data": pkg.MountPath}).String(),
		imageapi.IMAGE_USED_BY_POST_OVERLAY: "true",
	}
	imageObj, err := imagemodules.Images.Upload(s, jsonutils.Marshal(imgParams), f, size)
	if err != nil {
		return "", 0, errors.Wrap(err, "Upload")
	}
	imageId, err := imageObj.GetString("id")
	if err != nil {
		return "", 0, errors.Wrap(err, "GetString id")
	}
	return imageId, size, nil
}

func (pkg *SLLMBenchmarkPackage) syncImageStatus(ctx context.Context, userCred mcclient.TokenCredential) error {
	img, err := fetchImage(ctx, userCred, pkg.ImageId)
	if err != nil {
		return err
	}
	_, err = db.Update(pkg, func() error {
		pkg.Status = img.Status
		pkg.Size = img.Size
		pkg.ActualSizeMb = img.MinDiskMB
		return nil
	})
	return err
}

func (pkg *SLLMBenchmarkPackage) WaitImageStatus(ctx context.Context, userCred mcclient.TokenCredential, targetStatus []string, timeoutSecs int) (*imageapi.ImageDetails, error) {
	expire := time.Now().Add(time.Second * time.Duration(timeoutSecs))
	for time.Now().Before(expire) {
		img, err := fetchImage(ctx, userCred, pkg.ImageId)
		if err != nil {
			return nil, err
		}
		for _, status := range targetStatus {
			if img.Status == status {
				return img, nil
			}
		}
		if img.Status == imageapi.IMAGE_STATUS_KILLED || img.Status == imageapi.IMAGE_STATUS_DEACTIVATED {
			return nil, errors.Wrap(errors.ErrInvalidStatus, img.Status)
		}
		time.Sleep(2 * time.Second)
	}
	return nil, errors.Wrapf(httperrors.ErrTimeout, "wait image status %s timeout", targetStatus)
}

func (pkg *SLLMBenchmarkPackage) CleanupImportTmpDir(ctx context.Context, userCred mcclient.TokenCredential, dir string) {
	if dir == "" {
		return
	}
	if err := os.RemoveAll(dir); err != nil {
		log.Warningf("cleanup benchmark package import dir %s: %s", dir, err)
	}
}

func downloadHuggingFaceBenchmarkPackageFile(ctx context.Context, dst string, input api.LLMBenchmarkPackageCreateInput) error {
	endpoint := strings.TrimRight(options.Options.HuggingFaceEndpoint, "/")
	if endpoint == "" {
		endpoint = huggingFaceMirrorEndpoint
	}
	fileURL := buildHuggingFaceDatasetFileURL(endpoint, input.RepoId, input.Revision, input.FilePath)
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return errors.Wrap(err, "create tmp")
	}
	client := httputils.GetTimeoutClient(0)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		_ = f.Close()
		return err
	}
	if options.Options.HuggingFaceToken != "" {
		req.Header.Set("Authorization", "Bearer "+options.Options.HuggingFaceToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		_ = f.Close()
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = f.Close()
		return errors.Errorf("download %s failed: %s", fileURL, resp.Status)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

func buildHuggingFaceDatasetFileURL(endpoint, repoID, revision, filePath string) string {
	return fmt.Sprintf("%s/datasets/%s/resolve/%s/%s",
		strings.TrimRight(endpoint, "/"),
		escapeURLPathPreserveSlash(repoID),
		url.PathEscape(revision),
		escapeURLPathPreserveSlash(filePath),
	)
}

func validateBenchmarkPackageDeleteStatus(status string) error {
	if status == imageapi.IMAGE_STATUS_SAVING || status == imageapi.IMAGE_STATUS_QUEUED || status == commonapis.STATUS_DELETING {
		return httperrors.NewInvalidStatusError("benchmark package is %s", status)
	}
	return nil
}

func (pkg *SLLMBenchmarkPackage) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	if err := validateBenchmarkPackageDeleteStatus(pkg.Status); err != nil {
		return err
	}
	if err := ValidateBenchmarkPackageUnused(pkg.Id, ""); err != nil {
		return err
	}
	return pkg.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx, info)
}

func (pkg *SLLMBenchmarkPackage) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	manager := GetLLMBenchmarkManager()
	lockKey := db.GetLockClassKey(manager, pkg.GetOwnerId())
	lockman.LockClass(ctx, manager, lockKey)
	defer lockman.ReleaseClass(ctx, manager, lockKey)

	if err := ValidateBenchmarkPackageUnused(pkg.Id, ""); err != nil {
		return err
	}
	return pkg.StartDeleteTask(ctx, userCred, query, "")
}

func (pkg *SLLMBenchmarkPackage) StartDeleteTask(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, parentTaskId string) error {
	if err := validateBenchmarkPackageDeleteStatus(pkg.Status); err != nil {
		return err
	}
	previousStatus := pkg.Status
	if err := pkg.SetStatus(ctx, userCred, commonapis.STATUS_DELETING, ""); err != nil {
		return errors.Wrap(err, "set benchmark package deleting")
	}
	rollbackStatus := func(startErr error) error {
		if err := pkg.SetStatus(ctx, userCred, previousStatus, startErr.Error()); err != nil {
			return errors.Wrapf(startErr, "rollback benchmark package status: %s", err)
		}
		return startErr
	}
	params := jsonutils.NewDict()
	if pkg.ImageId != "" {
		params.Set("image_id", jsonutils.NewString(pkg.ImageId))
	}
	if jsonutils.QueryBoolean(query, "purge", false) {
		params.Set("purge", jsonutils.JSONTrue)
	}
	task, err := taskman.TaskManager.NewTask(ctx, "LLMBenchmarkPackageDeleteTask", pkg, userCred, params, parentTaskId, "")
	if err != nil {
		return rollbackStatus(errors.Wrap(err, "NewTask LLMBenchmarkPackageDeleteTask"))
	}
	if err := task.ScheduleRun(nil); err != nil {
		return rollbackStatus(errors.Wrap(err, "ScheduleRun LLMBenchmarkPackageDeleteTask"))
	}
	return nil
}

func (pkg *SLLMBenchmarkPackage) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}

func (pkg *SLLMBenchmarkPackage) RealDelete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return pkg.SSharableVirtualResourceBase.Delete(ctx, userCred)
}
