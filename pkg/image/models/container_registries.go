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

package models

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/streamutils"
	"yunion.io/x/pkg/util/stringutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/apis"
	identityapi "yunion.io/x/onecloud/pkg/apis/identity"
	api "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/image/drivers/container_registries/client"
	"yunion.io/x/onecloud/pkg/image/options"
	"yunion.io/x/onecloud/pkg/image/utils/registry"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	identitymodules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	kubemod "yunion.io/x/onecloud/pkg/mcclient/modules/k8s"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SContainerRegistryManager struct {
	db.SSharableVirtualResourceBaseManager
}

var (
	containerRegistryManager *SContainerRegistryManager
	ctrRegStreamingWorkerMan = appsrv.NewWorkerManager("container_registry_streaming_worker", 10, 1024, true)
)

func GetContainerRegistryManager() *SContainerRegistryManager {
	if containerRegistryManager == nil {
		containerRegistryManager = &SContainerRegistryManager{
			SSharableVirtualResourceBaseManager: db.NewSharableVirtualResourceBaseManager(
				SContainerRegistry{},
				"container_registries_tbl",
				"container_registry",
				"container_registries",
			),
		}
		containerRegistryManager.SetVirtualObject(containerRegistryManager)
	}
	return containerRegistryManager
}

func init() {
	GetContainerRegistryManager()
}

type SContainerRegistry struct {
	db.SSharableVirtualResourceBase

	Url          string               `width:"256" charset:"ascii" nullable:"false" create:"required" update:"user" list:"user"`
	Type         string               `charset:"ascii" width:"128" create:"required" nullable:"true" list:"user"`
	CredentialId string               `width:"256" charset:"ascii" nullable:"true" create:"optional" list:"user"`
	Config       jsonutils.JSONObject `nullable:"true" create:"optional"`
}

func (man *SContainerRegistryManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, input *api.ContainerRegistryListInput) (*sqlchemy.SQuery, error) {
	q, err := man.SSharableVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, input.SharableVirtualResourceListInput)
	if err != nil {
		return nil, err
	}
	if input.Type != "" {
		q = q.Equals("type", input.Type)
	}
	if input.Url != "" {
		q = q.Contains("url", input.Url)
	}
	return q, nil
}

func (man *SContainerRegistryManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.ContainerRegistryDetails {
	rows := make([]api.ContainerRegistryDetails, len(objs))
	virtRows := man.SSharableVirtualResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)

	credIds := make([]string, 0)
	credIdSet := make(map[string]struct{})
	for i := range objs {
		reg := objs[i].(*SContainerRegistry)
		if reg.CredentialId != "" {
			if _, ok := credIdSet[reg.CredentialId]; !ok {
				credIdSet[reg.CredentialId] = struct{}{}
				credIds = append(credIds, reg.CredentialId)
			}
		}
	}
	credNames := make(map[string]string, len(credIds))
	if len(credIds) > 0 {
		s := auth.GetAdminSession(ctx, options.Options.Region)
		for _, id := range credIds {
			obj, err := identitymodules.Credentials.Get(s, id, nil)
			if err != nil {
				log.Errorf("get credential %s: %v", id, err)
				continue
			}
			name, _ := obj.GetString("name")
			if name != "" {
				credNames[id] = name
			}
		}
	}

	for i := range rows {
		reg := objs[i].(*SContainerRegistry)
		rows[i] = api.ContainerRegistryDetails{
			SharableVirtualResourceDetails: virtRows[i],
			Url:                            reg.Url,
			Type:                           reg.Type,
			CredentialId:                   reg.CredentialId,
		}
		if name, ok := credNames[reg.CredentialId]; ok {
			rows[i].Credential = name
		}
	}
	return rows
}

func (man *SContainerRegistryManager) GetDriver(rType api.ContainerRegistryType) (IContainerRegistryDriver, error) {
	return GetContainerRegistryDriver(rType)
}

func (man *SContainerRegistryManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data *api.ContainerRegistryCreateInput) (*api.ContainerRegistryCreateInput, error) {
	shareInput, err := man.SSharableVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerId, query, data.SharableVirtualResourceCreateInput)
	if err != nil {
		return nil, err
	}
	data.SharableVirtualResourceCreateInput = shareInput
	if data.Url == "" {
		return nil, httperrors.NewInputParameterError("Missing repo url")
	}
	if _, err := url.Parse(data.Url); err != nil {
		return nil, httperrors.NewNotAcceptableError("Invalid repo url: %v", err)
	}

	// Import path: reuse existing credential_id without driver ping
	if data.CredentialId != "" {
		return data, nil
	}

	driver, err := man.GetDriver(data.Type)
	if err != nil {
		return nil, httperrors.NewInputParameterError("Get driver by type: %q", data.Type)
	}

	data, err = driver.ValidateCreateData(ctx, userCred, ownerId, query, data)
	if err != nil {
		return nil, errors.Wrapf(err, "validate %q create data", driver.GetType())
	}

	return data, err
}

