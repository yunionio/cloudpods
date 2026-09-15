package ocidir

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"

	// crypto libraries included for go-digest
	_ "crypto/sha256"
	_ "crypto/sha512"

	"github.com/opencontainers/go-digest"
	"github.com/regclient/regclient/internal/rwfs"
	"github.com/regclient/regclient/internal/wraperr"
	"github.com/regclient/regclient/scheme"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/ref"
	"github.com/sirupsen/logrus"
)

// ManifestDelete removes a manifest, including all tags that point to that manifest
func (o *OCIDir) ManifestDelete(ctx context.Context, r ref.Ref, opts ...scheme.ManifestOpts) error {
	if r.Digest == "" {
		return wraperr.New(fmt.Errorf("digest required to delete manifest, reference %s", r.CommonName()), types.ErrMissingDigest)
	}

	mc := scheme.ManifestConfig{}
	for _, opt := range opts {
		opt(&mc)
	}

	// always check for refers with ocidir
	if mc.Manifest == nil {
		m, err := o.ManifestGet(ctx, r)
		if err != nil {
			return fmt.Errorf("failed to pull manifest for refers: %w", err)
		}
		mc.Manifest = m
	}
	if mc.Manifest != nil {
		if ms, ok := mc.Manifest.(manifest.Subjecter); ok {
			sDesc, err := ms.GetSubject()
			if err == nil && sDesc != nil && sDesc.MediaType != "" && sDesc.Size > 0 {
				// attempt to delete the referrer, but ignore if the referrer entry wasn't found
				err = o.referrerDelete(ctx, r, mc.Manifest)
				if err != nil && !errors.Is(err, types.ErrNotFound) {
					return err
				}
			}
		}
	}

	// get index
	changed := false
	index, err := o.readIndex(r)
	if err != nil {
		return fmt.Errorf("failed to read index: %w", err)
	}
	for i := len(index.Manifests) - 1; i >= 0; i-- {
		// remove matching entry from index
		if r.Digest != "" && index.Manifests[i].Digest.String() == r.Digest {
			changed = true
			index.Manifests = append(index.Manifests[:i], index.Manifests[i+1:]...)
		}
	}
	// push manifest back out
	if changed {
		err = o.writeIndex(r, index)
		if err != nil {
			return fmt.Errorf("failed to write index: %w", err)
		}
	}

	// delete from filesystem like a registry would do
	d := digest.Digest(r.Digest)
	file := path.Join(r.Path, "blobs", d.Algorithm().String(), d.Encoded())
	err = o.fs.Remove(file)
	if err != nil {
		return fmt.Errorf("failed to delete manifest: %w", err)
	}
	o.refMod(r)
	return nil
}

// ManifestGet retrieves a manifest from a repository
func (o *OCIDir) ManifestGet(ctx context.Context, r ref.Ref) (manifest.Manifest, error) {
	index, err := o.readIndex(r)
	if err != nil {
		return nil, fmt.Errorf("unable to read oci index: %w", err)
	}
	if r.Digest == "" && r.Tag == "" {
		r.Tag = "latest"
	}
	desc, err := indexGet(index, r)
	if err != nil {
		if r.Digest != "" {
			desc.Digest = digest.Digest(r.Digest)
		} else {
			return nil, err
		}
	}
	if desc.Digest == "" {
		return nil, types.ErrNotFound
	}
	file := path.Join(r.Path, "blobs", desc.Digest.Algorithm().String(), desc.Digest.Encoded())
	fd, err := o.fs.Open(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest: %w", err)
	}
	defer fd.Close()
	mb, err := io.ReadAll(fd)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}
	if desc.Size == 0 {
		desc.Size = int64(len(mb))
	}
	o.log.WithFields(logrus.Fields{
		"ref":  r.CommonName(),
		"file": file,
	}).Debug("retrieved manifest")
	return manifest.New(
		manifest.WithRef(r),
		manifest.WithDesc(desc),
		manifest.WithRaw(mb),
	)
}

