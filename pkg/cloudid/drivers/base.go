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

package drivers

import (
	"context"
	"fmt"
	"strings"

	cloudid "yunion.io/x/cloudmux/pkg/apis/cloudid"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/cloudid"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/cloudid/models"
	"yunion.io/x/onecloud/pkg/cloudid/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SBaseProviderDriver struct {
}

func (base SBaseProviderDriver) RequestSyncCloudaccountResources(ctx context.Context, userCred mcclient.TokenCredential, account *models.SCloudaccount, provider cloudprovider.ICloudProvider) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestSyncCloudaccountResources for %s", account.Provider)
}

func (base SBaseProviderDriver) RequestSyncCloudproviderResources(ctx context.Context, userCred mcclient.TokenCredential, cp *models.SCloudprovider, provider cloudprovider.ICloudProvider) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestSyncCloudproviderResources for %s", cp.Provider)
}

func (base SBaseProviderDriver) ValidateCreateCloudgroup(ctx context.Context, userCred mcclient.TokenCredential, cp *models.SCloudprovider, input *api.CloudgroupCreateInput) (*api.CloudgroupCreateInput, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "ValidateCreateCloudgroup for %s", cp.Provider)
}

func (base SBaseProviderDriver) RequestCreateCloudgroup(ctx context.Context, userCred mcclient.TokenCredential, cp *models.SCloudprovider, group *models.SCloudgroup) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestCreateCloudgroup for %s", cp.Provider)
}

func (base SBaseProviderDriver) ValidateCreateClouduser(ctx context.Context, userCred mcclient.TokenCredential, cp *models.SCloudprovider, input *api.ClouduserCreateInput) (*api.ClouduserCreateInput, error) {
	return nil, errors.Wrapf(cloudprovider.ErrNotImplemented, "ValidateCreateClouduser for %s", cp.Provider)
}

func (base SBaseProviderDriver) RequestCreateClouduser(ctx context.Context, userCred mcclient.TokenCredential, cp *models.SCloudprovider, user *models.SClouduser) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestCreateClouduser for %s", cp.Provider)
}

func (base SBaseProviderDriver) RequestCreateSAMLProvider(ctx context.Context, userCred mcclient.TokenCredential, account *models.SCloudaccount) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestCreateSAMLProvider for %s", account.Provider)
}

func (base SBaseProviderDriver) RequestCreateRoleForSamlUser(ctx context.Context, userCred mcclient.TokenCredential, account *models.SCloudaccount, group *models.SCloudgroup, user *models.SSamluser) error {
	return errors.Wrapf(cloudprovider.ErrNotImplemented, "RequestCreateRoleForSamlUser for %s", account.Provider)
}

type SAccountBaseProviderDriver struct {
	SBaseProviderDriver
}

func (base SAccountBaseProviderDriver) ValidateCreateCloudgroup(ctx context.Context, userCred mcclient.TokenCredential, cp *models.SCloudprovider, input *api.CloudgroupCreateInput) (*api.CloudgroupCreateInput, error) {
	for i := range input.CloudpolicyIds {
		policyObj, err := validators.ValidateModel(ctx, userCred, models.CloudpolicyManager, &input.CloudpolicyIds[i])
		if err != nil {
			return nil, err
		}
		policy := policyObj.(*models.SCloudpolicy)
		if policy.CloudaccountId != cp.CloudaccountId {
			return nil, httperrors.NewConflictError("policy %s do not belong to accounts %s", policy.Name, cp.Name)
		}
	}
	return input, nil
}

