package types

import (
	"fmt"
	"strings"
	"text/tabwriter"

	// crypto libraries included for go-digest
	_ "crypto/sha256"
	_ "crypto/sha512"

	"github.com/opencontainers/go-digest"
	"github.com/regclient/regclient/internal/units"
	"github.com/regclient/regclient/types/platform"
)

// Descriptor is used in manifests to refer to content by media type, size, and digest.
type Descriptor struct {
	// MediaType describe the type of the content.
	MediaType string `json:"mediaType,omitempty"`

	// Size in bytes of content.
	Size int64 `json:"size,omitempty"`

	// Digest uniquely identifies the content.
	Digest digest.Digest `json:"digest,omitempty"`

	// URLs contains the source URLs of this content.
	URLs []string `json:"urls,omitempty"`

	// Annotations contains arbitrary metadata relating to the targeted content.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Data is an embedding of the targeted content. This is encoded as a base64
	// string when marshalled to JSON (automatically, by encoding/json). If
	// present, Data can be used directly to avoid fetching the targeted content.
	Data []byte `json:"data,omitempty"`

	// Platform describes the platform which the image in the manifest runs on.
	// This should only be used when referring to a manifest.
	Platform *platform.Platform `json:"platform,omitempty"`

	// ArtifactType is the media type of the artifact this descriptor refers to.
	ArtifactType string `json:"artifactType,omitempty"`
}

var ScratchData = []byte("{}")
var ScratchDigest = digest.FromBytes(ScratchData)
var emptyDigest = digest.FromBytes([]byte{})
var mtToOCI map[string]string

func init() {
	mtToOCI = map[string]string{
		MediaTypeDocker2ManifestList: MediaTypeOCI1ManifestList,
		MediaTypeDocker2Manifest:     MediaTypeOCI1Manifest,
		MediaTypeDocker2ImageConfig:  MediaTypeOCI1ImageConfig,
		MediaTypeDocker2LayerGzip:    MediaTypeOCI1LayerGzip,
		MediaTypeOCI1ManifestList:    MediaTypeOCI1ManifestList,
		MediaTypeOCI1Manifest:        MediaTypeOCI1Manifest,
		MediaTypeOCI1ImageConfig:     MediaTypeOCI1ImageConfig,
		MediaTypeOCI1LayerGzip:       MediaTypeOCI1LayerGzip,
	}
}

// GetData decodes the Data field from the descriptor if available
func (d Descriptor) GetData() ([]byte, error) {
	if len(d.Data) == 0 && d.Digest != emptyDigest {
		return nil, ErrParsingFailed
	}
	// verify length
	if int64(len(d.Data)) != d.Size {
		return nil, ErrParsingFailed
	}
	// generate and verify digest
	dDig := digest.FromBytes(d.Data)
	if d.Digest != dDig {
		return nil, ErrParsingFailed
	}
	// return data
	return d.Data, nil
}

// Equal indicates the two descriptors are identical, effectively a DeepEqual.
func (d Descriptor) Equal(d2 Descriptor) bool {
	if !d.Same(d2) {
		return false
	}
	if d.MediaType != d2.MediaType {
		return false
	}
	if d.ArtifactType != d2.ArtifactType {
		return false
	}
	if d.Platform == nil || d2.Platform == nil {
		if d.Platform != nil || d2.Platform != nil {
			return false
		}
	} else if !platform.Match(*d.Platform, *d2.Platform) {
		return false
	}
	if d.URLs == nil || d2.URLs == nil {
		if d.URLs != nil || d2.URLs != nil {
			return false
		}
	} else if len(d.URLs) != len(d2.URLs) {
		return false
	} else {
		for i := range d.URLs {
			if d.URLs[i] != d2.URLs[i] {
				return false
			}
		}
	}
	if d.Annotations == nil || d2.Annotations == nil {
		if d.Annotations != nil || d2.Annotations != nil {
			return false
		}
	} else if len(d.Annotations) != len(d2.Annotations) {
		return false
	} else {
		for i := range d.Annotations {
			if d.Annotations[i] != d2.Annotations[i] {
				return false
			}
		}
	}
	return true
}

// Same indicates two descriptors point to the same CAS object.
// This verifies the digest, media type, and size all match
func (d Descriptor) Same(d2 Descriptor) bool {
	if d.Digest != d2.Digest || d.Size != d2.Size {
		return false
	}
	// loosen the check on media type since this can be converted from a build
	if d.MediaType != d2.MediaType {
		if _, ok := mtToOCI[d.MediaType]; !ok {
			return false
		} else if mtToOCI[d.MediaType] != mtToOCI[d2.MediaType] {
			return false
		}
	}
	return true
}

func (d Descriptor) MarshalPrettyTW(tw *tabwriter.Writer, prefix string) error {
	fmt.Fprintf(tw, "%sDigest:\t%s\n", prefix, string(d.Digest))
	fmt.Fprintf(tw, "%sMediaType:\t%s\n", prefix, d.MediaType)
	if d.ArtifactType != "" {
		fmt.Fprintf(tw, "%sArtifactType:\t%s\n", prefix, d.ArtifactType)
	}
	switch d.MediaType {
	case MediaTypeDocker1Manifest, MediaTypeDocker1ManifestSigned,
		MediaTypeDocker2Manifest, MediaTypeDocker2ManifestList,
		MediaTypeOCI1Manifest, MediaTypeOCI1ManifestList:
		// skip printing size for descriptors to manifests
	default:
		if d.Size > 100000 {
			fmt.Fprintf(tw, "%sSize:\t%s\n", prefix, units.HumanSize(float64(d.Size)))
		} else {
			fmt.Fprintf(tw, "%sSize:\t%dB\n", prefix, d.Size)
		}
	}
	if p := d.Platform; p != nil && p.OS != "" {
		fmt.Fprintf(tw, "%sPlatform:\t%s\n", prefix, p.String())
		if p.OSVersion != "" {
			fmt.Fprintf(tw, "%sOSVersion:\t%s\n", prefix, p.OSVersion)
		}
		if len(p.OSFeatures) > 0 {
			fmt.Fprintf(tw, "%sOSFeatures:\t%s\n", prefix, strings.Join(p.OSFeatures, ", "))
		}
	}
	if len(d.URLs) > 0 {
		fmt.Fprintf(tw, "%sURLs:\t%s\n", prefix, strings.Join(d.URLs, ", "))
	}
	if d.Annotations != nil {
		fmt.Fprintf(tw, "%sAnnotations:\t\n", prefix)
		for k, v := range d.Annotations {
			fmt.Fprintf(tw, "%s  %s:\t%s\n", prefix, k, v)
		}
	}
	return nil
}
