// Package manifest abstracts the various types of supported manifests.
// Supported types include OCI index and image, and Docker manifest list and manifest.
package manifest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	// crypto libraries included for go-digest
	_ "crypto/sha256"
	_ "crypto/sha512"

	digest "github.com/opencontainers/go-digest"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/docker/schema1"
	"github.com/regclient/regclient/types/docker/schema2"
	v1 "github.com/regclient/regclient/types/oci/v1"
	"github.com/regclient/regclient/types/platform"
	"github.com/regclient/regclient/types/ref"
)

// Manifest interface is implemented by all supported manifests but
// many calls are only supported by certain underlying media types.
type Manifest interface {
	GetDescriptor() types.Descriptor
	GetOrig() interface{}
	GetRef() ref.Ref
	IsList() bool
	IsSet() bool
	MarshalJSON() ([]byte, error)
	RawBody() ([]byte, error)
	RawHeaders() (http.Header, error)
	SetOrig(interface{}) error

	// Deprecated: GetConfig should be accessed using Imager interface
	GetConfig() (types.Descriptor, error)
	// Deprecated: GetLayers should be accessed using Imager interface
	GetLayers() ([]types.Descriptor, error)

	// Deprecated: GetManifestList should be accessed using Indexer interface
	GetManifestList() ([]types.Descriptor, error)

	// Deprecated: GetConfigDigest should be replaced with GetConfig
	GetConfigDigest() (digest.Digest, error)
	// Deprecated: GetDigest should be replaced with GetDescriptor().Digest
	GetDigest() digest.Digest
	// Deprecated: GetMediaType should be replaced with GetDescriptor().MediaType
	GetMediaType() string
	// Deprecated: GetPlatformDesc method should be replaced with manifest.GetPlatformDesc function
	GetPlatformDesc(p *platform.Platform) (*types.Descriptor, error)
	// Deprecated: GetPlatformList method should be replaced with manifest.GetPlatformList function
	GetPlatformList() ([]*platform.Platform, error)
	// Deprecated: GetRateLimit method should be replaced with manifest.GetRateLimit function
	GetRateLimit() types.RateLimit
	// Deprecated: HasRateLimit method should be replaced with manifest.HasRateLimit function
	HasRateLimit() bool
}

type Annotator interface {
	GetAnnotations() (map[string]string, error)
	SetAnnotation(key, val string) error
}

type Indexer interface {
	GetManifestList() ([]types.Descriptor, error)
	SetManifestList(dl []types.Descriptor) error
}

type Imager interface {
	GetConfig() (types.Descriptor, error)
	GetLayers() ([]types.Descriptor, error)
	SetConfig(d types.Descriptor) error
	SetLayers(dl []types.Descriptor) error
}

type Subjecter interface {
	GetSubject() (*types.Descriptor, error)
	SetSubject(d *types.Descriptor) error
}

type manifestConfig struct {
	r      ref.Ref
	desc   types.Descriptor
	raw    []byte
	orig   interface{}
	header http.Header
}
type Opts func(*manifestConfig)

// New creates a new manifest based on provided options
func New(opts ...Opts) (Manifest, error) {
	mc := manifestConfig{}
	for _, opt := range opts {
		opt(&mc)
	}
	c := common{
		r:         mc.r,
		desc:      mc.desc,
		rawBody:   mc.raw,
		rawHeader: mc.header,
	}
	// extract fields from header where available
	if mc.header != nil {
		if c.desc.MediaType == "" {
			c.desc.MediaType = mc.header.Get("Content-Type")
		}
		if c.desc.Size == 0 {
			cl, _ := strconv.Atoi(mc.header.Get("Content-Length"))
			c.desc.Size = int64(cl)
		}
		if c.desc.Digest == "" {
			c.desc.Digest, _ = digest.Parse(mc.header.Get("Docker-Content-Digest"))
		}
		c.setRateLimit(mc.header)
	}
	if mc.orig != nil {
		return fromOrig(c, mc.orig)
	}
	return fromCommon(c)
}

// WithDesc specifies the descriptor for the manifest
func WithDesc(desc types.Descriptor) Opts {
	return func(mc *manifestConfig) {
		mc.desc = desc
	}
}

// WithHeader provides the headers from the response when pulling the manifest
func WithHeader(header http.Header) Opts {
	return func(mc *manifestConfig) {
		mc.header = header
	}
}

// WithOrig provides the original manifest variable
func WithOrig(orig interface{}) Opts {
	return func(mc *manifestConfig) {
		mc.orig = orig
	}
}

// WithRaw provides the manifest bytes or HTTP response body
func WithRaw(raw []byte) Opts {
	return func(mc *manifestConfig) {
		mc.raw = raw
	}
}

