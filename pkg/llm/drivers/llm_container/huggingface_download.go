package llm_container

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/models"
)

type hfModelAPIResponse struct {
	Siblings []hfModelSibling `json:"siblings"`
}

type hfModelSibling struct {
	RFilename string `json:"rfilename"`
	Size      int64  `json:"size"`
}

func resolveHfdRevision(modelTag string) string {
	if strings.TrimSpace(modelTag) == "" {
		return "main"
	}
	return strings.TrimSpace(modelTag)
}

func escapeURLPathPreserveSlash(p string) string {
	if p == "" {
		return ""
	}
	parts := strings.Split(p, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}

func buildHuggingFaceModelAPIURL(endpoint, modelName, revision string) string {
	return fmt.Sprintf("%s/api/models/%s?revision=%s&blobs=true",
		strings.TrimRight(endpoint, "/"),
		escapeURLPathPreserveSlash(modelName),
		url.QueryEscape(revision),
	)
}

func isNonEmptyFile(p string) bool {
	st, err := os.Stat(p)
	if err != nil {
		return false
	}
	return !st.IsDir() && st.Size() > 0
}

func isCompleteFile(p string, expectedSize int64) bool {
	st, err := os.Stat(p)
	if err != nil || st.IsDir() {
		return false
	}
	if expectedSize > 0 {
		return st.Size() == expectedSize
	}
	return true
}

func isHuggingFaceImportComplete(localDir string, siblings []hfModelSibling) bool {
	if len(siblings) == 0 {
		return false
	}
	for _, sibling := range siblings {
		rf := strings.TrimSpace(sibling.RFilename)
		if rf == "" {
			continue
		}
		dst := filepath.Join(localDir, filepath.FromSlash(rf))
		if !isCompleteFile(dst, sibling.Size) {
			return false
		}
	}
	return true
}

func hfSiblingDownloadProgress(localDir string, siblings []hfModelSibling) (int64, int64, int, int) {
	totalSize := int64(0)
	completedSize := int64(0)
	totalFiles := 0
	completedFiles := 0
	for _, sibling := range siblings {
		rf := strings.TrimSpace(sibling.RFilename)
		if rf == "" {
			continue
		}
		totalFiles++
		if sibling.Size > 0 {
			totalSize += sibling.Size
		}
		dst := filepath.Join(localDir, filepath.FromSlash(rf))
		if isCompleteFile(dst, sibling.Size) {
			completedFiles++
			if sibling.Size > 0 {
				completedSize += sibling.Size
			}
		}
	}
	return totalSize, completedSize, totalFiles, completedFiles
}

func resolveImportModelName(input api.InstantModelImportInput) string {
	if repo := strings.TrimSpace(input.RepoId); repo != "" {
		return repo
	}
	return strings.TrimSpace(input.ModelName)
}

func resolveImportRevision(input api.InstantModelImportInput, defaultRev string) string {
	if rev := strings.TrimSpace(input.Revision); rev != "" {
		return rev
	}
	if rev := strings.TrimSpace(input.ModelTag); rev != "" {
		return rev
	}
	return defaultRev
}

func downloadHuggingFaceSnapshot(
	ctx context.Context,
	llm *models.SLLM,
	tmpDir string,
	input api.InstantModelImportInput,
	endpoint string,
	modelsPath string,
	progress func(progress float32),
) (string, []string, error) {
	if strings.TrimSpace(tmpDir) == "" {
		return "", nil, errors.Error("tmpDir is empty")
	}
	modelName := resolveImportModelName(input)
	if modelName == "" {
		return "", nil, errors.Error("modelName is empty")
	}

	modelBase := filepath.Base(modelName)
	localDir := filepath.Join(tmpDir, "huggingface", modelBase)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return "", nil, errors.Wrap(err, "mkdir local model dir")
	}

	rev := resolveImportRevision(input, resolveHfdRevision(""))
	apiURL := buildHuggingFaceModelAPIURL(endpoint, modelName, rev)
	log.Infof("Downloading HF model via HF Mirror API: %s", func() string {
		b, _ := json.Marshal(map[string]string{
			"model":    modelName,
			"revision": rev,
			"dir":      localDir,
			"endpoint": endpoint,
			"api":      apiURL,
		})
		return string(b)
	}())
	metaBody, err := llm.HttpGet(ctx, apiURL)
	if err != nil {
		return "", nil, errors.Wrapf(err, "fetch hf model metadata failed: %s", apiURL)
	}
	meta := hfModelAPIResponse{}
	if err := json.Unmarshal(metaBody, &meta); err != nil {
		return "", nil, errors.Wrap(err, "unmarshal hf model metadata")
	}
	if len(meta.Siblings) == 0 {
		return "", nil, errors.Errorf("hf model metadata has no siblings: %s", apiURL)
	}
	totalSize, completedSize, totalFiles, completedFiles := hfSiblingDownloadProgress(localDir, meta.Siblings)
	if totalSize > 0 {
		reportInstantModelDownloadProgress(progress, completedSize, totalSize)
	} else {
		reportInstantModelStepProgress(progress, completedFiles, totalFiles)
	}
	if isHuggingFaceImportComplete(localDir, meta.Siblings) {
		targetDir := path.Join(modelsPath, modelBase)
		log.Infof("Model %s already exists in import dir %s", modelName, localDir)
		return modelName, []string{targetDir}, nil
	}

	for _, s := range meta.Siblings {
		rf := strings.TrimSpace(s.RFilename)
		if rf == "" {
			continue
		}
		dst := filepath.Join(localDir, filepath.FromSlash(rf))
		if isCompleteFile(dst, s.Size) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return "", nil, errors.Wrapf(err, "mkdir for %s", dst)
		}
		fileURL := fmt.Sprintf("%s/%s/resolve/%s/%s", endpoint, escapeURLPathPreserveSlash(modelName), url.PathEscape(rev), escapeURLPathPreserveSlash(rf))
		fileCompleted := completedSize
		fileCompletedSteps := completedFiles
		downloadProgress := instantModelFileDownloadProgress(progress, fileCompleted, totalSize, s.Size)
		if totalSize <= 0 {
			downloadProgress = instantModelStepDownloadProgress(progress, fileCompletedSteps, totalFiles)
		}
		if err := llm.HttpDownloadFileWithProgress(ctx, fileURL, dst, downloadProgress); err != nil {
			return "", nil, errors.Wrapf(err, "download file failed: %s -> %s", fileURL, dst)
		}
		if totalSize > 0 && s.Size > 0 {
			completedSize += s.Size
			reportInstantModelDownloadProgress(progress, completedSize, totalSize)
		} else if totalSize <= 0 {
			completedFiles++
			reportInstantModelStepProgress(progress, completedFiles, totalFiles)
		}
	}

	targetDir := path.Join(modelsPath, modelBase)
	return modelName, []string{targetDir}, nil
}
