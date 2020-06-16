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

import (
	"encoding/xml"
	"testing"
)

var (
	spMetadata1 = `
<!-- This is the metadata for the SAMLtest SP, named by entityID -->
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" ID="SAMLtestSP" entityID="https://samltest.id/saml/sp">
<!-- This list enumerates the cryptographic algorithms acceptable to this SP -->
  <md:Extensions xmlns:alg="urn:oasis:names:tc:SAML:metadata:algsupport">
    <alg:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha512"/>
    <alg:DigestMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#sha384"/>
    <alg:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256"/>
    <alg:DigestMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#sha224"/>
    <alg:DigestMethod Algorithm="http://www.w3.org/2000/09/xmldsig#sha1"/>
    <alg:SigningMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha512"/>
    <alg:SigningMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha384"/>
    <alg:SigningMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha256"/>
    <alg:SigningMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha224"/>
    <alg:SigningMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha512"/>
    <alg:SigningMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha384"/>
    <alg:SigningMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>
    <alg:SigningMethod Algorithm="http://www.w3.org/2009/xmldsig11#dsa-sha256"/>
    <alg:SigningMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha1"/>
    <alg:SigningMethod Algorithm="http://www.w3.org/2000/09/xmldsig#rsa-sha1"/>
    <alg:SigningMethod Algorithm="http://www.w3.org/2000/09/xmldsig#dsa-sha1"/>
  </md:Extensions>

  <md:SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <md:Extensions>

<!-- The location to redirect users to for invocation of an AuthnRequest -->
      <init:RequestInitiator xmlns:init="urn:oasis:names:tc:SAML:profiles:SSO:request-init" Binding="urn:oasis:names:tc:SAML:profiles:SSO:request-init" Location="https://samltest.id/Shibboleth.sso/Login"/>

<!-- Display information about this SP that the IdP can present to users -->
      <mdui:UIInfo xmlns:mdui="urn:oasis:names:tc:SAML:metadata:ui">
          <mdui:DisplayName xml:lang="en">SAMLtest SP</mdui:DisplayName>
          <mdui:Description xml:lang="en">A free and basic SP for testing SAML deployments</mdui:Description>
          <mdui:Logo height="90" width="225">https://samltest.id/saml/logo.png</mdui:Logo>
       </mdui:UIInfo>

    </md:Extensions>
<!-- A certificate containing the public key for verification of signed messages from this SP.
This is rarely used because the SP sends few signed messages, but using a separate key is better
security hygiene.  In practice, many SP's use only one key for both encryption and signature.
Most SAML implementations don't rely on the rest of the certificate's contents. -->
    <md:KeyDescriptor use="signing">
      <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
        <ds:X509Data>
          <ds:X509Certificate>
MIIERTCCAq2gAwIBAgIJAKmtzjCD1+tqMA0GCSqGSIb3DQEBCwUAMDUxMzAxBgNV
BAMTKmlwLTE3Mi0zMS0yOC02NC51cy13ZXN0LTIuY29tcHV0ZS5pbnRlcm5hbDAe
Fw0xODA4MTgyMzI0MjNaFw0yODA4MTUyMzI0MjNaMDUxMzAxBgNVBAMTKmlwLTE3
Mi0zMS0yOC02NC51cy13ZXN0LTIuY29tcHV0ZS5pbnRlcm5hbDCCAaIwDQYJKoZI
hvcNAQEBBQADggGPADCCAYoCggGBALhUlY3SkIOze+l8y6dBzM6p7B8OykJWlwiz
szU16Lih8D7KLhNJfahoVxbPxB3YFM/81PJLOeK2krvJ5zY6CJyQY3sPQAkZKI7I
8qq9lmZ2g4QPqybNstXS6YUXJNUt/ixbbK/N97+LKTiSutbD1J7AoFnouMuLjlhN
5VRZ43jez4xLSHVZaYuUFKn01Y9oLKbj46LQnZnJCAGpTgPqEQJr6GpVGw43bKyU
pGoaPrdDRgRgtPMUWgFDkgcI3QiV1lsKfBs1t1E2UA7ACFnlJZpEuBtwgivzo3Ve
itiSaF3Jxh25EY5/vABpcgQQRz3RH2l8MMKdRsxb8VT3yh2S+CX55s+cN67LiCPr
6f2u+KS1iKfB9mWN6o2S4lcmo82HIBbsuXJV0oA1HrGMyyc4Y9nng/I8iuAp8or1
JrWRHQ+8NzO85DWK0rtvtLPxkvw0HK32glyuOP/9F05Z7+tiVIgn67buC0EdoUm1
RSpibqmB1ST2PikslOlVbJuy4Ah93wIDAQABo1gwVjA1BgNVHREELjAsgippcC0x
NzItMzEtMjgtNjQudXMtd2VzdC0yLmNvbXB1dGUuaW50ZXJuYWwwHQYDVR0OBBYE
FAdsTxYfulJ5yunYtgYJHC9IcevzMA0GCSqGSIb3DQEBCwUAA4IBgQB3J6i7Krei
HL8NPMglfWLHk1PZOgvIEEpKL+GRebvcbyqgcuc3VVPylq70VvGqhJxp1q/mzLfr
aUiypzfWFGm9zfwIg0H5TqRZYEPTvgIhIICjaDWRwZBDJG8D5G/KoV60DlUG0crP
BlIuCCr/SRa5ZoDQqvucTfr3Rx4Ha6koXFSjoSXllR+jn4GnInhm/WH137a+v35P
UcffNxfuehoGn6i4YeXF3cwJK4e35cOFW+dLbnaLk+Ty7HOGvpw86h979C6mJ9qE
HYgq9rQyzlSPbLZGZSgVcIezunOaOsWm81BsXRNNJjzHGCqKf8RMhd8oZP55+2/S
VRBwnkGyUNCuDPrJcymC95ZT2NW/KeWkz28HF2i31xQmecT2r3lQRSM8acvOXQsN
EDCDvJvCzJT9c2AnsnO24r6arPXs/UWAxOI+MjclXPLkLD6uTHV+Oo8XZ7bOjegD
5hL6/bKUWnNMurQNGrmi/jvqsCFLDKftl7ajuxKjtodnSuwhoY7NQy8=
</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </md:KeyDescriptor>
<!-- A certificate containing the public key for encryption of messages sent to the SAMLtest SP.
This key is crucial for securing assertions from IdP's.  Multiple encryption keys can be listed
and this will often be necessary for key rollovers. -->
    <md:KeyDescriptor use="encryption">
      <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
        <ds:X509Data>
          <ds:X509Certificate>
MIIERTCCAq2gAwIBAgIJAKGA/tV7hXUvMA0GCSqGSIb3DQEBCwUAMDUxMzAxBgNV
BAMTKmlwLTE3Mi0zMS0yOC02NC51cy13ZXN0LTIuY29tcHV0ZS5pbnRlcm5hbDAe
Fw0xODA4MTgyMzI0MjVaFw0yODA4MTUyMzI0MjVaMDUxMzAxBgNVBAMTKmlwLTE3
Mi0zMS0yOC02NC51cy13ZXN0LTIuY29tcHV0ZS5pbnRlcm5hbDCCAaIwDQYJKoZI
hvcNAQEBBQADggGPADCCAYoCggGBANoi7TtbPz5DD5b+pGj2bWHUWcOm135Dl+kf
KWcJV6x4Z4VRMa33nwSfFg6U0DhPaA6rYr8BfcmCIY4V4cGlJkLNsYbgbZNnrLh2
3mj7jkaUeyv/DlGtLBcqr0gP6eDtcOf3MMGAkhROcicMj6i+uF6hqLDh4eNcpqEV
DVn+ADBsosIPiAx+RkcyZkfAF3UeGEV5WTSiQw7qYpI7x+c4ViiBzV4waBgXjvNN
72Dqlc01AylpmMKaUPfxIpPC+Ctr0bHu5xn7NxMS8Zt5NDWsP9T15qrpYatW68sX
VyE5nJRYpiRiRbo8i7QpUEya+TkXEI8PVD3KBw9UwhqL8qPPe0T+EeaawF6BVRTE
Pc+Mn4lGBr4cCFcGk/PLHeyksgPdjNmO1g7y5TWQzu21WzkXRTWJq7wGwWeW6Nrc
NqweYPLbXEo0JlmHqunkUs+NsLQAFqSPX02P2xzkA/eOU2o/jN4jAPNpzqxJouvm
iWGXl8Qy4U7vQZ0tGvlTDSltATOQ/QIDAQABo1gwVjA1BgNVHREELjAsgippcC0x
NzItMzEtMjgtNjQudXMtd2VzdC0yLmNvbXB1dGUuaW50ZXJuYWwwHQYDVR0OBBYE
FBBtS9YNKSIwViH37GJCTxjNBzLAMA0GCSqGSIb3DQEBCwUAA4IBgQDWXcaI7zMn
hGsLVTUA6dgzZCa88QkN/Z6n7lCY2oaKj1neBAWA1Mxg7GBJsmLOrHN8ie0D/uKA
F+7NqKCXYqd0PpTX7c1NICL92DvbugG/Ow50j5Dw6rU4Y8dPS7Y/T1ddbT2F9/5l
HCIWP/O2E9HREJ0JAIbu/Mi0CE1qui2aSJMDWKuiGK63M/7fvP51m6xSJOfZBhmj
gllIwEhIzfh4hVPhH0C7iqVls34UyLCZ8IZOCuGPJyTaJN6Pi3Uo1Otkz/1igN5M
pQhVaeYG7SMgha6skTLrVXTt4CuMVsOZ6cG3kHqw8XZoRld+I50iyHqansf5qwzm
NoPeXyjGRFQzV/EH3SUu8eAISTt9pfirwjKsVNHrmMRnQEB/hJYYbTWSsvdS8ghw
7a/A0EKQPVaZGCP/hcpt9JMMb66y2L8VgBbb6aTsR+Uabf6aiMnj1UBMUz9yaMka
kKM7e66uHdXUDZ/s8F5rPOGCK+O8O6EsLRf8XetRWLa1TXRDkJZVPX4=
</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
      <md:EncryptionMethod Algorithm="http://www.w3.org/2009/xmlenc11#aes128-gcm"/>
      <md:EncryptionMethod Algorithm="http://www.w3.org/2009/xmlenc11#aes192-gcm"/>
      <md:EncryptionMethod Algorithm="http://www.w3.org/2009/xmlenc11#aes256-gcm"/>
      <md:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#aes128-cbc"/>
      <md:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#aes192-cbc"/>
      <md:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#aes256-cbc"/>
      <md:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#tripledes-cbc"/>
      <md:EncryptionMethod Algorithm="http://www.w3.org/2009/xmlenc11#rsa-oaep"/>
      <md:EncryptionMethod Algorithm="http://www.w3.org/2001/04/xmlenc#rsa-oaep-mgf1p"/>
    </md:KeyDescriptor>

<!-- These endpoints tell IdP's where to send messages, either directly or via
a browser redirect.  The locations must match the address of the SP as seen from the outside
world if this host is behind a reverse proxy. -->
    <md:ArtifactResolutionService Binding="urn:oasis:names:tc:SAML:2.0:bindings:SOAP" Location="https://samltest.id/Shibboleth.sso/Artifact/SOAP" index="1"/>
    <md:SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:SOAP" Location="https://samltest.id/Shibboleth.sso/SLO/SOAP"/>
    <md:SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://samltest.id/Shibboleth.sso/SLO/Redirect"/>
    <md:SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="https://samltest.id/Shibboleth.sso/SLO/POST"/>
    <md:SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Artifact" Location="https://samltest.id/Shibboleth.sso/SLO/Artifact"/>
<!-- The primary endpoint to which SAML assertions will be delivered. -->
    <md:AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="https://samltest.id/Shibboleth.sso/SAML2/POST" index="1"/>
    <md:AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST-SimpleSign" Location="https://samltest.id/Shibboleth.sso/SAML2/POST-SimpleSign" index="2"/>
    <md:AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Artifact" Location="https://samltest.id/Shibboleth.sso/SAML2/Artifact" index="3"/>
    <md:AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:PAOS" Location="https://samltest.id/Shibboleth.sso/SAML2/ECP" index="4"/>
  </md:SPSSODescriptor>

</md:EntityDescriptor>
`

	idpMetadata = `<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" ID="SAMLtestIdP" xmlns:ds="http://www.w3.org/2000/09/xmldsig#" xmlns:shibmd="urn:mace:shibboleth:metadata:1.0" xmlns:xml="http://www.w3.org/XML/1998/namespace" xmlns:mdui="urn:oasis:names:tc:SAML:metadata:ui" entityID="https://samltest.id/saml/idp">

    <IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol urn:oasis:names:tc:SAML:1.1:protocol urn:mace:shibboleth:1.0">

        <Extensions>
<!-- An enumeration of the domains this IdP is able to assert scoped attributes, which are
typically those with a @ delimiter, like mail.  Most IdP's serve only a single domain.  It's crucial
for the SP to check received attribute values match permitted domains to prevent a recognized IdP from 
sending attribute values for which a different recognized IdP is authoritative. -->
            <shibmd:Scope regexp="false">samltest.id</shibmd:Scope>

<!-- Display information about this IdP that can be used by SP's and discovery
services to identify the IdP meaningfully for end users --> 
            <mdui:UIInfo>
                <mdui:DisplayName xml:lang="en">SAMLtest IdP</mdui:DisplayName>
                <mdui:Description xml:lang="en">A free and basic IdP for testing SAML deployments</mdui:Description>
                <mdui:Logo height="90" width="225">https://samltest.id/saml/logo.png</mdui:Logo>
            </mdui:UIInfo>
        </Extensions>

        <KeyDescriptor use="signing">
            <ds:KeyInfo>
                    <ds:X509Data>
                        <ds:X509Certificate>
MIIDETCCAfmgAwIBAgIUZRpDhkNKl5eWtJqk0Bu1BgTTargwDQYJKoZIhvcNAQEL
BQAwFjEUMBIGA1UEAwwLc2FtbHRlc3QuaWQwHhcNMTgwODI0MjExNDEwWhcNMzgw
ODI0MjExNDEwWjAWMRQwEgYDVQQDDAtzYW1sdGVzdC5pZDCCASIwDQYJKoZIhvcN
AQEBBQADggEPADCCAQoCggEBAJrh9/PcDsiv3UeL8Iv9rf4WfLPxuOm9W6aCntEA
8l6c1LQ1Zyrz+Xa/40ZgP29ENf3oKKbPCzDcc6zooHMji2fBmgXp6Li3fQUzu7yd
+nIC2teejijVtrNLjn1WUTwmqjLtuzrKC/ePoZyIRjpoUxyEMJopAd4dJmAcCq/K
k2eYX9GYRlqvIjLFoGNgy2R4dWwAKwljyh6pdnPUgyO/WjRDrqUBRFrLQJorR2kD
c4seZUbmpZZfp4MjmWMDgyGM1ZnR0XvNLtYeWAyt0KkSvFoOMjZUeVK/4xR74F8e
8ToPqLmZEg9ZUx+4z2KjVK00LpdRkH9Uxhh03RQ0FabHW6UCAwEAAaNXMFUwHQYD
VR0OBBYEFJDbe6uSmYQScxpVJhmt7PsCG4IeMDQGA1UdEQQtMCuCC3NhbWx0ZXN0
LmlkhhxodHRwczovL3NhbWx0ZXN0LmlkL3NhbWwvaWRwMA0GCSqGSIb3DQEBCwUA
A4IBAQBNcF3zkw/g51q26uxgyuy4gQwnSr01Mhvix3Dj/Gak4tc4XwvxUdLQq+jC
cxr2Pie96klWhY/v/JiHDU2FJo9/VWxmc/YOk83whvNd7mWaNMUsX3xGv6AlZtCO
L3JhCpHjiN+kBcMgS5jrtGgV1Lz3/1zpGxykdvS0B4sPnFOcaCwHe2B9SOCWbDAN
JXpTjz1DmJO4ImyWPJpN1xsYKtm67Pefxmn0ax0uE2uuzq25h0xbTkqIQgJzyoE/
DPkBFK1vDkMfAW11dQ0BXatEnW7Gtkc0lh2/PIbHWj4AzxYMyBf5Gy6HSVOftwjC
voQR2qr2xJBixsg+MIORKtmKHLfU
                        </ds:X509Certificate>
                    </ds:X509Data>
            </ds:KeyInfo>

        </KeyDescriptor>
        <KeyDescriptor use="signing">
            <ds:KeyInfo>
                    <ds:X509Data>
                        <ds:X509Certificate>
MIIDEjCCAfqgAwIBAgIVAMECQ1tjghafm5OxWDh9hwZfxthWMA0GCSqGSIb3DQEB
CwUAMBYxFDASBgNVBAMMC3NhbWx0ZXN0LmlkMB4XDTE4MDgyNDIxMTQwOVoXDTM4
MDgyNDIxMTQwOVowFjEUMBIGA1UEAwwLc2FtbHRlc3QuaWQwggEiMA0GCSqGSIb3
DQEBAQUAA4IBDwAwggEKAoIBAQC0Z4QX1NFKs71ufbQwoQoW7qkNAJRIANGA4iM0
ThYghul3pC+FwrGv37aTxWXfA1UG9njKbbDreiDAZKngCgyjxj0uJ4lArgkr4AOE
jj5zXA81uGHARfUBctvQcsZpBIxDOvUUImAl+3NqLgMGF2fktxMG7kX3GEVNc1kl
bN3dfYsaw5dUrw25DheL9np7G/+28GwHPvLb4aptOiONbCaVvh9UMHEA9F7c0zfF
/cL5fOpdVa54wTI0u12CsFKt78h6lEGG5jUs/qX9clZncJM7EFkN3imPPy+0HC8n
spXiH/MZW8o2cqWRkrw3MzBZW3Ojk5nQj40V6NUbjb7kfejzAgMBAAGjVzBVMB0G
A1UdDgQWBBQT6Y9J3Tw/hOGc8PNV7JEE4k2ZNTA0BgNVHREELTArggtzYW1sdGVz
dC5pZIYcaHR0cHM6Ly9zYW1sdGVzdC5pZC9zYW1sL2lkcDANBgkqhkiG9w0BAQsF
AAOCAQEASk3guKfTkVhEaIVvxEPNR2w3vWt3fwmwJCccW98XXLWgNbu3YaMb2RSn
7Th4p3h+mfyk2don6au7Uyzc1Jd39RNv80TG5iQoxfCgphy1FYmmdaSfO8wvDtHT
TNiLArAxOYtzfYbzb5QrNNH/gQEN8RJaEf/g/1GTw9x/103dSMK0RXtl+fRs2nbl
D1JJKSQ3AdhxK/weP3aUPtLxVVJ9wMOQOfcy02l+hHMb6uAjsPOpOVKqi3M8XmcU
ZOpx4swtgGdeoSpeRyrtMvRwdcciNBp9UZome44qZAYH1iqrpmmjsfI9pJItsgWu
3kXPjhSfj1AJGR1l9JGvJrHki1iHTA==
                        </ds:X509Certificate>
                    </ds:X509Data>
            </ds:KeyInfo>

        </KeyDescriptor>
        <KeyDescriptor use="encryption">
            <ds:KeyInfo>
                    <ds:X509Data>
                        <ds:X509Certificate>
MIIDEjCCAfqgAwIBAgIVAPVbodo8Su7/BaHXUHykx0Pi5CFaMA0GCSqGSIb3DQEB
CwUAMBYxFDASBgNVBAMMC3NhbWx0ZXN0LmlkMB4XDTE4MDgyNDIxMTQwOVoXDTM4
MDgyNDIxMTQwOVowFjEUMBIGA1UEAwwLc2FtbHRlc3QuaWQwggEiMA0GCSqGSIb3
DQEBAQUAA4IBDwAwggEKAoIBAQCQb+1a7uDdTTBBFfwOUun3IQ9nEuKM98SmJDWa
MwM877elswKUTIBVh5gB2RIXAPZt7J/KGqypmgw9UNXFnoslpeZbA9fcAqqu28Z4
sSb2YSajV1ZgEYPUKvXwQEmLWN6aDhkn8HnEZNrmeXihTFdyr7wjsLj0JpQ+VUlc
4/J+hNuU7rGYZ1rKY8AA34qDVd4DiJ+DXW2PESfOu8lJSOteEaNtbmnvH8KlwkDs
1NvPTsI0W/m4SK0UdXo6LLaV8saIpJfnkVC/FwpBolBrRC/Em64UlBsRZm2T89ca
uzDee2yPUvbBd5kLErw+sC7i4xXa2rGmsQLYcBPhsRwnmBmlAgMBAAGjVzBVMB0G
A1UdDgQWBBRZ3exEu6rCwRe5C7f5QrPcAKRPUjA0BgNVHREELTArggtzYW1sdGVz
dC5pZIYcaHR0cHM6Ly9zYW1sdGVzdC5pZC9zYW1sL2lkcDANBgkqhkiG9w0BAQsF
AAOCAQEABZDFRNtcbvIRmblnZItoWCFhVUlq81ceSQddLYs8DqK340//hWNAbYdj
WcP85HhIZnrw6NGCO4bUipxZXhiqTA/A9d1BUll0vYB8qckYDEdPDduYCOYemKkD
dmnHMQWs9Y6zWiYuNKEJ9mf3+1N8knN/PK0TYVjVjXAf2CnOETDbLtlj6Nqb8La3
sQkYmU+aUdopbjd5JFFwbZRaj6KiHXHtnIRgu8sUXNPrgipUgZUOVhP0C0N5OfE4
JW8ZBrKgQC/6vJ2rSa9TlzI6JAa5Ww7gMXMP9M+cJUNQklcq+SBnTK8G+uBHgPKR
zBDsMIEzRtQZm4GIoHJae4zmnCekkQ==
                        </ds:X509Certificate>
                    </ds:X509Data>
            </ds:KeyInfo>

        </KeyDescriptor>

<!-- An endpoint for artifact resolution.  Please see Wikipedia for more details about SAML
     artifacts and when you may find them useful. -->

        <ArtifactResolutionService Binding="urn:oasis:names:tc:SAML:2.0:bindings:SOAP" Location="https://samltest.id/idp/profile/SAML2/SOAP/ArtifactResolution" index="1" />

<!-- A set of endpoints where the IdP can receive logout messages. These must match the public
facing addresses if this IdP is hosted behind a reverse proxy.  --> 
        <SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://samltest.id/idp/profile/SAML2/Redirect/SLO"/>
        <SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="https://samltest.id/idp/profile/SAML2/POST/SLO"/>
        <SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST-SimpleSign" Location="https://samltest.id/idp/profile/SAML2/POST-SimpleSign/SLO"/>

<!-- A set of endpoints the SP can send AuthnRequests to in order to trigger user authentication. -->
        <SingleSignOnService Binding="urn:mace:shibboleth:1.0:profiles:AuthnRequest" Location="https://samltest.id/idp/profile/Shibboleth/SSO"/>
        <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="https://samltest.id/idp/profile/SAML2/POST/SSO"/>
        <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST-SimpleSign" Location="https://samltest.id/idp/profile/SAML2/POST-SimpleSign/SSO"/>
        <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://samltest.id/idp/profile/SAML2/Redirect/SSO"/>
        <SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:SOAP" Location="https://samltest.id/idp/profile/SAML2/SOAP/ECP"/>

    </IDPSSODescriptor>

</EntityDescriptor>`

	hwSpMetadata = `<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" ID="M349b8f2417d474387360e922ef39baa" entityID="https://auth.huaweicloud.com/">
<ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
<ds:SignedInfo>
<ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
<ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"/>
<ds:Reference URI="#M349b8f2417d474387360e922ef39baa">
<ds:Transforms>
<ds:Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"/>
<ds:Transform Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#"/>
</ds:Transforms>
<ds:DigestMethod Algorithm="http://www.w3.org/2000/09/xmldsig#sha1"/>
<ds:DigestValue>diBMwyuN633Q/kBf0M+SQZ4fNCI=</ds:DigestValue>
</ds:Reference>
</ds:SignedInfo>
<ds:SignatureValue>
k6+6QX8T5+S5wEIWlxwvd5a48xLWmp4Jd0bcs33vNaQVQ93YiztGcroUUqnLYK8uqAsmk2ZP4NtKFL/MyqoznrStglSi7uycwX6X3fswMwvkHJ9UVhH6Gp2Txl2JQ9/0vXy/tlXjQNFbfqzLhqxKC/K/PmKd684XkPYXlkiEJa6nGibu0gse9rvP2hE0G9bDMfdiyQnanfBvN+oNRte3xyI3DWh2P/jDTAPJMYzGM76JIneS8jLPa4gKDz5KcumG/8JV7GUWTDTZZ4ftujsjEuUFdPzKqwMYogMXOnTt8wbisSF8ZlLSMH/TIj6xAO9aSkEAo8HtlHJbekYlVA5Tz5IamLLzfaPpf7+NghLSuATIwY8/pvBZ8qhY8PVOjzRCeoEZJUrlFOcZ2CvO9zVKYdkEa1JC3mUW7CgeQc7G2/9niub7Vu00eyp9AAd9nkPfLoiWam8/yg2TBPRRJ9VKsv2UMFuITWrcFJayjezlTzY3dI3tI8lfoIMEVRaEP1v1D8XbxnxiaiKZXsGQHtpTSLc1ZL441jeZDLa661raUJDlGA6mBc1QukJklEocGg+Q+FeU33MJCaN/rIZCGvrjIZNng3yKFw1+R7/CqeJJlFWw0hPJQGiy6wfrDYSmhwuXJ2vQn3dTHjlT3kWsniiZtuOUi2TjHbq8gVHqlxmPxSs=
</ds:SignatureValue>
<ds:KeyInfo>
<ds:X509Data>
<ds:X509Certificate>
MIIF6TCCA9GgAwIBAgIEPHSCijANBgkqhkiG9w0BAQsFADCBpDELMAkGA1UEBhMCQ04xEjAQBgNV BAgTCUd1YW5nRG9uZzERMA8GA1UEBxMIU2hlblpoZW4xJTAjBgNVBAoTHEh1YXdlaSBUZWNobm9s b2dpZXMgQ28uLCBMdGQxKDAmBgNVBAsTH1NlcnZpY2UgUHJvdmlkZXIgT3BlcmF0aW9uIERlcHQx HTAbBgNVBAMTFGF1dGguaHVhd2VpY2xvdWQuY29tMB4XDTE4MDYyMTEzMjUwMFoXDTI4MDYxODEz MjUwMFowgaQxCzAJBgNVBAYTAkNOMRIwEAYDVQQIEwlHdWFuZ0RvbmcxETAPBgNVBAcTCFNoZW5a aGVuMSUwIwYDVQQKExxIdWF3ZWkgVGVjaG5vbG9naWVzIENvLiwgTHRkMSgwJgYDVQQLEx9TZXJ2 aWNlIFByb3ZpZGVyIE9wZXJhdGlvbiBEZXB0MR0wGwYDVQQDExRhdXRoLmh1YXdlaWNsb3VkLmNv bTCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIBAJisHObeLOPs2mJ4eJBcruMZl9FjvRQe DmofxgOcIcmybt6qlDqAv7275JMMQfQFcEoxH3GqcCTYqvSaSnHPaJB1xljPIKAWtd7p1ymevcSy F2HdCZ8gPJKg3Q6+ZjwTipKS/nZr7xmpUQ0WvwRkgZ7zfslAW+y5PCgkDrgmEjG92rLArj8iPhmu jajXaTPQKVHuZMxBzS735uo4yjVwYE3+0mE4HTZjK+6n6Ffu+JhLzhcKGulmrT/6qHisMbIXAZyE egBDavomb+5zu/CUQwii5IAPrRTwwegYpG4+uYJ2cHfrUdHqw9lSCSbQzu1yW1AS4zB16sjoHZdV rxYyktlswNmJ1/MyRH5bO90e2kvVwV4l34Hi5HEFvFFjL8TAsbN4mGvA9fgohXp30x97UdPVV8Ji NNAKZjdZSEdG9xqrTRfe4+LQg/hzLNSwsko3nDnH8qhCgtb8qIipQ3s7niCa53AQWYR82lEViols /dbWU9qYeldVvGNAgJSqHLB7qLwcQW78+2V1446KhQqzqPeLI4ANGaLFKw8fGzgh85RKOjrIetb4 wAOZmhrrUJrRg47DYQQjNv3glDg53ijLPFunzRqUoqLphrZ1XpEA4y21OtTP6OMYAM0lSOj1gjvb ubTDo8XOQs5YGtTOyHn4CQ4GR8NNo7UrwVEmoZHA+AKbAgMBAAGjITAfMB0GA1UdDgQWBBTt05QW 9dyXi7eMekKOH0bn4xKrkDANBgkqhkiG9w0BAQsFAAOCAgEACv9zgzUgxxQ8t9ldOXmirxzSOrHx MCL8SKsu+c+Y4hoHma5LFjylv6x76NWTAFSE6GgqfuNI/gPj/2AWqObAvsHd8lsPjJ96ZoSaTmS8 NrtU6HuT1Lc+CmVfeGd3/G+KspQECjg2JeBrfyEw9B8KUAQV20DQukGfAHtKKQPOZmKm0Qm3ExWC eXz1TR2KP+Lrhny/yG43g4iVUKq65HFHs5cRzRk0iR0/NLpggl5+Op0rxMxBn+bCrnJBi0n9/PIM fWNhkEBl+B++EifPUQxQaOEsnxTgFo1O4ksK9hDFcLbr+1qCDgVIMkyC1xMBBikCgLvIdzy3SXBN ndIEUq+QgOORxLlq1WqrfLFO22TxZm8XaB5g36UMk5PGoVHGnjALReAHjC0C5sIiKMJSgQOPd71X mQSsw3G9NsMKf/H3xJXkq/b672ls/l1JBslm52DAk2k5UlLkf/1p4I7WHOfm5ZNpDjj1rTP6SiAc tWLtqXIU28fLa2sA+zHXA5acDGOm6eIrMme5HpsV/KoUOW1MXGugK59zofeueCFDGRfbyoS2lj0S W+CbJVa72CLf3xPh2nWH0cK9de+wyCx8uI0KGPyV4I9/XBLHhvkb3XPaUfnkzYkcrG/39cOaxuPF z+haXwI1lvI964zvmTgwdDjdf/0asA09S7EEK2KyzXUREM4=
</ds:X509Certificate>
</ds:X509Data>
</ds:KeyInfo>
</ds:Signature>
<md:SPSSODescriptor AuthnRequestsSigned="true" WantAssertionsSigned="true" protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
<md:KeyDescriptor use="signing">
<ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
<ds:X509Data>
<ds:X509Certificate>
MIIF6TCCA9GgAwIBAgIEPHSCijANBgkqhkiG9w0BAQsFADCBpDELMAkGA1UEBhMCQ04xEjAQBgNV BAgTCUd1YW5nRG9uZzERMA8GA1UEBxMIU2hlblpoZW4xJTAjBgNVBAoTHEh1YXdlaSBUZWNobm9s b2dpZXMgQ28uLCBMdGQxKDAmBgNVBAsTH1NlcnZpY2UgUHJvdmlkZXIgT3BlcmF0aW9uIERlcHQx HTAbBgNVBAMTFGF1dGguaHVhd2VpY2xvdWQuY29tMB4XDTE4MDYyMTEzMjUwMFoXDTI4MDYxODEz MjUwMFowgaQxCzAJBgNVBAYTAkNOMRIwEAYDVQQIEwlHdWFuZ0RvbmcxETAPBgNVBAcTCFNoZW5a aGVuMSUwIwYDVQQKExxIdWF3ZWkgVGVjaG5vbG9naWVzIENvLiwgTHRkMSgwJgYDVQQLEx9TZXJ2 aWNlIFByb3ZpZGVyIE9wZXJhdGlvbiBEZXB0MR0wGwYDVQQDExRhdXRoLmh1YXdlaWNsb3VkLmNv bTCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIBAJisHObeLOPs2mJ4eJBcruMZl9FjvRQe DmofxgOcIcmybt6qlDqAv7275JMMQfQFcEoxH3GqcCTYqvSaSnHPaJB1xljPIKAWtd7p1ymevcSy F2HdCZ8gPJKg3Q6+ZjwTipKS/nZr7xmpUQ0WvwRkgZ7zfslAW+y5PCgkDrgmEjG92rLArj8iPhmu jajXaTPQKVHuZMxBzS735uo4yjVwYE3+0mE4HTZjK+6n6Ffu+JhLzhcKGulmrT/6qHisMbIXAZyE egBDavomb+5zu/CUQwii5IAPrRTwwegYpG4+uYJ2cHfrUdHqw9lSCSbQzu1yW1AS4zB16sjoHZdV rxYyktlswNmJ1/MyRH5bO90e2kvVwV4l34Hi5HEFvFFjL8TAsbN4mGvA9fgohXp30x97UdPVV8Ji NNAKZjdZSEdG9xqrTRfe4+LQg/hzLNSwsko3nDnH8qhCgtb8qIipQ3s7niCa53AQWYR82lEViols /dbWU9qYeldVvGNAgJSqHLB7qLwcQW78+2V1446KhQqzqPeLI4ANGaLFKw8fGzgh85RKOjrIetb4 wAOZmhrrUJrRg47DYQQjNv3glDg53ijLPFunzRqUoqLphrZ1XpEA4y21OtTP6OMYAM0lSOj1gjvb ubTDo8XOQs5YGtTOyHn4CQ4GR8NNo7UrwVEmoZHA+AKbAgMBAAGjITAfMB0GA1UdDgQWBBTt05QW 9dyXi7eMekKOH0bn4xKrkDANBgkqhkiG9w0BAQsFAAOCAgEACv9zgzUgxxQ8t9ldOXmirxzSOrHx MCL8SKsu+c+Y4hoHma5LFjylv6x76NWTAFSE6GgqfuNI/gPj/2AWqObAvsHd8lsPjJ96ZoSaTmS8 NrtU6HuT1Lc+CmVfeGd3/G+KspQECjg2JeBrfyEw9B8KUAQV20DQukGfAHtKKQPOZmKm0Qm3ExWC eXz1TR2KP+Lrhny/yG43g4iVUKq65HFHs5cRzRk0iR0/NLpggl5+Op0rxMxBn+bCrnJBi0n9/PIM fWNhkEBl+B++EifPUQxQaOEsnxTgFo1O4ksK9hDFcLbr+1qCDgVIMkyC1xMBBikCgLvIdzy3SXBN ndIEUq+QgOORxLlq1WqrfLFO22TxZm8XaB5g36UMk5PGoVHGnjALReAHjC0C5sIiKMJSgQOPd71X mQSsw3G9NsMKf/H3xJXkq/b672ls/l1JBslm52DAk2k5UlLkf/1p4I7WHOfm5ZNpDjj1rTP6SiAc tWLtqXIU28fLa2sA+zHXA5acDGOm6eIrMme5HpsV/KoUOW1MXGugK59zofeueCFDGRfbyoS2lj0S W+CbJVa72CLf3xPh2nWH0cK9de+wyCx8uI0KGPyV4I9/XBLHhvkb3XPaUfnkzYkcrG/39cOaxuPF z+haXwI1lvI964zvmTgwdDjdf/0asA09S7EEK2KyzXUREM4=
</ds:X509Certificate>
</ds:X509Data>
</ds:KeyInfo>
</md:KeyDescriptor>
<md:KeyDescriptor use="encryption">
<ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
<ds:X509Data>
<ds:X509Certificate>
MIIF6TCCA9GgAwIBAgIEPHSCijANBgkqhkiG9w0BAQsFADCBpDELMAkGA1UEBhMCQ04xEjAQBgNV BAgTCUd1YW5nRG9uZzERMA8GA1UEBxMIU2hlblpoZW4xJTAjBgNVBAoTHEh1YXdlaSBUZWNobm9s b2dpZXMgQ28uLCBMdGQxKDAmBgNVBAsTH1NlcnZpY2UgUHJvdmlkZXIgT3BlcmF0aW9uIERlcHQx HTAbBgNVBAMTFGF1dGguaHVhd2VpY2xvdWQuY29tMB4XDTE4MDYyMTEzMjUwMFoXDTI4MDYxODEz MjUwMFowgaQxCzAJBgNVBAYTAkNOMRIwEAYDVQQIEwlHdWFuZ0RvbmcxETAPBgNVBAcTCFNoZW5a aGVuMSUwIwYDVQQKExxIdWF3ZWkgVGVjaG5vbG9naWVzIENvLiwgTHRkMSgwJgYDVQQLEx9TZXJ2 aWNlIFByb3ZpZGVyIE9wZXJhdGlvbiBEZXB0MR0wGwYDVQQDExRhdXRoLmh1YXdlaWNsb3VkLmNv bTCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIBAJisHObeLOPs2mJ4eJBcruMZl9FjvRQe DmofxgOcIcmybt6qlDqAv7275JMMQfQFcEoxH3GqcCTYqvSaSnHPaJB1xljPIKAWtd7p1ymevcSy F2HdCZ8gPJKg3Q6+ZjwTipKS/nZr7xmpUQ0WvwRkgZ7zfslAW+y5PCgkDrgmEjG92rLArj8iPhmu jajXaTPQKVHuZMxBzS735uo4yjVwYE3+0mE4HTZjK+6n6Ffu+JhLzhcKGulmrT/6qHisMbIXAZyE egBDavomb+5zu/CUQwii5IAPrRTwwegYpG4+uYJ2cHfrUdHqw9lSCSbQzu1yW1AS4zB16sjoHZdV rxYyktlswNmJ1/MyRH5bO90e2kvVwV4l34Hi5HEFvFFjL8TAsbN4mGvA9fgohXp30x97UdPVV8Ji NNAKZjdZSEdG9xqrTRfe4+LQg/hzLNSwsko3nDnH8qhCgtb8qIipQ3s7niCa53AQWYR82lEViols /dbWU9qYeldVvGNAgJSqHLB7qLwcQW78+2V1446KhQqzqPeLI4ANGaLFKw8fGzgh85RKOjrIetb4 wAOZmhrrUJrRg47DYQQjNv3glDg53ijLPFunzRqUoqLphrZ1XpEA4y21OtTP6OMYAM0lSOj1gjvb ubTDo8XOQs5YGtTOyHn4CQ4GR8NNo7UrwVEmoZHA+AKbAgMBAAGjITAfMB0GA1UdDgQWBBTt05QW 9dyXi7eMekKOH0bn4xKrkDANBgkqhkiG9w0BAQsFAAOCAgEACv9zgzUgxxQ8t9ldOXmirxzSOrHx MCL8SKsu+c+Y4hoHma5LFjylv6x76NWTAFSE6GgqfuNI/gPj/2AWqObAvsHd8lsPjJ96ZoSaTmS8 NrtU6HuT1Lc+CmVfeGd3/G+KspQECjg2JeBrfyEw9B8KUAQV20DQukGfAHtKKQPOZmKm0Qm3ExWC eXz1TR2KP+Lrhny/yG43g4iVUKq65HFHs5cRzRk0iR0/NLpggl5+Op0rxMxBn+bCrnJBi0n9/PIM fWNhkEBl+B++EifPUQxQaOEsnxTgFo1O4ksK9hDFcLbr+1qCDgVIMkyC1xMBBikCgLvIdzy3SXBN ndIEUq+QgOORxLlq1WqrfLFO22TxZm8XaB5g36UMk5PGoVHGnjALReAHjC0C5sIiKMJSgQOPd71X mQSsw3G9NsMKf/H3xJXkq/b672ls/l1JBslm52DAk2k5UlLkf/1p4I7WHOfm5ZNpDjj1rTP6SiAc tWLtqXIU28fLa2sA+zHXA5acDGOm6eIrMme5HpsV/KoUOW1MXGugK59zofeueCFDGRfbyoS2lj0S W+CbJVa72CLf3xPh2nWH0cK9de+wyCx8uI0KGPyV4I9/XBLHhvkb3XPaUfnkzYkcrG/39cOaxuPF z+haXwI1lvI964zvmTgwdDjdf/0asA09S7EEK2KyzXUREM4=
</ds:X509Certificate>
</ds:X509Data>
</ds:KeyInfo>
</md:KeyDescriptor>
<md:ArtifactResolutionService xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Artifact" Location="https://auth.huaweicloud.com/authui/saml/SAMLAssertionConsumer" index="0" isDefault="true"/>
<md:SingleLogoutService xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://auth.huaweicloud.com/authui/saml/LogoutServiceHTTPRedirect" ResponseLocation="https://auth.huaweicloud.com/authui/saml/LogoutServiceHTTPRedirectResponse"/>
<md:SingleLogoutService xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="https://auth.huaweicloud.com/authui/saml/LogoutServiceHTTPPost" ResponseLocation="https://auth.huaweicloud.com/authui/saml/LogoutServiceHTTPPostResponse"/>
<md:ManageNameIDService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="https://auth.huaweicloud.com/authui/saml/ManageNameIDServiceHTTPPost" ResponseLocation="https://auth.huaweicloud.com/authui/saml/ManageNameIDServiceHTTPPostResponse"/>
<md:ManageNameIDService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://auth.huaweicloud.com/authui/saml/ManageNameIDServiceHTTPRedirect" ResponseLocation="https://auth.huaweicloud.com/authui/saml/ManageNameIDServiceHTTPRedirectResponse"/>
<md:ManageNameIDService xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" Binding="urn:oasis:names:tc:SAML:2.0:bindings:SOAP" Location="https://auth.huaweicloud.com/authui/saml/ManageNameIDServiceSOAP"/>
<md:NameIDFormat xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata">
urn:oasis:names:tc:SAML:2.0:nameid-format:transient
</md:NameIDFormat>
<md:AssertionConsumerService xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="https://auth.huaweicloud.com/authui/saml/SAMLAssertionConsumer" index="0" isDefault="true"/>
</md:SPSSODescriptor>
</md:EntityDescriptor>`
	hwIdpMetadata = `<md:EntityDescriptor
    xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata"
    xmlns:ds="http://www.w3.org/2000/09/xmldsig#" entityID="https://www.test.com">
    <md:IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
        <md:KeyDescriptor use="signing">
            <ds:KeyInfo
                xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
                <ds:X509Data>
                    <ds:X509Certificate>
MIICsDCCAhmgAwIBAgIJAKNbH+B0Vm9HMA0GCSqGSIb3DQEBBQUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIEwpTb21lLVN0YXRlMSEwHwYDVQQKExhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMTgxMDMwMDIxMzA4WhcNMzMxMDMxMDIxMzA4WjBF
MQswCQYDVQQGEwJBVTETMBEGA1UECBMKU29tZS1TdGF0ZTEhMB8GA1UEChMYSW50
ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKB
gQDIIZtsLpqLDpXB1LI8tbtwoeOyJbM2PIxJTOqRm1ZM0r7rpvt4kFCgAd68gAsl
YAEeSqUawxV3FUgt62DLMOT2auwBcpywVW7L/ZF4IUziwuFQLWdw5NIGMP5lpt1M
HSel8k4paokoXAwZ2B+Vtku+kDTGLc3cp1T5/ClYE/ofdQIDAQABo4GnMIGkMB0G
A1UdDgQWBBRVZlu4B6TzuNHasJz5tHoMilKLdjB1BgNVHSMEbjBsgBRVZlu4B6Tz
uNHasJz5tHoMilKLdqFJpEcwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgTClNvbWUt
U3RhdGUxITAfBgNVBAoTGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZIIJAKNbH+B0
Vm9HMAwGA1UdEwQFMAMBAf8wDQYJKoZIhvcNAQEFBQADgYEAhyVdBqW4r94XdwMy
LK42mwqNnHy4WjM8eq9X5FhBckZX+TyM909iH2AsMjpkv8BDIxTiX6tpmNyYhOCp
vCPMmQHl9450maIA7At//sEgL94FNRJbTYkme7F3xI90X0htMr23Yan31lRwdj53
DgagnkMlzQ8QccUXrdQgzXzKb0w=</ds:X509Certificate>
                </ds:X509Data>
            </ds:KeyInfo>
        </md:KeyDescriptor>
        <md:KeyDescriptor use="encryption">
            <ds:KeyInfo
                xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
                <ds:X509Data>
                    <ds:X509Certificate>
MIICsDCCAhmgAwIBAgIJAKNbH+B0Vm9HMA0GCSqGSIb3DQEBBQUAMEUxCzAJBgNV
BAYTAkFVMRMwEQYDVQQIEwpTb21lLVN0YXRlMSEwHwYDVQQKExhJbnRlcm5ldCBX
aWRnaXRzIFB0eSBMdGQwHhcNMTgxMDMwMDIxMzA4WhcNMzMxMDMxMDIxMzA4WjBF
MQswCQYDVQQGEwJBVTETMBEGA1UECBMKU29tZS1TdGF0ZTEhMB8GA1UEChMYSW50
ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKB
gQDIIZtsLpqLDpXB1LI8tbtwoeOyJbM2PIxJTOqRm1ZM0r7rpvt4kFCgAd68gAsl
YAEeSqUawxV3FUgt62DLMOT2auwBcpywVW7L/ZF4IUziwuFQLWdw5NIGMP5lpt1M
HSel8k4paokoXAwZ2B+Vtku+kDTGLc3cp1T5/ClYE/ofdQIDAQABo4GnMIGkMB0G
A1UdDgQWBBRVZlu4B6TzuNHasJz5tHoMilKLdjB1BgNVHSMEbjBsgBRVZlu4B6Tz
uNHasJz5tHoMilKLdqFJpEcwRTELMAkGA1UEBhMCQVUxEzARBgNVBAgTClNvbWUt
U3RhdGUxITAfBgNVBAoTGEludGVybmV0IFdpZGdpdHMgUHR5IEx0ZIIJAKNbH+B0
Vm9HMAwGA1UdEwQFMAMBAf8wDQYJKoZIhvcNAQEFBQADgYEAhyVdBqW4r94XdwMy
LK42mwqNnHy4WjM8eq9X5FhBckZX+TyM909iH2AsMjpkv8BDIxTiX6tpmNyYhOCp
vCPMmQHl9450maIA7At//sEgL94FNRJbTYkme7F3xI90X0htMr23Yan31lRwdj53
DgagnkMlzQ8QccUXrdQgzXzKb0w=
                    </ds:X509Certificate>
                </ds:X509Data>
            </ds:KeyInfo>
        </md:KeyDescriptor>
        <md:SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://www.test.com/saml/logout"/>
        <md:NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:transient</md:NameIDFormat>
        <md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="https://www.test.com/saml/login"/>
    </md:IDPSSODescriptor>
</md:EntityDescriptor>`
	aliyunSpMeta = `<?xml version="1.0"?>
<EntityDescriptor xmlns="urn:oasis:names:tc:SAML:2.0:metadata" xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion" xmlns:ds="http://www.w3.org/2000/09/xmldsig#" entityID="urn:alibaba:cloudcomputing">
    <SPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol" WantAssertionsSigned="true">
        <KeyDescriptor use="signing">
            <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
                <ds:X509Data>
                    <ds:X509Certificate>MIIDXjCCAkagAwIBAgIEXHToLjANBgkqhkiG9w0BAQsFADBwMQswCQYDVQQGEwJDTjERMA8GA1UE
                        CBMISGFuZ3pob3UxKTAnBgNVBAsTIEFsaWJhYmEgQ2xvdWQgQ29tcHV0aW5nIENvLiBMdGQuMSMw
                        IQYDVQQDExp1cm46YWxpYmFiYTpjbG91ZGNvbXB1dGluZzAgFw0xOTAyMjYwNzE4MDZaGA8yMTE5
                        MDIwMjA3MTgwNlowcDELMAkGA1UEBhMCQ04xETAPBgNVBAgTCEhhbmd6aG91MSkwJwYDVQQLEyBB
                        bGliYWJhIENsb3VkIENvbXB1dGluZyBDby4gTHRkLjEjMCEGA1UEAxMadXJuOmFsaWJhYmE6Y2xv
                        dWRjb21wdXRpbmcwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQDDjD53ZENEHoYNXAOf
                        OIbVBJhj7SCWKmdjnbnxq8WAFWEeZtS6hZLPpWh1z7b0NjJkvCf5oFVqBJYbbW5kEKV+9CpV6VHZ
                        qOXmsIRlkvZB+Wnc3SduwiiiUR9JojSPxVQvSf4WLT+HDASlrBztuRV2vHj9utLbvy+6bgVBqF8g
                        emL9Pcif1robDH8HlqUcADXLAt18E4MbToldVoHjpFc6fAKUXujWH5feAL8g0CKlmf/JVlHLEtu4
                        vKPxBQ8sgkysk6EnrjXl6Q4a4t+vbPG5uczA1ouTkDupMCRlaWHIHaJL/AoDGabn8sVXdaVJUKC5
                        54FNkRznBhRQll+Nuc2rAgMBAAEwDQYJKoZIhvcNAQELBQADggEBAE8k4S4HvOglthJwF3aMQXGi
                        LKW6Becs9SljA0/5VtZQDrDf2By/1BIMvWfZ/dFnO+MylDLVdS6XWvWat/DW0fOGxU4s1WNfshX7
                        DJDGR2G1XgtGoDZEYIDahUp5katAPypCkY57fGZlI0d3nq46/2qT/Zpne+pFE3DI/x8klZMniniw
                        YjNXbG96y/M4DYi1J7RR8mLIfVvz5o1SMGT4Ta2p/USE2M9F6O7/zc2j62dQgXiYa9OONo31RiXR
                        TmvGEUNuQoBZhVrFIvOnNjIfFT7Xd3CUowwJKP1floserrx4B5jScRAi9yK3x2a2lhfc+PksXfTs
                        aqIr5TQL4OorLEE=</ds:X509Certificate>
                </ds:X509Data>
            </ds:KeyInfo>
        </KeyDescriptor>
        <NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:transient</NameIDFormat>
        <NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:persistent</NameIDFormat>
        <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</NameIDFormat>
        <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified</NameIDFormat>
        <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:X509SubjectName</NameIDFormat>
        <NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:WindowsDomainQualifiedName</NameIDFormat>
        <NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:kerberos</NameIDFormat>
        <NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:entity</NameIDFormat>
        <AssertionConsumerService index="1" isDefault="true" Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="https://signin.aliyun.com/saml-role/sso"/>
        <AttributeConsumingService index="1">
            <ServiceName xml:lang="en">Alibaba Cloud Console Single Sign-On</ServiceName>
            <RequestedAttribute isRequired="true" Name="https://www.aliyun.com/SAML-Role/Attributes/Role" FriendlyName="RoleEntitlement"/>
            <RequestedAttribute isRequired="true" Name="https://www.aliyun.com/SAML-Role/Attributes/RoleSessionName" FriendlyName="RoleSessionName"/>
            <RequestedAttribute isRequired="false" Name="https://www.aliyun.com/SAML-Role/Attributes/SessionDuration" FriendlyName="SessionDuration"/>
        </AttributeConsumingService>
    </SPSSODescriptor>
    <Organization>
        <OrganizationName xml:lang="en">Alibaba Cloud Computing Co. Ltd.</OrganizationName>
        <OrganizationDisplayName xml:lang="en">AlibabaCloud</OrganizationDisplayName>
        <OrganizationURL xml:lang="en">https://www.aliyun.com</OrganizationURL>
    </Organization>
</EntityDescriptor>`

	aliyunUserMeta = `<?xml version="1.0" encoding="UTF-8"?>
<md:EntityDescriptor ID="https___signin.aliyun.com_1661281931531610_saml_SSO" entityID="https://signin.aliyun.com/1661281931531610/saml/SSO" xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata"><md:SPSSODescriptor WantAssertionsSigned="true" protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol"><md:KeyDescriptor use="signing"><ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#"><ds:X509Data><ds:X509Certificate>MIIDUTCCAjmgAwIBAgIEIv2v9DANBgkqhkiG9w0BAQsFADBZMQswCQYDVQQGEwJDTjERMA8GA1UE
BxMISGFuZ3pob3UxFDASBgNVBAoTC0FsaWJhYmEgSW5jMQ8wDQYDVQQLEwZBcHNhcmExEDAOBgNV
BAMTB0FsaWJhYmEwHhcNMTcwMzE0MTc1OTE5WhcNMjcwMzEyMTc1OTE5WjBZMQswCQYDVQQGEwJD
TjERMA8GA1UEBxMISGFuZ3pob3UxFDASBgNVBAoTC0FsaWJhYmEgSW5jMQ8wDQYDVQQLEwZBcHNh
cmExEDAOBgNVBAMTB0FsaWJhYmEwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCqK2HR
tf4smv9pCQtPenFE1w6lvxsHiv0J/knvpC1BU4iAWcS8LxAElKb49QbKHuUxcwEGJfm0+zZpqS+J
I3jmGc4aHYACyL2WxtKNx/5EK1Qs5ugCipn7g+ySOqXxc/Rv2S7muw6LTrGVTT7vo09EUDkZM34s
TupuU7tzX0ktYhimxwskG9o7bvZuQKQf66gN8l/DUzyUl59/0wA1+x5A5B3pvaABCA6dq4mi8mtJ
fTXcqWm06+FgVNPgKo59uP6y08rQJXjKDwLIf0owuoiRrPLR5JKC1vQ6PSz0cGv8tGUts5dr/0zG
FHy4h3aufQiXCSi44WUB3FejQQfgEiBdAgMBAAGjITAfMB0GA1UdDgQWBBShWN61nZsWz9MYnSrV
kCkJnSdFtDANBgkqhkiG9w0BAQsFAAOCAQEAMMAl+C3oyI6kZNmvX05Sb0q6UAM8wqjFKbPhSSiy
srjVZwjEjiZnOSnoX8vO07fsZpcVmByHzGXWuBxxKCviCpQCS9hyOTF6bvAoXwe37h02Uhv3tKI0
7FRkXJA7HeB0HEuHPCBxxWVWJfgtkeUETnGV06CrUlGON7Du3h37EUzfTqmKhlsqKeK8uqw3gLYq
Bp6ULrP1PbNo2AaHMYaZhFL1dSUtNYvekZppregZKMIDqtEm6Pwpw2lj8gjTC40PQ0GuXEeTsfE5
dhw42xc9RkyUg1Go04k9Z/UMxTX0KVMiRZ9DF2FWjWp1AAQJ3TvZ2Ao/XOhmk4GWRehUoHr7Hw==</ds:X509Certificate></ds:X509Data></ds:KeyInfo></md:KeyDescriptor><md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress</md:NameIDFormat><md:NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:transient</md:NameIDFormat><md:NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:persistent</md:NameIDFormat><md:NameIDFormat>urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified</md:NameIDFormat><md:AssertionConsumerService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST" Location="https://signin.aliyun.com/saml/SSO" index="0" isDefault="true"/></md:SPSSODescriptor></md:EntityDescriptor>`
)