func (r *SContainerRegistry) CustomizeCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	input := new(api.ContainerRegistryCreateInput)
	if err := data.Unmarshal(input); err != nil {
		return errors.Wrap(err, "unmarshal container registry create input")
	}
	if input.CredentialId != "" {
		r.CredentialId = input.CredentialId
		r.Config = nil
		return nil
	}
	drv, err := GetContainerRegistryManager().GetDriver(input.Type)
	if err != nil {
		return errors.Wrap(err, "get container registry driver")
	}
	credentialId, err := drv.CreateCredential(ctx, userCred, ownerId, query, input)
	if err != nil {
		return errors.Wrap(err, "create credential")
	}
	r.CredentialId = credentialId
	r.Config = nil
	return nil
}

func (r *SContainerRegistry) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	// Skip deleting credentials imported from kubeserver so the shared Keystone secret remains usable.
	importedFrom := r.GetMetadata(ctx, "imported_from", userCred)
	if r.CredentialId != "" && importedFrom != "kubeserver" {
		s := auth.GetAdminSession(ctx, options.Options.Region)
		if _, err := identitymodules.Credentials.Delete(s, r.CredentialId, nil); err != nil {
			return errors.Wrapf(err, "delete credential %s", r.CredentialId)
		}
	}
	return r.SSharableVirtualResourceBase.CustomizeDelete(ctx, userCred, query, data)
}

func (r *SContainerRegistry) ValidateDeleteCondition(ctx context.Context, info jsonutils.JSONObject) error {
	cnt, err := GetContainerImageManager().Query().Equals("registry_id", r.GetId()).CountWithError()
	if err != nil {
		return errors.Wrap(err, "count container_images by registry_id")
	}
	if cnt > 0 {
		return httperrors.NewNotEmptyError("container registry still has %d container image(s)", cnt)
	}
	return r.SSharableVirtualResourceBase.ValidateDeleteCondition(ctx, info)
}

func (r *SContainerRegistry) GetConfig() (*api.ContainerRegistryConfig, error) {
	if r.Config != nil {
		conf := new(api.ContainerRegistryConfig)
		if err := r.Config.Unmarshal(conf); err != nil {
			return nil, err
		}
		conf.Type = api.ContainerRegistryType(r.Type)
		return conf, nil
	}
	if r.CredentialId == "" {
		return &api.ContainerRegistryConfig{
			Type: api.ContainerRegistryType(r.Type),
		}, nil
	}
	return r.getConfigByCredential()
}

func (r *SContainerRegistry) getConfigByCredential() (*api.ContainerRegistryConfig, error) {
	if r.CredentialId == "" {
		return nil, errors.Error("both config and credential_id are empty")
	}
	s := auth.GetAdminSession(context.Background(), options.Options.Region)
	obj, err := identitymodules.Credentials.Get(s, r.CredentialId, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "get credential %s", r.CredentialId)
	}
	blobStr, err := obj.GetString("blob")
	if err != nil {
		return nil, errors.Wrap(err, "get credential blob")
	}
	blob := new(identityapi.CredentialContainerImageBlob)
	if err := jsonutils.NewString(blobStr).Unmarshal(blob); err != nil {
		blobJson, parseErr := jsonutils.ParseString(blobStr)
		if parseErr != nil {
			return nil, errors.Wrap(parseErr, "parse credential blob json")
		}
		if err := blobJson.Unmarshal(blob); err != nil {
			return nil, errors.Wrap(err, "unmarshal credential blob")
		}
	}
	commonConf := &api.ContainerRegistryConfigCommon{
		ContainerPullImageAuthConfig: apis.ContainerPullImageAuthConfig{
			Username: blob.Username,
			Password: blob.Password,
		},
	}
	conf := &api.ContainerRegistryConfig{
		Type: api.ContainerRegistryType(r.Type),
	}
	switch conf.Type {
	case api.ContainerRegistryTypeCommon:
		conf.Common = commonConf
	case api.ContainerRegistryTypeHarbor:
		conf.Harbor = &api.ContainerRegistryConfigHarbor{ContainerRegistryConfigCommon: *commonConf}
	case api.ContainerRegistryTypeCustom:
		conf.Custom = &api.ContainerRegistryConfigCustom{ContainerRegistryConfigCommon: *commonConf}
	}
	return conf, nil
}