// ManifestHead gets metadata about the manifest (existence, digest, mediatype, size)
func (o *OCIDir) ManifestHead(ctx context.Context, r ref.Ref) (manifest.Manifest, error) {
	index, err := o.readIndex(r)
	if err != nil {
		return nil, fmt.Errorf("unable to read oci index: %w", err)
	}
	if r.Digest == "" && r.Tag == "" {
		r.Tag = "latest"
	}
	desc, err := indexGet(index, r)
	if err != nil {
		if r.Digest != "" {
			desc.Digest = digest.Digest(r.Digest)
		} else {
			return nil, err
		}
	}
	if desc.Digest == "" {
		return nil, types.ErrNotFound
	}
	// verify underlying file exists
	file := path.Join(r.Path, "blobs", desc.Digest.Algorithm().String(), desc.Digest.Encoded())
	fi, err := rwfs.Stat(o.fs, file)
	if err != nil || fi.IsDir() {
		return nil, types.ErrNotFound
	}
	// if missing, set media type on desc
	if desc.MediaType == "" {
		raw, err := rwfs.ReadFile(o.fs, file)
		if err != nil {
			return nil, err
		}
		mt := struct {
			MediaType     string        `json:"mediaType,omitempty"`
			SchemaVersion int           `json:"schemaVersion,omitempty"`
			Signatures    []interface{} `json:"signatures,omitempty"`
		}{}
		err = json.Unmarshal(raw, &mt)
		if err != nil {
			return nil, err
		}
		if mt.MediaType != "" {
			desc.MediaType = mt.MediaType
			desc.Size = int64(len(raw))
		} else if mt.SchemaVersion == 1 && len(mt.Signatures) > 0 {
			desc.MediaType = types.MediaTypeDocker1ManifestSigned
		} else if mt.SchemaVersion == 1 {
			desc.MediaType = types.MediaTypeDocker1Manifest
			desc.Size = int64(len(raw))
		}
	}
	return manifest.New(
		manifest.WithRef(r),
		manifest.WithDesc(desc),
	)
}

// ManifestPut sends a manifest to the repository
func (o *OCIDir) ManifestPut(ctx context.Context, r ref.Ref, m manifest.Manifest, opts ...scheme.ManifestOpts) error {
	config := scheme.ManifestConfig{}
	for _, opt := range opts {
		opt(&config)
	}
	if !config.Child && r.Digest == "" && r.Tag == "" {
		r.Tag = "latest"
	}

	indexChanged := false
	index, err := o.readIndex(r)
	if err != nil {
		index = indexCreate()
		indexChanged = true
	}
	desc := m.GetDescriptor()
	b, err := m.RawBody()
	if err != nil {
		return fmt.Errorf("could not serialize manifest: %w", err)
	}
	if r.Tag == "" {
		// force digest to match manifest value
		r.Digest = desc.Digest.String()
	}
	if r.Tag != "" {
		desc.Annotations = map[string]string{
			aOCIRefName: r.Tag,
		}
	}
	// create manifest CAS file
	dir := path.Join(r.Path, "blobs", desc.Digest.Algorithm().String())
	err = rwfs.MkdirAll(o.fs, dir, 0777)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("failed creating %s: %w", dir, err)
	}
	file := path.Join(dir, desc.Digest.Encoded())
	fd, err := o.fs.Create(file)
	if err != nil {
		return fmt.Errorf("failed to create manifest: %w", err)
	}
	defer fd.Close()
	_, err = fd.Write(b)
	if err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}
	// replace existing tag or create a new entry
	if !config.Child {
		err := indexSet(&index, r, desc)
		if err != nil {
			return fmt.Errorf("failed to update index: %w", err)
		}
		indexChanged = true
	}
	// write the index.json and oci-layout if it's been changed
	if indexChanged {
		err = o.writeIndex(r, index)
		if err != nil {
			return fmt.Errorf("failed to write index: %w", err)
		}
	}
	o.refMod(r)
	o.log.WithFields(logrus.Fields{
		"ref":  r.CommonName(),
		"file": file,
	}).Debug("pushed manifest")

	// update referrers if defined on this manifest
	if ms, ok := m.(manifest.Subjecter); ok {
		mDesc, err := ms.GetSubject()
		if err != nil {
			return err
		}
		if mDesc != nil && mDesc.MediaType != "" && mDesc.Size > 0 {
			err = o.referrerPut(ctx, r, m)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