// WithRef provides the reference used to get the manifest
func WithRef(r ref.Ref) Opts {
	return func(mc *manifestConfig) {
		mc.r = r
	}
}

// GetDigest returns the digest from the manifest descriptor
func GetDigest(m Manifest) digest.Digest {
	d := m.GetDescriptor()
	return d.Digest
}

// GetMediaType returns the media type from the manifest descriptor
func GetMediaType(m Manifest) string {
	d := m.GetDescriptor()
	return d.MediaType
}

// GetPlatformDesc returns the descriptor for a specific platform from an index
func GetPlatformDesc(m Manifest, p *platform.Platform) (*types.Descriptor, error) {
	dl, err := m.GetManifestList()
	if err != nil {
		return nil, err
	}
	return getPlatformDesc(p, dl)
}

// GetPlatformList returns the list of platforms from an index
func GetPlatformList(m Manifest) ([]*platform.Platform, error) {
	dl, err := m.GetManifestList()
	if err != nil {
		return nil, err
	}
	var l []*platform.Platform
	for _, d := range dl {
		if d.Platform != nil {
			l = append(l, d.Platform)
		}
	}
	return l, nil
}

// GetRateLimit returns the current rate limit seen in headers
func GetRateLimit(m Manifest) types.RateLimit {
	rl := types.RateLimit{}
	header, err := m.RawHeaders()
	if err != nil {
		return rl
	}
	// check for rate limit headers
	rlLimit := header.Get("RateLimit-Limit")
	rlRemain := header.Get("RateLimit-Remaining")
	rlReset := header.Get("RateLimit-Reset")
	if rlLimit != "" {
		lpSplit := strings.Split(rlLimit, ",")
		lSplit := strings.Split(lpSplit[0], ";")
		rlLimitI, err := strconv.Atoi(lSplit[0])
		if err != nil {
			rl.Limit = 0
		} else {
			rl.Limit = rlLimitI
		}
		if len(lSplit) > 1 {
			rl.Policies = lpSplit
		} else if len(lpSplit) > 1 {
			rl.Policies = lpSplit[1:]
		}
	}
	if rlRemain != "" {
		rSplit := strings.Split(rlRemain, ";")
		rlRemainI, err := strconv.Atoi(rSplit[0])
		if err != nil {
			rl.Remain = 0
		} else {
			rl.Remain = rlRemainI
			rl.Set = true
		}
	}
	if rlReset != "" {
		rlResetI, err := strconv.Atoi(rlReset)
		if err != nil {
			rl.Reset = 0
		} else {
			rl.Reset = rlResetI
		}
	}
	return rl
}

// HasRateLimit indicates whether the rate limit is set and available
func HasRateLimit(m Manifest) bool {
	rl := GetRateLimit(m)
	return rl.Set
}

func OCIIndexFromAny(orig interface{}) (v1.Index, error) {
	ociI := v1.Index{
		Versioned: v1.IndexSchemaVersion,
		MediaType: types.MediaTypeOCI1ManifestList,
	}
	switch orig := orig.(type) {
	case schema2.ManifestList:
		ociI.Manifests = orig.Manifests
		ociI.Annotations = orig.Annotations
	case v1.Index:
		ociI = orig
	default:
		return ociI, fmt.Errorf("unable to convert %T to OCI index", orig)
	}
	return ociI, nil
}

func OCIIndexToAny(ociI v1.Index, origP interface{}) error {
	// reflect is used to handle both *interface{} and *Manifest
	rv := reflect.ValueOf(origP)
	for rv.IsValid() && rv.Type().Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return fmt.Errorf("invalid manifest output parameter: %T", origP)
	}
	if !rv.CanSet() {
		return fmt.Errorf("manifest output must be a pointer: %T", origP)
	}
	origR := rv.Interface()
	switch orig := (origR).(type) {
	case schema2.ManifestList:
		orig.Versioned = schema2.ManifestListSchemaVersion
		orig.Manifests = ociI.Manifests
		orig.Annotations = ociI.Annotations
		rv.Set(reflect.ValueOf(orig))
	case v1.Index:
		rv.Set(reflect.ValueOf(ociI))
	default:
		return fmt.Errorf("unable to convert OCI index to %T", origR)
	}
	return nil
}

func OCIManifestFromAny(orig interface{}) (v1.Manifest, error) {
	ociM := v1.Manifest{
		Versioned: v1.ManifestSchemaVersion,
		MediaType: types.MediaTypeOCI1Manifest,
	}
	switch orig := orig.(type) {
	case schema2.Manifest:
		ociM.Config = orig.Config
		ociM.Layers = orig.Layers
		ociM.Annotations = orig.Annotations
	case v1.Manifest:
		ociM = orig
	default:
		// TODO: consider supporting Docker schema v1 media types
		return ociM, fmt.Errorf("unable to convert %T to OCI image", orig)
	}
	return ociM, nil
}

