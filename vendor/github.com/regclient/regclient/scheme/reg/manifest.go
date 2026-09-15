package reg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/regclient/regclient/internal/reghttp"
	"github.com/regclient/regclient/internal/wraperr"
	"github.com/regclient/regclient/scheme"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/ref"
	"github.com/sirupsen/logrus"
)

// ManifestDelete removes a manifest by reference (digest) from a registry.
// This will implicitly delete all tags pointing to that manifest.
func (reg *Reg) ManifestDelete(ctx context.Context, r ref.Ref, opts ...scheme.ManifestOpts) error {
	if r.Digest == "" {
		return wraperr.New(fmt.Errorf("digest required to delete manifest, reference %s", r.CommonName()), types.ErrMissingDigest)
	}

	mc := scheme.ManifestConfig{}
	for _, opt := range opts {
		opt(&mc)
	}

	if mc.CheckReferrers && mc.Manifest == nil {
		m, err := reg.ManifestGet(ctx, r)
		if err != nil {
			return fmt.Errorf("failed to pull manifest for refers: %w", err)
		}
		mc.Manifest = m
	}
	if mc.Manifest != nil {
		if mr, ok := mc.Manifest.(manifest.Subjecter); ok {
			sDesc, err := mr.GetSubject()
			if err == nil && sDesc != nil && sDesc.MediaType != "" && sDesc.Size > 0 {
				// attempt to delete the referrer, but ignore if the referrer entry wasn't found
				err = reg.referrerDelete(ctx, r, mc.Manifest)
				if err != nil && !errors.Is(err, types.ErrNotFound) {
					return err
				}
			}
		}
	}

	// build/send request
	req := &reghttp.Req{
		Host:      r.Registry,
		NoMirrors: true,
		APIs: map[string]reghttp.ReqAPI{
			"": {
				Method:     "DELETE",
				Repository: r.Repository,
				Path:       "manifests/" + r.Digest,
			},
		},
	}
	resp, err := reg.reghttp.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete manifest %s: %w", r.CommonName(), err)
	}
	defer resp.Close()
	if resp.HTTPResponse().StatusCode != 202 {
		return fmt.Errorf("failed to delete manifest %s: %w", r.CommonName(), reghttp.HTTPError(resp.HTTPResponse().StatusCode))
	}

	return nil
}

// ManifestGet retrieves a manifest from the registry
func (reg *Reg) ManifestGet(ctx context.Context, r ref.Ref) (manifest.Manifest, error) {
	var tagOrDigest string
	if r.Digest != "" {
		tagOrDigest = r.Digest
	} else if r.Tag != "" {
		tagOrDigest = r.Tag
	} else {
		return nil, wraperr.New(fmt.Errorf("reference missing tag and digest: %s", r.CommonName()), types.ErrMissingTagOrDigest)
	}

	// build/send request
	headers := http.Header{
		"Accept": []string{
			types.MediaTypeDocker1Manifest,
			types.MediaTypeDocker1ManifestSigned,
			types.MediaTypeDocker2Manifest,
			types.MediaTypeDocker2ManifestList,
			types.MediaTypeOCI1Manifest,
			types.MediaTypeOCI1ManifestList,
		},
	}
	req := &reghttp.Req{
		Host: r.Registry,
		APIs: map[string]reghttp.ReqAPI{
			"": {
				Method:     "GET",
				Repository: r.Repository,
				Path:       "manifests/" + tagOrDigest,
				Headers:    headers,
			},
		},
	}
	resp, err := reg.reghttp.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest %s: %w", r.CommonName(), err)
	}
	defer resp.Close()
	if resp.HTTPResponse().StatusCode != 200 {
		return nil, fmt.Errorf("failed to get manifest %s: %w", r.CommonName(), reghttp.HTTPError(resp.HTTPResponse().StatusCode))
	}

	// read manifest
	rawBody, err := io.ReadAll(resp)
	if err != nil {
		return nil, fmt.Errorf("error reading manifest for %s: %w", r.CommonName(), err)
	}

	return manifest.New(
		manifest.WithRef(r),
		manifest.WithHeader(resp.HTTPResponse().Header),
		manifest.WithRaw(rawBody),
	)
}

