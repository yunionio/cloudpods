package manifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"text/tabwriter"

	// crypto libraries included for go-digest
	_ "crypto/sha256"
	_ "crypto/sha512"

	digest "github.com/opencontainers/go-digest"
	"github.com/regclient/regclient/internal/units"
	"github.com/regclient/regclient/types"
	v1 "github.com/regclient/regclient/types/oci/v1"
	"github.com/regclient/regclient/types/platform"
)

const (
	// MediaTypeOCI1Manifest OCI v1 manifest media type
	MediaTypeOCI1Manifest = types.MediaTypeOCI1Manifest
	// MediaTypeOCI1ManifestList OCI v1 manifest list media type
	MediaTypeOCI1ManifestList = types.MediaTypeOCI1ManifestList
)

type oci1Manifest struct {
	common
	v1.Manifest
}
type oci1Index struct {
	common
	v1.Index
}

// oci1Artifact is EXPERIMENTAL
type oci1Artifact struct {
	common
	v1.ArtifactManifest
}

func (m *oci1Manifest) GetAnnotations() (map[string]string, error) {
	if !m.manifSet {
		return nil, types.ErrManifestNotSet
	}
	return m.Annotations, nil
}
func (m *oci1Manifest) GetConfig() (types.Descriptor, error) {
	if !m.manifSet {
		return types.Descriptor{}, types.ErrManifestNotSet
	}
	return m.Config, nil
}
func (m *oci1Manifest) GetConfigDigest() (digest.Digest, error) {
	if !m.manifSet {
		return digest.Digest(""), types.ErrManifestNotSet
	}
	return m.Config.Digest, nil
}
func (m *oci1Index) GetAnnotations() (map[string]string, error) {
	if !m.manifSet {
		return nil, types.ErrManifestNotSet
	}
	return m.Annotations, nil
}
func (m *oci1Index) GetConfig() (types.Descriptor, error) {
	return types.Descriptor{}, fmt.Errorf("config digest not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}
func (m *oci1Index) GetConfigDigest() (digest.Digest, error) {
	return "", fmt.Errorf("config digest not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}
func (m *oci1Artifact) GetAnnotations() (map[string]string, error) {
	if !m.manifSet {
		return nil, types.ErrManifestNotSet
	}
	return m.Annotations, nil
}
func (m *oci1Artifact) GetConfig() (types.Descriptor, error) {
	return types.Descriptor{}, fmt.Errorf("config digest not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}
func (m *oci1Artifact) GetConfigDigest() (digest.Digest, error) {
	return "", fmt.Errorf("config digest not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}

func (m *oci1Manifest) GetManifestList() ([]types.Descriptor, error) {
	return []types.Descriptor{}, fmt.Errorf("platform descriptor list not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}
func (m *oci1Index) GetManifestList() ([]types.Descriptor, error) {
	if !m.manifSet {
		return nil, types.ErrManifestNotSet
	}
	return m.Manifests, nil
}
func (m *oci1Artifact) GetManifestList() ([]types.Descriptor, error) {
	return []types.Descriptor{}, fmt.Errorf("platform descriptor list not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}

func (m *oci1Manifest) GetLayers() ([]types.Descriptor, error) {
	if !m.manifSet {
		return nil, types.ErrManifestNotSet
	}
	return m.Layers, nil
}
func (m *oci1Index) GetLayers() ([]types.Descriptor, error) {
	return []types.Descriptor{}, fmt.Errorf("layers are not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}
func (m *oci1Artifact) GetLayers() ([]types.Descriptor, error) {
	if !m.manifSet {
		return nil, types.ErrManifestNotSet
	}
	return m.Blobs, nil
}

func (m *oci1Manifest) GetOrig() interface{} {
	return m.Manifest
}
func (m *oci1Index) GetOrig() interface{} {
	return m.Index
}
func (m *oci1Artifact) GetOrig() interface{} {
	return m.ArtifactManifest
}

func (m *oci1Manifest) GetPlatformDesc(p *platform.Platform) (*types.Descriptor, error) {
	return nil, fmt.Errorf("platform lookup not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}
func (m *oci1Index) GetPlatformDesc(p *platform.Platform) (*types.Descriptor, error) {
	dl, err := m.GetManifestList()
	if err != nil {
		return nil, err
	}
	return getPlatformDesc(p, dl)
}
func (m *oci1Artifact) GetPlatformDesc(p *platform.Platform) (*types.Descriptor, error) {
	return nil, fmt.Errorf("platform lookup not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}

func (m *oci1Manifest) GetPlatformList() ([]*platform.Platform, error) {
	return nil, fmt.Errorf("platform list not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}
func (m *oci1Index) GetPlatformList() ([]*platform.Platform, error) {
	dl, err := m.GetManifestList()
	if err != nil {
		return nil, err
	}
	return getPlatformList(dl)
}
func (m *oci1Artifact) GetPlatformList() ([]*platform.Platform, error) {
	return nil, fmt.Errorf("platform list not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}

func (m *oci1Manifest) MarshalJSON() ([]byte, error) {
	if !m.manifSet {
		return []byte{}, types.ErrManifestNotSet
	}

	if len(m.rawBody) > 0 {
		return m.rawBody, nil
	}

	return json.Marshal((m.Manifest))
}
func (m *oci1Manifest) GetSubject() (*types.Descriptor, error) {
	if !m.manifSet {
		return nil, types.ErrManifestNotSet
	}
	// TODO: remove fall back support for refers in a future release
	if m.Manifest.Subject == nil && m.Manifest.Refers != nil {
		return m.Manifest.Refers, nil
	}
	return m.Manifest.Subject, nil
}
func (m *oci1Artifact) GetSubject() (*types.Descriptor, error) {
	if !m.manifSet {
		return nil, types.ErrManifestNotSet
	}
	// TODO: remove fall back support for refers in a future release
	if m.ArtifactManifest.Subject == nil && m.ArtifactManifest.Refers != nil {
		return m.ArtifactManifest.Refers, nil
	}
	return m.ArtifactManifest.Subject, nil
}

func (m *oci1Index) MarshalJSON() ([]byte, error) {
	if !m.manifSet {
		return []byte{}, types.ErrManifestNotSet
	}

	if len(m.rawBody) > 0 {
		return m.rawBody, nil
	}

	return json.Marshal((m.Index))
}
func (m *oci1Artifact) MarshalJSON() ([]byte, error) {
	if !m.manifSet {
		return []byte{}, types.ErrManifestNotSet
	}

	if len(m.rawBody) > 0 {
		return m.rawBody, nil
	}

	return json.Marshal((m.ArtifactManifest))
}

func (m *oci1Manifest) MarshalPretty() ([]byte, error) {
	if m == nil {
		return []byte{}, nil
	}
	buf := &bytes.Buffer{}
	tw := tabwriter.NewWriter(buf, 0, 0, 1, ' ', 0)
	if m.r.Reference != "" {
		fmt.Fprintf(tw, "Name:\t%s\n", m.r.Reference)
	}
	fmt.Fprintf(tw, "MediaType:\t%s\n", m.desc.MediaType)
	fmt.Fprintf(tw, "Digest:\t%s\n", m.desc.Digest.String())
	if m.Annotations != nil && len(m.Annotations) > 0 {
		fmt.Fprintf(tw, "Annotations:\t\n")
		keys := make([]string, 0, len(m.Annotations))
		for k := range m.Annotations {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, name := range keys {
			val := m.Annotations[name]
			fmt.Fprintf(tw, "  %s:\t%s\n", name, val)
		}
	}
	var total int64
	for _, d := range m.Layers {
		total += d.Size
	}
	fmt.Fprintf(tw, "Total Size:\t%s\n", units.HumanSize(float64(total)))
	fmt.Fprintf(tw, "\t\n")
	fmt.Fprintf(tw, "Config:\t\n")
	err := m.Config.MarshalPrettyTW(tw, "  ")
	if err != nil {
		return []byte{}, err
	}
	fmt.Fprintf(tw, "\t\n")
	fmt.Fprintf(tw, "Layers:\t\n")
	for _, d := range m.Layers {
		fmt.Fprintf(tw, "\t\n")
		err := d.MarshalPrettyTW(tw, "  ")
		if err != nil {
			return []byte{}, err
		}
	}
	if m.Subject != nil {
		fmt.Fprintf(tw, "\t\n")
		fmt.Fprintf(tw, "Subject:\t\n")
		err := m.Subject.MarshalPrettyTW(tw, "  ")
		if err != nil {
			return []byte{}, err
		}
	}
	tw.Flush()
	return buf.Bytes(), nil
}
func (m *oci1Index) MarshalPretty() ([]byte, error) {
	if m == nil {
		return []byte{}, nil
	}
	buf := &bytes.Buffer{}
	tw := tabwriter.NewWriter(buf, 0, 0, 1, ' ', 0)
	if m.r.Reference != "" {
		fmt.Fprintf(tw, "Name:\t%s\n", m.r.Reference)
	}
	fmt.Fprintf(tw, "MediaType:\t%s\n", m.desc.MediaType)
	fmt.Fprintf(tw, "Digest:\t%s\n", m.desc.Digest.String())
	if m.Annotations != nil && len(m.Annotations) > 0 {
		fmt.Fprintf(tw, "Annotations:\t\n")
		keys := make([]string, 0, len(m.Annotations))
		for k := range m.Annotations {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, name := range keys {
			val := m.Annotations[name]
			fmt.Fprintf(tw, "  %s:\t%s\n", name, val)
		}
	}
	fmt.Fprintf(tw, "\t\n")
	fmt.Fprintf(tw, "Manifests:\t\n")
	for _, d := range m.Manifests {
		fmt.Fprintf(tw, "\t\n")
		dRef := m.r
		if dRef.Reference != "" {
			dRef.Digest = d.Digest.String()
			fmt.Fprintf(tw, "  Name:\t%s\n", dRef.CommonName())
		}
		err := d.MarshalPrettyTW(tw, "  ")
		if err != nil {
			return []byte{}, err
		}
	}
	tw.Flush()
	return buf.Bytes(), nil
}
func (m *oci1Artifact) MarshalPretty() ([]byte, error) {
	if m == nil {
		return []byte{}, nil
	}
	buf := &bytes.Buffer{}
	tw := tabwriter.NewWriter(buf, 0, 0, 1, ' ', 0)
	if m.r.Reference != "" {
		fmt.Fprintf(tw, "Name:\t%s\n", m.r.Reference)
	}
	fmt.Fprintf(tw, "MediaType:\t%s\n", m.desc.MediaType)
	fmt.Fprintf(tw, "Digest:\t%s\n", m.desc.Digest.String())
	if m.Annotations != nil && len(m.Annotations) > 0 {
		fmt.Fprintf(tw, "Annotations:\t\n")
		keys := make([]string, 0, len(m.Annotations))
		for k := range m.Annotations {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, name := range keys {
			val := m.Annotations[name]
			fmt.Fprintf(tw, "  %s:\t%s\n", name, val)
		}
	}
	var total int64
	for _, d := range m.Blobs {
		total += d.Size
	}
	fmt.Fprintf(tw, "Total Size:\t%s\n", units.HumanSize(float64(total)))
	fmt.Fprintf(tw, "\t\n")
	fmt.Fprintf(tw, "Blobs:\t\n")
	for _, d := range m.Blobs {
		fmt.Fprintf(tw, "\t\n")
		err := d.MarshalPrettyTW(tw, "  ")
		if err != nil {
			return []byte{}, err
		}
	}
	if m.Subject != nil {
		fmt.Fprintf(tw, "\t\n")
		fmt.Fprintf(tw, "Subject:\t\n")
		err := m.Subject.MarshalPrettyTW(tw, "  ")
		if err != nil {
			return []byte{}, err
		}
	}
	tw.Flush()
	return buf.Bytes(), nil
}

func (m *oci1Manifest) SetAnnotation(key, val string) error {
	if !m.manifSet {
		return types.ErrManifestNotSet
	}
	if m.Annotations == nil {
		m.Annotations = map[string]string{}
	}
	m.Annotations[key] = val
	return m.updateDesc()
}
func (m *oci1Index) SetAnnotation(key, val string) error {
	if !m.manifSet {
		return types.ErrManifestNotSet
	}
	if m.Annotations == nil {
		m.Annotations = map[string]string{}
	}
	m.Annotations[key] = val
	return m.updateDesc()
}
func (m *oci1Artifact) SetAnnotation(key, val string) error {
	if !m.manifSet {
		return types.ErrManifestNotSet
	}
	if m.Annotations == nil {
		m.Annotations = map[string]string{}
	}
	m.Annotations[key] = val
	return m.updateDesc()
}

func (m *oci1Artifact) SetConfig(d types.Descriptor) error {
	return fmt.Errorf("set config not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}

func (m *oci1Manifest) SetConfig(d types.Descriptor) error {
	if !m.manifSet {
		return types.ErrManifestNotSet
	}
	m.Config = d
	return m.updateDesc()
}

func (m *oci1Artifact) SetLayers(dl []types.Descriptor) error {
	if !m.manifSet {
		return types.ErrManifestNotSet
	}
	m.Blobs = dl
	return m.updateDesc()
}

func (m *oci1Manifest) SetLayers(dl []types.Descriptor) error {
	if !m.manifSet {
		return types.ErrManifestNotSet
	}
	m.Layers = dl
	return m.updateDesc()
}

func (m *oci1Index) SetManifestList(dl []types.Descriptor) error {
	if !m.manifSet {
		return types.ErrManifestNotSet
	}
	m.Manifests = dl
	return m.updateDesc()
}

func (m *oci1Manifest) SetOrig(origIn interface{}) error {
	orig, ok := origIn.(v1.Manifest)
	if !ok {
		return types.ErrUnsupportedMediaType
	}
	if orig.MediaType != types.MediaTypeOCI1Manifest {
		// TODO: error?
		orig.MediaType = types.MediaTypeOCI1Manifest
	}
	m.manifSet = true
	m.Manifest = orig

	return m.updateDesc()
}

func (m *oci1Index) SetOrig(origIn interface{}) error {
	orig, ok := origIn.(v1.Index)
	if !ok {
		return types.ErrUnsupportedMediaType
	}
	if orig.MediaType != types.MediaTypeOCI1ManifestList {
		// TODO: error?
		orig.MediaType = types.MediaTypeOCI1ManifestList
	}
	m.manifSet = true
	m.Index = orig

	return m.updateDesc()
}

func (m *oci1Artifact) SetSubject(d *types.Descriptor) error {
	if !m.manifSet {
		return types.ErrManifestNotSet
	}
	m.ArtifactManifest.Subject = d
	return m.updateDesc()
}
func (m *oci1Manifest) SetSubject(d *types.Descriptor) error {
	if !m.manifSet {
		return types.ErrManifestNotSet
	}
	m.Manifest.Subject = d
	return m.updateDesc()
}

func (m *oci1Artifact) SetOrig(origIn interface{}) error {
	orig, ok := origIn.(v1.ArtifactManifest)
	if !ok {
		return types.ErrUnsupportedMediaType
	}
	if orig.MediaType != types.MediaTypeOCI1Artifact {
		// TODO: error?
		orig.MediaType = types.MediaTypeOCI1Artifact
	}
	m.manifSet = true
	m.ArtifactManifest = orig

	return m.updateDesc()
}

func (m *oci1Manifest) updateDesc() error {
	mj, err := json.Marshal(m.Manifest)
	if err != nil {
		return err
	}
	m.rawBody = mj
	m.desc = types.Descriptor{
		MediaType: types.MediaTypeOCI1Manifest,
		Digest:    digest.FromBytes(mj),
		Size:      int64(len(mj)),
	}
	return nil
}
func (m *oci1Index) updateDesc() error {
	mj, err := json.Marshal(m.Index)
	if err != nil {
		return err
	}
	m.rawBody = mj
	m.desc = types.Descriptor{
		MediaType: types.MediaTypeOCI1ManifestList,
		Digest:    digest.FromBytes(mj),
		Size:      int64(len(mj)),
	}
	return nil
}
func (m *oci1Artifact) updateDesc() error {
	mj, err := json.Marshal(m.ArtifactManifest)
	if err != nil {
		return err
	}
	m.rawBody = mj
	m.desc = types.Descriptor{
		MediaType: types.MediaTypeOCI1Artifact,
		Digest:    digest.FromBytes(mj),
		Size:      int64(len(mj)),
	}
	return nil
}
