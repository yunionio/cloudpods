package types

const (
	// MediaTypeDocker1Manifest deprecated media type for docker schema1 manifests
	MediaTypeDocker1Manifest = "application/vnd.docker.distribution.manifest.v1+json"
	// MediaTypeDocker1ManifestSigned is a deprecated schema1 manifest with jws signing
	MediaTypeDocker1ManifestSigned = "application/vnd.docker.distribution.manifest.v1+prettyjws"
	// MediaTypeDocker2Manifest is the media type when pulling manifests from a v2 registry
	MediaTypeDocker2Manifest = "application/vnd.docker.distribution.manifest.v2+json"
	// MediaTypeDocker2ManifestList is the media type when pulling a manifest list from a v2 registry
	MediaTypeDocker2ManifestList = "application/vnd.docker.distribution.manifest.list.v2+json"
	// MediaTypeDocker2ImageConfig is for the configuration json object media type
	MediaTypeDocker2ImageConfig = "application/vnd.docker.container.image.v1+json"
	// MediaTypeOCI1Artifact EXPERIMENTAL OCI v1 artifact media type
	MediaTypeOCI1Artifact = "application/vnd.oci.artifact.manifest.v1+json"
	// MediaTypeOCI1Manifest OCI v1 manifest media type
	MediaTypeOCI1Manifest = "application/vnd.oci.image.manifest.v1+json"
	// MediaTypeOCI1ManifestList OCI v1 manifest list media type
	MediaTypeOCI1ManifestList = "application/vnd.oci.image.index.v1+json"
	// MediaTypeOCI1ImageConfig OCI v1 configuration json object media type
	MediaTypeOCI1ImageConfig = "application/vnd.oci.image.config.v1+json"
	// MediaTypeDocker2LayerGzip is the default compressed layer for docker schema2
	MediaTypeDocker2LayerGzip = "application/vnd.docker.image.rootfs.diff.tar.gzip"
	// MediaTypeDocker2ForeignLayer is the default compressed layer for foreign layers in docker schema2
	MediaTypeDocker2ForeignLayer = "application/vnd.docker.image.rootfs.foreign.diff.tar.gzip"
	// MediaTypeOCI1Layer is the uncompressed layer for OCIv1
	MediaTypeOCI1Layer = "application/vnd.oci.image.layer.v1.tar"
	// MediaTypeOCI1LayerGzip is the gzip compressed layer for OCI v1
	MediaTypeOCI1LayerGzip = "application/vnd.oci.image.layer.v1.tar+gzip"
	// MediaTypeOCI1LayerZstd is the zstd compressed layer for OCI v1
	MediaTypeOCI1LayerZstd = "application/vnd.oci.image.layer.v1.tar+zstd"
	// MediaTypeOCI1ForeignLayer is the foreign layer for OCI v1
	MediaTypeOCI1ForeignLayer = "application/vnd.oci.image.layer.nondistributable.v1.tar"
	// MediaTypeOCI1ForeignLayerGzip is the gzip compressed foreign layer for OCI v1
	MediaTypeOCI1ForeignLayerGzip = "application/vnd.oci.image.layer.nondistributable.v1.tar+gzip"
	// MediaTypeOCI1ForeignLayerZstd is the zstd compressed foreign layer for OCI v1
	MediaTypeOCI1ForeignLayerZstd = "application/vnd.oci.image.layer.nondistributable.v1.tar+zstd"
	// MediaTypeBuildkitCacheConfig is used by buildkit cache images
	MediaTypeBuildkitCacheConfig = "application/vnd.buildkit.cacheconfig.v0"
)