func (r *SContainerRegistry) GetType() api.ContainerRegistryType {
	return api.ContainerRegistryType(r.Type)
}

func (r *SContainerRegistry) GetDriver() IContainerRegistryDriver {
	drv, err := GetContainerRegistryManager().GetDriver(r.GetType())
	if err != nil {
		panic(fmt.Sprintf("Get container registry driver for %s/%s", r.GetId(), r.GetName()))
	}
	return drv
}

func (r *SContainerRegistry) GetDockerRegistryClient() (client.Client, error) {
	conf, err := r.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "get config")
	}
	return r.GetDriver().GetDockerRegistryClient(r.Url, conf)
}

func (r *SContainerRegistry) GetDetailsImages(ctx context.Context, userCred mcclient.TokenCredential, input *api.ContainerRegistryListImagesInput) (jsonutils.JSONObject, error) {
	rgCli, err := r.GetDockerRegistryClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetDockerRegistryClient")
	}
	return rgCli.ListImages(ctx, input)
}

func (r *SContainerRegistry) GetDetailsImageTags(ctx context.Context, userCred mcclient.TokenCredential, query *api.ContainerRegistryGetImageTagsInput) (jsonutils.JSONObject, error) {
	if query.Repository == "" {
		return nil, httperrors.NewNotEmptyError("repository is empty")
	}
	rgCli, err := r.GetDockerRegistryClient()
	if err != nil {
		return nil, errors.Wrap(err, "GetDockerRegistryClient")
	}
	return rgCli.ListImageTags(ctx, query.Repository)
}

func (r *SContainerRegistry) GetDetailsConfig(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*api.ContainerRegistryConfig, error) {
	return r.GetConfig()
}

func (m *SContainerRegistryManager) CustomizeHandlerInfo(info *appsrv.SHandlerInfo) {
	m.SSharableVirtualResourceBaseManager.CustomizeHandlerInfo(info)

	switch info.GetName(nil) {
	case "perform_action", "get_specific", "get_property", "get_details":
		info.SetProcessTimeout(time.Minute * 120).SetWorkerManager(ctrRegStreamingWorkerMan)
	}
}

func (r *SContainerRegistry) PerformUploadImage(ctx context.Context, userCred mcclient.TokenCredential, query, data api.ContainerRegistryUploadImageInput) (*client.ImageMetadata, error) {
	appParams := appsrv.AppContextGetParams(ctx)
	savedPath, err := saveImageFromStream(appParams.Request.Body, appParams.Request.ContentLength)
	defer func() {
		if savedPath != "" {
			os.RemoveAll(savedPath)
		}
	}()
	if err != nil {
		return nil, errors.Wrap(err, "save from stream")
	}
	return r.uploadImage(ctx, savedPath, data)
}

func saveImageFromStream(reader io.Reader, totalSize int64) (string, error) {
	imgName := stringutils.UUID4()
	tarPath := fmt.Sprintf("/tmp/%s", imgName)
	fp, err := os.Create(tarPath)
	if err != nil {
		return "", err
	}
	defer fp.Close()
	lastSaveTime := time.Now()
	_, err = streamutils.StreamPipe(reader, fp, false, func(saved int64) {
		now := time.Now()
		if now.Sub(lastSaveTime) > 5*time.Second {
			log.Infof("saved %d / %d", saved, totalSize)
			lastSaveTime = now
		}
	})
	return tarPath, err
}

func (r *SContainerRegistry) uploadImage(ctx context.Context, imgPath string, input api.ContainerRegistryUploadImageInput) (*client.ImageMetadata, error) {
	dstPath := fmt.Sprintf("%s.tar", imgPath)
	isGzip, err := registry.IsGzipFile(imgPath)
	if err != nil {
		return nil, errors.Wrapf(err, "check %q is gzip file", imgPath)
	}
	if isGzip {
		log.Infof("%q is gzip, decompress it", imgPath)
		if err := registry.UnZip(imgPath, dstPath); err != nil {
			return nil, errors.Wrapf(err, "unzip %q", imgPath)
		}
	} else {
		if err := os.Rename(imgPath, dstPath); err != nil {
			return nil, errors.Wrapf(err, "rename %q to %q", imgPath, dstPath)
		}
	}
	imgPath = dstPath
	defer func() {
		if err := os.RemoveAll(imgPath); err != nil {
			log.Errorf("remove %q: %v", imgPath, err)
		}
	}()

	cli, err := r.GetDockerRegistryClient()
	if err != nil {
		return nil, errors.Wrap(err, "get docker registry client")
	}
	meta, err := cli.AnalysisImageTarMetadata(imgPath)
	if err != nil {
		return nil, errors.Wrapf(err, "analysis image metadata")
	}
	if input.Tag != "" {
		meta.Ref.Tag = input.Tag
	}
	if input.Repository != "" {
		meta.Ref.Repository = input.Repository
	}

	driver := r.GetDriver()
	conf, _ := r.GetConfig()
	if err := driver.PreparePushImage(ctx, r.Url, conf, meta); err != nil {
		return nil, errors.Wrapf(err, "prepare push image to %q", driver.GetType())
	}

	if err := cli.PushImage(ctx, meta, imgPath); err != nil {
		return nil, errors.Wrapf(err, "push image by input: %s", jsonutils.Marshal(input))
	}
	return meta, nil
}

