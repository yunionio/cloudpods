package options

import (
	"fmt"
	"io/ioutil"

	"yunion.io/x/jsonutils"
)

type ServiceCertificateCreateOptions struct {
	NAME string

	Cert   string `required:"true" json:"-" help:"path to certificate file"`
	Pkey   string `required:"true" json:"-" help:"path to private key file"`
	CaCert string `help:"path to ca certificate file" json:"-"`
	CaPkey string `help:"paht to ca private key file" json:"-"`
}

func serviceCertificateLoadFiles(pathM map[string]string, allowEmpty bool) (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	for fieldName, path := range pathM {
		if path == "" {
			if allowEmpty {
				continue
			} else {
				return nil, fmt.Errorf("%s: empty path", fieldName)
			}
		}
		d, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s: read %s: %s", fieldName, path, err)
		}
		if len(d) == 0 {
			return nil, fmt.Errorf("%s: empty file %s", fieldName, path)
		}
		params.Set(fieldName, jsonutils.NewString(string(d)))
	}
	return params, nil
}

func (opts *ServiceCertificateCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := StructToParams(opts)
	if err != nil {
		return nil, err
	}
	pathM := map[string]string{
		"certificate": opts.Cert,
		"private_key": opts.Pkey,
	}
	paramsCertKey, err := serviceCertificateLoadFiles(pathM, false)
	if err != nil {
		return nil, err
	}
	params.Update(paramsCertKey)

	pathM = map[string]string{
		"ca_certificate": opts.CaCert,
		"ca_private_key": opts.CaPkey,
	}
	paramsCertKey, err = serviceCertificateLoadFiles(pathM, false)
	if err != nil {
		return nil, err
	}
	params.Update(paramsCertKey)
	return params, nil
}