func (base SAccountBaseProviderDriver) RequestCreateCloudgroup(ctx context.Context, userCred mcclient.TokenCredential, cp *models.SCloudprovider, group *models.SCloudgroup) error {
	_, err := db.Update(group, func() error {
		group.ManagerId = ""
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	driver := &SProviderBaseProviderDriver{}
	return driver.RequestCreateCloudgroup(ctx, userCred, cp, group)
}

func (base SAccountBaseProviderDriver) RequestCreateClouduser(ctx context.Context, userCred mcclient.TokenCredential, cp *models.SCloudprovider, user *models.SClouduser) error {
	_, err := db.Update(user, func() error {
		user.ManagerId = ""
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	driver := &SProviderBaseProviderDriver{}
	return driver.RequestCreateClouduser(ctx, userCred, cp, user)
}

func (base SAccountBaseProviderDriver) ValidateCreateClouduser(ctx context.Context, userCred mcclient.TokenCredential, cp *models.SCloudprovider, input *api.ClouduserCreateInput) (*api.ClouduserCreateInput, error) {
	for i := range input.CloudpolicyIds {
		policyObj, err := validators.ValidateModel(ctx, userCred, models.CloudpolicyManager, &input.CloudpolicyIds[i])
		if err != nil {
			return nil, err
		}
		policy := policyObj.(*models.SCloudpolicy)
		if policy.CloudaccountId != cp.CloudaccountId {
			return nil, httperrors.NewConflictError("policy %s do not belong to accounts %s", policy.Name, cp.Name)
		}
	}
	for i := range input.CloudgroupIds {
		groupObj, err := validators.ValidateModel(ctx, userCred, models.CloudgroupManager, &input.CloudgroupIds[i])
		if err != nil {
			return nil, err
		}
		group := groupObj.(*models.SCloudgroup)
		if group.CloudaccountId != cp.CloudaccountId {
			return nil, httperrors.NewConflictError("group %s do not belong to accounts %s", group.Name, cp.Name)
		}
	}
	return input, nil
}

func (base SAccountBaseProviderDriver) RequestSyncCloudaccountResources(ctx context.Context, userCred mcclient.TokenCredential, account *models.SCloudaccount, provider cloudprovider.ICloudProvider) error {

	func() {
		policies, err := provider.GetICloudpolicies()
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				log.Errorf("get policies for account %s error: %v", account.Name, err)
			}
			return
		}
		result := account.SyncPolicies(ctx, userCred, policies, "")
		log.Infof("Sync %s policies for account %s result: %s", account.Provider, account.Name, result.Result())
	}()

	func() {
		lockman.LockRawObject(ctx, account.Id, models.SAMLProviderManager.Keyword())
		defer lockman.ReleaseRawObject(ctx, account.Id, models.SAMLProviderManager.Keyword())

		samls, err := provider.GetICloudSAMLProviders()
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				log.Errorf("get saml providers for account %s error: %v", account.Name, err)
			}
			return
		}
		result := account.SyncSAMLProviders(ctx, userCred, samls, "")
		log.Infof("Sync SAMLProviders for account %s(%s) result: %s", account.Name, account.Provider, result.Result())
	}()

	func() {
		roles, err := provider.GetICloudroles()
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				log.Errorf("get roles for account %s error: %v", account.Name, err)
			}
			return
		}
		result := account.SyncCloudroles(ctx, userCred, roles, "")
		log.Infof("SyncCloudroles for account %s(%s) result: %s", account.Name, account.Provider, result.Result())
	}()

	func() {
		iGroups, err := provider.GetICloudgroups()
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				log.Errorf("get groups for account %s error: %v", account.Name, err)
			}
			return
		}
		localGroups, remoteGroups, result := account.SyncCloudgroups(ctx, userCred, iGroups, "")
		log.Infof("SyncCloudgroups for account %s(%s) result: %s", account.Name, account.Provider, result.Result())
		for i := 0; i < len(localGroups); i += 1 {
			func() {
				// lock cloudgroup
				lockman.LockObject(ctx, &localGroups[i])
				defer lockman.ReleaseObject(ctx, &localGroups[i])

				localGroups[i].SyncCloudpolicies(ctx, userCred, remoteGroups[i])
			}()
		}
	}()

	func() {
		iUsers, err := provider.GetICloudusers()
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				log.Errorf("get users for account %s error: %v", account.Name, err)
			}
			return
		}
		localUsers, remoteUsers, result := account.SyncCloudusers(ctx, userCred, iUsers, "")
		log.Infof("SyncCloudusers for account %s(%s) result: %s", account.Name, account.Provider, result.Result())
		for i := 0; i < len(localUsers); i += 1 {
			func() {
				// lock clouduser
				lockman.LockObject(ctx, &localUsers[i])
				defer lockman.ReleaseObject(ctx, &localUsers[i])

				localUsers[i].SyncCloudpolicies(ctx, userCred, remoteUsers[i])
				localUsers[i].SyncCloudgroups(ctx, userCred, remoteUsers[i])
			}()
		}
	}()

	return nil
}

