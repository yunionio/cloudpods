package manifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/tabwriter"

	// crypto libraries included for go-digest
	_ "crypto/sha256"
	_ "crypto/sha512"

	digest "github.com/opencontainers/go-digest"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/docker/schema1"
	"github.com/regclient/regclient/types/platform"
)

const (
	// MediaTypeDocker1Manifest deprecated media type for docker schema1 manifests
	MediaTypeDocker1Manifest = "application/vnd.docker.distribution.manifest.v1+json"
	// MediaTypeDocker1ManifestSigned is a deprecated schema1 manifest with jws signing
	MediaTypeDocker1ManifestSigned = "application/vnd.docker.distribution.manifest.v1+prettyjws"
)

type docker1Manifest struct {
	common
	schema1.Manifest
}
type docker1SignedManifest struct {
	common
	schema1.SignedManifest
}

func (m *docker1Manifest) GetConfig() (types.Descriptor, error) {
	return types.Descriptor{}, fmt.Errorf("config digest not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}
func (m *docker1Manifest) GetConfigDigest() (digest.Digest, error) {
	return "", fmt.Errorf("config digest not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}
func (m *docker1SignedManifest) GetConfig() (types.Descriptor, error) {
	return types.Descriptor{}, fmt.Errorf("config digest not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}
func (m *docker1SignedManifest) GetConfigDigest() (digest.Digest, error) {
	return "", fmt.Errorf("config digest not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}

func (m *docker1Manifest) GetManifestList() ([]types.Descriptor, error) {
	return []types.Descriptor{}, fmt.Errorf("platform descriptor list not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}
func (m *docker1SignedManifest) GetManifestList() ([]types.Descriptor, error) {
	return []types.Descriptor{}, fmt.Errorf("platform descriptor list not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}

func (m *docker1Manifest) GetLayers() ([]types.Descriptor, error) {
	if !m.manifSet {
		return []types.Descriptor{}, types.ErrManifestNotSet
	}

	var dl []types.Descriptor
	for _, sd := range m.FSLayers {
		dl = append(dl, types.Descriptor{
			Digest: sd.BlobSum,
		})
	}
	return dl, nil
}
func (m *docker1SignedManifest) GetLayers() ([]types.Descriptor, error) {
	if !m.manifSet {
		return []types.Descriptor{}, types.ErrManifestNotSet
	}

	var dl []types.Descriptor
	for _, sd := range m.FSLayers {
		dl = append(dl, types.Descriptor{
			Digest: sd.BlobSum,
		})
	}
	return dl, nil
}

func (m *docker1Manifest) GetOrig() interface{} {
	return m.Manifest
}
func (m *docker1SignedManifest) GetOrig() interface{} {
	return m.SignedManifest
}

func (m *docker1Manifest) GetPlatformDesc(p *platform.Platform) (*types.Descriptor, error) {
	return nil, fmt.Errorf("platform lookup not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}
func (m *docker1SignedManifest) GetPlatformDesc(p *platform.Platform) (*types.Descriptor, error) {
	return nil, fmt.Errorf("platform lookup not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}

func (m *docker1Manifest) GetPlatformList() ([]*platform.Platform, error) {
	return nil, fmt.Errorf("platform list not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}
func (m *docker1SignedManifest) GetPlatformList() ([]*platform.Platform, error) {
	return nil, fmt.Errorf("platform list not available for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}

func (m *docker1Manifest) MarshalJSON() ([]byte, error) {
	if !m.manifSet {
		return []byte{}, types.ErrManifestNotSet
	}

	if len(m.rawBody) > 0 {
		return m.rawBody, nil
	}

	return json.Marshal((m.Manifest))
}

func (m *docker1SignedManifest) MarshalJSON() ([]byte, error) {
	if !m.manifSet {
		return []byte{}, types.ErrManifestNotSet
	}

	return m.SignedManifest.MarshalJSON()
}

func (m *docker1Manifest) MarshalPretty() ([]byte, error) {
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
	fmt.Fprintf(tw, "\t\n")
	fmt.Fprintf(tw, "Layers:\t\n")
	for _, d := range m.FSLayers {
		fmt.Fprintf(tw, "  Digest:\t%s\n", string(d.BlobSum))
	}
	tw.Flush()
	return buf.Bytes(), nil
}
func (m *docker1SignedManifest) MarshalPretty() ([]byte, error) {
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
	fmt.Fprintf(tw, "\t\n")
	fmt.Fprintf(tw, "Layers:\t\n")
	for _, d := range m.FSLayers {
		fmt.Fprintf(tw, "  Digest:\t%s\n", string(d.BlobSum))
	}
	tw.Flush()
	return buf.Bytes(), nil
}

func (m *docker1Manifest) SetConfig(d types.Descriptor) error {
	return fmt.Errorf("set methods not supported for for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}

func (m *docker1SignedManifest) SetConfig(d types.Descriptor) error {
	return fmt.Errorf("set methods not supported for for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}

func (m *docker1Manifest) SetLayers(dl []types.Descriptor) error {
	return fmt.Errorf("set methods not supported for for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}

func (m *docker1SignedManifest) SetLayers(dl []types.Descriptor) error {
	return fmt.Errorf("set methods not supported for for media type %s%.0w", m.desc.MediaType, types.ErrUnsupportedMediaType)
}

func (m *docker1Manifest) SetOrig(origIn interface{}) error {
	orig, ok := origIn.(schema1.Manifest)
	if !ok {
		return types.ErrUnsupportedMediaType
	}
	if orig.MediaType != types.MediaTypeDocker1Manifest {
		// TODO: error?
		orig.MediaType = types.MediaTypeDocker1Manifest
	}
	mj, err := json.Marshal(orig)
	if err != nil {
		return err
	}
	m.manifSet = true
	m.rawBody = mj
	m.desc = types.Descriptor{
		MediaType: types.MediaTypeDocker1Manifest,
		Digest:    digest.FromBytes(mj),
		Size:      int64(len(mj)),
	}
	m.Manifest = orig

	return nil
}

func (m *docker1SignedManifest) SetOrig(origIn interface{}) error {
	orig, ok := origIn.(schema1.SignedManifest)
	if !ok {
		return types.ErrUnsupportedMediaType
	}
	if orig.MediaType != types.MediaTypeDocker1ManifestSigned {
		// TODO: error?
		orig.MediaType = types.MediaTypeDocker1ManifestSigned
	}
	mj, err := json.Marshal(orig)
	if err != nil {
		return err
	}
	m.manifSet = true
	m.rawBody = mj
	m.desc = types.Descriptor{
		MediaType: types.MediaTypeDocker1ManifestSigned,
		Digest:    digest.FromBytes(mj),
		Size:      int64(len(mj)),
	}
	m.SignedManifest = orig

	return nil
}
