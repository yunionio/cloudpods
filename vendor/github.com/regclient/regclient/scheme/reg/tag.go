package reg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	// crypto libraries included for go-digest
	_ "crypto/sha256"
	_ "crypto/sha512"

	"github.com/opencontainers/go-digest"
	"github.com/regclient/regclient/internal/httplink"
	"github.com/regclient/regclient/internal/reghttp"
	"github.com/regclient/regclient/scheme"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/docker/schema2"
	"github.com/regclient/regclient/types/manifest"
	v1 "github.com/regclient/regclient/types/oci/v1"
	"github.com/regclient/regclient/types/ref"
	"github.com/regclient/regclient/types/tag"
	"github.com/sirupsen/logrus"
)

// TagDelete removes a tag from a repository.
// It first attempts the newer OCI API to delete by tag name (not widely supported).
// If the OCI API fails, it falls back to pushing a unique empty manifest and deleting that.
func (reg *Reg) TagDelete(ctx context.Context, r ref.Ref) error {
	var tempManifest manifest.Manifest
	if r.Tag == "" {
		return types.ErrMissingTag
	}

	// attempt to delete the tag directly, available in OCI distribution-spec, and Hub API
	req := &reghttp.Req{
		Host:      r.Registry,
		NoMirrors: true,
		APIs: map[string]reghttp.ReqAPI{
			"": {
				Method:     "DELETE",
				Repository: r.Repository,
				Path:       "manifests/" + r.Tag,
				IgnoreErr:  true, // do not trigger backoffs if this fails
			},
			"hub": {
				Method: "DELETE",
				Path:   "repositories/" + r.Repository + "/tags/" + r.Tag + "/",
			},
		},
	}

	resp, err := reg.reghttp.Do(ctx, req)
	if resp != nil {
		defer resp.Close()
	}
	// TODO: Hub may return a different status
	if err == nil && resp != nil && resp.HTTPResponse().StatusCode == 202 {
		return nil
	}
	// ignore errors, fallback to creating a temporary manifest to replace the tag and deleting that manifest

	// lookup the current manifest media type
	curManifest, err := reg.ManifestHead(ctx, r)
	if err != nil && errors.Is(err, types.ErrUnsupportedAPI) {
		curManifest, err = reg.ManifestGet(ctx, r)
	}
	if err != nil {
		return err
	}

	// create empty image config with single label
	// Note, this should be MediaType specific, but it appears that docker uses OCI for the config
	now := time.Now()
	conf := v1.Image{
		Created: &now,
		Config: v1.ImageConfig{
			Labels: map[string]string{
				"delete-tag":  r.Tag,
				"delete-date": now.String(),
			},
		},
		OS:           "linux",
		Architecture: "amd64",
		History: []v1.History{
			{
				Created:   &now,
				CreatedBy: "# regclient",
				Comment:   "scratch blob",
			},
		},
		RootFS: v1.RootFS{
			Type: "layers",
			DiffIDs: []digest.Digest{
				types.ScratchDigest,
			},
		},
	}
	confB, err := json.Marshal(conf)
	if err != nil {
		return err
	}
	digester := digest.Canonical.Digester()
	confBuf := bytes.NewBuffer(confB)
	_, err = confBuf.WriteTo(digester.Hash())
	if err != nil {
		return err
	}
	confDigest := digester.Digest()

	// create manifest with config, matching the original tag manifest type
	switch manifest.GetMediaType(curManifest) {
	case types.MediaTypeOCI1Manifest, types.MediaTypeOCI1ManifestList:
		tempManifest, err = manifest.New(manifest.WithOrig(v1.Manifest{
			Versioned: v1.ManifestSchemaVersion,
			MediaType: types.MediaTypeOCI1Manifest,
			Config: types.Descriptor{
				MediaType: types.MediaTypeOCI1ImageConfig,
				Digest:    confDigest,
				Size:      int64(len(confB)),
			},
			Layers: []types.Descriptor{
				{
					MediaType: types.MediaTypeOCI1Layer,
					Size:      int64(len(types.ScratchData)),
					Digest:    types.ScratchDigest,
				},
			},
		}))
		if err != nil {
			return err
		}
	default: // default to the docker v2 schema
		tempManifest, err = manifest.New(manifest.WithOrig(schema2.Manifest{
			Versioned: schema2.ManifestSchemaVersion,
			Config: types.Descriptor{
				MediaType: types.MediaTypeDocker2ImageConfig,
				Digest:    confDigest,
				Size:      int64(len(confB)),
			},
			Layers: []types.Descriptor{
				{
					MediaType: types.MediaTypeDocker2LayerGzip,
					Size:      int64(len(types.ScratchData)),
					Digest:    types.ScratchDigest,
				},
			},
		}))
		if err != nil {
			return err
		}
	}
	reg.log.WithFields(logrus.Fields{
		"ref": r.Reference,
	}).Debug("Sending dummy manifest to replace tag")

	// push scratch layer
	_, err = reg.BlobPut(ctx, r, types.Descriptor{Digest: types.ScratchDigest, Size: int64(len(types.ScratchData))}, bytes.NewReader(types.ScratchData))
	if err != nil {
		return err
	}

	// push config
	_, err = reg.BlobPut(ctx, r, types.Descriptor{Digest: confDigest, Size: int64(len(confB))}, bytes.NewReader(confB))
	if err != nil {
		return fmt.Errorf("failed sending dummy config to delete %s: %w", r.CommonName(), err)
	}

	// push manifest to tag
	err = reg.ManifestPut(ctx, r, tempManifest)
	if err != nil {
		return fmt.Errorf("failed sending dummy manifest to delete %s: %w", r.CommonName(), err)
	}

	r.Digest = tempManifest.GetDescriptor().Digest.String()

	// delete manifest by digest
	reg.log.WithFields(logrus.Fields{
		"ref":    r.Reference,
		"digest": r.Digest,
	}).Debug("Deleting dummy manifest")
	err = reg.ManifestDelete(ctx, r)
	if err != nil {
		return fmt.Errorf("failed deleting dummy manifest for %s: %w", r.CommonName(), err)
	}

	return nil
}

