package regclient

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	// crypto libraries included for go-digest
	_ "crypto/sha256"
	_ "crypto/sha512"

	digest "github.com/opencontainers/go-digest"
	"github.com/regclient/regclient/pkg/archive"
	"github.com/regclient/regclient/scheme"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/docker/schema2"
	"github.com/regclient/regclient/types/manifest"
	v1 "github.com/regclient/regclient/types/oci/v1"
	"github.com/regclient/regclient/types/platform"
	"github.com/regclient/regclient/types/ref"
	"github.com/regclient/regclient/types/warning"
	"github.com/sirupsen/logrus"
)

const (
	dockerManifestFilename = "manifest.json"
	ociLayoutVersion       = "1.0.0"
	ociIndexFilename       = "index.json"
	ociLayoutFilename      = "oci-layout"
	annotationRefName      = "org.opencontainers.image.ref.name"
	annotationImageName    = "io.containerd.image.name"
)

// used by import/export to match docker tar expected format
type dockerTarManifest struct {
	Config       string
	RepoTags     []string
	Layers       []string
	Parent       digest.Digest                      `json:",omitempty"`
	LayerSources map[digest.Digest]types.Descriptor `json:",omitempty"`
}

type tarFileHandler func(header *tar.Header, trd *tarReadData) error
type tarReadData struct {
	tr          *tar.Reader
	handleAdded bool
	handlers    map[string]tarFileHandler
	processed   map[string]bool
	finish      []func() error
	// data processed from various handlers
	manifests           map[digest.Digest]manifest.Manifest
	ociIndex            v1.Index
	ociManifest         manifest.Manifest
	dockerManifestFound bool
	dockerManifestList  []dockerTarManifest
	dockerManifest      schema2.Manifest
}
type tarWriteData struct {
	tw    *tar.Writer
	dirs  map[string]bool
	files map[string]bool
	// uid, gid  int
	mode      int64
	timestamp time.Time
}

type imageOpt struct {
	checkBaseDigest string
	checkBaseRef    string
	checkSkipConfig bool
	child           bool
	exportRef       ref.Ref
	forceRecursive  bool
	includeExternal bool
	digestTags      bool
	platform        string
	platforms       []string
	referrerConfs   []scheme.ReferrerConfig
	tagList         []string
}

// ImageOpts define options for the Image* commands
type ImageOpts func(*imageOpt)

// ImageWithCheckBaseDigest provides a base digest to compare
func ImageWithCheckBaseDigest(d string) ImageOpts {
	return func(opts *imageOpt) {
		opts.checkBaseDigest = d
	}
}

// ImageWithCheckBaseRef provides a base reference to compare
func ImageWithCheckBaseRef(r string) ImageOpts {
	return func(opts *imageOpt) {
		opts.checkBaseRef = r
	}
}

// ImageWithCheckSkipConfig skips the configuration check
func ImageWithCheckSkipConfig() ImageOpts {
	return func(opts *imageOpt) {
		opts.checkSkipConfig = true
	}
}

// ImageWithChild attempts to copy every manifest and blob even if parent manifests already exist.
func ImageWithChild() ImageOpts {
	return func(opts *imageOpt) {
		opts.child = true
	}
}

// ImageWithExportRef overrides the image name embedded in the export file
func ImageWithExportRef(r ref.Ref) ImageOpts {
	return func(opts *imageOpt) {
		opts.exportRef = r
	}
}

// ImageWithForceRecursive attempts to copy every manifest and blob even if parent manifests already exist.
func ImageWithForceRecursive() ImageOpts {
	return func(opts *imageOpt) {
		opts.forceRecursive = true
	}
}

// ImageWithIncludeExternal attempts to copy every manifest and blob even if parent manifests already exist.
func ImageWithIncludeExternal() ImageOpts {
	return func(opts *imageOpt) {
		opts.includeExternal = true
	}
}

// ImageWithDigestTags looks for "sha-<digest>.*" tags in the repo to copy with any manifest.
// These are used by some artifact systems like sigstore/cosign.
func ImageWithDigestTags() ImageOpts {
	return func(opts *imageOpt) {
		opts.digestTags = true
	}
}

// ImageWithPlatform requests specific platforms from a manifest list.
// This is used by ImageCheckBase.
func ImageWithPlatform(p string) ImageOpts {
	return func(opts *imageOpt) {
		opts.platform = p
	}
}

// ImageWithPlatforms only copies specific platforms from a manifest list.
// This will result in a failure on many registries that validate manifests.
// Use the empty string to indicate images without a platform definition should be copied.
func ImageWithPlatforms(p []string) ImageOpts {
	return func(opts *imageOpt) {
		opts.platforms = p
	}
}

// ImageWithReferrers recursively includes images that refer to this.
func ImageWithReferrers(rOpts ...scheme.ReferrerOpts) ImageOpts {
	return func(opts *imageOpt) {
		if opts.referrerConfs == nil {
			opts.referrerConfs = []scheme.ReferrerConfig{}
		}
		rConf := scheme.ReferrerConfig{}
		for _, rOpt := range rOpts {
			rOpt(&rConf)
		}
		opts.referrerConfs = append(opts.referrerConfs, rConf)
	}
}