func (man *SContainerRegistryManager) GetPropertyDownloadImage(ctx context.Context, userCred mcclient.TokenCredential, query api.ContainerRegistryManagerDownloadImageInput) (jsonutils.JSONObject, error) {
	if query.Image == "" {
		return nil, httperrors.NewNotEmptyError("image is not provided")
	}
	imgParts := strings.Split(query.Image, "/")
	lastPart := imgParts[len(imgParts)-1]
	lastParts := strings.Split(lastPart, ":")
	if len(lastParts) != 2 {
		return nil, httperrors.NewInputParameterError("invalid image: %s, last part is %s", query.Image, lastPart)
	}
	imgTag := lastParts[1]
	imgPath := strings.TrimSuffix(query.Image, ":"+imgTag)
	proto := "https"
	if query.Insecure {
		proto = "http"
	}
	drv, _ := man.GetDriver(api.ContainerRegistryTypeCommon)
	conf := &api.ContainerRegistryConfig{
		Common: &api.ContainerRegistryConfigCommon{
			ContainerPullImageAuthConfig: apis.ContainerPullImageAuthConfig{
				Username: query.Username,
				Password: query.Password,
			},
		},
	}
	imgPathParts := strings.Split(imgPath, "/")
	imgName := imgPathParts[len(imgPathParts)-1]
	regUrl := fmt.Sprintf("%s://%s", proto, strings.TrimSuffix(imgPath, imgName))
	input := api.ContainerRegistryDownloadImageInput{
		ImageName: imgName,
		Tag:       imgTag,
	}
	return man.downloadImage(ctx, userCred, drv, conf, regUrl, input)
}

func (r *SContainerRegistry) GetDetailsDownloadImage(ctx context.Context, userCred mcclient.TokenCredential, query api.ContainerRegistryDownloadImageInput) (jsonutils.JSONObject, error) {
	drv := r.GetDriver()
	conf, _ := r.GetConfig()
	return GetContainerRegistryManager().downloadImage(ctx, userCred, drv, conf, r.Url, query)
}

func (r *SContainerRegistryManager) downloadImage(ctx context.Context, userCred mcclient.TokenCredential,
	drv IContainerRegistryDriver,
	conf *api.ContainerRegistryConfig,
	regUrl string,
	query api.ContainerRegistryDownloadImageInput) (jsonutils.JSONObject, error) {
	if query.ImageName == "" {
		return nil, httperrors.NewNotEmptyError("image name required")
	}
	if query.Tag == "" {
		return nil, httperrors.NewNotEmptyError("image tag required")
	}
	savedPath, err := drv.DownloadImage(ctx, regUrl, conf, query)
	if err != nil {
		return nil, errors.Wrap(err, "download image")
	}

	fStat, err := os.Stat(savedPath)
	if err != nil {
		return nil, errors.Wrapf(err, "os.Stat %s", savedPath)
	}
	f, err := os.Open(savedPath)
	if err != nil {
		return nil, errors.Wrapf(err, "os.Open %s", savedPath)
	}
	defer f.Close()
	fSize := fStat.Size()

	appParams := appsrv.AppContextGetParams(ctx)
	header := appParams.Response.Header()
	header.Set("Content-Length", strconv.FormatInt(fSize, 10))
	header.Set("Image-Filename", filepath.Base(savedPath))

	defer func() {
		for _, sp := range []string{
			savedPath,
			strings.TrimSuffix(savedPath, ".gz"),
		} {
			if err := os.RemoveAll(sp); err != nil {
				log.Errorf("remove %q: %v", sp, err)
			}
		}
	}()

	_, err = streamutils.StreamPipe(f, appParams.Response, false, nil)
	if err != nil {
		return nil, httperrors.NewGeneralError(err)
	}
	return nil, nil
}

