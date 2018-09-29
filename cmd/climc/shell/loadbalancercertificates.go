package shell

import (
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	R(&options.LoadbalancerCertificateCreateOptions{}, "lbcert-create", "Create lbcert", func(s *mcclient.ClientSession, opts *options.LoadbalancerCertificateCreateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		lbcert, err := modules.LoadbalancerCertificates.Create(s, params)
		if err != nil {
			return err
		}
		printObject(lbcert)
		return nil
	})
	R(&options.LoadbalancerCertificateGetOptions{}, "lbcert-show", "Show lbcert", func(s *mcclient.ClientSession, opts *options.LoadbalancerCertificateGetOptions) error {
		lbcert, err := modules.LoadbalancerCertificates.Get(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lbcert)
		return nil
	})
	R(&options.LoadbalancerCertificateListOptions{}, "lbcert-list", "List lbcerts", func(s *mcclient.ClientSession, opts *options.LoadbalancerCertificateListOptions) error {
		params, err := options.ListStructToParams(opts)
		if err != nil {
			return err
		}
		result, err := modules.LoadbalancerCertificates.List(s, params)
		if err != nil {
			return err
		}
		printList(result, modules.LoadbalancerCertificates.GetColumns(s))
		return nil
	})
	R(&options.LoadbalancerCertificateUpdateOptions{}, "lbcert-update", "Update lbcert", func(s *mcclient.ClientSession, opts *options.LoadbalancerCertificateUpdateOptions) error {
		params, err := opts.Params()
		if err != nil {
			return err
		}
		lbcert, err := modules.LoadbalancerCertificates.Update(s, opts.ID, params)
		if err != nil {
			return err
		}
		printObject(lbcert)
		return nil
	})
	R(&options.LoadbalancerCertificateDeleteOptions{}, "lbcert-delete", "Show lbcert", func(s *mcclient.ClientSession, opts *options.LoadbalancerCertificateDeleteOptions) error {
		lbcert, err := modules.LoadbalancerCertificates.Delete(s, opts.ID, nil)
		if err != nil {
			return err
		}
		printObject(lbcert)
		return nil
	})
}