type SProviderBaseProviderDriver struct {
	SBaseProviderDriver
}

func (base SProviderBaseProviderDriver) RequestSyncCloudproviderResources(ctx context.Context, userCred mcclient.TokenCredential, cp *models.SCloudprovider, provider cloudprovider.ICloudProvider) error {
	account, err := cp.GetCloudaccount()
	if err != nil {
		return errors.Wrapf(err, "GetCloudaccount")
	}

	func() {
		policies, err := provider.GetICloudpolicies()
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				log.Errorf("get system policies for manager %s error: %v", cp.Name, err)
			}
			return
		}
		result := account.SyncPolicies(ctx, userCred, policies, cp.Id)
		log.Infof("Sync %s policies for manager %s result: %s", cp.Provider, cp.Name, result.Result())
	}()

	func() {
		lockman.LockRawObject(ctx, cp.Id, models.SAMLProviderManager.Keyword())
		defer lockman.ReleaseRawObject(ctx, cp.Id, models.SAMLProviderManager.Keyword())

		samls, err := provider.GetICloudSAMLProviders()
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				log.Errorf("get saml providers for manager %s error: %v", cp.Name, err)
			}
			return
		}
		result := account.SyncSAMLProviders(ctx, userCred, samls, cp.Id)
		log.Infof("Sync SAMLProviders for manager %s(%s) result: %s", cp.Name, cp.Provider, result.Result())
	}()

	func() {
		roles, err := provider.GetICloudroles()
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				log.Errorf("get roles for manager %s error: %v", cp.Name, err)
			}
			return
		}
		result := account.SyncCloudroles(ctx, userCred, roles, cp.Id)
		log.Infof("SyncCloudroles for manager %s(%s) result: %s", cp.Name, cp.Provider, result.Result())
	}()

	func() {
		iGroups, err := provider.GetICloudgroups()
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				log.Errorf("get groups for manager %s error: %v", cp.Name, err)
			}
			return
		}
		localGroups, remoteGroups, result := account.SyncCloudgroups(ctx, userCred, iGroups, cp.Id)
		log.Infof("SyncCloudgroups for manager %s(%s) result: %s", cp.Name, cp.Provider, result.Result())

		for i := 0; i < len(localGroups); i += 1 {
			func() {
				// lock cloudgroup
				lockman.LockObject(ctx, &localGroups[i])
				defer lockman.ReleaseObject(ctx, &localGroups[i])

				localGroups[i].SyncCloudpolicies(ctx, userCred, remoteGroups[i])
			}()
		}
	}()

	func() {
		iUsers, err := provider.GetICloudusers()
		if err != nil {
			if errors.Cause(err) != cloudprovider.ErrNotSupported && errors.Cause(err) != cloudprovider.ErrNotImplemented {
				log.Errorf("get users for manager %s error: %v", cp.Name, err)
			}
			return
		}
		localUsers, remoteUsers, result := account.SyncCloudusers(ctx, userCred, iUsers, cp.Id)
		log.Infof("SyncCloudusers for manger %s(%s) result: %s", cp.Name, cp.Provider, result.Result())
		for i := 0; i < len(localUsers); i += 1 {
			func() {
				// lock clouduser
				lockman.LockObject(ctx, &localUsers[i])
				defer lockman.ReleaseObject(ctx, &localUsers[i])

				localUsers[i].SyncCloudpolicies(ctx, userCred, remoteUsers[i])
				localUsers[i].SyncCloudgroups(ctx, userCred, remoteUsers[i])
			}()
		}
	}()

	return nil
}