// ManifestHead returns metadata on the manifest from the registry
func (reg *Reg) ManifestHead(ctx context.Context, r ref.Ref) (manifest.Manifest, error) {
	// build the request
	var tagOrDigest string
	if r.Digest != "" {
		tagOrDigest = r.Digest
	} else if r.Tag != "" {
		tagOrDigest = r.Tag
	} else {
		return nil, wraperr.New(fmt.Errorf("reference missing tag and digest: %s", r.CommonName()), types.ErrMissingTagOrDigest)
	}

	// build/send request
	headers := http.Header{
		"Accept": []string{
			types.MediaTypeDocker1Manifest,
			types.MediaTypeDocker1ManifestSigned,
			types.MediaTypeDocker2Manifest,
			types.MediaTypeDocker2ManifestList,
			types.MediaTypeOCI1Manifest,
			types.MediaTypeOCI1ManifestList,
		},
	}
	req := &reghttp.Req{
		Host: r.Registry,
		APIs: map[string]reghttp.ReqAPI{
			"": {
				Method:     "HEAD",
				Repository: r.Repository,
				Path:       "manifests/" + tagOrDigest,
				Headers:    headers,
			},
		},
	}
	resp, err := reg.reghttp.Do(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to request manifest head %s: %w", r.CommonName(), err)
	}
	defer resp.Close()
	if resp.HTTPResponse().StatusCode != 200 {
		return nil, fmt.Errorf("failed to request manifest head %s: %w", r.CommonName(), reghttp.HTTPError(resp.HTTPResponse().StatusCode))
	}

	return manifest.New(
		manifest.WithRef(r),
		manifest.WithHeader(resp.HTTPResponse().Header),
	)
}

// ManifestPut uploads a manifest to a registry
func (reg *Reg) ManifestPut(ctx context.Context, r ref.Ref, m manifest.Manifest, opts ...scheme.ManifestOpts) error {
	var tagOrDigest string
	if r.Digest != "" {
		tagOrDigest = r.Digest
	} else if r.Tag != "" {
		tagOrDigest = r.Tag
	} else {
		reg.log.WithFields(logrus.Fields{
			"ref": r.Reference,
		}).Warn("Manifest put requires a tag")
		return types.ErrMissingTag
	}

	// create the request body
	mj, err := m.MarshalJSON()
	if err != nil {
		reg.log.WithFields(logrus.Fields{
			"ref": r.Reference,
			"err": err,
		}).Warn("Error marshaling manifest")
		return fmt.Errorf("error marshalling manifest for %s: %w", r.CommonName(), err)
	}

	// build/send request
	headers := http.Header{
		"Content-Type": []string{manifest.GetMediaType(m)},
	}
	req := &reghttp.Req{
		Host:      r.Registry,
		NoMirrors: true,
		APIs: map[string]reghttp.ReqAPI{
			"": {
				Method:     "PUT",
				Repository: r.Repository,
				Path:       "manifests/" + tagOrDigest,
				Headers:    headers,
				BodyLen:    int64(len(mj)),
				BodyBytes:  mj,
			},
		},
	}
	resp, err := reg.reghttp.Do(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to put manifest %s: %w", r.CommonName(), err)
	}
	defer resp.Close()
	if resp.HTTPResponse().StatusCode != 201 {
		return fmt.Errorf("failed to put manifest %s: %w", r.CommonName(), reghttp.HTTPError(resp.HTTPResponse().StatusCode))
	}

	// update referrers if defined on this manifest
	if mr, ok := m.(manifest.Subjecter); ok {
		mDesc, err := mr.GetSubject()
		if err != nil {
			return err
		}
		if mDesc != nil && mDesc.MediaType != "" && mDesc.Size > 0 {
			err = reg.referrerPut(ctx, r, m)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
