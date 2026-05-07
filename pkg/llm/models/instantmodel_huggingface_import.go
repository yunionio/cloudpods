package models

import (
	"strings"

	"yunion.io/x/jsonutils"

	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	apis "yunion.io/x/onecloud/pkg/apis/llm"
)

const defaultHuggingFaceRevision = "main"

func normalizeInstantModelSource(source string, repoID string) string {
	source = strings.TrimSpace(source)
	if strings.EqualFold(source, apis.InstantModelSourceHuggingFace) || (source == "" && strings.TrimSpace(repoID) != "") {
		return apis.InstantModelSourceHuggingFace
	}
	return source
}

func resolveImportRepoAndRevision(input apis.InstantModelImportInput) (string, string, string) {
	repoID := strings.TrimSpace(input.RepoId)
	source := normalizeInstantModelSource(input.Source, repoID)
	revision := strings.TrimSpace(input.Revision)
	if repoID == "" {
		repoID = strings.TrimSpace(input.ModelName)
	}
	if revision == "" {
		revision = strings.TrimSpace(input.ModelTag)
	}
	if source == apis.InstantModelSourceHuggingFace && repoID != "" && revision == "" {
		revision = defaultHuggingFaceRevision
	}
	return source, repoID, revision
}

func buildInstantModelImportInputFromCreate(input apis.InstantModelCreateInput) apis.InstantModelImportInput {
	importInput := apis.InstantModelImportInput{
		Source:    input.Source,
		RepoId:    input.RepoId,
		Revision:  input.Revision,
		ModelName: input.ModelName,
		ModelTag:  input.ModelTag,
		LlmType:   input.LlmType,
	}
	source, repoID, revision := resolveImportRepoAndRevision(importInput)
	importInput.Source = source
	importInput.RepoId = repoID
	importInput.Revision = revision
	importInput.ModelName = repoID
	importInput.ModelTag = revision
	return importInput
}

func normalizeInstantModelCreateInput(input apis.InstantModelCreateInput) apis.InstantModelCreateInput {
	importInput := buildInstantModelImportInputFromCreate(input)
	input.Source = importInput.Source
	input.RepoId = importInput.RepoId
	input.Revision = importInput.Revision
	input.ModelName = importInput.ModelName
	input.ModelTag = importInput.ModelTag
	return input
}

func buildInstantModelImageProperties(input apis.InstantModelImportInput, repoID string, resolvedRevision string) map[string]string {
	source, _, requestedRevision := resolveImportRepoAndRevision(input)
	properties := map[string]string{
		"llm_type": string(input.LlmType),
	}
	if input.ModelName != "" {
		properties["model_name"] = input.ModelName
	}
	if input.ModelTag != "" {
		properties["model_tag"] = input.ModelTag
	}
	if repoID != "" {
		properties["model_name"] = repoID
		properties["source_repo_id"] = repoID
	}
	if requestedRevision != "" {
		properties["model_tag"] = requestedRevision
		properties["source_requested_revision"] = requestedRevision
	}
	if source != "" {
		properties["source"] = source
	}
	if resolvedRevision != "" {
		properties["source_resolved_revision"] = resolvedRevision
	}
	return properties
}

func withInstantModelPostOverlayImageProperties(properties map[string]string, pathMap map[string]string) map[string]string {
	if len(pathMap) == 0 {
		return properties
	}
	if properties == nil {
		properties = make(map[string]string)
	}
	properties[imageapi.IMAGE_INTERNAL_PATH_MAP] = jsonutils.Marshal(pathMap).String()
	properties[imageapi.IMAGE_USED_BY_POST_OVERLAY] = "true"
	return properties
}
