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

package identity

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type IdentityProviderListOptions struct {
		options.BaseListOptions
		SsoDomain string `help:"Filter SSO IDP by domain" json:"sso_domain"`
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

	type IdentityProviderUpdateOptions struct {
		ID string `help:"Id or name of identity provider to update" json:"-"`
		api.IdentityProviderUpdateInput
	}
	R(&IdentityProviderUpdateOptions{}, "idp-update", "Update a identity provider", func(s *mcclient.ClientSession, args *IdentityProviderUpdateOptions) error {
		resp, err := modules.IdentityProviders.Update(s, args.ID, jsonutils.Marshal(args))
		if err != nil {
			return err
		}
		printObject(resp)
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

		AutoCreateProject   bool `help:"automatically create a project if the default_project not exists" json:"-"`
		NoAutoCreateProject bool `help:"do not create default project when importing domain" json:"-"`

		AutoCreateUser   bool `help:"automatically create a user" json:"-"`
		NoAutoCreateUser bool `help:"do not automatically create a user" json:"-"`

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
		if args.AutoCreateUser {
			params.Add(jsonutils.JSONTrue, "auto_create_user")
		} else if args.NoAutoCreateUser {
			params.Add(jsonutils.JSONFalse, "auto_create_user")
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

	type IdentityProviderConfigEditOptions struct {
		IDP string `help:"identity provider name or ID"`
	}
	R(&IdentityProviderConfigEditOptions{}, "idp-config-edit", "Edit config yaml of an identity provider", func(s *mcclient.ClientSession, args *IdentityProviderConfigEditOptions) error {
		idp, err := modules.IdentityProviders.Get(s, args.IDP, nil)
		if err != nil {
			return err
		}
		enabled, _ := idp.GetString("enabled")
		if enabled != "false" {
			return errors.Wrap(httperrors.ErrInvalidStatus, "idp must be disabled")
		}
		params := jsonutils.NewDict()
		params.Add(jsonutils.JSONTrue, "sensitive")
		conf, err := modules.IdentityProviders.GetSpecific(s, args.IDP, "config", params)
		if err != nil {
			return err
		}
		confJson, err := conf.Get("config")
		if err != nil {
			return err
		}
		content, err := shellutils.Edit(confJson.YAMLString())
		if err != nil {
			return err
		}
		yamlJson, err := jsonutils.ParseYAML(content)
		if err != nil {
			return err
		}
		config := jsonutils.NewDict()
		config.Add(yamlJson, "config")
		nconf, err := modules.IdentityProviders.PerformAction(s, args.IDP, "config", config)
		if err != nil {
			return err
		}
		fmt.Println(nconf.PrettyString())
		return nil
	})

	type IdentityProviderCreateSAMLOptions struct {
		NAME string `help:"name of identity provider" json:"-"`

		AutoCreateProject   bool `help:"automatically create a default project when importing domain" json:"-"`
		NoAutoCreateProject bool `help:"do not create default project when importing domain" json:"-"`

		TargetDomain string `help:"target domain without creating new domain" json:"-"`

		api.SSAMLIdpConfigOptions
	}
	R(&IdentityProviderCreateSAMLOptions{}, "idp-create-saml", "Create an identity provider with SAML driver", func(s *mcclient.ClientSession, args *IdentityProviderCreateSAMLOptions) error {
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

		params.Add(jsonutils.NewString("saml"), "driver")
		params.Add(jsonutils.Marshal(args), "config", "saml")

		idp, err := modules.IdentityProviders.Create(s, params)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	type IdentityProviderConfigSAMLOptions struct {
		ID string `help:"ID of idp to config" json:"-"`
		api.SSAMLIdpConfigOptions
	}
	R(&IdentityProviderConfigSAMLOptions{}, "idp-config-saml", "Config an Identity provider with SAML driver", func(s *mcclient.ClientSession, args *IdentityProviderConfigSAMLOptions) error {
		config := jsonutils.NewDict()
		config.Add(jsonutils.Marshal(args), "config", "saml")
		nconf, err := modules.IdentityProviders.PerformAction(s, args.ID, "config", config)
		if err != nil {
			return err
		}
		fmt.Println(nconf.PrettyString())
		return nil
	})

	type IdentityProviderGetSAMLMetadataOptions struct {
		ID string `help:"ID of idp to config" json:"-"`
		api.GetIdpSamlMetadataInput
	}
	R(&IdentityProviderGetSAMLMetadataOptions{}, "idp-saml-metadata", "Get SAML service provider metadata", func(s *mcclient.ClientSession, args *IdentityProviderGetSAMLMetadataOptions) error {
		nconf, err := modules.IdentityProviders.GetSpecific(s, args.ID, "saml-metadata", jsonutils.Marshal(args))
		if err != nil {
			return err
		}
		spMeta, _ := nconf.GetString("metadata")
		fmt.Println(spMeta)
		return nil
	})

	type IdentityProviderCreateSAMLTestOptions struct {
		NAME string `help:"name of identity provider" json:"-"`
		// TEMPLATE string `help:"configuration template name" choices:"msad_multi_domain" json:"-"`

		api.SSAMLTestIdpConfigOptions
	}
	R(&IdentityProviderCreateSAMLTestOptions{}, "idp-create-saml-test", "Create an identity provider with samltest.id (test only)", func(s *mcclient.ClientSession, args *IdentityProviderCreateSAMLTestOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString("saml"), "driver")
		params.Add(jsonutils.NewString(api.IdpTemplateSAMLTest), "template")

		params.Add(jsonutils.Marshal(args), "config", "saml")

		idp, err := modules.IdentityProviders.Create(s, params)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	type IdentityProviderCreateAzureADSAMLOptions struct {
		NAME string `help:"name of identity provider" json:"-"`
		// TEMPLATE string `help:"configuration template name" choices:"msad_multi_domain" json:"-"`

		api.SSAMLAzureADConfigOptions
	}
	R(&IdentityProviderCreateAzureADSAMLOptions{}, "idp-create-azure-ad-saml", "Create an identity provider with Azure AD SAML", func(s *mcclient.ClientSession, args *IdentityProviderCreateAzureADSAMLOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString("saml"), "driver")
		params.Add(jsonutils.NewString(api.IdpTemplateAzureADSAML), "template")

		params.Add(jsonutils.Marshal(args), "config", "saml")

		idp, err := modules.IdentityProviders.Create(s, params)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	type IdentityProviderCreateOIDCOptions struct {
		NAME string `help:"name of identity provider" json:"-"`

		AutoCreateProject   bool `help:"automatically create a default project when importing domain" json:"-"`
		NoAutoCreateProject bool `help:"do not create default project when importing domain" json:"-"`

		AutoCreateUser   bool `help:"automatically create a user" json:"-"`
		NoAutoCreateUser bool `help:"do not automatically create a user" json:"-"`

		TargetDomain string `help:"target domain without creating new domain" json:"-"`

		api.SOIDCIdpConfigOptions
	}
	R(&IdentityProviderCreateOIDCOptions{}, "idp-create-oidc", "Create an identity provider with OpenID connect driver", func(s *mcclient.ClientSession, args *IdentityProviderCreateOIDCOptions) error {
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
		if args.AutoCreateUser {
			params.Add(jsonutils.JSONTrue, "auto_create_user")
		} else if args.NoAutoCreateUser {
			params.Add(jsonutils.JSONFalse, "auto_create_user")
		}

		params.Add(jsonutils.NewString("oidc"), "driver")
		params.Add(jsonutils.Marshal(args), "config", "oidc")

		idp, err := modules.IdentityProviders.Create(s, params)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	type IdentityProviderConfigOIDCOptions struct {
		ID string `help:"ID of idp to config" json:"-"`
		api.SOIDCIdpConfigOptions
	}
	R(&IdentityProviderConfigOIDCOptions{}, "idp-config-oidc", "Config an Identity provider with OpenID connect driver", func(s *mcclient.ClientSession, args *IdentityProviderConfigOIDCOptions) error {
		config := jsonutils.NewDict()
		config.Add(jsonutils.Marshal(args), "config", "oidc")
		nconf, err := modules.IdentityProviders.PerformAction(s, args.ID, "config", config)
		if err != nil {
			return err
		}
		fmt.Println(nconf.PrettyString())
		return nil
	})

	type IdentityProviderCreateDexOIDCOptions struct {
		NAME string `help:"name of identity provider" json:"-"`

		api.SOIDCDexConfigOptions
	}
	R(&IdentityProviderCreateDexOIDCOptions{}, "idp-create-dex-oidc", "Create an identity provider with DEX OpenID Connect", func(s *mcclient.ClientSession, args *IdentityProviderCreateDexOIDCOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString("oidc"), "driver")
		params.Add(jsonutils.NewString(api.IdpTemplateDex), "template")

		params.Add(jsonutils.Marshal(args), "config", "oidc")

		idp, err := modules.IdentityProviders.Create(s, params)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	type IdentityProviderCreateCommonOptions struct {
		TargetDomainId    string `help:"target domain id"`
		AutoCreateProject bool   `help:"create project if no project presents" negative:"no-auto-create-project"`
		AutoCreateUser    bool   `help:"create user if no user presents" negative:"no-auto-create-user"`
	}

	type IdentityProviderCreateGithubOIDCOptions struct {
		NAME string `help:"name of identity provider" json:"name"`

		api.SOIDCGithubConfigOptions

		IdentityProviderCreateCommonOptions
	}
	R(&IdentityProviderCreateGithubOIDCOptions{}, "idp-create-github-oidc", "Create an identity provider with Github OpenID Connect", func(s *mcclient.ClientSession, args *IdentityProviderCreateGithubOIDCOptions) error {
		params := jsonutils.NewDict()
		// params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString("oidc"), "driver")
		params.Add(jsonutils.NewString(api.IdpTemplateGithub), "template")

		params.Update(jsonutils.Marshal(args))

		params.Add(jsonutils.Marshal(args.SOIDCGithubConfigOptions), "config", "oidc")

		idp, err := modules.IdentityProviders.Create(s, params)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	type IdentityProviderCreateGoogleOIDCOptions struct {
		NAME string `help:"name of identity provider" json:"name"`

		api.SOIDCGoogleConfigOptions

		IdentityProviderCreateCommonOptions
	}
	R(&IdentityProviderCreateGoogleOIDCOptions{}, "idp-create-google-oidc", "Create an identity provider with Google OpenID Connect", func(s *mcclient.ClientSession, args *IdentityProviderCreateGoogleOIDCOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString("oidc"), "driver")
		params.Add(jsonutils.NewString(api.IdpTemplateGoogle), "template")

		params.Update(jsonutils.Marshal(args))

		params.Add(jsonutils.Marshal(args.SOIDCGoogleConfigOptions), "config", "oidc")

		idp, err := modules.IdentityProviders.Create(s, params)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	type IdentityProviderCreateAzureOIDCOptions struct {
		NAME string `help:"name of identity provider" json:"name"`

		api.SOIDCAzureConfigOptions

		IdentityProviderCreateCommonOptions
	}
	R(&IdentityProviderCreateAzureOIDCOptions{}, "idp-create-azure-oidc", "Create an identity provider with Azure AD OpenID Connect", func(s *mcclient.ClientSession, args *IdentityProviderCreateAzureOIDCOptions) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString("oidc"), "driver")
		params.Add(jsonutils.NewString(api.IdpTemplateAzureOAuth2), "template")

		params.Update(jsonutils.Marshal(args))

		params.Add(jsonutils.Marshal(args.SOIDCAzureConfigOptions), "config", "oidc")

		idp, err := modules.IdentityProviders.Create(s, params)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	type IdentityProviderCreateAlipayOAuth2Options struct {
		NAME    string `help:"name of identity provider"`
		APPID   string `help:"Alipay app_id"`
		KEYFILE string `json:"Alipay app private key file"`
	}
	R(&IdentityProviderCreateAlipayOAuth2Options{}, "idp-create-alipay-oauth2", "Create an identity provider with Alipay OAuth2.0", func(s *mcclient.ClientSession, args *IdentityProviderCreateAlipayOAuth2Options) error {
		opts := api.SOAuth2IdpConfigOptions{}
		opts.AppId = args.APPID
		var err error
		opts.Secret, err = fileutils2.FileGetContents(args.KEYFILE)
		if err != nil {
			return err
		}
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString("oauth2"), "driver")
		params.Add(jsonutils.NewString(api.IdpTemplateAlipay), "template")
		params.Add(jsonutils.Marshal(opts), "config", "oauth2")
		idp, err := modules.IdentityProviders.Create(s, params)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	type IdentityProviderCreateFeishuOAuth2Options struct {
		NAME string `help:"name of identity provider" json:"name"`

		api.SOAuth2IdpConfigOptions

		IdentityProviderCreateCommonOptions
	}
	R(&IdentityProviderCreateFeishuOAuth2Options{}, "idp-create-feishu-oauth2", "Create an identity provider with Feishu OAuth2.0", func(s *mcclient.ClientSession, args *IdentityProviderCreateFeishuOAuth2Options) error {
		params := jsonutils.NewDict()
		// params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString("oauth2"), "driver")
		params.Add(jsonutils.NewString(api.IdpTemplateFeishu), "template")
		params.Update(jsonutils.Marshal(args))
		params.Add(jsonutils.Marshal(args.SOAuth2IdpConfigOptions), "config", "oauth2")
		idp, err := modules.IdentityProviders.Create(s, params)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	type IdentityProviderCreateDingtalkOAuth2Options struct {
		NAME string `help:"name of identity provider"`

		api.SOAuth2IdpConfigOptions
	}
	R(&IdentityProviderCreateDingtalkOAuth2Options{}, "idp-create-dingtalk-oauth2", "Create an identity provider with Feishu OAuth2.0", func(s *mcclient.ClientSession, args *IdentityProviderCreateDingtalkOAuth2Options) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString("oauth2"), "driver")
		params.Add(jsonutils.NewString(api.IdpTemplateDingtalk), "template")
		params.Add(jsonutils.Marshal(args), "config", "oauth2")
		idp, err := modules.IdentityProviders.Create(s, params)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	type IdentityProviderCreateWechatOAuth2Options struct {
		NAME string `help:"name of identity provider"`

		api.SOAuth2IdpConfigOptions
	}
	R(&IdentityProviderCreateWechatOAuth2Options{}, "idp-create-wechat-oauth2", "Create an identity provider with Wechat OAuth2.0", func(s *mcclient.ClientSession, args *IdentityProviderCreateWechatOAuth2Options) error {
		params := jsonutils.NewDict()
		params.Add(jsonutils.NewString(args.NAME), "name")
		params.Add(jsonutils.NewString("oauth2"), "driver")
		params.Add(jsonutils.NewString(api.IdpTemplateWechat), "template")
		params.Add(jsonutils.Marshal(args), "config", "oauth2")
		idp, err := modules.IdentityProviders.Create(s, params)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	type IdentityProviderCreateQywechatOAuth2Options struct {
		api.IdentityProviderCreateInput
		CorpId  string `help:"corp id of qywechat"`
		AgentId string `help:"agent id of app"`
		Secret  string `help:"secret of qywechat"`
	}
	R(&IdentityProviderCreateQywechatOAuth2Options{}, "idp-create-qywechat-oauth2", "Create an identity provider with Qiye Wechat OAuth2.0", func(s *mcclient.ClientSession, args *IdentityProviderCreateQywechatOAuth2Options) error {
		conf := api.SOAuth2IdpConfigOptions{
			AppId:  fmt.Sprintf("%s/%s", args.CorpId, args.AgentId),
			Secret: args.Secret,
		}
		params := jsonutils.Marshal(args).(*jsonutils.JSONDict)
		params.Add(jsonutils.NewString("oauth2"), "driver")
		params.Add(jsonutils.NewString(api.IdpTemplateQywechat), "template")
		params.Add(jsonutils.Marshal(conf), "config", "oauth2")
		idp, err := modules.IdentityProviders.Create(s, params)
		if err != nil {
			return err
		}
		printObject(idp)
		return nil
	})

	type IdpGetRedirectUriOptions struct {
		ID string `help:"id or name of idp to query" json:"-"`

		api.GetIdpSsoRedirectUriInput
	}
	R(&IdpGetRedirectUriOptions{}, "idp-sso-url", "Get sso url of a SSO idp", func(s *mcclient.ClientSession, args *IdpGetRedirectUriOptions) error {
		result, err := modules.IdentityProviders.GetSpecific(s, args.ID, "sso-redirect-uri", jsonutils.Marshal(args))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type IdpGetCallbackUriOptions struct {
		ID string `help:"id or name of idp to query" json:"-"`

		api.GetIdpSsoCallbackUriInput
	}
	R(&IdpGetCallbackUriOptions{}, "idp-sso-callback-url", "Get sso callback url of a SSO idp", func(s *mcclient.ClientSession, args *IdpGetCallbackUriOptions) error {
		result, err := modules.IdentityProviders.GetSpecific(s, args.ID, "sso-callback-uri", jsonutils.Marshal(args))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})

	type IdpSetDefaultSsoOptions struct {
		ID string `help:"id or name of idp to set default Sso" json:"-"`

		api.PerformDefaultSsoInput
	}
	R(&IdpSetDefaultSsoOptions{}, "idp-default-sso", "Enable/disable default SSO", func(s *mcclient.ClientSession, args *IdpSetDefaultSsoOptions) error {
		result, err := modules.IdentityProviders.PerformAction(s, args.ID, "default-sso", jsonutils.Marshal(args))
		if err != nil {
			return err
		}
		printObject(result)
		return nil
	})
}
