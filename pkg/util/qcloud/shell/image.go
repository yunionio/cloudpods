package shell

import (
	"fmt"

	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/util/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ImageListOptions struct {
		Status string `help:"Image status"`
		Owner  string `help:"Image owner" choices:"PRIVATE_IMAGE|PUBLIC_IMAGE|MARKET_IMAGE|SHARED_IMAGE"`
		Image  string `help:"Image Id"`
		Name   string `help:"Image Name"`
		Limit  int    `help:"page size"`
		Offset int    `help:"page offset"`
	}
	shellutils.R(&ImageListOptions{}, "image-list", "List images", func(cli *qcloud.SRegion, args *ImageListOptions) error {
		imageIds := []string{}
		if len(args.Image) > 0 {
			imageIds = append(imageIds, args.Image)
		}
		images, total, err := cli.GetImages(args.Status, args.Owner, imageIds, args.Name, args.Offset, args.Limit)
		if err != nil {
			return err
		}
		printList(images, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type ImageCreateOptions struct {
		NAME      string `helo:"Image name"`
		OSTYPE    string `helo:"Operation system" choices:"CentOS|Ubuntu|Debian|OpenSUSE|SUSE|CoreOS|FreeBSD|Other Linux|Windows Server 2008|Windows Server 2012|Windows Server 2016"`
		OSARCH    string `help:"OS Architecture" choices:"x86_64|i386"`
		osVersion string `help:"OS Version"`
		URL       string `helo:"Cos URL"`
	}

	shellutils.R(&ImageCreateOptions{}, "image-create", "Create image", func(cli *qcloud.SRegion, args *ImageCreateOptions) error {
		image, err := cli.ImportImage(args.NAME, args.OSARCH, args.OSTYPE, args.osVersion, args.URL)
		if err != nil {
			return err
		}
		printObject(image)
		return nil
	})

	type ImageDeleteOptions struct {
		ID string `helo:"Image ID"`
	}

	shellutils.R(&ImageDeleteOptions{}, "image-delete", "Delete image", func(cli *qcloud.SRegion, args *ImageDeleteOptions) error {
		return cli.DeleteImage(args.ID)
	})

	type ImageSupportSetOptions struct {
	}

	shellutils.R(&ImageSupportSetOptions{}, "image-support-set", "Show image support set", func(cli *qcloud.SRegion, args *ImageSupportSetOptions) error {
		imageSet, err := cli.GetSupportImageSet()
		if err != nil {
			return err
		}
		printObject(imageSet)
		return nil
	})

	type ImageParam struct {
		osArch          string
		osDist          string
		osVersion       string
		OutputArch      string
		OutputDist      string
		OutputOsVersion string
	}

	shellutils.R(&ImageSupportSetOptions{}, "image-import-test", "Test image import params", func(cli *qcloud.SRegion, args *ImageSupportSetOptions) error {
		imageParams := []ImageParam{
			{
				osArch:          "test",
				osDist:          "Centos",
				osVersion:       "3.4",
				OutputArch:      "x86_64",
				OutputDist:      "CentOS",
				OutputOsVersion: "-",
			},
			{
				osArch:          "i386",
				osDist:          "Centos",
				osVersion:       "6.9",
				OutputArch:      "i386",
				OutputDist:      "CentOS",
				OutputOsVersion: "6",
			},
			{
				osArch:          "i386",
				osDist:          "Centos",
				osVersion:       "7.1.1503",
				OutputArch:      "i386",
				OutputDist:      "CentOS",
				OutputOsVersion: "7",
			},
			{
				osArch:          "",
				osDist:          "",
				osVersion:       "7.1",
				OutputArch:      "x86_64",
				OutputDist:      "Other Linux",
				OutputOsVersion: "-",
			},
			{
				osArch:          "x86_64",
				osDist:          "Windows Server",
				osVersion:       "2008 R2 Datacenter Evaluation",
				OutputArch:      "x86_64",
				OutputDist:      "Windows Server 2008",
				OutputOsVersion: "-",
			},
			{
				osArch:          "x86_64",
				osDist:          "Ubuntu",
				osVersion:       "16.04.5",
				OutputArch:      "x86_64",
				OutputDist:      "Ubuntu",
				OutputOsVersion: "16",
			},
			{
				osArch:          "x86_64",
				osDist:          "Windows%20Server%202008%20R2%20Datacenter",
				osVersion:       "6.1",
				OutputArch:      "x86_64",
				OutputDist:      "Windows Server 2008",
				OutputOsVersion: "-",
			},
			{
				osArch:          "x86_64",
				osDist:          "Windows Server 2012 R2 Datacenter Evaluation",
				osVersion:       "6.2",
				OutputArch:      "x86_64",
				OutputDist:      "Windows Server 2012",
				OutputOsVersion: "-",
			},
		}
		for _, imageParam := range imageParams {
			log.Debugf("process %s", imageParam.osDist)
			params, err := cli.GetImportImageParams("", imageParam.osArch, imageParam.osDist, imageParam.osVersion, "")
			if err != nil {
				return err
			}
			if params["OsType"] != imageParam.OutputDist {
				return fmt.Errorf("params: %s osType should be %s not %s", imageParam, imageParam.OutputDist, params["OsType"])
			}
			if params["OsVersion"] != imageParam.OutputOsVersion {
				return fmt.Errorf("params: %s OsVersion should be %s not %s", imageParam, imageParam.OutputOsVersion, params["OsVersion"])
			}
			if params["Architecture"] != imageParam.OutputArch {
				return fmt.Errorf("params: %s Architecture should be %s not %s", imageParam, imageParam.OutputArch, params["Architecture"])
			}
		}
		return nil
	})

}
