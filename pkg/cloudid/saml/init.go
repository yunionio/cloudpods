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

package saml

import (
	"os"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/samlutils"

	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/cloudid/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/samlutils/idp"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

var (
	saml        *samlutils.SSAMLInstance
	idpInstance *idp.SSAMLIdpInstance
)

func initSAMLInstance() error {
	certfile := options.Options.SslCertfile
	if len(options.Options.SslCaCerts) > 0 {
		var err error
		certfile, err = seclib2.MergeCaCertFiles(options.Options.SslCaCerts, options.Options.SslCertfile)
		if err != nil {
			return errors.Wrapf(httperrors.ErrInputParameter, "fail to merge ca+cert content: %s", err)
		}
		defer os.Remove(certfile)
	}
	if len(certfile) == 0 {
		return errors.Wrap(httperrors.ErrInputParameter, "Missing ssl-certfile")
	}
	if len(options.Options.SslKeyfile) == 0 {
		return errors.Wrap(httperrors.ErrInputParameter, "Missing ssl-keyfile")
	}

	var err error
	saml, err = samlutils.NewSAMLInstance(options.Options.ApiServer, certfile, options.Options.SslKeyfile)
	if err != nil {
		return errors.Wrap(err, "samlutils.NewSAMLInstance")
	}

	models.SamlIdpInstance = SAMLIdpInstance
	return nil
}

func SAMLInstance() *samlutils.SSAMLInstance {
	if saml.GetEntityId() != options.Options.ApiServer {
		saml.SetEntityId(options.Options.ApiServer)
	}
	return saml
}

func SAMLIdpInstance() *idp.SSAMLIdpInstance {
	return idpInstance
}

func IsSAMLEnabled() bool {
	return saml != nil
}

func InitSAML(app *appsrv.Application, prefix string) error {
	err := initSAMLInstance()
	if err != nil {
		return errors.Wrap(err, "initSAMLInstance")
	}
	err = initSAMLIdp(app, prefix)
	if err != nil {
		return errors.Wrap(err, "initSAMLIdp")
	}
	return nil
}
