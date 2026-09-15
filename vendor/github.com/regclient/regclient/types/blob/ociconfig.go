package blob

import (
	"encoding/json"
	"fmt"

	// crypto libraries included for go-digest
	_ "crypto/sha256"
	_ "crypto/sha512"

	"github.com/opencontainers/go-digest"
	"github.com/regclient/regclient/types"
	v1 "github.com/regclient/regclient/types/oci/v1"
)

// OCIConfig wraps an OCI Config struct extracted from a Blob
type OCIConfig interface {
	Blob
	GetConfig() v1.Image
	SetConfig(v1.Image)
}

// ociConfig includes an OCI Config struct extracted from a Blob
// Image is included as an anonymous field to facilitate json and templating calls transparently
type ociConfig struct {
	common
	rawBody []byte
	v1.Image
}

// NewOCIConfig creates a new BlobOCIConfig from an OCI Image
func NewOCIConfig(opts ...Opts) OCIConfig {
	bc := blobConfig{}
	for _, opt := range opts {
		opt(&bc)
	}
	if bc.image != nil && len(bc.rawBody) == 0 {
		var err error
		bc.rawBody, err = json.Marshal(bc.image)
		if err != nil {
			bc.rawBody = []byte{}
		}
	}
	if len(bc.rawBody) > 0 {
		if bc.image == nil {
			bc.image = &v1.Image{}
			err := json.Unmarshal(bc.rawBody, bc.image)
			if err != nil {
				bc.image = nil
			}
		}
		// force descriptor to match raw body, even if we generated the raw body
		bc.desc.Digest = digest.FromBytes(bc.rawBody)
		bc.desc.Size = int64(len(bc.rawBody))
		if bc.desc.MediaType == "" {
			bc.desc.MediaType = types.MediaTypeOCI1ImageConfig
		}
	}

	c := common{
		desc:      bc.desc,
		r:         bc.r,
		rawHeader: bc.header,
		resp:      bc.resp,
	}
	b := ociConfig{
		common:  c,
		rawBody: bc.rawBody,
	}
	if bc.image != nil {
		b.Image = *bc.image
		b.blobSet = true
	}
	return &b
}

// GetConfig returns OCI config
func (b *ociConfig) GetConfig() v1.Image {
	return b.Image
}

// RawBody returns the original body from the request
func (b *ociConfig) RawBody() ([]byte, error) {
	var err error
	if !b.blobSet {
		return []byte{}, fmt.Errorf("Blob is not defined")
	}
	if len(b.rawBody) == 0 {
		b.rawBody, err = json.Marshal(b.Image)
	}
	return b.rawBody, err
}

// SetConfig updates the config, including raw body and descriptor
func (b *ociConfig) SetConfig(c v1.Image) {
	b.Image = c
	b.rawBody, _ = json.Marshal(b.Image)
	if b.desc.MediaType == "" {
		b.desc.MediaType = types.MediaTypeOCI1ImageConfig
	}
	b.desc.Digest = digest.FromBytes(b.rawBody)
	b.desc.Size = int64(len(b.rawBody))
	b.blobSet = true
}