func (base SProviderBaseProviderDriver) ValidateCreateCloudgroup(ctx context.Context, userCred mcclient.TokenCredential, cp *models.SCloudprovider, input *api.CloudgroupCreateInput) (*api.CloudgroupCreateInput, error) {
	for i := range input.CloudpolicyIds {
		policyObj, err := validators.ValidateModel(ctx, userCred, models.CloudpolicyManager, &input.CloudpolicyIds[i])
		if err != nil {
			return nil, err
		}
		policy := policyObj.(*models.SCloudpolicy)
		if policy.ManagerId != cp.Id {
			return nil, httperrors.NewConflictError("policy %s do not belong to subaccounts %s", policy.Name, cp.Name)
		}
	}
	return input, nil
}

func (base SProviderBaseProviderDriver) ValidateCreateClouduser(ctx context.Context, userCred mcclient.TokenCredential, cp *models.SCloudprovider, input *api.ClouduserCreateInput) (*api.ClouduserCreateInput, error) {
	for i := range input.CloudpolicyIds {
		policyObj, err := validators.ValidateModel(ctx, userCred, models.CloudpolicyManager, &input.CloudpolicyIds[i])
		if err != nil {
			return nil, err
		}
		policy := policyObj.(*models.SCloudpolicy)
		if policy.ManagerId != cp.Id {
			return nil, httperrors.NewConflictError("policy %s do not belong to subaccounts %s", policy.Name, cp.Name)
		}
	}
	for i := range input.CloudgroupIds {
		groupObj, err := validators.ValidateModel(ctx, userCred, models.CloudgroupManager, &input.CloudgroupIds[i])
		if err != nil {
			return nil, err
		}
		group := groupObj.(*models.SCloudgroup)
		if group.ManagerId != cp.Id {
			return nil, httperrors.NewConflictError("group %s do not belong to subaccounts %s", group.Name, cp.Name)
		}
	}

	return input, nil
}