func OCIManifestToAny(ociM v1.Manifest, origP interface{}) error {
	// reflect is used to handle both *interface{} and *Manifest
	rv := reflect.ValueOf(origP)
	for rv.IsValid() && rv.Type().Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return fmt.Errorf("invalid manifest output parameter: %T", origP)
	}
	if !rv.CanSet() {
		return fmt.Errorf("manifest output must be a pointer: %T", origP)
	}
	origR := rv.Interface()
	switch orig := (origR).(type) {
	case schema2.Manifest:
		orig.Versioned = schema2.ManifestSchemaVersion
		orig.Config = ociM.Config
		orig.Layers = ociM.Layers
		orig.Annotations = ociM.Annotations
		rv.Set(reflect.ValueOf(orig))
	case v1.Manifest:
		rv.Set(reflect.ValueOf(ociM))
	default:
		// Docker schema v1 will not be supported, can't resign, and no need for unsigned
		return fmt.Errorf("unable to convert OCI image to %T", origR)
	}
	return nil
}

// FromOrig creates a new manifest from the original upstream manifest type.
// This method should be used if you are creating a new manifest rather than pulling one from a registry.
func fromOrig(c common, orig interface{}) (Manifest, error) {
	var mt string
	var m Manifest
	origDigest := c.desc.Digest

	mj, err := json.Marshal(orig)
	if err != nil {
		return nil, err
	}
	c.manifSet = true
	if len(c.rawBody) == 0 {
		c.rawBody = mj
	}
	if _, ok := orig.(schema1.SignedManifest); !ok {
		c.desc.Digest = digest.FromBytes(mj)
	}
	if c.desc.Size == 0 {
		c.desc.Size = int64(len(mj))
	}
	// create manifest based on type
	switch mOrig := orig.(type) {
	case schema1.Manifest:
		mt = mOrig.MediaType
		c.desc.MediaType = types.MediaTypeDocker1Manifest
		m = &docker1Manifest{
			common:   c,
			Manifest: mOrig,
		}
	case schema1.SignedManifest:
		mt = mOrig.MediaType
		c.desc.MediaType = types.MediaTypeDocker1ManifestSigned
		// recompute digest on the canonical data
		c.desc.Digest = digest.FromBytes(mOrig.Canonical)
		m = &docker1SignedManifest{
			common:         c,
			SignedManifest: mOrig,
		}
	case schema2.Manifest:
		mt = mOrig.MediaType
		c.desc.MediaType = types.MediaTypeDocker2Manifest
		m = &docker2Manifest{
			common:   c,
			Manifest: mOrig,
		}
	case schema2.ManifestList:
		mt = mOrig.MediaType
		c.desc.MediaType = types.MediaTypeDocker2ManifestList
		m = &docker2ManifestList{
			common:       c,
			ManifestList: mOrig,
		}
	case v1.Manifest:
		mt = mOrig.MediaType
		c.desc.MediaType = types.MediaTypeOCI1Manifest
		m = &oci1Manifest{
			common:   c,
			Manifest: mOrig,
		}
	case v1.Index:
		mt = mOrig.MediaType
		c.desc.MediaType = types.MediaTypeOCI1ManifestList
		m = &oci1Index{
			common: c,
			Index:  orig.(v1.Index),
		}
	case v1.ArtifactManifest:
		mt = mOrig.MediaType
		c.desc.MediaType = types.MediaTypeOCI1Artifact
		m = &oci1Artifact{
			common:           c,
			ArtifactManifest: mOrig,
		}
	default:
		return nil, fmt.Errorf("unsupported type to convert to a manifest: %T", orig)
	}
	// verify media type
	err = verifyMT(c.desc.MediaType, mt)
	if err != nil {
		return nil, err
	}
	// verify digest didn't change
	if origDigest != "" && origDigest != c.desc.Digest {
		return nil, fmt.Errorf("manifest digest mismatch, expected %s, computed %s", origDigest, c.desc.Digest)
	}
	return m, nil
}

