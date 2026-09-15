package ocidir

import (
	"context"
	"fmt"
	"io/fs"
	"path"

	"github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/ref"
	"github.com/sirupsen/logrus"
)

// Close triggers a garbage collection if the underlying path has been modified
func (o *OCIDir) Close(ctx context.Context, r ref.Ref) error {
	if !o.gc {
		return nil
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	if _, ok := o.modRefs[r.Path]; !ok {
		// unmodified, no need to gc ref
		return nil
	}

	// perform GC
	o.log.WithFields(logrus.Fields{
		"ref": r.CommonName(),
	}).Debug("running GC")
	dl := map[string]bool{}
	// recurse through index, manifests, and blob lists, generating a digest list
	index, err := o.readIndex(r)
	if err != nil {
		return err
	}
	im, err := manifest.New(manifest.WithOrig(index))
	if err != nil {
		return err
	}
	err = o.closeProcManifest(ctx, r, im, &dl)
	if err != nil {
		return err
	}

	// go through filesystem digest list, removing entries not seen in recursive pass
	blobsPath := path.Join(r.Path, "blobs")
	blobDirs, err := fs.ReadDir(o.fs, blobsPath)
	if err != nil {
		return err
	}
	for _, blobDir := range blobDirs {
		if !blobDir.IsDir() {
			// should this warn or delete unexpected files in the blobs folder?
			continue
		}
		digestFiles, err := fs.ReadDir(o.fs, path.Join(blobsPath, blobDir.Name()))
		if err != nil {
			return err
		}
		for _, digestFile := range digestFiles {
			digest := fmt.Sprintf("%s:%s", blobDir.Name(), digestFile.Name())
			if !dl[digest] {
				o.log.WithFields(logrus.Fields{
					"digest": digest,
				}).Debug("ocidir garbage collect")
				// delete
				o.fs.Remove(path.Join(blobsPath, blobDir.Name(), digestFile.Name()))
			}
		}
	}
	delete(o.modRefs, r.Path)
	return nil
}

func (o *OCIDir) closeProcManifest(ctx context.Context, r ref.Ref, m manifest.Manifest, dl *map[string]bool) error {
	if mi, ok := m.(manifest.Indexer); ok {
		// go through manifest list, updating dl, and recursively processing nested manifests
		ml, err := mi.GetManifestList()
		if err != nil {
			return err
		}
		for _, cur := range ml {
			cr, _ := ref.New(r.CommonName())
			cr.Tag = ""
			cr.Digest = cur.Digest.String()
			(*dl)[cr.Digest] = true
			cm, err := o.ManifestGet(ctx, cr)
			if err != nil {
				// ignore errors in case a manifest has been deleted or sparse copy
				o.log.WithFields(logrus.Fields{
					"ref": cr.CommonName(),
					"err": err,
				}).Debug("could not retrieve manifest")
				continue
			}
			err = o.closeProcManifest(ctx, cr, cm, dl)
			if err != nil {
				return err
			}
		}
	}
	if mi, ok := m.(manifest.Imager); ok {
		// get config from manifest if it exists
		cd, err := mi.GetConfig()
		if err == nil {
			(*dl)[cd.Digest.String()] = true
		}
		// finally add all layers to digest list
		layers, err := mi.GetLayers()
		if err != nil {
			return err
		}
		for _, layer := range layers {
			(*dl)[layer.Digest.String()] = true
		}
	}
	return nil
}
