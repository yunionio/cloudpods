package volume_mount

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis"
	imageapi "yunion.io/x/onecloud/pkg/apis/image"
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
	return nil
}

func (p povImage) getContainerTargetDirs(ov *apis.ContainerVolumeMountDiskPostOverlay) []string {
	pathMap := ov.Image.PathMap
	ctrPaths := []string{}
	for _, ctrPath := range pathMap {
		ctrPaths = append(ctrPaths, ctrPath)
	}
	return ctrPaths
}