// ImageCheckBase returns nil if the base image is unchanged.
// A base image mismatch returns an error that wraps types.ErrMismatch.
func (rc *RegClient) ImageCheckBase(ctx context.Context, r ref.Ref, opts ...ImageOpts) error {
	var opt imageOpt
	for _, optFn := range opts {
		optFn(&opt)
	}
	var m manifest.Manifest
	var err error

	// if the base name is not provided, check image for base annotations
	if opt.checkBaseRef == "" {
		m, err = rc.ManifestGet(ctx, r)
		if err != nil {
			return err
		}
		ma, ok := m.(manifest.Annotator)
		if !ok {
			return fmt.Errorf("image does not support annotations, base image must be provided%.0w", types.ErrMissingAnnotation)
		}
		annot, err := ma.GetAnnotations()
		if err != nil {
			return err
		}
		if baseName, ok := annot[types.AnnotationBaseImageName]; ok {
			opt.checkBaseRef = baseName
		} else {
			return fmt.Errorf("image does not have a base annotation, base image must be provided%.0w", types.ErrMissingAnnotation)
		}
		if baseDig, ok := annot[types.AnnotationBaseImageDigest]; ok {
			opt.checkBaseDigest = baseDig
		}
	}
	baseR, err := ref.New(opt.checkBaseRef)
	if err != nil {
		return err
	}
	defer rc.Close(ctx, baseR)

	// if the digest is available, check if that matches the base name
	if opt.checkBaseDigest != "" {
		baseMH, err := rc.ManifestHead(ctx, baseR, WithManifestRequireDigest())
		if err != nil {
			return err
		}
		expectDig, err := digest.Parse(opt.checkBaseDigest)
		if err != nil {
			return err
		}
		if baseMH.GetDescriptor().Digest == expectDig {
			rc.log.WithFields(logrus.Fields{
				"name":   baseR.CommonName(),
				"digest": baseMH.GetDescriptor().Digest.String(),
			}).Debug("base image digest matches")
			return nil
		} else {
			rc.log.WithFields(logrus.Fields{
				"name":     baseR.CommonName(),
				"digest":   baseMH.GetDescriptor().Digest.String(),
				"expected": expectDig.String(),
			}).Debug("base image digest changed")
			return fmt.Errorf("base digest changed, %s, expected %s, received %s%.0w",
				baseR.CommonName(), expectDig.String(), baseMH.GetDescriptor().Digest.String(), types.ErrMismatch)
		}
	}

	// if the digest is not available, compare layers of each manifest
	if m == nil {
		m, err = rc.ManifestGet(ctx, r)
		if err != nil {
			return err
		}
	}
	if m.IsList() && opt.platform != "" {
		p, err := platform.Parse(opt.platform)
		if err != nil {
			return err
		}
		d, err := manifest.GetPlatformDesc(m, &p)
		if err != nil {
			return err
		}
		rp := r
		rp.Digest = d.Digest.String()
		m, err = rc.ManifestGet(ctx, rp)
		if err != nil {
			return err
		}
	}
	if m.IsList() {
		// loop through each platform
		ml, ok := m.(manifest.Indexer)
		if !ok {
			return fmt.Errorf("manifest list is not an Indexer")
		}
		dl, err := ml.GetManifestList()
		if err != nil {
			return err
		}
		rp := r
		for _, d := range dl {
			rp.Digest = d.Digest.String()
			optP := append(opts, ImageWithPlatform(d.Platform.String()))
			err = rc.ImageCheckBase(ctx, rp, optP...)
			if err != nil {
				return fmt.Errorf("platform %s mismatch: %w", d.Platform.String(), err)
			}
		}
		return nil
	}
	img, ok := m.(manifest.Imager)
	if !ok {
		return fmt.Errorf("manifest must be an image")
	}
	layers, err := img.GetLayers()
	if err != nil {
		return err
	}
	baseM, err := rc.ManifestGet(ctx, baseR)
	if err != nil {
		return err
	}
	if baseM.IsList() && opt.platform != "" {
		p, err := platform.Parse(opt.platform)
		if err != nil {
			return err
		}
		d, err := manifest.GetPlatformDesc(baseM, &p)
		if err != nil {
			return err
		}
		rp := baseR
		rp.Digest = d.Digest.String()
		baseM, err = rc.ManifestGet(ctx, rp)
		if err != nil {
			return err
		}
	}
	baseImg, ok := baseM.(manifest.Imager)
	if !ok {
		return fmt.Errorf("base image manifest must be an image")
	}
	baseLayers, err := baseImg.GetLayers()
	if err != nil {
		return err
	}
	if len(baseLayers) <= 0 {
		return fmt.Errorf("base image has no layers")
	}
	for i := range baseLayers {
		if i >= len(layers) {
			return fmt.Errorf("image has fewer layers than base image")
		}
		if !layers[i].Same(baseLayers[i]) {
			rc.log.WithFields(logrus.Fields{
				"layer":    i,
				"expected": layers[i].Digest.String(),
				"digest":   baseLayers[i].Digest.String(),
			}).Debug("image layer changed")
			return fmt.Errorf("base layer changed, %s[%d], expected %s, received %s%.0w",
				baseR.CommonName(), i, layers[i].Digest.String(), baseLayers[i].Digest.String(), types.ErrMismatch)
		}
	}

	if opt.checkSkipConfig {
		return nil
	}

	// if the layers match, compare the config history
	confDesc, err := img.GetConfig()
	if err != nil {
		return err
	}
	conf, err := rc.BlobGetOCIConfig(ctx, r, confDesc)
	if err != nil {
		return err
	}
	confOCI := conf.GetConfig()
	baseConfDesc, err := baseImg.GetConfig()
	if err != nil {
		return err
	}
	baseConf, err := rc.BlobGetOCIConfig(ctx, baseR, baseConfDesc)
	if err != nil {
		return err
	}
	baseConfOCI := baseConf.GetConfig()
	for i := range baseConfOCI.History {
		if i >= len(confOCI.History) {
			return fmt.Errorf("image has fewer history entries than base image")
		}
		if baseConfOCI.History[i].Author != confOCI.History[i].Author ||
			baseConfOCI.History[i].Comment != confOCI.History[i].Comment ||
			!baseConfOCI.History[i].Created.Equal(*confOCI.History[i].Created) ||
			baseConfOCI.History[i].CreatedBy != confOCI.History[i].CreatedBy ||
			baseConfOCI.History[i].EmptyLayer != confOCI.History[i].EmptyLayer {
			rc.log.WithFields(logrus.Fields{
				"index":    i,
				"expected": confOCI.History[i],
				"history":  baseConfOCI.History[i],
			}).Debug("image history changed")
			return fmt.Errorf("base history changed, %s[%d], expected %v, received %v%.0w",
				baseR.CommonName(), i, confOCI.History[i], baseConfOCI.History[i], types.ErrMismatch)
		}
	}

	rc.log.WithFields(logrus.Fields{
		"base": baseR.CommonName(),
	}).Debug("base image layers and history matches")
	return nil
}