func TestParseMetadata(t *testing.T) {
	cases := []struct {
		In string
	}{
		{
			In: spMetadata1,
		},
		{
			In: idpMetadata,
		},
		{
			In: hwSpMetadata,
		},
		{
			In: hwIdpMetadata,
		},
		{
			In: aliyunSpMeta,
		},
		{
			In: aliyunUserMeta,
		},
	}
	for _, c := range cases {
		ed, err := ParseMetadata([]byte(c.In))
		if err != nil {
			t.Errorf("ParseMetadata fail %s", err)
		} else {
			xmlstr, err := xml.MarshalIndent(ed, "", "  ")
			if err != nil {
				t.Errorf("xml.Marshal fail %s", err)
			} else {
				t.Logf("%s", xmlstr)
			}
		}
	}
}

var (
	resp1 = `<?xml version="1.0" encoding="UTF-8"?>
<!-- InResponseTo需要与samlRequest请求的AuthnRequest中的ID配置项保持一致 -->
<!-- Destination需要与SPMetadata中AssertionConsumerService标签下的Location的值保持一致 -->
<saml2p:Response Consent="urn:oasis:names:tc:SAML:2.0:consent:unspecified" ID="_d794dc393ae6724e236003bf0b917cf0" Destination="https://auth.huaweicloud.com/authui/saml/SAMLAssertionConsumer"
    InResponseTo="_dck4mm08qmdhc8k4nuir07hghetdqqg8umg5" IssueInstant="2018-10-30T08:21:41.740Z" Version="2.0"
    xmlns:saml2p="urn:oasis:names:tc:SAML:2.0:protocol">
  <!-- 必须与IDP Metadata.xml中的entityID保持一致 -->
  <saml2:Issuer Format="urn:oasis:names:tc:SAML:2.0:nameid-format:entity" 
    xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">https://www.test.com</saml2:Issuer>
  <saml2p:Status>
    <saml2p:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success" />
    <saml2p:StatusMessage>urn:oasis:names:tc:SAML:2.0:status:Success</saml2p:StatusMessage>
  </saml2p:Status>
  <saml2:Assertion ID="_2320c40ac7b5e857b2d0d4ea0c8758c3" IssueInstant="2018-10-30T08:21:41.740Z" Version="2.0" 
    xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion" 
    xmlns:xsd="http://www.w3.org/2001/XMLSchema">
    <!-- 必须与IDP Metadata.xml的entityID保持一致 -->
    <saml2:Issuer>https://www.test.com</saml2:Issuer>
    <ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
      <ds:SignedInfo>
        <ds:CanonicalizationMethod Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#" />
        <ds:SignatureMethod Algorithm="http://www.w3.org/2001/04/xmldsig-more#rsa-sha256" />
        <!-- URI #号后面的值必须和Assertion标签中的ID保持一致 -->
        <ds:Reference URI="#_2320c40ac7b5e857b2d0d4ea0c8758c3">
          <ds:Transforms>
            <ds:Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature" />
            <ds:Transform Algorithm="http://www.w3.org/2001/10/xml-exc-c14n#">
              <ec:InclusiveNamespaces PrefixList="xsd" 
                xmlns:ec="http://www.w3.org/2001/10/xml-exc-c14n#" />
            </ds:Transform>
          </ds:Transforms>
          <ds:DigestMethod Algorithm="http://www.w3.org/2001/04/xmlenc#sha256" />
          <!-- DigestValue的值为Assertion标签对象做的摘要，摘要算法和DigestMethod一致 -->
          <ds:DigestValue>rFxrycznfGNYOnprZIFJJou4ro0Mz65+43MIR5F0+H4=</ds:DigestValue>
        </ds:Reference>
      </ds:SignedInfo>
      <!-- 合作伙伴签名值，具体生成方法请参见下文描述 -->
      <ds:SignatureValue>
        YqTWQngAPfGqQmWa610PM7LeefqWdKuveUVINrqL67NoHJIDa2WxLwdVzoJIlJh64QiNPr6+ndmL DCMgIC5F/9ijuzhIICZcc6lHNIjy6EsPkKRjfo9oeoVAqLgG/kmVQYeHLBID0y11RNXXpAVY4nhJ 26KiIVGt7ywyKAmhichE+eW/UYAGiOI5vkfgD2gZUGV+yPkv64k7xK4yAH3mL2NaCPuw/90e4enm iUx0YuazDwM5FiRUSMpcJs0rcNmS6clWAUcCzbOx+y2vJGtTjHb7k3UsmpnTop5eYNp94+sDPEat 8FaV4SgafMEL5z54gpe8+//9yOWEvlBs1b0RYg==
      </ds:SignatureValue>
      <ds:KeyInfo>
        <ds:X509Data>
        <!-- 合作伙伴公钥证书，必须与IDP Metadata.xml的公钥证书保持一致 -->
          <ds:X509Certificate>
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAhK3L160NjP9EhBGQOC2s4r+Wc62bkRkc nUxfhiZwCwJdQCykzuLOAoATnfoEamV5W25xtSS5kFs+4OC0mYVpKcI3SWoydX+UE5Qik5UfJ8Dt G1AvSEKhSluyO9axrV5Uv089jMxBnlm/R+xND73WcZM11yIbKJEZSTCEDfh+KnFbMw108umFMden RZCrNWUJoSp/90XeG0V2Nmj7Fkq72skSifwIASLRq9KqLbmh1QwUX+AoWpHK/jRUBustMBmG1n1i
            AqpD4EBjjBOB27k1wXZ30+IoJt8IZmfSZRFoNn5VFWXNeEmZ1aQvGSvd3Tyyw2/Wr+w/8Mags69C mpeX6QIDAQAB
          </ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </ds:Signature>
    <saml2:Subject>
    <!-- NameQualifier取值必须与SP Metadata.xml的entityID保持一致 -->
      <saml2:NameID Format="urn:oasis:names:tc:SAML:2.0:nameid-format:transient" NameQualifier="https://auth.huaweicloud.com/">Some NameID value</saml2:NameID>
      <saml2:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
        <!-- InResponseTo需要与samlRequest请求的AuthnRequest中的ID配置项保持一致 -->
        <saml2:SubjectConfirmationData InResponseTo="_dck4mm08qmdhc8k4nuir07hghetdqqg8umg5" NotBefore="2018-10-28T08:21:41.740Z" NotOnOrAfter="2018-11-01T08:21:41.740Z" Recipient="https://auth.huaweicloud.com/authui/saml/SAMLAssertionConsumer" />
      </saml2:SubjectConfirmation>
    </saml2:Subject>
    <saml2:Conditions NotBefore="2018-10-28T08:21:41.740Z" NotOnOrAfter="2018-11-01T08:21:41.740Z">
      <saml2:AudienceRestriction>
      <!-- 必须与SP Metadata.xml的entityID保持一致  -->
        <saml2:Audience>https://auth.huaweicloud.com/</saml2:Audience>
      </saml2:AudienceRestriction>
    </saml2:Conditions>
    <saml2:AttributeStatement>
      <!-- <saml2:AttributeValue></saml2:AttributeValue>之间的取值请参见下表的说明  -->
      <saml2:Attribute FriendlyName="xUserId" Name="xUserId" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:uri">
        <saml2:AttributeValue xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xsd:string">*******</saml2:AttributeValue>
      </saml2:Attribute>
      <!-- xAccountId和xUserId属性的值须一致  --> 
      <saml2:Attribute FriendlyName="xAccountId" Name="xAccountId" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:uri">
        <saml2:AttributeValue xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xsd:string">********</saml2:AttributeValue>
      </saml2:Attribute>
      <saml2:Attribute FriendlyName="bpId" Name="bpId" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:uri">
        <saml2:AttributeValue xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xsd:string">******</saml2:AttributeValue>
      </saml2:Attribute>
      <saml2:Attribute FriendlyName="email" Name="email" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:uri">
        <saml2:AttributeValue xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xsd:string" />
      </saml2:Attribute>
      <saml2:Attribute FriendlyName="name" Name="name" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:uri">
        <saml2:AttributeValue xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xsd:string">******</saml2:AttributeValue>
      </saml2:Attribute>
      <saml2:Attribute FriendlyName="mobile" Name="mobile" NameFormat="urn:oasis:names:tc:SAML:2.0:attrname-format:uri">
        <saml2:AttributeValue xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:type="xsd:string">*****</saml2:AttributeValue>
       </saml2:Attribute>
    </saml2:AttributeStatement>
    <saml2:AuthnStatement AuthnInstant="2018-10-30T08:21:41.741Z">
      <!-- 必须与SP Metadata.xml的entityID保持一致 -->
      <saml2:SubjectLocality Address="https://auth.huaweicloud.com/" />
      <saml2:AuthnContext>
        <saml2:AuthnContextClassRef>urn:oasis:names:tc:SAML:2.0:ac:classes:unspecified</saml2:AuthnContextClassRef>
      </saml2:AuthnContext>
    </saml2:AuthnStatement>
  </saml2:Assertion>
</saml2p:Response>`
	resp2 = `<samlp:Response
    xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
    xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
    ID="identifier_2"
    InResponseTo="identifier_1"
    Version="2.0"
    IssueInstant="2004-12-05T09:22:05Z"
    Destination="https://sp.example.com/SAML2/SSO/POST">
    <saml:Issuer>https://idp.example.org/SAML2</saml:Issuer>
    <samlp:Status>
      <samlp:StatusCode
        Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
    </samlp:Status>
    <saml:Assertion
      xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
      ID="identifier_3"
      Version="2.0"
      IssueInstant="2004-12-05T09:22:05Z">
      <saml:Issuer>https://idp.example.org/SAML2</saml:Issuer>
      <!-- a POSTed assertion MUST be signed -->
      <ds:Signature
        xmlns:ds="http://www.w3.org/2000/09/xmldsig#">...</ds:Signature>
      <saml:Subject>
        <saml:NameID
          Format="urn:oasis:names:tc:SAML:2.0:nameid-format:transient">
          3f7b3dcf-1674-4ecd-92c8-1544f346baf8
        </saml:NameID>
        <saml:SubjectConfirmation
          Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
          <saml:SubjectConfirmationData
            InResponseTo="identifier_1"
            Recipient="https://sp.example.com/SAML2/SSO/POST"
            NotOnOrAfter="2004-12-05T09:27:05Z"/>
        </saml:SubjectConfirmation>
      </saml:Subject>
      <saml:Conditions
        NotBefore="2004-12-05T09:17:05Z"
        NotOnOrAfter="2004-12-05T09:27:05Z">
        <saml:AudienceRestriction>
          <saml:Audience>https://sp.example.com/SAML2</saml:Audience>
        </saml:AudienceRestriction>
      </saml:Conditions>
      <saml:AuthnStatement
        AuthnInstant="2004-12-05T09:22:00Z"
        SessionIndex="identifier_3">
        <saml:AuthnContext>
          <saml:AuthnContextClassRef>
            urn:oasis:names:tc:SAML:2.0:ac:classes:PasswordProtectedTransport
         </saml:AuthnContextClassRef>
        </saml:AuthnContext>
      </saml:AuthnStatement>
    </saml:Assertion>
  </samlp:Response>`
)

func TestParseResponse(t *testing.T) {
	cases := []struct {
		In string
	}{
		{
			In: resp1,
		},
		{
			In: resp2,
		},
	}
	for _, c := range cases {
		resp := Response{}
		err := xml.Unmarshal([]byte(c.In), &resp)
		if err != nil {
			t.Errorf("xml.Unmarshal %s", err)
		} else {
			xmlstr, err := xml.MarshalIndent(resp, "", "  ")
			if err != nil {
				t.Errorf("xml.Marshal fail %s", err)
			} else {
				t.Logf("%s", xmlstr)
			}
		}
	}
}
