// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package volume_mount

import (
	"context"
	"path/filepath"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	imagemod "yunion.io/x/onecloud/pkg/mcclient/modules/image"
)

type povImage struct {
}

func newDiskPostOverlayImage() iDiskPostOverlay {
	return &povImage{}
}

func (p povImage) validateData(ctx context.Context, userCred mcclient.TokenCredential, pov *apis.ContainerVolumeMountDiskPostOverlay) error {
	img := pov.Image
	if img.Id == "" {
		return httperrors.NewMissingParameterError("image id")
	}
	s := auth.GetAdminSession(ctx, options.Options.Region)
	obj, err := imagemod.Images.Get(s, img.Id, nil)
	if err != nil {
		return errors.Wrapf(err, "Get image by id %s", img.Id)
	}
	imgObj := new(imageapi.ImageDetails)
	if err := obj.Unmarshal(imgObj); err != nil {
		return errors.Wrap(err, "unmarshal image details")
	}
	pov.Image.Id = imgObj.Id
	props := imgObj.Properties
	usedByStr, ok := props[imageapi.IMAGE_USED_BY_POST_OVERLAY]
	if !ok {
		return errors.Wrapf(err, "Get %s", imageapi.IMAGE_USED_BY_POST_OVERLAY)
	}
	if usedByStr != "true" {
		return errors.Errorf("image isn't used by post overlay")
	}
	pathMapStr := props[imageapi.IMAGE_INTERNAL_PATH_MAP]
	pathMapObj, err := jsonutils.ParseString(pathMapStr)
	if err != nil {
		return errors.Wrapf(err, "json parse path_map: %s", pathMapStr)
	}
	pathMap := make(map[string]string)
	if err := pathMapObj.Unmarshal(pathMap); err != nil {
		return errors.Wrapf(err, "unmarshal pathMapObj")
	}
	if len(pov.Image.PathMap) == 0 {
		pov.Image.PathMap = pathMap
	}
	if len(pov.Image.PathMap) > 0 {
		cleanedMap := make(map[string]string, len(pov.Image.PathMap))
		for k, v := range pov.Image.PathMap {
			clean, err := cleanPathMapKey(userCred, k)
			if err != nil {
				return err
			}
			cleanedMap[clean] = v
		}
		pov.Image.PathMap = cleanedMap
	}
	if len(pov.Image.HostLowerMap) != 0 {
		cleanedLower := make(map[string]*apis.HostLowerPath, len(pov.Image.HostLowerMap))
		for hostPath, hlp := range pov.Image.HostLowerMap {
			clean, err := cleanPathMapKey(userCred, hostPath)
			if err != nil {
				return err
			}
			_, ok := pov.Image.PathMap[clean]
			if !ok {
				return httperrors.NewNotFoundError("host_path %s of host_lower_map doesn't found in path_map", hostPath)
			}
			if hlp != nil {
				pre, err := validateColonSeparatedHostBindPaths(userCred, hlp.PrePath)
				if err != nil {
					return err
				}
				hlp.PrePath = pre
				post, err := validateColonSeparatedHostBindPaths(userCred, hlp.PostPath)
				if err != nil {
					return err
				}
				hlp.PostPath = post
			}
			cleanedLower[clean] = hlp
		}
		pov.Image.HostLowerMap = cleanedLower
	}
	if img.UpperConfig != nil && img.UpperConfig.Disk != nil && img.UpperConfig.Disk.SubPath != "" {
		clean, err := models.ValidateRelSubpath(img.UpperConfig.Disk.SubPath)
		if err != nil {
			return err
		}
		img.UpperConfig.Disk.SubPath = clean
	}
	return nil
}

func cleanPathMapKey(userCred mcclient.TokenCredential, k string) (string, error) {
	rel := strings.TrimPrefix(strings.TrimSpace(k), string(filepath.Separator))
	return models.ValidateRelSubpath(rel)
}

func validateColonSeparatedHostBindPaths(userCred mcclient.TokenCredential, pathStr string) (string, error) {
	if pathStr == "" {
		return "", nil
	}
	parts := strings.Split(pathStr, ":")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		clean, err := models.ValidateHostBindPath(userCred, p)
		if err != nil {
			return "", err
		}
		out = append(out, clean)
	}
	return strings.Join(out, ":"), nil
}

func (p povImage) getContainerTargetDirs(ov *apis.ContainerVolumeMountDiskPostOverlay) []string {
	pathMap := ov.Image.PathMap
	ctrPaths := []string{}
	for _, ctrPath := range pathMap {
		ctrPaths = append(ctrPaths, ctrPath)
	}
	return ctrPaths
}