// ImportFromKubeserver imports container registries from kubeserver for compatibility.
// Existing registries with the same URL are skipped unless force is true.
func (man *SContainerRegistryManager) ImportFromKubeserver(ctx context.Context, userCred mcclient.TokenCredential, force bool) (*api.ContainerRegistryImportFromKubeserverOutput, error) {
	s := auth.GetAdminSession(ctx, options.Options.Region)
	params := jsonutils.NewDict()
	params.Set("limit", jsonutils.NewInt(0))
	params.Set("scope", jsonutils.NewString("system"))
	result, err := kubemod.ContainerRegistries.List(s, params)
	if err != nil {
		return nil, errors.Wrap(err, "list kubeserver container_registries")
	}

	out := &api.ContainerRegistryImportFromKubeserverOutput{
		Imported: make([]string, 0),
		Skipped:  make([]string, 0),
		Failed:   make([]string, 0),
	}

	for _, obj := range result.Data {
		srcId, _ := obj.GetString("id")
		name, _ := obj.GetString("name")
		urlStr, _ := obj.GetString("url")
		typ, _ := obj.GetString("type")
		credId, _ := obj.GetString("credential_id")
		domainId, _ := obj.GetString("domain_id")
		projectId, _ := obj.GetString("tenant_id")
		if projectId == "" {
			projectId, _ = obj.GetString("project_id")
		}

		if urlStr == "" {
			out.Failed = append(out.Failed, fmt.Sprintf("%s: empty url", srcId))
			continue
		}

		cnt, err := man.Query().Equals("url", urlStr).CountWithError()
		if err != nil {
			out.Failed = append(out.Failed, fmt.Sprintf("%s: query existing: %v", name, err))
			continue
		}
		if cnt > 0 && !force {
			out.Skipped = append(out.Skipped, fmt.Sprintf("%s (%s)", name, urlStr))
			continue
		}

		createData := jsonutils.NewDict()
		createData.Set("name", jsonutils.NewString(name))
		createData.Set("type", jsonutils.NewString(typ))
		createData.Set("url", jsonutils.NewString(urlStr))
		if credId != "" {
			createData.Set("credential_id", jsonutils.NewString(credId))
		}
		if domainId != "" {
			createData.Set("domain_id", jsonutils.NewString(domainId))
		}
		if projectId != "" {
			createData.Set("project_id", jsonutils.NewString(projectId))
			createData.Set("tenant_id", jsonutils.NewString(projectId))
		}
		createData.Set("__meta__", jsonutils.Marshal(map[string]string{
			"imported_from": "kubeserver",
			"source_id":     srcId,
		}))

		owner := &db.SOwnerId{
			DomainId:  domainId,
			ProjectId: projectId,
		}
		model, err := db.DoCreate(man, ctx, userCred, nil, createData, owner)
		if err != nil {
			out.Failed = append(out.Failed, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		if reg, ok := model.(*SContainerRegistry); ok {
			reg.SetMetadata(ctx, "imported_from", "kubeserver", userCred)
			reg.SetMetadata(ctx, "source_id", srcId, userCred)
		}
		out.Imported = append(out.Imported, fmt.Sprintf("%s (%s) from %s", name, urlStr, srcId))
	}

	return out, nil
}

// PerformImportFromKubeserver imports container registries from kubeserver for compatibility.
func (man *SContainerRegistryManager) PerformImportFromKubeserver(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, input *api.ContainerRegistryImportFromKubeserverInput) (*api.ContainerRegistryImportFromKubeserverOutput, error) {
	force := false
	if input != nil {
		force = input.Force
	}
	return man.ImportFromKubeserver(ctx, userCred, force)
}

// AutoImportFromKubeserver is a cron job that idempotently imports registries from kubeserver.
// Failures are logged and do not block glance.
func (man *SContainerRegistryManager) AutoImportFromKubeserver(ctx context.Context, userCred mcclient.TokenCredential, _ bool) {
	out, err := man.ImportFromKubeserver(ctx, userCred, false)
	if err != nil {
		log.Warningf("AutoImportFromKubeserver: %v", err)
		return
	}
	log.Infof("AutoImportFromKubeserver: imported=%d skipped=%d failed=%d",
		len(out.Imported), len(out.Skipped), len(out.Failed))
	if len(out.Failed) > 0 {
		log.Warningf("AutoImportFromKubeserver failed items: %v", out.Failed)
	}
}