func (base SProviderBaseProviderDriver) RequestCreateCloudgroup(ctx context.Context, userCred mcclient.TokenCredential, cp *models.SCloudprovider, group *models.SCloudgroup) error {
	provider, err := group.GetProvider()
	if err != nil {
		return errors.Wrapf(err, "GetProvider")
	}
	iGroup, err := provider.CreateICloudgroup(group.Name, group.Description)
	if err != nil {
		return errors.Wrapf(err, "CreateICloudgroup")
	}
	_, err = db.Update(group, func() error {
		group.ExternalId = iGroup.GetGlobalId()
		group.Status = apis.STATUS_AVAILABLE
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	policies, err := group.GetCloudpolicies()
	if err != nil {
		return errors.Wrapf(err, "GetCloudpolicies")
	}
	for i := range policies {
		err = iGroup.AttachPolicy(policies[i].ExternalId, cloudid.TPolicyType(policies[i].PolicyType))
		if err != nil {
			return errors.Wrapf(err, "Attach %s policy %s", policies[i].PolicyType, policies[i].ExternalId)
		}
	}
	group.SyncCloudpolicies(ctx, userCred, iGroup)
	return nil
}

func (base SProviderBaseProviderDriver) RequestCreateClouduser(ctx context.Context, userCred mcclient.TokenCredential, cp *models.SCloudprovider, user *models.SClouduser) error {
	provider, err := user.GetProvider()
	if err != nil {
		return errors.Wrapf(err, "GetProvider")
	}

	opts := &cloudprovider.SClouduserCreateConfig{
		Name:           user.Name,
		Desc:           user.Description,
		IsConsoleLogin: user.IsConsoleLogin.Bool(),
		Email:          user.Email,
		MobilePhone:    user.MobilePhone,
	}
	opts.Password, _ = user.GetPassword()

	iUser, err := provider.CreateIClouduser(opts)
	if err != nil {
		return errors.Wrapf(err, "CreateIClouduser")
	}
	_, err = db.Update(user, func() error {
		user.ExternalId = iUser.GetGlobalId()
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "db.Update")
	}
	policies, err := user.GetCloudpolicies()
	if err != nil {
		return errors.Wrapf(err, "GetCloudpolicies")
	}
	for i := range policies {
		err = iUser.AttachPolicy(policies[i].ExternalId, cloudid.TPolicyType(policies[i].PolicyType))
		if err != nil {
			return errors.Wrapf(err, "Attach %s policy %s", policies[i].PolicyType, policies[i].ExternalId)
		}
	}
	user.SyncCloudpolicies(ctx, userCred, iUser)

	groups, err := user.GetCloudgroups()
	if err != nil {
		return errors.Wrapf(err, "GetCloudgroups")
	}

	for i := range groups {
		iGroup, err := groups[i].GetICloudgroup()
		if err != nil {
			return errors.Wrapf(err, "GetICloudgroup")
		}
		err = iGroup.AddUser(user.Name)
		if err != nil {
			return errors.Wrapf(err, "join group %s", groups[i].Name)
		}
	}

	user.SyncCloudgroups(ctx, userCred, iUser)

	return nil
}

func (base SProviderBaseProviderDriver) RequestCreateSAMLProvider(ctx context.Context, userCred mcclient.TokenCredential, account *models.SCloudaccount) error {
	providers, err := account.GetCloudproviders()
	if err != nil {
		return errors.Wrapf(err, "GetCloudproviders")
	}
	for i := range providers {
		err = func() error {
			lockman.LockRawObject(ctx, providers[i].Id, models.SAMLProviderManager.Keyword())
			defer lockman.ReleaseRawObject(ctx, providers[i].Id, models.SAMLProviderManager.Keyword())

			samlProviders, err := providers[i].GetSamlProviders()
			if err != nil {
				return errors.Wrapf(err, "GetSamlProviders")
			}

			iProvider, err := providers[i].GetProvider()
			if err != nil {
				return errors.Wrapf(err, "GetProvider")
			}

			for j := range samlProviders {
				if samlProviders[j].EntityId != options.Options.ApiServer {
					continue
				}
				if strings.Contains(samlProviders[j].MetadataDocument, "login/"+providers[i].Id) {
					return nil
				}
				iSamlProviders, err := iProvider.GetICloudSAMLProviders()
				if err != nil {
					return errors.Wrapf(err, "GetICloudSAMLProviders")
				}
				for _, iSaml := range iSamlProviders {
					if iSaml.GetGlobalId() != samlProviders[j].ExternalId {
						continue
					}
					doc := models.SamlIdpInstance().GetMetadata(providers[i].Id)
					err := iSaml.UpdateMetadata(doc)
					if err != nil {
						return errors.Wrapf(err, "UpdateMetadata")
					}
					_, err = db.Update(&samlProviders[j], func() error {
						samlProviders[j].Status = apis.STATUS_AVAILABLE
						samlProviders[j].MetadataDocument = doc.String()
						return nil
					})
					return err
				}
			}

			opts := &cloudprovider.SAMLProviderCreateOptions{
				Metadata: models.SamlIdpInstance().GetMetadata(providers[i].Id),
				Name:     strings.TrimPrefix(options.Options.ApiServer, "https://"),
				Desc:     "create by cloudpods",
			}

			log.Debugf("create saml provider for manager %s(%s) %s", providers[i].Name, providers[i].Id, opts.Metadata.String())

			opts.Name = strings.TrimPrefix(opts.Name, "http://")

			iSaml, err := iProvider.CreateICloudSAMLProvider(opts)
			if err != nil {
				return errors.Wrapf(err, "CreateICloudSAMLProvider")
			}

			saml := &models.SSAMLProvider{}
			saml.SetModelManager(models.SAMLProviderManager, saml)
			saml.Name = opts.Name
			saml.DomainId = account.DomainId
			saml.ManagerId = providers[i].Id
			saml.CloudaccountId = account.Id
			saml.ExternalId = iSaml.GetGlobalId()
			saml.Status = apis.STATUS_AVAILABLE
			saml.EntityId = options.Options.ApiServer
			err = models.SAMLProviderManager.TableSpec().Insert(ctx, saml)
			if err != nil {
				return err
			}
			return saml.SyncWithCloudSAMLProvider(ctx, userCred, iSaml, providers[i].Id)
		}()
		if err != nil {
			log.Errorf("create saml provider for manager %s error: %v", providers[i].Name, err)
		}
	}
	return nil
}

func (base SProviderBaseProviderDriver) RequestCreateRoleForSamlUser(ctx context.Context, userCred mcclient.TokenCredential, account *models.SCloudaccount, group *models.SCloudgroup, user *models.SSamluser) error {
	provider, err := group.GetCloudprovider()
	if err != nil {
		return errors.Wrapf(err, "GetCloudprovider")
	}
	samlProvider, err := provider.GetSamlProvider()
	if err != nil {
		return errors.Wrapf(err, "GetSamlProvider")
	}
	policies, err := group.GetCloudpolicies()
	if err != nil {
		return errors.Wrapf(err, "GetCloudpolicies")
	}
	roles, err := account.GetCloudroles(provider.Id)
	if err != nil {
		return errors.Wrapf(err, "GetCloudroles")
	}
	for i := range roles {
		if roles[i].Status == apis.STATUS_AVAILABLE && len(roles[i].ExternalId) > 0 && roles[i].SAMLProviderId == samlProvider.Id && roles[i].CloudgroupId == group.Id {
			_, err := db.Update(user, func() error {
				user.CloudroleId = roles[i].Id
				return nil
			})
			if err != nil {
				return err
			}
			existPolicies := []string{}
			iRole, err := roles[i].GetICloudrole()
			if err != nil {
				return err
			}
			iPolicies, err := iRole.GetICloudpolicies()
			if err != nil {
				return errors.Wrapf(err, "GetICloudpolicies")
			}
			for _, policy := range iPolicies {
				existPolicies = append(existPolicies, policy.GetGlobalId())
			}
			for _, policy := range policies {
				if !utils.IsInStringArray(policy.ExternalId, existPolicies) {
					err = iRole.AttachPolicy(policy.ExternalId, cloudid.TPolicyType(policy.PolicyType))
					if err != nil {
						return errors.Wrapf(err, "attach %s policy %s", policy.PolicyType, policy.ExternalId)
					}
				}
			}
			return nil
		}
	}
	iProvider, err := group.GetProvider()
	if err != nil {
		return errors.Wrapf(err, "GetProvider")
	}
	opts := &cloudprovider.SRoleCreateOptions{
		Name:         fmt.Sprintf("%s-%s", group.Name, utils.GenRequestId(5)),
		Desc:         fmt.Sprintf("auto create by cloudpods"),
		SAMLProvider: samlProvider.ExternalId,
	}
	iRole, err := iProvider.CreateICloudrole(opts)
	if err != nil {
		return errors.Wrapf(err, "CreateICloudrole")
	}
	role := &models.SCloudrole{}
	role.SetModelManager(models.CloudroleManager, role)
	role.ExternalId = iRole.GetGlobalId()
	role.Name = iRole.GetName()
	role.Document = iRole.GetDocument()
	role.CloudaccountId = account.Id
	role.ManagerId = group.ManagerId
	role.SAMLProviderId = samlProvider.Id
	role.CloudgroupId = group.Id
	role.Status = apis.STATUS_AVAILABLE
	role.SetEnabled(true)
	err = models.CloudroleManager.TableSpec().Insert(ctx, role)
	if err != nil {
		return errors.Wrapf(err, "")
	}
	_, err = db.Update(user, func() error {
		user.CloudroleId = role.Id
		return nil
	})
	if err != nil {
		return err
	}
	for _, policy := range policies {
		err := iRole.AttachPolicy(policy.ExternalId, cloudid.TPolicyType(policy.PolicyType))
		if err != nil {
			return errors.Wrapf(err, "attach %s policy %s", policy.PolicyType, policy.ExternalId)
		}
	}
	return nil
}
