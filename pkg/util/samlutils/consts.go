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

package samlutils

const (
	XMLNS_MD     = "urn:oasis:names:tc:SAML:2.0:metadata"
	XMLNS_DS     = "http://www.w3.org/2000/09/xmldsig#"
	XMLNS_PROTO  = "urn:oasis:names:tc:SAML:2.0:protocol"
	XMLNS_ASSERT = "urn:oasis:names:tc:SAML:2.0:assertion"

	PROTOCOL_SAML2 = "urn:oasis:names:tc:SAML:2.0:protocol"

	KEY_USE_SIGNING    = "signing"
	KEY_USE_ENCRYPTION = "encryption"

	NAME_ID_FORMAT_PERSISTENT = "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent"
	NAME_ID_FORMAT_TRANSIENT  = "urn:oasis:names:tc:SAML:2.0:nameid-format:transient"
	NAME_ID_FORMAT_EMAIL      = "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
	NAME_ID_FORMAT_UNSPEC     = "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified"
	NAME_ID_FORMAT_X509       = "urn:oasis:names:tc:SAML:1.1:nameid-format:X509SubjectName"
	NAME_ID_FORMAT_WINDOWS    = "urn:oasis:names:tc:SAML:1.1:nameid-format:WindowsDomainQualifiedName"
	NAME_ID_FORMAT_KERBEROS   = "urn:oasis:names:tc:SAML:2.0:nameid-format:kerberos"
	NAME_ID_FORMAT_ENTITY     = "urn:oasis:names:tc:SAML:2.0:nameid-format:entity"

	SAML2_VERSION = "2.0"

	STATUS_SUCCESS = "urn:oasis:names:tc:SAML:2.0:status:Success"

	BINDING_HTTP_POST     = "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"
	BINDING_HTTP_REDIRECT = "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"

	HTML_SAML_FORM_TOKEN  = "$FORM$"
	DEFAULT_HTML_TEMPLATE = `<!DOCTYPE html><html lang="en-US"><body>$FORM$</body></html>`
)

var (
	NAME_ID_FORMATS = []string{
		NAME_ID_FORMAT_PERSISTENT,
		NAME_ID_FORMAT_TRANSIENT,
		NAME_ID_FORMAT_EMAIL,
		NAME_ID_FORMAT_UNSPEC,
		NAME_ID_FORMAT_X509,
		NAME_ID_FORMAT_WINDOWS,
		NAME_ID_FORMAT_KERBEROS,
		NAME_ID_FORMAT_ENTITY,
	}
)
