package v1

import (
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/oci"
)

// IndexSchemaVersion is a pre-configured versioned field for manifests
var IndexSchemaVersion = oci.Versioned{
	SchemaVersion: 2,
}

// Index references manifests for various platforms.
// This structure provides `application/vnd.oci.image.index.v1+json` mediatype when marshalled to JSON.
type Index struct {
	oci.Versioned

	// MediaType specifies the type of this document data structure e.g. `application/vnd.oci.image.index.v1+json`
	MediaType string `json:"mediaType,omitempty"`

	// Manifests references platform specific manifests.
	Manifests []types.Descriptor `json:"manifests"`

	// Annotations contains arbitrary metadata for the image index.
	Annotations map[string]string `json:"annotations,omitempty"`
}
