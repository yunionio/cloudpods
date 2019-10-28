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

package shell

import (
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {
	type IdentityProviderListOptions struct {
		options.BaseListOptions
	}
	R(&IdentityProviderListOptions{}, "idp-list", "List all identity provider", func(s *mcclient.ClientSession, args *IdentityProviderListOptions) error {
		params, err := options.ListStructToParams(args)
		if err != nil {
			return err
		}
		results, err := modules.IdentityProviders.List(s, params)
		if err != nil {
			return err
		}
		printList(results, modules.IdentityProviders.GetColumns(s))
		return nil
	})

	type IdentityProviderDetailOptions struct {
		ID string `help:"Id or name of identity provider to show"`
	}
	R(&IdentityProviderDetailOptions{}, "idp-show", "Show details of idp", func(s *mcclient.ClientSession, args *IdentityProviderDetailOptions) error {
		detail, err := modules.IdentityProviders.Get(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(detail)
		return nil
	})

	R(&IdentityProviderDetailOptions{}, "idp-config-show", "Show detail of a domain config", func(s *mcclient.ClientSession, args *IdentityProviderDetailOptions) error {
		conf, err := modules.IdentityProviders.GetSpecific(s, args.ID, "config", nil)
		if err != nil {
			return err
		}
		fmt.Println(conf.PrettyString())
		return nil
	})

	R(&IdentityProviderDetailOptions{}, "idp-enable", "Enable an identity provider", func(s *mcclient.ClientSession, args *IdentityProviderDetailOptions) error {
		idp, err := modules.IdentityProviders.PerformAction(s, args.ID, "enable", nil)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	R(&IdentityProviderDetailOptions{}, "idp-disable", "Disable an identity provider", func(s *mcclient.ClientSession, args *IdentityProviderDetailOptions) error {
		idp, err := modules.IdentityProviders.PerformAction(s, args.ID, "disable", nil)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	R(&IdentityProviderDetailOptions{}, "idp-delete", "Delete an identity provider", func(s *mcclient.ClientSession, args *IdentityProviderDetailOptions) error {
		idp, err := modules.IdentityProviders.Delete(s, args.ID, nil)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	R(&IdentityProviderDetailOptions{}, "idp-sync", "Sync an identity provider", func(s *mcclient.ClientSession, args *IdentityProviderDetailOptions) error {
		idp, err := modules.IdentityProviders.PerformAction(s, args.ID, "sync", nil)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	type IdentityProviderConfigLDAPOptions struct {
		ID string `help:"ID of idp to config" json:"-"`
		api.SLDAPIdpConfigOptions
	}
	R(&IdentityProviderConfigLDAPOptions{}, "idp-config-ldap", "Config an Identity provider with LDAP driver", func(s *mcclient.ClientSession, args *IdentityProviderConfigLDAPOptions) error {
		config := jsonutils.NewDict()
		config.Add(jsonutils.Marshal(args), "config", "ldap")
		nconf, err := modules.IdentityProviders.PerformAction(s, args.ID, "config", config)
		if err != nil {
			return err
		}
		fmt.Println(nconf.PrettyString())
		return nil
	})

	type IdentityProviderCreateLDAPOptions struct {
		NAME string `help:"name of identity provider" json:"-"`

		AutoCreateProject   bool `help:"automatically create a default project when importing domain" json:"-"`
		NoAutoCreateProject bool `help:"do not create default project when importing domain" json:"-"`

		TargetDomain string `help:"target domain without creating new domain" json:"-"`

		api.SLDAPIdpConfigOptions
	}
	R(&IdentityProviderCreateLDAPOptions{}, "idp-create-ldap", "Create an identity provider with LDAP driver", func(s *mcclient.ClientSession, args *IdentityProviderCreateLDAPOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")

		if len(args.TargetDomain) > 0 {
			params.Add(jsonutils.NewString(args.TargetDomain), "target_domain")
		}
		if args.AutoCreateProject {
			params.Add(jsonutils.JSONTrue, "auto_create_project")
		} else if args.NoAutoCreateProject {
			params.Add(jsonutils.JSONFalse, "auto_create_project")
		}

		params.Add(jsonutils.NewString("ldap"), "driver")
		params.Add(jsonutils.Marshal(args), "config", "ldap")

		idp, err := modules.IdentityProviders.Create(s, params)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	type IdentityProviderConfigLDAPSingleDomainOptions struct {
		ID string `help:"ID of idp to config" json:"-"`
		api.SLDAPIdpConfigSingleDomainOptions
	}
	R(&IdentityProviderConfigLDAPSingleDomainOptions{}, "idp-config-ldap-single-domain", "Config an Identity provider with LDAP driver/single domain template", func(s *mcclient.ClientSession, args *IdentityProviderConfigLDAPSingleDomainOptions) error {
		config := jsonutils.NewDict()
		config.Add(jsonutils.Marshal(args), "config", "ldap")
		nconf, err := modules.IdentityProviders.PerformAction(s, args.ID, "config", config)
		if err != nil {
			return err
		}
		fmt.Println(nconf.PrettyString())
		return nil
	})

	type IdentityProviderCreateLDAPSingleDomainOptions struct {
		NAME     string `help:"name of identity provider" json:"-"`
		TEMPLATE string `help:"configuration template name" choices:"msad_one_domain|openldap_one_domain" json:"-"`

		AutoCreateProject   bool `help:"automatically create a default project when importing domain" json:"-"`
		NoAutoCreateProject bool `help:"do not create default project when importing domain" json:"-"`

		TargetDomain string `help:"target domain without creating new domain" json:"-"`

		api.SLDAPIdpConfigSingleDomainOptions
	}
	R(&IdentityProviderCreateLDAPSingleDomainOptions{}, "idp-create-ldap-single-domain", "Create an identity provider with LDAP driver/single domain template", func(s *mcclient.ClientSession, args *IdentityProviderCreateLDAPSingleDomainOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString("ldap"), "driver")
		params.Add(jsonutils.NewString(args.TEMPLATE), "template")

		if len(args.TargetDomain) > 0 {
			params.Add(jsonutils.NewString(args.TargetDomain), "target_domain")
		}
		if args.AutoCreateProject {
			params.Add(jsonutils.JSONTrue, "auto_create_project")
		} else if args.NoAutoCreateProject {
			params.Add(jsonutils.JSONFalse, "auto_create_project")
		}

		params.Add(jsonutils.Marshal(args), "config", "ldap")

		idp, err := modules.IdentityProviders.Create(s, params)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	type IdentityProviderConfigLDAPMultiDomainOptions struct {
		ID string `help:"ID of idp to config" json:"-"`
		api.SLDAPIdpConfigMultiDomainOptions
	}
	R(&IdentityProviderConfigLDAPMultiDomainOptions{}, "idp-config-ldap-multi-domain", "Config an Identity provider with LDAP driver/multi domain template", func(s *mcclient.ClientSession, args *IdentityProviderConfigLDAPMultiDomainOptions) error {
		config := jsonutils.NewDict()
		config.Add(jsonutils.Marshal(args), "config", "ldap")
		nconf, err := modules.IdentityProviders.PerformAction(s, args.ID, "config", config)
		if err != nil {
			return err
		}
		fmt.Println(nconf.PrettyString())
		return nil
	})

	type IdentityProviderCreateLDAPMultiDomainOptions struct {
		NAME     string `help:"name of identity provider" json:"-"`
		TEMPLATE string `help:"configuration template name" choices:"msad_multi_domain" json:"-"`

		AutoCreateProject   bool `help:"automatically create a default project when importing domain" json:"-"`
		NoAutoCreateProject bool `help:"do not create default project when importing domain" json:"-"`

		api.SLDAPIdpConfigMultiDomainOptions
	}
	R(&IdentityProviderCreateLDAPMultiDomainOptions{}, "idp-create-ldap-multi-domain", "Create an identity provider with LDAP driver/single domain template", func(s *mcclient.ClientSession, args *IdentityProviderCreateLDAPMultiDomainOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString("ldap"), "driver")
		params.Add(jsonutils.NewString(args.TEMPLATE), "template")

		if args.AutoCreateProject {
			params.Add(jsonutils.JSONTrue, "auto_create_project")
		} else if args.NoAutoCreateProject {
			params.Add(jsonutils.JSONFalse, "auto_create_project")
		}

		params.Add(jsonutils.Marshal(args), "config", "ldap")

		idp, err := modules.IdentityProviders.Create(s, params)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	type IdentityProviderCreateCASOptions struct {
		NAME string `help:"name of identity provider" json:"-"`

		AutoCreateProject   bool `help:"automatically create a default project when importing domain" json:"-"`
		NoAutoCreateProject bool `help:"do not create default project when importing domain" json:"-"`

		TargetDomain string `help:"target domain without creating new domain" json:"-"`

		api.SCASIdpConfigOptions
	}
	R(&IdentityProviderCreateCASOptions{}, "idp-create-cas", "Create an identity provider with CAS driver", func(s *mcclient.ClientSession, args *IdentityProviderCreateCASOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")

		if len(args.TargetDomain) > 0 {
			params.Add(jsonutils.NewString(args.TargetDomain), "target_domain")
		}
		if args.AutoCreateProject {
			params.Add(jsonutils.JSONTrue, "auto_create_project")
		} else if args.NoAutoCreateProject {
			params.Add(jsonutils.JSONFalse, "auto_create_project")
		}

		params.Add(jsonutils.NewString("cas"), "driver")
		params.Add(jsonutils.Marshal(args), "config", "cas")

		idp, err := modules.IdentityProviders.Create(s, params)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	type IdentityProviderConfigCASOptions struct {
		ID string `help:"ID of idp to config" json:"-"`
		api.SCASIdpConfigOptions
	}
	R(&IdentityProviderConfigCASOptions{}, "idp-config-cas", "Config an Identity provider with CAS driver", func(s *mcclient.ClientSession, args *IdentityProviderConfigCASOptions) error {
		config := jsonutils.NewDict()
		config.Add(jsonutils.Marshal(args), "config", "cas")
		nconf, err := modules.IdentityProviders.PerformAction(s, args.ID, "config", config)
		if err != nil {
			return err
		}
		fmt.Println(nconf.PrettyString())
		return nil
	})

}