// ImageCopy copies an image
// This will retag an image in the same repository, only pushing and pulling the top level manifest
// On the same registry, it will attempt to use cross-repository blob mounts to avoid pulling blobs
// Blobs are only pulled when they don't exist on the target and a blob mount fails
func (rc *RegClient) ImageCopy(ctx context.Context, refSrc ref.Ref, refTgt ref.Ref, opts ...ImageOpts) error {
	var opt imageOpt
	for _, optFn := range opts {
		optFn(&opt)
	}
	// dedup warnings
	if w := warning.FromContext(ctx); w == nil {
		ctx = warning.NewContext(ctx, &warning.Warning{Hook: warning.DefaultHook()})
	}
	return rc.imageCopyOpt(ctx, refSrc, refTgt, types.Descriptor{}, opt.child, &opt)
}

func (rc *RegClient) imageCopyOpt(ctx context.Context, refSrc ref.Ref, refTgt ref.Ref, d types.Descriptor, child bool, opt *imageOpt) error {
	mOpts := []ManifestOpts{}
	if child {
		mOpts = append(mOpts, WithManifestChild())
	}
	// check if scheme/refTgt prefers parent manifests pushed first
	// if so, this should automatically set forceRecursive
	tgtSI, err := rc.schemeInfo(refTgt)
	if err != nil {
		return fmt.Errorf("failed looking up scheme for %s: %v", refTgt.CommonName(), err)
	}
	if tgtSI.ManifestPushFirst {
		opt.forceRecursive = true
	}
	// check if source and destination already match
	mdh, errD := rc.ManifestHead(ctx, refTgt, WithManifestRequireDigest())
	if opt.forceRecursive {
		// copy forced, unable to run below skips
	} else if errD == nil && refTgt.Digest != "" && digest.Digest(refTgt.Digest) == mdh.GetDescriptor().Digest {
		rc.log.WithFields(logrus.Fields{
			"target": refTgt.Reference,
			"digest": mdh.GetDescriptor().Digest.String(),
		}).Info("Copy not needed, target already up to date")
		return nil
	} else if errD == nil && refTgt.Digest == "" {
		msh, errS := rc.ManifestHead(ctx, refSrc, WithManifestRequireDigest())
		if errS == nil && msh.GetDescriptor().Digest == mdh.GetDescriptor().Digest {
			rc.log.WithFields(logrus.Fields{
				"source": refSrc.Reference,
				"target": refTgt.Reference,
				"digest": mdh.GetDescriptor().Digest.String(),
			}).Info("Copy not needed, target already up to date")
			return nil
		}
	}

	// get the manifest for the source
	m, err := rc.ManifestGet(ctx, refSrc, WithManifestDesc(d))
	if err != nil {
		rc.log.WithFields(logrus.Fields{
			"ref": refSrc.Reference,
			"err": err,
		}).Warn("Failed to get source manifest")
		return err
	}

	if tgtSI.ManifestPushFirst {
		// push manifest to target
		err = rc.ManifestPut(ctx, refTgt, m, mOpts...)
		if err != nil {
			rc.log.WithFields(logrus.Fields{
				"target": refTgt.Reference,
				"err":    err,
			}).Warn("Failed to push manifest")
			return err
		}
	}

	if !ref.EqualRepository(refSrc, refTgt) {
		// copy components of the image if the repository is different
		if mi, ok := m.(manifest.Indexer); ok {
			// manifest lists need to recursively copy nested images by digest
			pd, err := mi.GetManifestList()
			if err != nil {
				return err
			}
			for _, entry := range pd {
				// skip copy of platforms not specifically included
				if len(opt.platforms) > 0 {
					match, err := imagePlatformInList(entry.Platform, opt.platforms)
					if err != nil {
						return err
					}
					if !match {
						rc.log.WithFields(logrus.Fields{
							"platform": entry.Platform,
						}).Debug("Platform excluded from copy")
						continue
					}
				}
				rc.log.WithFields(logrus.Fields{
					"platform": entry.Platform,
					"digest":   entry.Digest.String(),
				}).Debug("Copy platform")
				entrySrc := refSrc
				entryTgt := refTgt
				entrySrc.Tag = ""
				entryTgt.Tag = ""
				entrySrc.Digest = entry.Digest.String()
				entryTgt.Digest = entry.Digest.String()
				switch entry.MediaType {
				case types.MediaTypeDocker1Manifest, types.MediaTypeDocker1ManifestSigned,
					types.MediaTypeDocker2Manifest, types.MediaTypeDocker2ManifestList,
					types.MediaTypeOCI1Manifest, types.MediaTypeOCI1ManifestList:
					// known manifest media type
					err = rc.imageCopyOpt(ctx, entrySrc, entryTgt, entry, true, opt)
				case types.MediaTypeDocker2ImageConfig, types.MediaTypeOCI1ImageConfig,
					types.MediaTypeDocker2LayerGzip, types.MediaTypeOCI1Layer, types.MediaTypeOCI1LayerGzip,
					types.MediaTypeBuildkitCacheConfig:
					// known blob media type
					err = rc.BlobCopy(ctx, entrySrc, entryTgt, entry)
				default:
					// unknown media type, first try an image copy
					err = rc.imageCopyOpt(ctx, entrySrc, entryTgt, entry, true, opt)
					if err != nil {
						// fall back to trying to copy a blob
						err = rc.BlobCopy(ctx, entrySrc, entryTgt, entry)
					}
				}
				if err != nil {
					return err
				}
			}
		}
		if mi, ok := m.(manifest.Imager); ok {
			// copy components of an image
			// transfer the config
			cd, err := mi.GetConfig()
			if err != nil {
				// docker schema v1 does not have a config object, ignore if it's missing
				if !errors.Is(err, types.ErrUnsupportedMediaType) {
					rc.log.WithFields(logrus.Fields{
						"ref": refSrc.Reference,
						"err": err,
					}).Warn("Failed to get config digest from manifest")
					return fmt.Errorf("failed to get config digest for %s: %w", refSrc.CommonName(), err)
				}
			} else {
				rc.log.WithFields(logrus.Fields{
					"source": refSrc.Reference,
					"target": refTgt.Reference,
					"digest": cd.Digest.String(),
				}).Info("Copy config")
				if err := rc.BlobCopy(ctx, refSrc, refTgt, cd); err != nil {
					rc.log.WithFields(logrus.Fields{
						"source": refSrc.Reference,
						"target": refTgt.Reference,
						"digest": cd.Digest.String(),
						"err":    err,
					}).Warn("Failed to copy config")
					return err
				}
			}

			// copy filesystem layers
			l, err := mi.GetLayers()
			if err != nil {
				return err
			}
			for _, layerSrc := range l {
				if len(layerSrc.URLs) > 0 && !opt.includeExternal {
					// skip blobs where the URLs are defined, these aren't hosted and won't be pulled from the source
					rc.log.WithFields(logrus.Fields{
						"source":        refSrc.Reference,
						"target":        refTgt.Reference,
						"layer":         layerSrc.Digest.String(),
						"external-urls": layerSrc.URLs,
					}).Debug("Skipping external layer")
					continue
				}
				rc.log.WithFields(logrus.Fields{
					"source": refSrc.Reference,
					"target": refTgt.Reference,
					"layer":  layerSrc.Digest.String(),
				}).Info("Copy layer")
				if err := rc.BlobCopy(ctx, refSrc, refTgt, layerSrc); err != nil {
					rc.log.WithFields(logrus.Fields{
						"source": refSrc.Reference,
						"target": refTgt.Reference,
						"layer":  layerSrc.Digest.String(),
						"err":    err,
					}).Warn("Failed to copy layer")
					return err
				}
			}
		}
	}

	if !tgtSI.ManifestPushFirst {
		// push manifest to target
		err = rc.ManifestPut(ctx, refTgt, m, mOpts...)
		if err != nil {
			rc.log.WithFields(logrus.Fields{
				"target": refTgt.Reference,
				"err":    err,
			}).Warn("Failed to push manifest")
			return err
		}
	}

	// copy referrers
	referrerTags := []string{}
	if opt.referrerConfs != nil {
		rl, err := rc.ReferrerList(ctx, refSrc)
		if err != nil {
			return err
		}
		referrerTags = append(referrerTags, rl.Tags...)
		descList := []types.Descriptor{}
		if len(opt.referrerConfs) == 0 {
			descList = rl.Descriptors
		} else {
			for _, rConf := range opt.referrerConfs {
				rlFilter := scheme.ReferrerFilter(rConf, rl)
				descList = append(descList, rlFilter.Descriptors...)
			}
		}
		for _, rDesc := range descList {
			referrerSrc := refSrc
			referrerSrc.Tag = ""
			referrerSrc.Digest = rDesc.Digest.String()
			referrerTgt := refTgt
			referrerTgt.Tag = ""
			referrerTgt.Digest = rDesc.Digest.String()
			err = rc.imageCopyOpt(ctx, referrerSrc, referrerTgt, rDesc, true, opt)
			if err != nil {
				rc.log.WithFields(logrus.Fields{
					"digest": rDesc.Digest.String(),
					"src":    referrerSrc.CommonName(),
					"tgt":    referrerTgt.CommonName(),
				}).Warn("Failed to copy referrer")
				return err
			}
		}
	}

	// lookup digest tags to include artifacts with image
	if opt.digestTags {
		if len(opt.tagList) == 0 {
			tl, err := rc.TagList(ctx, refSrc)
			if err != nil {
				rc.log.WithFields(logrus.Fields{
					"source": refSrc.Reference,
					"err":    err,
				}).Warn("Failed to list tags for digest-tag copy")
				return err
			}
			tags, err := tl.GetTags()
			if err != nil {
				rc.log.WithFields(logrus.Fields{
					"source": refSrc.Reference,
					"err":    err,
				}).Warn("Failed to list tags for digest-tag copy")
				return err
			}
			opt.tagList = tags
		}
		prefix := fmt.Sprintf("%s-%s", m.GetDescriptor().Digest.Algorithm(), m.GetDescriptor().Digest.Encoded())
		for _, tag := range opt.tagList {
			if strings.HasPrefix(tag, prefix) {
				// skip referrers that were copied above
				for _, referrerTag := range referrerTags {
					if referrerTag == tag {
						continue
					}
				}
				refTagSrc := refSrc
				refTagSrc.Tag = tag
				refTagSrc.Digest = ""
				refTagTgt := refTgt
				refTagTgt.Tag = tag
				refTagTgt.Digest = ""
				err = rc.imageCopyOpt(ctx, refTagSrc, refTagTgt, types.Descriptor{}, false, opt)
				if err != nil {
					rc.log.WithFields(logrus.Fields{
						"tag": tag,
						"src": refTagSrc.CommonName(),
						"tgt": refTagTgt.CommonName(),
					}).Warn("Failed to copy digest-tag")
					return err
				}
			}
		}
	}

	return nil
}

