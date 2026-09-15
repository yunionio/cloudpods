package v1

import (
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/oci"
)

// ManifestSchemaVersion is a pre-configured versioned field for manifests
var ManifestSchemaVersion = oci.Versioned{
	SchemaVersion: 2,
}

// Manifest defines an OCI image
type Manifest struct {
	oci.Versioned

	// MediaType specifies the type of this document data structure e.g. `application/vnd.oci.image.manifest.v1+json`
	MediaType string `json:"mediaType,omitempty"`

	// Config references a configuration object for a container, by digest.
	// The referenced configuration object is a JSON blob that the runtime uses to set up the container.
	Config types.Descriptor `json:"config"`

	// Layers is an indexed list of layers referenced by the manifest.
	Layers []types.Descriptor `json:"layers"`

	// Annotations contains arbitrary metadata for the image manifest.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Refers indicates this manifest references another manifest
	// TODO: deprecated, delete this from future releases
	Refers *types.Descriptor `json:"refers,omitempty"`

	// Subject is an optional link from the image manifest to another manifest forming an association between the image manifest and the other manifest.
	Subject *types.Descriptor `json:"subject,omitempty"`
}
