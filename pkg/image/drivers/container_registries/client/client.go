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

package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"
	"github.com/regclient/regclient/types/ref"
	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/image/utils/registry"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type Client interface {
	Ping(ctx context.Context) error
	ListImages(ctx context.Context, input *api.ContainerRegistryListImagesInput) (jsonutils.JSONObject, error)
	ListImageTags(ctx context.Context, image string) (jsonutils.JSONObject, error)
	AnalysisImageTarMetadata(tarPath string) (*ImageMetadata, error)
	PushImage(ctx context.Context, input *ImageMetadata, tarPath string) error
}

// DockerAuthConfig contains authorization information for connecting to a registry.
type DockerAuthConfig struct {
	Username string
	Password string
	// IdentityToken can be used as a refresh_token in place of username and password.
	IdentityToken string
}

type client struct {
	regUrl     string
	regHost    string
	authConfig DockerAuthConfig
	rc         *regclient.RegClient
	pathPrefix string
}

func NewClient(regUrl string, authConf DockerAuthConfig) (Client, error) {
	regURL, err := url.Parse(regUrl)
	if err != nil {
		return nil, errors.Wrapf(err, "parse registry url %s", regUrl)
	}

	regHost := regURL.Host

	confHost := config.Host{
		Name:       regHost,
		Hostname:   regHost,
		ReqPerSec:  100,
		User:       authConf.Username,
		Pass:       authConf.Password,
		RepoAuth:   true,
		PathPrefix: regURL.Path,
	}
	if regURL.Scheme == "https" {
		confHost.TLS = config.TLSInsecure
	}
	if regURL.Scheme == "http" {
		confHost.TLS = config.TLSDisabled
	}

	rc := regclient.New(regclient.WithConfigHost(confHost))
	return &client{
		regUrl:     regUrl,
		regHost:    regHost,
		authConfig: authConf,
		rc:         rc,
		pathPrefix: confHost.PathPrefix,
	}, nil
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func joinUrl(baseUrl, pathStr string) string {
	base, version := mcclient.SplitVersionedURL(baseUrl)
	if len(version) > 0 {
		if strings.HasPrefix(pathStr, fmt.Sprintf("/%s/", version)) {
			baseUrl = base
		}
	}
	return fmt.Sprintf("%s%s", baseUrl, pathStr)
}

func (c client) getHeader(header http.Header) http.Header {
	if header == nil {
		header = http.Header{}
	}
	if c.authConfig.Username != "" {
		header.Set("Authorization", fmt.Sprintf("Basic %s", basicAuth(c.authConfig.Username, c.authConfig.Password)))
	}
	return header
}

func (c client) Request(ctx context.Context, method httputils.THttpMethod, urlPath string, header http.Header, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	cli := httputils.GetDefaultClient()
	regUrl := strings.TrimSuffix(c.regUrl, c.pathPrefix)
	return httputils.JSONRequest(cli, ctx, method, joinUrl(regUrl, urlPath), c.getHeader(header), body, true)
}

func (c client) Get(ctx context.Context, urlPath string, header http.Header, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	return c.Request(ctx, httputils.GET, urlPath, header, body)
}

func (c client) Ping(ctx context.Context) error {
	_, _, err := c.Get(ctx, "/v2/", nil, nil)
	return err
}

type CatalogResult struct {
	Repositories []string          `json:"repositories"`
	Details      []*ImageTagResult `json:"details"`
}

var catalogLinkRegexp = regexp.MustCompile(`<(.*)>;`)

func parseCatalogLink(respHeader http.Header) string {
	if respHeader == nil {
		return ""
	}
	val := respHeader.Get("Link")
	if val == "" {
		return ""
	}
	matches := catalogLinkRegexp.FindStringSubmatch(val)
	if len(matches) < 1 {
		return ""
	}
	return matches[1]
}

func (c client) ListImages(ctx context.Context, input *api.ContainerRegistryListImagesInput) (jsonutils.JSONObject, error) {
	catalogUrl := "/v2/_catalog"
	result := &CatalogResult{
		Repositories: make([]string, 0),
		Details:      make([]*ImageTagResult, 0),
	}
	for {
		header, resp, err := c.Get(ctx, catalogUrl, nil, nil)
		if err != nil {
			return nil, err
		}
		tmpResult := new(CatalogResult)
		if err := resp.Unmarshal(tmpResult); err != nil {
			return nil, err
		}
		catalogUrl = parseCatalogLink(header)
		result.Repositories = append(result.Repositories, tmpResult.Repositories...)
		if catalogUrl == "" {
			break
		}
	}
	newRepos := []string{}
	if c.pathPrefix != "" {
		pathPrefix := strings.TrimPrefix(c.pathPrefix, "/")
		pathPrefix = strings.TrimSuffix(pathPrefix, "/")
		if len(result.Repositories) != 0 {
			for _, repo := range result.Repositories {
				if strings.HasPrefix(repo, pathPrefix+"/") {
					newRepos = append(newRepos, repo)
				}
			}
		} else {
			newRepos = result.Repositories
		}
	}
	if input.RepositoryName != "" {
		filterRepos := []string{}
		for _, name := range newRepos {
			if strings.Contains(name, input.RepositoryName) {
				filterRepos = append(filterRepos, name)
			}
		}
		result.Repositories = filterRepos
	} else {
		result.Repositories = newRepos
	}
	if input.Details {
		result.Details = make([]*ImageTagResult, 0)
		var errgrp errgroup.Group
		for _, r := range result.Repositories {
			tmpR := r
			errgrp.Go(func() error {
				tags, err := c.ListImageTags(ctx, tmpR)
				if err != nil {
					return errors.Wrapf(err, "List image tags: %s", tmpR)
				}
				detail := new(ImageTagResult)
				if err := tags.Unmarshal(detail); err != nil {
					return errors.Wrapf(err, "Unmarshal %#v", tags)
				}
				result.Details = append(result.Details, detail)
				return nil
			})
		}
		if err := errgrp.Wait(); err != nil {
			return nil, err
		}
	}
	return jsonutils.Marshal(result), nil
}

type ImageTagResult struct {
	Name  string   `json:"name"`
	Tags  []string `json:"tags"`
	Image string   `json:"image"`
}

func (c client) ListImageTags(ctx context.Context, image string) (jsonutils.JSONObject, error) {
	tagsUrl := "/v2/%s/tags/list"
	if c.pathPrefix != "" {
		parts := strings.Split(image, "/")
		prefix := strings.TrimPrefix(c.pathPrefix, "/")
		image = fmt.Sprintf("%s/%s", prefix, parts[len(parts)-1])
	}
	image = url.QueryEscape(image)
	tagsUrl = fmt.Sprintf(tagsUrl, image)
	_, resp, err := c.Get(ctx, tagsUrl, nil, nil)
	if err != nil {
		return nil, err
	}
	result := new(ImageTagResult)
	if err := resp.Unmarshal(result); err != nil {
		return nil, err
	}
	result.Image = path.Join(c.regHost, result.Name)
	return jsonutils.Marshal(result), nil
}

func (c client) AnalysisTar(tarPath string) (*registry.TarReadData, error) {
	return registry.AnalysisTar(tarPath)
}

type ImageMetadata struct {
	Ref ref.Ref
}

func newImageMetadata(data *registry.TarReadData) (*ImageMetadata, error) {
	manifests := data.GetDockerManifestList()
	if len(manifests) == 0 {
		return nil, errors.Errorf("not found manifests")
	}
	if len(manifests[0].RepoTags) == 0 {
		return nil, errors.Errorf("not found RepoTags from")
	}
	repoTag := manifests[0].RepoTags[0]
	r, err := ref.New(repoTag)
	if err != nil {
		return nil, errors.Errorf("ref.New(%s)", repoTag)
	}
	return &ImageMetadata{Ref: r}, nil
}

func (c client) AnalysisImageTarMetadata(tarPath string) (*ImageMetadata, error) {
	data, err := c.AnalysisTar(tarPath)
	if err != nil {
		return nil, errors.Wrapf(err, "analysis tar %q data", tarPath)
	}
	meta, err := newImageMetadata(data)
	if err != nil {
		return nil, errors.Wrapf(err, "new image metadata")
	}
	return meta, nil
}

func (c client) PushImage(ctx context.Context, meta *ImageMetadata, tarPath string) error {
	r := meta.Ref
	r.Registry = c.regHost

	f, err := os.Open(tarPath)
	if err != nil {
		return errors.Wrapf(err, "open file %q", tarPath)
	}
	defer f.Close()
	log.Infof("import %q to %#v", tarPath, r)
	return c.rc.ImageImport(ctx, r, f)
}