// ImageExport exports an image to an output stream.
// The format is compatible with "docker load" if a single image is selected and not a manifest list.
// The ref must include a tag for exporting to docker (defaults to latest), and may also include a digest.
// The export is also formatted according to OCI layout which supports multi-platform images.
// <https://github.com/opencontainers/image-spec/blob/master/image-layout.md>
// A tar file will be sent to outStream.
//
// Resulting filesystem:
// oci-layout: created at top level, can be done at the start
// index.json: created at top level, single descriptor with org.opencontainers.image.ref.name annotation pointing to the tag
// manifest.json: created at top level, based on every layer added, only works for a single arch image
// blobs/$algo/$hash: each content addressable object (manifest, config, or layer), created recursively
func (rc *RegClient) ImageExport(ctx context.Context, r ref.Ref, outStream io.Writer, opts ...ImageOpts) error {
	var ociIndex v1.Index

	var opt imageOpt
	for _, optFn := range opts {
		optFn(&opt)
	}
	if opt.exportRef.IsZero() {
		opt.exportRef = r
	}

	// create tar writer object
	tw := tar.NewWriter(outStream)
	defer tw.Close()
	twd := &tarWriteData{
		tw:    tw,
		dirs:  map[string]bool{},
		files: map[string]bool{},
		mode:  0644,
	}

	// retrieve image manifest
	m, err := rc.ManifestGet(ctx, r)
	if err != nil {
		rc.log.WithFields(logrus.Fields{
			"ref": r.CommonName(),
			"err": err,
		}).Warn("Failed to get manifest")
		return err
	}

	// build/write oci-layout
	ociLayout := v1.ImageLayout{Version: ociLayoutVersion}
	err = twd.tarWriteFileJSON(ociLayoutFilename, ociLayout)
	if err != nil {
		return err
	}

	// create a manifest descriptor
	mDesc := m.GetDescriptor()
	if mDesc.Annotations == nil {
		mDesc.Annotations = map[string]string{}
	}
	mDesc.Annotations[annotationImageName] = opt.exportRef.CommonName()
	mDesc.Annotations[annotationRefName] = opt.exportRef.Tag

	// generate/write an OCI index
	ociIndex.Versioned = v1.IndexSchemaVersion
	ociIndex.Manifests = []types.Descriptor{mDesc} // initialize with the descriptor to the manifest list
	err = twd.tarWriteFileJSON(ociIndexFilename, ociIndex)
	if err != nil {
		return err
	}

	// append to docker manifest with tag, config filename, each layer filename, and layer descriptors
	if mi, ok := m.(manifest.Imager); ok {
		conf, err := mi.GetConfig()
		if err != nil {
			return err
		}
		refTag := opt.exportRef.ToReg()
		if refTag.Digest != "" {
			refTag.Digest = ""
		}
		if refTag.Tag == "" {
			refTag.Tag = "latest"
		}
		dockerManifest := dockerTarManifest{
			RepoTags:     []string{refTag.CommonName()},
			Config:       tarOCILayoutDescPath(conf),
			Layers:       []string{},
			LayerSources: map[digest.Digest]types.Descriptor{},
		}
		dl, err := mi.GetLayers()
		if err != nil {
			return err
		}
		for _, d := range dl {
			dockerManifest.Layers = append(dockerManifest.Layers, tarOCILayoutDescPath(d))
			dockerManifest.LayerSources[d.Digest] = d
		}

		// marshal manifest and write manifest.json
		err = twd.tarWriteFileJSON(dockerManifestFilename, []dockerTarManifest{dockerManifest})
		if err != nil {
			return err
		}
	}

	// recursively include manifests and nested blobs
	err = rc.imageExportDescriptor(ctx, r, mDesc, twd)
	if err != nil {
		return err
	}

	return nil
}