// TagList returns a listing to tags from the repository
func (reg *Reg) TagList(ctx context.Context, r ref.Ref, opts ...scheme.TagOpts) (*tag.List, error) {
	var config scheme.TagConfig
	for _, opt := range opts {
		opt(&config)
	}

	tl, err := reg.tagListOCI(ctx, r, config)
	if err != nil {
		return tl, err
	}

	for {
		// if limit reached, stop searching
		if config.Limit > 0 && len(tl.Tags) >= config.Limit {
			break
		}
		tlHead, err := tl.RawHeaders()
		if err != nil {
			return tl, err
		}
		links, err := httplink.Parse(tlHead.Values("Link"))
		if err != nil {
			return tl, err
		}
		next, err := links.Get("rel", "next")
		// if Link header with rel="next" is defined
		if err == nil {
			link := tl.GetURL()
			if link == nil {
				return tl, fmt.Errorf("tag list, failed to get URL of previous request")
			}
			link, err = link.Parse(next.URI)
			if err != nil {
				return tl, fmt.Errorf("tag list failed to parse Link: %w", err)
			}
			tlAdd, err := reg.tagListLink(ctx, r, config, link)
			if err != nil {
				return tl, fmt.Errorf("tag list failed to get Link: %w", err)
			}
			err = tl.Append(tlAdd)
			if err != nil {
				return tl, fmt.Errorf("tag list failed to append entries: %w", err)
			}
		} else {
			// do not automatically expand tags with OCI methods,
			// OCI registries should send all possible entries up to the specified limit
			break
		}
	}

	return tl, nil
}

func (reg *Reg) tagListOCI(ctx context.Context, r ref.Ref, config scheme.TagConfig) (*tag.List, error) {
	query := url.Values{}
	if config.Last != "" {
		query.Set("last", config.Last)
	}
	if config.Limit > 0 {
		query.Set("n", strconv.Itoa(config.Limit))
	}
	headers := http.Header{
		"Accept": []string{"application/json"},
	}
	req := &reghttp.Req{
		Host: r.Registry,
		APIs: map[string]reghttp.ReqAPI{
			"": {
				Method:     "GET",
				Repository: r.Repository,
				Path:       "tags/list",
				Query:      query,
				Headers:    headers,
			},
		},
	}
	resp, err := reg.reghttp.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags for %s: %w", r.CommonName(), err)
	}
	defer resp.Close()
	if resp.HTTPResponse().StatusCode != 200 {
		return nil, fmt.Errorf("failed to list tags for %s: %w", r.CommonName(), reghttp.HTTPError(resp.HTTPResponse().StatusCode))
	}
	respBody, err := io.ReadAll(resp)
	if err != nil {
		reg.log.WithFields(logrus.Fields{
			"err": err,
			"ref": r.CommonName(),
		}).Warn("Failed to read tag list")
		return nil, fmt.Errorf("failed to read tags for %s: %w", r.CommonName(), err)
	}
	tl, err := tag.New(
		tag.WithRef(r),
		tag.WithRaw(respBody),
		tag.WithResp(resp.HTTPResponse()),
	)
	if err != nil {
		reg.log.WithFields(logrus.Fields{
			"err":  err,
			"body": respBody,
			"ref":  r.CommonName(),
		}).Warn("Failed to unmarshal tag list")
		return tl, fmt.Errorf("failed to unmarshal tag list for %s: %w", r.CommonName(), err)
	}

	return tl, nil
}

func (reg *Reg) tagListLink(ctx context.Context, r ref.Ref, config scheme.TagConfig, link *url.URL) (*tag.List, error) {
	headers := http.Header{
		"Accept": []string{"application/json"},
	}
	req := &reghttp.Req{
		Host: r.Registry,
		APIs: map[string]reghttp.ReqAPI{
			"": {
				Method:     "GET",
				DirectURL:  link,
				Repository: r.Repository,
				Headers:    headers,
			},
		},
	}
	resp, err := reg.reghttp.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to list tags for %s: %w", r.CommonName(), err)
	}
	defer resp.Close()
	if resp.HTTPResponse().StatusCode != 200 {
		return nil, fmt.Errorf("failed to list tags for %s: %w", r.CommonName(), reghttp.HTTPError(resp.HTTPResponse().StatusCode))
	}
	respBody, err := io.ReadAll(resp)
	if err != nil {
		reg.log.WithFields(logrus.Fields{
			"err": err,
			"ref": r.CommonName(),
		}).Warn("Failed to read tag list")
		return nil, fmt.Errorf("failed to read tags for %s: %w", r.CommonName(), err)
	}
	tl, err := tag.New(
		tag.WithRef(r),
		tag.WithRaw(respBody),
		tag.WithResp(resp.HTTPResponse()),
	)
	if err != nil {
		reg.log.WithFields(logrus.Fields{
			"err":  err,
			"body": respBody,
			"ref":  r.CommonName(),
		}).Warn("Failed to unmarshal tag list")
		return tl, fmt.Errorf("failed to unmarshal tag list for %s: %w", r.CommonName(), err)
	}

	return tl, nil
}