func fromCommon(c common) (Manifest, error) {
	var err error
	var m Manifest
	var mt string
	origDigest := c.desc.Digest
	// extract common data from from rawBody
	if len(c.rawBody) > 0 {
		c.manifSet = true
		// extract media type from body if needed
		if c.desc.MediaType == "" {
			mt := struct {
				MediaType     string        `json:"mediaType,omitempty"`
				SchemaVersion int           `json:"schemaVersion,omitempty"`
				Signatures    []interface{} `json:"signatures,omitempty"`
			}{}
			err = json.Unmarshal(c.rawBody, &mt)
			if mt.MediaType != "" {
				c.desc.MediaType = mt.MediaType
			} else if mt.SchemaVersion == 1 && len(mt.Signatures) > 0 {
				c.desc.MediaType = types.MediaTypeDocker1ManifestSigned
			} else if mt.SchemaVersion == 1 {
				c.desc.MediaType = types.MediaTypeDocker1Manifest
			}
		}
		// compute digest
		if c.desc.MediaType != types.MediaTypeDocker1ManifestSigned {
			d := digest.FromBytes(c.rawBody)
			c.desc.Digest = d
			c.desc.Size = int64(len(c.rawBody))
		}
	}
	switch c.desc.MediaType {
	case types.MediaTypeDocker1Manifest:
		var mOrig schema1.Manifest
		if len(c.rawBody) > 0 {
			err = json.Unmarshal(c.rawBody, &mOrig)
			mt = mOrig.MediaType
		}
		m = &docker1Manifest{common: c, Manifest: mOrig}
	case types.MediaTypeDocker1ManifestSigned:
		var mOrig schema1.SignedManifest
		if len(c.rawBody) > 0 {
			err = json.Unmarshal(c.rawBody, &mOrig)
			mt = mOrig.MediaType
			d := digest.FromBytes(mOrig.Canonical)
			c.desc.Digest = d
			c.desc.Size = int64(len(mOrig.Canonical))
		}
		m = &docker1SignedManifest{common: c, SignedManifest: mOrig}
	case types.MediaTypeDocker2Manifest:
		var mOrig schema2.Manifest
		if len(c.rawBody) > 0 {
			err = json.Unmarshal(c.rawBody, &mOrig)
			mt = mOrig.MediaType
		}
		m = &docker2Manifest{common: c, Manifest: mOrig}
	case types.MediaTypeDocker2ManifestList:
		var mOrig schema2.ManifestList
		if len(c.rawBody) > 0 {
			err = json.Unmarshal(c.rawBody, &mOrig)
			mt = mOrig.MediaType
		}
		m = &docker2ManifestList{common: c, ManifestList: mOrig}
	case types.MediaTypeOCI1Manifest:
		var mOrig v1.Manifest
		if len(c.rawBody) > 0 {
			err = json.Unmarshal(c.rawBody, &mOrig)
			mt = mOrig.MediaType
		}
		m = &oci1Manifest{common: c, Manifest: mOrig}
	case types.MediaTypeOCI1ManifestList:
		var mOrig v1.Index
		if len(c.rawBody) > 0 {
			err = json.Unmarshal(c.rawBody, &mOrig)
			mt = mOrig.MediaType
		}
		m = &oci1Index{common: c, Index: mOrig}
	case types.MediaTypeOCI1Artifact:
		var mOrig v1.ArtifactManifest
		if len(c.rawBody) > 0 {
			err = json.Unmarshal(c.rawBody, &mOrig)
			mt = mOrig.MediaType
		}
		m = &oci1Artifact{common: c, ArtifactManifest: mOrig}
	default:
		return nil, fmt.Errorf("%w: \"%s\"", types.ErrUnsupportedMediaType, c.desc.MediaType)
	}
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling manifest for %s: %w", c.r.CommonName(), err)
	}
	// verify media type
	err = verifyMT(c.desc.MediaType, mt)
	if err != nil {
		return nil, err
	}
	// verify digest didn't change
	if origDigest != "" && origDigest != c.desc.Digest {
		return nil, fmt.Errorf("manifest digest mismatch, expected %s, computed %s", origDigest, c.desc.Digest)
	}
	return m, nil
}

func verifyMT(expected, received string) error {
	if received != "" && expected != received {
		return fmt.Errorf("manifest contains an unexpected media type: expected %s, received %s", expected, received)
	}
	return nil
}

func getPlatformDesc(p *platform.Platform, dl []types.Descriptor) (*types.Descriptor, error) {
	if p == nil {
		return nil, fmt.Errorf("invalid input, platform is nil%.0w", types.ErrNotFound)
	}
	for _, d := range dl {
		if d.Platform != nil && platform.Match(*p, *d.Platform) {
			return &d, nil
		}
	}
	// if no platforms match, fall back to searching for a compatible platform (Mac runs Linux images)
	for _, d := range dl {
		if d.Platform != nil && platform.Compatible(*p, *d.Platform) {
			return &d, nil
		}
	}
	return nil, fmt.Errorf("platform not found: %s%.0w", *p, types.ErrNotFound)
}

func getPlatformList(dl []types.Descriptor) ([]*platform.Platform, error) {
	var l []*platform.Platform
	for _, d := range dl {
		if d.Platform != nil {
			l = append(l, d.Platform)
		}
	}
	return l, nil
}