// imageExportDescriptor pulls a manifest or blob, outputs to a tar file, and recursively processes any nested manifests or blobs
func (rc *RegClient) imageExportDescriptor(ctx context.Context, ref ref.Ref, desc types.Descriptor, twd *tarWriteData) error {
	tarFilename := tarOCILayoutDescPath(desc)
	if twd.files[tarFilename] {
		// blob has already been imported into tar, skip
		return nil
	}
	switch desc.MediaType {
	case types.MediaTypeDocker1Manifest, types.MediaTypeDocker1ManifestSigned, types.MediaTypeDocker2Manifest, types.MediaTypeOCI1Manifest:
		// Handle single platform manifests
		// retrieve manifest
		m, err := rc.ManifestGet(ctx, ref, WithManifestDesc(desc))
		if err != nil {
			return err
		}
		mi, ok := m.(manifest.Imager)
		if !ok {
			return fmt.Errorf("manifest doesn't support image methods%.0w", types.ErrUnsupportedMediaType)
		}
		// write manifest body by digest
		mBody, err := m.RawBody()
		if err != nil {
			return err
		}
		err = twd.tarWriteHeader(tarFilename, int64(len(mBody)))
		if err != nil {
			return err
		}
		_, err = twd.tw.Write(mBody)
		if err != nil {
			return err
		}

		// add config
		confD, err := mi.GetConfig()
		// ignore unsupported media type errors
		if err != nil && !errors.Is(err, types.ErrUnsupportedMediaType) {
			return err
		}
		if err == nil {
			err = rc.imageExportDescriptor(ctx, ref, confD, twd)
			if err != nil {
				return err
			}
		}

		// loop over layers
		layerDL, err := mi.GetLayers()
		// ignore unsupported media type errors
		if err != nil && !errors.Is(err, types.ErrUnsupportedMediaType) {
			return err
		}
		if err == nil {
			for _, layerD := range layerDL {
				err = rc.imageExportDescriptor(ctx, ref, layerD, twd)
				if err != nil {
					return err
				}
			}
		}

	case types.MediaTypeDocker2ManifestList, types.MediaTypeOCI1ManifestList:
		// handle OCI index and Docker manifest list
		// retrieve manifest
		m, err := rc.ManifestGet(ctx, ref, WithManifestDesc(desc))
		if err != nil {
			return err
		}
		mi, ok := m.(manifest.Indexer)
		if !ok {
			return fmt.Errorf("manifest doesn't support index methods%.0w", types.ErrUnsupportedMediaType)
		}
		// write manifest body by digest
		mBody, err := m.RawBody()
		if err != nil {
			return err
		}
		err = twd.tarWriteHeader(tarFilename, int64(len(mBody)))
		if err != nil {
			return err
		}
		_, err = twd.tw.Write(mBody)
		if err != nil {
			return err
		}
		// recurse over entries in the list/index
		mdl, err := mi.GetManifestList()
		if err != nil {
			return err
		}
		for _, md := range mdl {
			err = rc.imageExportDescriptor(ctx, ref, md, twd)
			if err != nil {
				return err
			}
		}

	default:
		// get blob
		blobR, err := rc.BlobGet(ctx, ref, desc)
		if err != nil {
			return err
		}
		defer blobR.Close()
		// write blob by digest
		err = twd.tarWriteHeader(tarFilename, int64(desc.Size))
		if err != nil {
			return err
		}
		size, err := io.Copy(twd.tw, blobR)
		if err != nil {
			return fmt.Errorf("failed to export blob %s: %w", desc.Digest.String(), err)
		}
		if size != desc.Size {
			return fmt.Errorf("blob size mismatch, descriptor %d, received %d", desc.Size, size)
		}
	}

	return nil
}

