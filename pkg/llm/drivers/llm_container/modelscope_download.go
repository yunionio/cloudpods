package llm_container

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"path/filepath"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/hub"
	"yunion.io/x/onecloud/pkg/llm/models"
)

func isModelScopeImportComplete(localDir string, files []hub.ModelScopeFileEntry) bool {
	if len(files) == 0 {
		return false
	}
	for _, f := range files {
		dst := filepath.Join(localDir, filepath.FromSlash(f.Path))
		if !isCompleteFile(dst, f.Size) {
			return false
		}
	}
	return true
}

func modelScopeDownloadProgress(localDir string, files []hub.ModelScopeFileEntry) (int64, int64, int, int) {
	totalSize := int64(0)
	completedSize := int64(0)
	totalFiles := 0
	completedFiles := 0
	for _, f := range files {
		totalFiles++
		if f.Size > 0 {
			totalSize += f.Size
		}
		dst := filepath.Join(localDir, filepath.FromSlash(f.Path))
		if isCompleteFile(dst, f.Size) {
			completedFiles++
			if f.Size > 0 {
				completedSize += f.Size
			}
		}
	}
	return totalSize, completedSize, totalFiles, completedFiles
}

func downloadModelScopeSnapshot(
	ctx context.Context,
	llm *models.SLLM,
	tmpDir string,
	input api.InstantModelImportInput,
	modelsPath string,
	progress func(progress float32),
) (string, []string, error) {
	if strings.TrimSpace(tmpDir) == "" {
		return "", nil, errors.Error("tmpDir is empty")
	}
	modelID := resolveImportModelName(input)
	if modelID == "" {
		return "", nil, errors.Error("model_id is empty")
	}

	modelBase := filepath.Base(modelID)
	localDir := filepath.Join(tmpDir, "modelscope", modelBase)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return "", nil, errors.Wrap(err, "mkdir local model dir")
	}

	revision := resolveImportRevision(input, hub.DefaultModelScopeRevision)
	endpoint := hub.ResolveModelScopeEndpoint()

	allFiles, err := hub.FetchModelScopeFiles(ctx, modelID, revision)
	if err != nil {
		return "", nil, err
	}
	if len(allFiles) == 0 {
		return "", nil, errors.Errorf("modelscope model %s has no downloadable files", modelID)
	}

	files, err := hub.MatchModelScopeFilePaths(allFiles, input.FilePath)
	if err != nil {
		return "", nil, err
	}

	log.Infof("Downloading ModelScope model: %s", func() string {
		b, _ := json.Marshal(map[string]string{
			"model_id": modelID,
			"revision": revision,
			"dir":      localDir,
			"endpoint": endpoint,
			"pattern":  input.FilePath,
		})
		return string(b)
	}())

	totalSize, completedSize, totalFiles, completedFiles := modelScopeDownloadProgress(localDir, files)
	if totalSize > 0 {
		reportInstantModelDownloadProgress(progress, completedSize, totalSize)
	} else {
		reportInstantModelStepProgress(progress, completedFiles, totalFiles)
	}
	if isModelScopeImportComplete(localDir, files) {
		targetDir := path.Join(modelsPath, modelBase)
		log.Infof("ModelScope model %s already exists in import dir %s", modelID, localDir)
		return modelID, []string{targetDir}, nil
	}

	for _, f := range files {
		rf := strings.TrimSpace(f.Path)
		if rf == "" {
			continue
		}
		dst := filepath.Join(localDir, filepath.FromSlash(rf))
		if isCompleteFile(dst, f.Size) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return "", nil, errors.Wrapf(err, "mkdir for %s", dst)
		}
		fileURL := hub.BuildModelScopeFileDownloadURL(endpoint, modelID, rf)
		fileCompleted := completedSize
		fileCompletedSteps := completedFiles
		downloadProgress := instantModelFileDownloadProgress(progress, fileCompleted, totalSize, f.Size)
		if totalSize <= 0 {
			downloadProgress = instantModelStepDownloadProgress(progress, fileCompletedSteps, totalFiles)
		}
		if err := llm.HttpDownloadFileWithProgress(ctx, fileURL, dst, downloadProgress); err != nil {
			return "", nil, errors.Wrapf(err, "download file failed: %s -> %s", fileURL, dst)
		}
		if totalSize > 0 && f.Size > 0 {
			completedSize += f.Size
			reportInstantModelDownloadProgress(progress, completedSize, totalSize)
		} else if totalSize <= 0 {
			completedFiles++
			reportInstantModelStepProgress(progress, completedFiles, totalFiles)
		}
	}

	targetDir := path.Join(modelsPath, modelBase)
	return modelID, []string{targetDir}, nil
}
