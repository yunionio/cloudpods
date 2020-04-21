package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	R(&options.ServiceCertificateCreateOptions{}, "service-cert-create", "Create service cert", func(s *mcclient.ClientSession, opts *options.ServiceCertificateCreateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		cert, err := modules.ServiceCertficatesV3.Create(s, params)
		if err != nil {
			return err
		}
		printObject(cert)
		return nil
	})
	type ServiceCertificateGetOptions struct {
		ID string `json:"-"`
	}
	R(&ServiceCertificateGetOptions{}, "service-cert-show", "Show service cert", func(s *mcclient.ClientSession, opts *ServiceCertificateGetOptions) error {
		cert, err := modules.ServiceCertficatesV3.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(cert)
		return nil
	})

	type ServiceCertificateListOptions struct {
		options.BaseListOptions
	}
	R(&ServiceCertificateListOptions{}, "service-cert-list", "List service certs", func(s *mcclient.ClientSession, opts *ServiceCertificateListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.ServiceCertficatesV3.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.ServiceCertficatesV3.GetColumns(s))
		return nil
	})
	type ServiceCertificateDeleteOptions struct {
		ID string `json:"-"`
	}
	R(&ServiceCertificateDeleteOptions{}, "service-cert-delete", "Delete service cert", func(s *mcclient.ClientSession, opts *ServiceCertificateDeleteOptions) error {
		cert, err := modules.ServiceCertficatesV3.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(cert)
		return nil
	})
}
