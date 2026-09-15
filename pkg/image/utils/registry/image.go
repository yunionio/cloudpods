package registry

import (
	"archive/tar"
	"context"
	"encoding/json"
	// "errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	digest "github.com/opencontainers/go-digest"
	"github.com/regclient/regclient/types"
	"github.com/regclient/regclient/types/docker/schema2"
	"github.com/regclient/regclient/types/manifest"
	v1 "github.com/regclient/regclient/types/oci/v1"
	"yunion.io/x/pkg/errors"
)

// reference from: https://github.com/regclient/regclient/blob/v0.4.8/image.go
func AnalysisTar(tarPath string) (*TarReadData, error) {
	rs, err := os.Open(tarPath)
	if err != nil {
		return nil, errors.Wrapf(err, "open tar file %q", tarPath)
	}
	defer rs.Close()

	trd, err := analysisTar(rs)
	if err != nil {
		return nil, errors.Wrapf(err, "analysis %q", tarPath)
	}
	return trd, nil
}

// used by import/export to match docker tar expected format
type DockerTarManifest struct {
	Config       string
	RepoTags     []string
	Layers       []string
	Parent       digest.Digest                      `json:",omitempty"`
	LayerSources map[digest.Digest]types.Descriptor `json:",omitempty"`
}

type tarFileHandler func(header *tar.Header, trd *TarReadData) error
type TarReadData struct {
	tr          *tar.Reader
	handleAdded bool
	handlers    map[string]tarFileHandler
	links       map[string][]string
	processed   map[string]bool
	finish      []func() error
	// data processed from various handlers
	manifests           map[digest.Digest]manifest.Manifest
	ociIndex            v1.Index
	ociManifest         manifest.Manifest
	dockerManifestFound bool
	dockerManifestList  []DockerTarManifest
	dockerManifest      schema2.Manifest
}

func (trd *TarReadData) GetDockerManifest() schema2.Manifest {
	return trd.dockerManifest
}

func (trd *TarReadData) GetDockerManifestList() []DockerTarManifest {
	return trd.dockerManifestList
}

// tarReadAll processes the tar file in a loop looking for matching filenames in the list of handlers
// handlers for filenames are added at the top level, and by manifest imports
func (trd *TarReadData) tarReadAll(rs io.ReadSeeker) error {
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
			// track symlinks
			if header.Typeflag == tar.TypeSymlink || header.Typeflag == tar.TypeLink {
				// normalize target relative to root of tar
				target := header.Linkname
				if !filepath.IsAbs(target) {
					target, err = filepath.Rel(filepath.Dir(name), target)
					if err != nil {
						return err
					}
				}
				target = filepath.Clean("/" + target)[1:]
				// track and set handleAdded if an existing handler points to the target
				if trd.linkAdd(name, target) && !trd.handleAdded {
					list, err := trd.linkList(target)
					if err != nil {
						return err
					}
					for _, src := range append(list, name) {
						if trd.handlers[src] != nil {
							trd.handleAdded = true
						}
					}
				}
			} else {
				// loop through filename and symlinks to file in search of handlers
				list, err := trd.linkList(name)
				if err != nil {
					return err
				}
				list = append(list, name)
				trdUsed := false
				for _, entry := range list {
					if trd.handlers[entry] != nil {
						// trd cannot be reused, force the loop to run again
						if trdUsed {
							trd.handleAdded = true
							break
						}
						trdUsed = true
						// run handler
						err = trd.handlers[entry](header, trd)
						if err != nil {
							return err
						}
						delete(trd.handlers, entry)
						trd.processed[entry] = true
						// return if last handler processed
						if len(trd.handlers) == 0 {
							return nil
						}
					}
				}
			}
		}
		// if entire file read without adding a new handler, fail
		if !trd.handleAdded {
			return fmt.Errorf("unable to read all files from tar: %w", types.ErrNotFound)
		}
	}
}

func (trd *TarReadData) linkAdd(src, tgt string) bool {
	for _, entry := range trd.links[tgt] {
		if entry == src {
			return false
		}
	}
	trd.links[tgt] = append(trd.links[tgt], src)
	return true
}

func (trd *TarReadData) linkList(tgt string) ([]string, error) {
	list := trd.links[tgt]
	for _, entry := range list {
		if entry == tgt {
			return nil, fmt.Errorf("symlink loop encountered for %s", tgt)
		}
		list = append(list, trd.links[entry]...)
	}
	return list, nil
}

// tarReadFileJSON reads the current tar entry and unmarshals json into provided interface
func (trd *TarReadData) tarReadFileJSON(data interface{}) error {
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

func analysisTar(rs io.ReadSeeker) (*TarReadData, error) {
	trd := &TarReadData{
		handlers:  map[string]tarFileHandler{},
		links:     map[string][]string{},
		processed: map[string]bool{},
		finish:    []func() error{},
		manifests: map[digest.Digest]manifest.Manifest{},
	}

	// add handler for manifest.json
	imageImportDockerAddHandler(trd)

	// process tar file looking for oci-layout and index.json, load manifests/blobs on success
	err := trd.tarReadAll(rs)
	if err != nil {
		return nil, errors.Wrapf(err, "read tar data")
	}
	log.Printf("===trd read: %v, %v", err, trd.dockerManifestFound)
	ctx := context.Background()
	if trd.dockerManifestFound {
		// import failed but manifest.json found, fall back to manifest.json processing
		// add handlers for the docker manifest layers
		imageImportDockerAddLayerHandlers(ctx, trd)
		// reprocess the tar looking for manifest.json files
		err = trd.tarReadAll(rs)
		if err != nil {
			return nil, fmt.Errorf("failed to import layers from docker tar: %w", err)
		}
		return trd, nil
	}
	return nil, fmt.Errorf("not found docker manifest from tar")
}

const (
	dockerManifestFilename = "manifest.json"
	ociLayoutVersion       = "1.0.0"
	ociIndexFilename       = "index.json"
	ociLayoutFilename      = "oci-layout"
	annotationRefName      = "org.opencontainers.image.ref.name"
	annotationImageName    = "io.containerd.image.name"
)

func imageImportDockerAddHandler(trd *TarReadData) {
	trd.handlers[dockerManifestFilename] = func(header *tar.Header, trd *TarReadData) error {
		err := trd.tarReadFileJSON(&trd.dockerManifestList)
		if err != nil {
			return err
		}
		trd.dockerManifestFound = true
		return nil
	}
}

// imageImportDockerAddLayerHandlers imports the docker layers when OCI import fails and docker manifest found
func imageImportDockerAddLayerHandlers(ctx context.Context, trd *TarReadData) {
	// remove handlers for OCI
	delete(trd.handlers, ociLayoutFilename)
	delete(trd.handlers, ociIndexFilename)

	// make a docker v2 manifest from first json array entry (can only tag one image)
	trd.dockerManifest.SchemaVersion = 2
	trd.dockerManifest.MediaType = types.MediaTypeDocker2Manifest
	trd.dockerManifest.Layers = make([]types.Descriptor, len(trd.dockerManifestList[0].Layers))
	content, _ := json.MarshalIndent(trd.dockerManifestList[0], "", "  ")
	log.Printf("%d content: %s", len(trd.dockerManifestList), content)
	trd.handleAdded = true
}
