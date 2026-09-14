package ocidir

import (
	"context"
	"errors"
	"fmt"

	"github.com/regclient/regclient/scheme"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/manifest"
	v1 "github.com/regclient/regclient/types/oci/v1"
	"github.com/regclient/regclient/types/platform"
	"github.com/regclient/regclient/types/ref"
	"github.com/regclient/regclient/types/referrer"
)

// ReferrerList returns a list of referrers to a given reference
func (o *OCIDir) ReferrerList(ctx context.Context, r ref.Ref, opts ...scheme.ReferrerOpts) (referrer.ReferrerList, error) {
	config := scheme.ReferrerConfig{}
	for _, opt := range opts {
		opt(&config)
	}
	rl := referrer.ReferrerList{
		Subject: r,
		Tags:    []string{},
	}
	// select a platform from a manifest list
	if config.Platform != "" {
		m, err := o.ManifestGet(ctx, r)
		if err != nil {
			return rl, err
		}
		if m.IsList() {
			plat, err := platform.Parse(config.Platform)
			if err != nil {
				return rl, err
			}
			d, err := manifest.GetPlatformDesc(m, &plat)
			if err != nil {
				return rl, err
			}
			r.Digest = d.Digest.String()
		} else {
			r.Digest = m.GetDescriptor().Digest.String()
		}
	}
	// if ref is a tag, run a head request for the digest
	if r.Digest == "" {
		m, err := o.ManifestHead(ctx, r)
		if err != nil {
			return rl, err
		}
		r.Digest = m.GetDescriptor().Digest.String()
	}

	// pull referrer list by tag
	rlTag, err := referrer.FallbackTag(r)
	if err != nil {
		return rl, err
	}
	m, err := o.ManifestGet(ctx, rlTag)
	if err != nil {
		if errors.Is(err, types.ErrNotFound) {
			// empty list, initialize a new manifest
			rl.Manifest, err = manifest.New(manifest.WithOrig(v1.Index{
				Versioned: v1.IndexSchemaVersion,
				MediaType: types.MediaTypeOCI1ManifestList,
			}))
			if err != nil {
				return rl, err
			}
			return rl, nil
		}
		return rl, err
	}
	ociML, ok := m.GetOrig().(v1.Index)
	if !ok {
		return rl, fmt.Errorf("manifest is not an OCI index: %s", rlTag.CommonName())
	}
	// update referrer list
	rl.Manifest = m
	rl.Descriptors = ociML.Manifests
	rl.Annotations = ociML.Annotations
	rl.Tags = append(rl.Tags, rlTag.Tag)
	rl = scheme.ReferrerFilter(config, rl)

	return rl, nil
}

// referrerDelete deletes a referrer associated with a manifest
func (o *OCIDir) referrerDelete(ctx context.Context, r ref.Ref, m manifest.Manifest) error {
	// get refers field
	mSubject, ok := m.(manifest.Subjecter)
	if !ok {
		return fmt.Errorf("manifest does not support subject: %w", types.ErrUnsupportedMediaType)
	}
	subject, err := mSubject.GetSubject()
	if err != nil {
		return err
	}
	// validate/set subject descriptor
	if subject == nil || subject.MediaType == "" || subject.Digest == "" || subject.Size <= 0 {
		return fmt.Errorf("subject is not set%.0w", types.ErrNotFound)
	}

	// get descriptor for subject
	rSubject := r
	rSubject.Tag = ""
	rSubject.Digest = subject.Digest.String()

	// pull existing referrer list
	rl, err := o.ReferrerList(ctx, rSubject)
	if err != nil {
		return err
	}
	err = rl.Delete(m)
	if err != nil {
		return err
	}

	// push updated referrer list by tag
	rlTag, err := referrer.FallbackTag(rSubject)
	if err != nil {
		return err
	}
	if rl.IsEmpty() {
		err = o.TagDelete(ctx, rlTag)
		if err == nil {
			return nil
		}
		// if delete is not supported, fall back to pushing empty list
	}
	return o.ManifestPut(ctx, rlTag, rl.Manifest)
}

// referrerPut pushes a new referrer associated with a given reference
func (o *OCIDir) referrerPut(ctx context.Context, r ref.Ref, m manifest.Manifest) error {
	// get subject field
	mSubject, ok := m.(manifest.Subjecter)
	if !ok {
		return fmt.Errorf("manifest does not support subject: %w", types.ErrUnsupportedMediaType)
	}
	subject, err := mSubject.GetSubject()
	if err != nil {
		return err
	}
	// validate/set subject descriptor
	if subject == nil || subject.MediaType == "" || subject.Digest == "" || subject.Size <= 0 {
		return fmt.Errorf("subject is not set%.0w", types.ErrNotFound)
	}

	// get descriptor for subject
	rSubject := r
	rSubject.Tag = ""
	rSubject.Digest = subject.Digest.String()

	// pull existing referrer list
	rl, err := o.ReferrerList(ctx, rSubject)
	if err != nil {
		return err
	}
	err = rl.Add(m)
	if err != nil {
		return err
	}

	// push updated referrer list by tag
	rlTag, err := referrer.FallbackTag(rSubject)
	if err != nil {
		return err
	}
	return o.ManifestPut(ctx, rlTag, rl.Manifest)
}