// ImageImport pushes an image from a tar file to a registry
func (rc *RegClient) ImageImport(ctx context.Context, ref ref.Ref, rs io.ReadSeeker) error {
	trd := &tarReadData{
		handlers:  map[string]tarFileHandler{},
		processed: map[string]bool{},
		finish:    []func() error{},
		manifests: map[digest.Digest]manifest.Manifest{},
	}

	// add handler for oci-layout, index.json, and manifest.json
	rc.imageImportOCIAddHandler(ctx, ref, trd)
	rc.imageImportDockerAddHandler(trd)

	// process tar file looking for oci-layout and index.json, load manifests/blobs on success
	err := trd.tarReadAll(rs)

	if err != nil && errors.Is(err, types.ErrNotFound) && trd.dockerManifestFound {
		// import failed but manifest.json found, fall back to manifest.json processing
		// add handlers for the docker manifest layers
		rc.imageImportDockerAddLayerHandlers(ctx, ref, trd)
		// reprocess the tar looking for manifest.json files
		err = trd.tarReadAll(rs)
		if err != nil {
			return fmt.Errorf("failed to import layers from docker tar: %w", err)
		}
		// push docker manifest
		m, err := manifest.New(manifest.WithOrig(trd.dockerManifest))
		if err != nil {
			return err
		}
		err = rc.ManifestPut(ctx, ref, m)
		if err != nil {
			return err
		}
	} else if err != nil {
		// unhandled error from tar read
		return err
	} else {
		// successful load of OCI blobs, now push manifest and tag
		err = rc.imageImportOCIPushManifests(ctx, ref, trd)
		if err != nil {
			return err
		}
	}
	return nil
}

func (rc *RegClient) imageImportBlob(ctx context.Context, ref ref.Ref, desc types.Descriptor, trd *tarReadData) error {
	// skip if blob already exists
	_, err := rc.BlobHead(ctx, ref, desc)
	if err == nil {
		return nil
	}
	// upload blob
	_, err = rc.BlobPut(ctx, ref, desc, trd.tr)
	if err != nil {
		return err
	}
	return nil
}

// imageImportDockerAddHandler processes tar files generated by docker
func (rc *RegClient) imageImportDockerAddHandler(trd *tarReadData) {
	trd.handlers[dockerManifestFilename] = func(header *tar.Header, trd *tarReadData) error {
		err := trd.tarReadFileJSON(&trd.dockerManifestList)
		if err != nil {
			return err
		}
		trd.dockerManifestFound = true
		return nil
	}
}

// imageImportDockerAddLayerHandlers imports the docker layers when OCI import fails and docker manifest found
func (rc *RegClient) imageImportDockerAddLayerHandlers(ctx context.Context, ref ref.Ref, trd *tarReadData) {
	// remove handlers for OCI
	delete(trd.handlers, ociLayoutFilename)
	delete(trd.handlers, ociIndexFilename)

	// make a docker v2 manifest from first json array entry (can only tag one image)
	trd.dockerManifest.SchemaVersion = 2
	trd.dockerManifest.MediaType = types.MediaTypeDocker2Manifest
	trd.dockerManifest.Layers = make([]types.Descriptor, len(trd.dockerManifestList[0].Layers))

	// add handler for config
	trd.handlers[filepath.Clean(trd.dockerManifestList[0].Config)] = func(header *tar.Header, trd *tarReadData) error {
		// upload blob, digest is unknown
		d, err := rc.BlobPut(ctx, ref, types.Descriptor{Size: header.Size}, trd.tr)
		if err != nil {
			return err
		}
		// save the resulting descriptor to the manifest
		if od, ok := trd.dockerManifestList[0].LayerSources[d.Digest]; ok {
			trd.dockerManifest.Config = od
		} else {
			d.MediaType = types.MediaTypeDocker2ImageConfig
			trd.dockerManifest.Config = d
		}
		return nil
	}
	// add handlers for each layer
	for i, layerFile := range trd.dockerManifestList[0].Layers {
		func(i int) {
			trd.handlers[filepath.Clean(layerFile)] = func(header *tar.Header, trd *tarReadData) error {
				// ensure blob is compressed with gzip to match media type
				gzipR, err := archive.Compress(trd.tr, archive.CompressGzip)
				if err != nil {
					return err
				}
				// upload blob, digest and size is unknown
				d, err := rc.BlobPut(ctx, ref, types.Descriptor{}, gzipR)
				if err != nil {
					return err
				}
				// save the resulting descriptor in the appropriate layer
				if od, ok := trd.dockerManifestList[0].LayerSources[d.Digest]; ok {
					trd.dockerManifest.Layers[i] = od
				} else {
					d.MediaType = types.MediaTypeDocker2LayerGzip
					trd.dockerManifest.Layers[i] = d
				}
				return nil
			}
		}(i)
	}
	trd.handleAdded = true
}

