package options

import (
	"fmt"
	"io/ioutil"

	"yunion.io/x/jsonutils"
)

func loadbalancerCertificateLoadFiles(cert, pkey string, allowEmpty bool) (*jsonutils.JSONDict, error) {
	params := jsonutils.NewDict()
	pathM := map[string]string{
		"certificate": cert,
		"private_key": pkey,
	}
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

type LoadbalancerCertificateCreateOptions struct {
	NAME string

	Cert      string `required:"true" json:"-" help:"path to certificate file"`
	Pkey      string `required:"true" json:"-" help:"path to private key file"`
	Region    string `json:"cloudregion"`
	ManagerId string
}

func (opts *LoadbalancerCertificateCreateOptions) Params() (*jsonutils.JSONDict, error) {
	params, err := StructToParams(opts)
	if err != nil {
		return nil, err
	}
	paramsCertKey, err := loadbalancerCertificateLoadFiles(opts.Cert, opts.Pkey, false)
	if err != nil {
		return nil, err
	}
	params.Update(paramsCertKey)
	return params, nil
}

type LoadbalancerCertificateGetOptions struct {
	ID string
}

type LoadbalancerCertificateDeleteOptions struct {
	ID string
}

type LoadbalancerCertificateListOptions struct {
	BaseListOptions

	PublicKeyAlgorithm string
	PublicKeyBitLen    *int
	SignatureAlgorithm string
}

type LoadbalancerCertificateUpdateOptions struct {
	ID   string
	Name string

	Cert string `json:"-" help:"path to certificate file"`
	Pkey string `json:"-" help:"path to private key file"`
}

func (opts *LoadbalancerCertificateUpdateOptions) Params() (*jsonutils.JSONDict, error) {
	paramsCertKey, err := loadbalancerCertificateLoadFiles(opts.Cert, opts.Pkey, true)
	if err != nil {
		return nil, err
	}
	return paramsCertKey, nil
}
