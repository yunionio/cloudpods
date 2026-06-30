package models

import (
	"context"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	"yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

// GetSkuPreferHosts returns host ids stored on sku.
func GetSkuPreferHosts(sku *SLLMSku) []string {
	if sku == nil || len(sku.PreferHosts) == 0 {
		return nil
	}
	return append([]string(nil), sku.PreferHosts...)
}

// GetDeploymentPreferHosts returns deployment prefer_hosts, falling back to sku.
func GetDeploymentPreferHosts(dep *SLLMDeployment, sku *SLLMSku) []string {
	if dep != nil && len(dep.PreferHosts) > 0 {
		return append([]string(nil), dep.PreferHosts...)
	}
	return GetSkuPreferHosts(sku)
}

// SelectPreferHostForInstanceIndex picks a host from prefer_hosts using round-robin.
func SelectPreferHostForInstanceIndex(hosts []string, index int) string {
	if len(hosts) == 0 {
		return ""
	}
	if index < 0 {
		index = 0
	}
	return hosts[index%len(hosts)]
}

func validatePreferHostsSubset(selected, allowed []string) error {
	if len(selected) == 0 {
		return httperrors.NewMissingParameterError("prefer_hosts")
	}
	if len(allowed) == 0 {
		return nil
	}
	allowedSet := sets.NewString(allowed...)
	for _, h := range selected {
		if !allowedSet.Has(h) {
			return httperrors.NewInputParameterError("prefer_hosts %q is not declared on the LLM SKU", h)
		}
	}
	return nil
}

func normalizePreferHostInputs(hosts []string) []string {
	seen := sets.NewString()
	out := make([]string, 0, len(hosts))
	for _, h := range hosts {
		h = strings.TrimSpace(h)
		if h == "" || seen.Has(h) {
			continue
		}
		seen.Insert(h)
		out = append(out, h)
	}
	return out
}

func resolvePreferHosts(ctx context.Context, userCred mcclient.TokenCredential, hosts []string) ([]string, error) {
	hosts = normalizePreferHostInputs(hosts)
	if len(hosts) == 0 {
		return nil, httperrors.NewMissingParameterError("prefer_hosts")
	}
	s := auth.GetSession(ctx, userCred, "")
	resolved := make([]string, 0, len(hosts))
	for _, hostRef := range hosts {
		hostJson, err := compute.Hosts.Get(s, hostRef, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "get host %s", hostRef)
		}
		hostDetails := computeapi.HostDetails{}
		if err := hostJson.Unmarshal(&hostDetails); err != nil {
			return nil, errors.Wrap(err, "unmarshal hostDetails")
		}
		if hostDetails.Enabled == nil || !*hostDetails.Enabled {
			return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "host %s is not enabled", hostRef)
		}
		if hostDetails.HostStatus != computeapi.HOST_ONLINE {
			return nil, errors.Wrapf(httperrors.ErrInvalidStatus, "host %s is not online", hostRef)
		}
		if hostDetails.HostType != computeapi.HOST_TYPE_CONTAINER {
			return nil, errors.Wrapf(httperrors.ErrNotAcceptable, "host %s type %s is not supported", hostRef, hostDetails.HostType)
		}
		resolved = append(resolved, hostDetails.Id)
	}
	return normalizePreferHostInputs(resolved), nil
}

func resolvePreferHost(ctx context.Context, userCred mcclient.TokenCredential, hostRef string) (string, error) {
	hosts, err := resolvePreferHosts(ctx, userCred, []string{hostRef})
	if err != nil {
		return "", err
	}
	if len(hosts) == 0 {
		return "", httperrors.NewMissingParameterError("prefer_host")
	}
	return hosts[0], nil
}

func validateLocalPathDeploymentPreferHosts(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	input *api.LLMDeploymentCreateInput,
	sku *SLLMSku,
) error {
	if input == nil || sku == nil || !SkuHasLocalHostPathModel(sku) {
		return nil
	}
	skuHosts := GetSkuPreferHosts(sku)
	if len(input.PreferHosts) == 0 {
		input.PreferHosts = append([]string(nil), skuHosts...)
	}
	if len(input.PreferHosts) == 0 {
		return httperrors.NewMissingParameterError("prefer_hosts is required for local_path SKU")
	}
	resolved, err := resolvePreferHosts(ctx, userCred, input.PreferHosts)
	if err != nil {
		return err
	}
	if err := validatePreferHostsSubset(resolved, skuHosts); err != nil {
		return err
	}
	input.PreferHosts = resolved
	return nil
}
