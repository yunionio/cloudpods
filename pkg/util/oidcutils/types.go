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

package oidcutils

type SOIDCConfiguration struct {
	Issuer string `json:"issuer"`

	AuthorizationEndpoint string `json:"authorization_endpoint"`

	TokenEndpoint string `json:"token_endpoint"`

	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`

	TokenEndpointAuthSigningAlgValuesSupported []string `json:"token_endpoint_auth_signing_alg_values_supported"`

	UserinfoEndpoint string `json:"userinfo_endpoint"`

	CheckSessionIframe string `json:"check_session_iframe"`

	EndSessionEndpoint string `json:"end_session_endpoint"`

	JwksUri string `json:"jwks_uri"`

	RegistrationEndpoint string `json:"registration_endpoint"`

	ScopesSupported []string `json:"scopes_supported"`

	ResponseTypesSupported []string `json:"response_types_supported"`

	AcrValuesSupported []string `json:"acr_values_supported"`

	SubjectTypesSupported []string `json:"subject_types_supported"`

	UserinfoSigningAlgValuesSupported []string `json:"userinfo_signing_alg_values_supported"`

	UserinfoEncryptionAlgValuesSupported []string `json:"userinfo_encryption_alg_values_supported"`

	UserinfoEncryptionEncValuesSupported []string `json:"userinfo_encryption_enc_values_supported"`

	IdTokenSigningAlgValuesSupported []string `json:"id_token_signing_alg_values_supported"`

	IdTokenEncryptionAlgValuesSupported []string `json:"id_token_encryption_alg_values_supported"`

	IdTokenEncryptionEncValuesSupported []string `json:"id_token_encryption_enc_values_supported"`

	RequestObjectSigningAlgValuesSupported []string `json:"request_object_signing_alg_values_supported"`

	DisplayValuesSupported []string `json:"display_values_supported"`

	ClaimTypesSupported []string `json:"claim_types_supported"`

	ClaimsSupported []string `json:"claims_supported"`

	ClaimsParameterSupported bool `json:"claims_parameter_supported"`

	ServiceDocumentation string `json:"service_documentation"`

	UiLocalesSupported []string `json:"ui_locales_supported"`
}

type SOIDCAccessTokenRequest struct {
	// grant_type
	// REQUIRED.  Value MUST be set to "authorization_code".
	GrantType string `json:"grant_type"`

	// code
	// REQUIRED.  The authorization code received from the
	// authorization server.
	Code string `json:"code"`

	// redirect_uri
	// REQUIRED, if the "redirect_uri" parameter was included in the
	// authorization request as described in Section 4.1.1, and their
	// values MUST be identical.
	RedirectUri string `json:"redirect_uri"`

	// client_id
	// REQUIRED, if the client is not authenticating with the
	// authorization server as described in Section 3.2.1.
	ClientId string `json:"client_id"`
}

type SOIDCAccessTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	IdToken      string `json:"id_token"`
}

const (
	OIDC_RESPONSE_TYPE_CODE = "code"
	OIDC_REQUEST_GRANT_TYPE = "authorization_code"
	OIDC_BEARER_TOKEN_TYPE  = "Bearer"
)

type SOIDCAuthRequest struct {
	ResponseType string `json:"response_type"`
	ClientId     string `json:"client_id"`
	RedirectUri  string `json:"redirect_uri"`
	State        string `json:"state"`
	Scope        string `json:"scope"`
}
