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

package models

import (
	"context"
	"io"
	"os"
	"path"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/samlutils"

	"yunion.io/x/onecloud/pkg/cloudid/options"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/samlutils/idp"
)

type SamlInstance func() *idp.SSAMLIdpInstance

var (
	SamlIdpInstance SamlInstance = nil
)

type ICloudSAMLLoginDriver interface {
	GetEntityID() string

	GetMetadataFilename() string
	GetMetadataUrl() string

	GetIdpInitiatedLoginData(ctx context.Context, userCred mcclient.TokenCredential, cloudAccountId string, sp *idp.SSAMLServiceProvider, redirectUrl string) (samlutils.SSAMLIdpInitiatedLoginData, error)
	GetSpInitiatedLoginData(ctx context.Context, userCred mcclient.TokenCredential, cloudAccoutId string, sp *idp.SSAMLServiceProvider) (samlutils.SSAMLSpInitiatedLoginData, error)
}

var (
	driverTable = make(map[string]ICloudSAMLLoginDriver)
)

func Register(driver ICloudSAMLLoginDriver) {
	driverTable[driver.GetEntityID()] = driver
}

func UnRegister(entityId string) {
	delete(driverTable, entityId)
}

func FindDriver(entityId string) ICloudSAMLLoginDriver {
	if driver, ok := driverTable[entityId]; ok {
		return driver
	}
	return nil
}

func AllDrivers() map[string]ICloudSAMLLoginDriver {
	return driverTable
}

func GetMetadata(driver ICloudSAMLLoginDriver) ([]byte, error) {
	filePath := path.Join(options.Options.CloudSAMLMetadataPath, driver.GetMetadataFilename())
	metaBytes, err := os.ReadFile(filePath)
	if err != nil || len(metaBytes) == 0 {
		metaUrl := driver.GetMetadataUrl()
		if len(metaUrl) > 0 {
			log.Debugf("[%s] metadata file load failed, try download from %s", driver.GetEntityID(), metaUrl)
			httpcli := httputils.GetDefaultClient()
			resp, err := httpcli.Get(metaUrl)
			if err != nil {
				return nil, errors.Wrapf(err, "http get %s fail", metaUrl)
			}
			metaBytes, err = io.ReadAll(resp.Body)
			if err != nil {
				return nil, errors.Wrapf(err, "read body %s fail", metaUrl)
			}
			os.WriteFile(filePath, metaBytes, 0644)
		} else {
			return nil, errors.Wrapf(err, "read file %s fail", filePath)
		}
	}
	return metaBytes, nil
}