// imageImportOCIAddHandler adds handlers for oci-layout and index.json found in OCI layout tar files
func (rc *RegClient) imageImportOCIAddHandler(ctx context.Context, ref ref.Ref, trd *tarReadData) {
	// add handler for oci-layout, index.json, and manifest.json
	var err error
	var foundLayout, foundIndex bool

	// common handler code when both oci-layout and index.json have been processed
	ociHandler := func(trd *tarReadData) error {
		// no need to process docker manifest.json when OCI layout is available
		delete(trd.handlers, dockerManifestFilename)
		// create a manifest from the index
		trd.ociManifest, err = manifest.New(manifest.WithOrig(trd.ociIndex))
		if err != nil {
			return err
		}
		// start recursively processing manifests starting with the index
		// there's no need to push the index.json by digest, it will be pushed by tag if needed
		err = rc.imageImportOCIHandleManifest(ctx, ref, trd.ociManifest, trd, false, false)
		if err != nil {
			return err
		}
		return nil
	}
	trd.handlers[ociLayoutFilename] = func(header *tar.Header, trd *tarReadData) error {
		var ociLayout v1.ImageLayout
		err := trd.tarReadFileJSON(&ociLayout)
		if err != nil {
			return err
		}
		if ociLayout.Version != ociLayoutVersion {
			// unknown version, ignore
			rc.log.WithFields(logrus.Fields{
				"version": ociLayout.Version,
			}).Warn("Unsupported oci-layout version")
			return nil
		}
		foundLayout = true
		if foundIndex {
			err = ociHandler(trd)
			if err != nil {
				return err
			}
		}
		return nil
	}
	trd.handlers[ociIndexFilename] = func(header *tar.Header, trd *tarReadData) error {
		err := trd.tarReadFileJSON(&trd.ociIndex)
		if err != nil {
			return err
		}
		foundIndex = true
		if foundLayout {
			err = ociHandler(trd)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

// imageImportOCIHandleManifest recursively processes index and manifest entries from an OCI layout tar
func (rc *RegClient) imageImportOCIHandleManifest(ctx context.Context, ref ref.Ref, m manifest.Manifest, trd *tarReadData, push bool, child bool) error {
	// cache the manifest to avoid needing to pull again later, this is used if index.json is a wrapper around some other manifest
	trd.manifests[m.GetDescriptor().Digest] = m

	handleManifest := func(d types.Descriptor, child bool) {
		filename := tarOCILayoutDescPath(d)
		if !trd.processed[filename] && trd.handlers[filename] == nil {
			trd.handlers[filename] = func(header *tar.Header, trd *tarReadData) error {
				b, err := io.ReadAll(trd.tr)
				if err != nil {
					return err
				}
				switch d.MediaType {
				case types.MediaTypeDocker1Manifest, types.MediaTypeDocker1ManifestSigned,
					types.MediaTypeDocker2Manifest, types.MediaTypeDocker2ManifestList,
					types.MediaTypeOCI1Manifest, types.MediaTypeOCI1ManifestList:
					// known manifest media types
					md, err := manifest.New(manifest.WithDesc(d), manifest.WithRaw(b))
					if err != nil {
						return err
					}
					return rc.imageImportOCIHandleManifest(ctx, ref, md, trd, true, child)
				case types.MediaTypeDocker2ImageConfig, types.MediaTypeOCI1ImageConfig,
					types.MediaTypeDocker2LayerGzip, types.MediaTypeOCI1Layer, types.MediaTypeOCI1LayerGzip,
					types.MediaTypeBuildkitCacheConfig:
					// known blob media types
					return rc.imageImportBlob(ctx, ref, d, trd)
				default:
					// attempt manifest import, fall back to blob import
					md, err := manifest.New(manifest.WithDesc(d), manifest.WithRaw(b))
					if err == nil {
						return rc.imageImportOCIHandleManifest(ctx, ref, md, trd, true, child)
					}
					return rc.imageImportBlob(ctx, ref, d, trd)
				}
			}
		}
	}

	if !push {
		mi, ok := m.(manifest.Indexer)
		if !ok {
			return fmt.Errorf("manifest doesn't support image methods%.0w", types.ErrUnsupportedMediaType)
		}
		// for root index, add handler for matching reference (or only reference)
		dl, err := mi.GetManifestList()
		if err != nil {
			return err
		}
		// locate the digest in the index
		var d types.Descriptor
		if len(dl) == 1 {
			d = dl[0]
		} else if ref.Digest != "" {
			d.Digest = digest.Digest(ref.Digest)
		} else {
			if ref.Tag == "" {
				ref.Tag = "latest"
			}
			// if more than one digest is in the index, use the first matching tag
			for _, cur := range dl {
				if cur.Annotations[annotationRefName] == ref.Tag {
					d = cur
					break
				}
			}
		}
		if d.Digest.String() == "" {
			return fmt.Errorf("could not find requested tag in index.json, %s", ref.Tag)
		}
		handleManifest(d, false)
		// add a finish step to tag the selected digest
		trd.finish = append(trd.finish, func() error {
			mRef, ok := trd.manifests[d.Digest]
			if !ok {
				return fmt.Errorf("could not find manifest to tag, ref: %s, digest: %s", ref.CommonName(), d.Digest)
			}
			return rc.ManifestPut(ctx, ref, mRef)
		})
	} else if m.IsList() {
		// for index/manifest lists, add handlers for each embedded manifest
		mi, ok := m.(manifest.Indexer)
		if !ok {
			return fmt.Errorf("manifest doesn't support index methods%.0w", types.ErrUnsupportedMediaType)
		}
		dl, err := mi.GetManifestList()
		if err != nil {
			return err
		}
		for _, d := range dl {
			handleManifest(d, true)
		}
	} else {
		// else if a single image/manifest
		mi, ok := m.(manifest.Imager)
		if !ok {
			return fmt.Errorf("manifest doesn't support image methods%.0w", types.ErrUnsupportedMediaType)
		}
		// add handler for the config descriptor if it's defined
		cd, err := mi.GetConfig()
		if err == nil {
			filename := tarOCILayoutDescPath(cd)
			if !trd.processed[filename] && trd.handlers[filename] == nil {
				func(cd types.Descriptor) {
					trd.handlers[filename] = func(header *tar.Header, trd *tarReadData) error {
						return rc.imageImportBlob(ctx, ref, cd, trd)
					}
				}(cd)
			}
		}
		// add handlers for each layer
		layers, err := mi.GetLayers()
		if err != nil {
			return err
		}
		for _, d := range layers {
			filename := tarOCILayoutDescPath(d)
			if !trd.processed[filename] && trd.handlers[filename] == nil {
				func(d types.Descriptor) {
					trd.handlers[filename] = func(header *tar.Header, trd *tarReadData) error {
						return rc.imageImportBlob(ctx, ref, d, trd)
					}
				}(d)
			}
		}
	}
	// add a finish func to push the manifest, this gets skipped for the index.json
	if push {
		trd.finish = append(trd.finish, func() error {
			mRef := ref
			mRef.Digest = string(m.GetDescriptor().Digest)
			_, err := rc.ManifestHead(ctx, mRef)
			if err == nil {
				return nil
			}
			opts := []ManifestOpts{}
			if child {
				opts = append(opts, WithManifestChild())
			}
			return rc.ManifestPut(ctx, mRef, m, opts...)
		})
	}
	trd.handleAdded = true
	return nil
}

// imageImportOCIPushManifests uploads manifests after OCI blobs were successfully loaded
func (rc *RegClient) imageImportOCIPushManifests(ctx context.Context, ref ref.Ref, trd *tarReadData) error {
	// run finish handlers in reverse order to upload nested manifests
	for i := len(trd.finish) - 1; i >= 0; i-- {
		err := trd.finish[i]()
		if err != nil {
			return err
		}
	}
	return nil
}

func imagePlatformInList(target *platform.Platform, list []string) (bool, error) {
	// special case for an unset platform
	if target == nil || target.OS == "" {
		for _, entry := range list {
			if entry == "" {
				return true, nil
			}
		}
		return false, nil
	}
	for _, entry := range list {
		if entry == "" {
			continue
		}
		plat, err := platform.Parse(entry)
		if err != nil {
			return false, err
		}
		if platform.Match(*target, plat) {
			return true, nil
		}
	}
	return false, nil
}

// tarReadAll processes the tar file in a loop looking for matching filenames in the list of handlers
// handlers for filenames are added at the top level, and by manifest imports
func (trd *tarReadData) tarReadAll(rs io.ReadSeeker) error {
	// return immediately if nothing to do
	if len(trd.handlers) == 0 {
		return nil
	}
	for {
		// reset back to beginning of tar file
		_, err := rs.Seek(0, 0)
		if err != nil {
			return err
		}
		trd.tr = tar.NewReader(rs)
		trd.handleAdded = false
		// loop over each entry of the tar file
		for {
			header, err := trd.tr.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}
			name := filepath.Clean(header.Name)
			// if a handler exists, run it, remove handler, and check if we are done
			if trd.handlers[name] != nil {
				err = trd.handlers[name](header, trd)
				if err != nil {
					return err
				}
				delete(trd.handlers, name)
				trd.processed[name] = true
				// return if last handler processed
				if len(trd.handlers) == 0 {
					return nil
				}
			}
		}
		// if entire file read without adding a new handler, fail
		if !trd.handleAdded {
			return fmt.Errorf("unable to read all files from tar: %w", types.ErrNotFound)
		}
	}
}

// tarReadFileJSON reads the current tar entry and unmarshals json into provided interface
func (trd *tarReadData) tarReadFileJSON(data interface{}) error {
	b, err := io.ReadAll(trd.tr)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, data)
	if err != nil {
		return err
	}
	return nil
}

var errTarFileExists = errors.New("tar file already exists")

func (td *tarWriteData) tarWriteHeader(filename string, size int64) error {
	dirname := filepath.Dir(filename)
	if !td.dirs[dirname] && dirname != "." {
		header := tar.Header{
			Format:     tar.FormatPAX,
			Typeflag:   tar.TypeDir,
			Name:       dirname,
			Size:       0,
			Mode:       td.mode | 0511,
			ModTime:    td.timestamp,
			AccessTime: td.timestamp,
			ChangeTime: td.timestamp,
		}
		err := td.tw.WriteHeader(&header)
		if err != nil {
			return err
		}
		td.dirs[dirname] = true
	}
	if td.files[filename] {
		return fmt.Errorf("%w: %s", errTarFileExists, filename)
	}
	td.files[filename] = true
	header := tar.Header{
		Format:     tar.FormatPAX,
		Typeflag:   tar.TypeReg,
		Name:       filename,
		Size:       size,
		Mode:       td.mode | 0400,
		ModTime:    td.timestamp,
		AccessTime: td.timestamp,
		ChangeTime: td.timestamp,
	}
	return td.tw.WriteHeader(&header)
}

func (td *tarWriteData) tarWriteFileJSON(filename string, data interface{}) error {
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = td.tarWriteHeader(filename, int64(len(dataJSON)))
	if err != nil {
		return err
	}
	_, err = td.tw.Write(dataJSON)
	if err != nil {
		return err
	}
	return nil
}

func tarOCILayoutDescPath(d types.Descriptor) string {
	return filepath.Clean(fmt.Sprintf("blobs/%s/%s", d.Digest.Algorithm(), d.Digest.Encoded()))
}
