package mcclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/pkg/util/httputils"
)

/*const (
	USER_AGENT = "yunioncloud-go/201708"
)

var (
	red    = color.New(color.FgRed, color.Bold).PrintlnFunc()
	green  = color.New(color.FgGreen, color.Bold).PrintlnFunc()
	yellow = color.New(color.FgYellow, color.Bold).PrintlnFunc()
	cyan   = color.New(color.FgHiCyan, color.Bold).PrintlnFunc()
)*/

/*type JSONClientError struct {
	Code    int
	Class   string
	Details string
}

func (e *JSONClientError) Error() string {
	return fmt.Sprintf("JSONClientError: %s %d %s", e.Details, e.Code, e.Class)
}*/

type Client struct {
	authUrl string
	timeout int
	debug   bool

	httpconn *http.Client
	// serviceCatalog *KeystoneServiceCatalog
}

func NewClient(authUrl string, timeout int, debug bool, insecure bool) *Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}
	client := Client{authUrl: authUrl,
		timeout:  timeout,
		debug:    debug,
		httpconn: &http.Client{Transport: tr}}
	return &client
}

func (this *Client) AuthVersion() string {
	pos := strings.LastIndexByte(this.authUrl, '/')
	if pos > 0 {
		return this.authUrl[pos+1:]
	} else {
		return ""
	}
}

func getDefaultHeader(header http.Header, token string) http.Header {
	if len(token) > 0 {
		if header == nil {
			header = http.Header{}
		}
		header.Add("X-Auth-Token", token)
	}
	return header
}

func joinUrl(baseUrl, path string) string {
	base, version := SplitVersionedURL(baseUrl)
	if len(version) > 0 {
		if strings.HasPrefix(path, fmt.Sprintf("/%s/", version)) {
			baseUrl = base
		}
	}
	return fmt.Sprintf("%s%s", baseUrl, path)
}

func (this *Client) rawRequest(endpoint string, token string, method string, url string, header http.Header, body io.Reader) (*http.Response, error) {
	ctx := context.Background()
	return httputils.Request(this.httpconn, ctx, method, joinUrl(endpoint, url), getDefaultHeader(header, token), body, this.debug)
}

func (this *Client) jsonRequest(endpoint string, token string, method string, url string, header http.Header, body jsonutils.JSONObject) (http.Header, jsonutils.JSONObject, error) {
	/*bodystr := ""
	if body != nil {
		bodystr = body.String()
	}
	jbody := strings.NewReader(bodystr)
	if header == nil {
		header = http.Header{}
	}
	header.Add("Content-Type", "application/json")
	resp, err := this.rawRequest(endpoint, token, method, url, header, jbody)
	return this.parseJSONResponse(resp, err)*/
	ctx := context.Background()
	return httputils.JSONRequest(this.httpconn, ctx, method, joinUrl(endpoint, url), getDefaultHeader(header, token), body, this.debug)
}

/*func (this *Client) parseJSONResponse(resp *http.Response, err error) (http.Header, jsonutils.JSONObject, error) {
	if err != nil {
		ce := JSONClientError{}
		ce.Code = 499
		ce.Details = err.Error()
		return nil, nil, &ce
	}
	defer resp.Body.Close()
	if this.debug {
		if resp.StatusCode < 300 {
			green("Status:", resp.StatusCode)
			green(resp.Header)
		} else if resp.StatusCode < 400 {
			yellow("Status:", resp.StatusCode)
			yellow(resp.Header)
		} else {
			red("Status:", resp.StatusCode)
			red(resp.Header)
		}
	}
	rbody, err := ioutil.ReadAll(resp.Body)
	if this.debug {
		fmt.Println(string(rbody))
	}
	if err != nil {
		return nil, nil, fmt.Errorf("Fail to read body: %s", err)
	}
	var jrbody jsonutils.JSONObject = nil
	if len(rbody) > 0 {
		jrbody, _ = jsonutils.Parse(rbody)
		///// XXX: ignore error case
		// if err != nil && resp.StatusCode < 300 {
		//     return nil, nil, fmt.Errorf("Fail to decode body: %s", err)
		// }
		if jrbody != nil && this.debug {
			fmt.Println(jrbody)
		}
	}
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		ce := JSONClientError{}
		ce.Code = resp.StatusCode
		ce.Details = resp.Header.Get("Location")
		ce.Class = "redirect"
		return nil, nil, &ce
	} else if resp.StatusCode >= 400 {
		ce := JSONClientError{}
		if jrbody == nil {
			ce.Code = resp.StatusCode
			ce.Details = resp.Status
			return nil, nil, &ce
		} else {
			jrbody2, e := jrbody.Get("error")
			if e == nil {
				ecode, e := jrbody2.Int("code")
				if e == nil {
					ce.Code = int(ecode)
					ce.Details, _ = jrbody2.GetString("message")
					ce.Class, _ = jrbody2.GetString("title")
					return nil, nil, &ce
				} else {
					ce.Code = resp.StatusCode
					ce.Details = jrbody2.String()
					return nil, nil, &ce
				}
			} else {
				err = jrbody.Unmarshal(&ce)
				if err != nil {
					return nil, nil, err
				} else {
					return nil, nil, &ce
				}
			}
		}
	} else {
		return resp.Header, jrbody, nil
	}
}*/

func (this *Client) _authV3(domainName, uname, passwd, projectId, projectName, token string) (TokenCredential, error) {
	body := jsonutils.NewDict()
	if len(uname) > 0 && len(passwd) > 0 { // Password authentication
		body.Add(jsonutils.NewArray(jsonutils.NewString("password")), "auth", "identity", "methods")
		body.Add(jsonutils.NewString(uname), "auth", "identity", "password", "user", "name")
		body.Add(jsonutils.NewString(passwd), "auth", "identity", "password", "user", "password")
		if len(domainName) > 0 {
			body.Add(jsonutils.NewString(domainName), "auth", "identity", "password", "user", "domain", "name")
		} else {
			body.Add(jsonutils.NewString("default"), "auth", "identity", "password", "user", "domain", "id")
		}
	} else if len(token) > 0 {
		body.Add(jsonutils.NewArray(jsonutils.NewString("token")), "auth", "identity", "methods")
		body.Add(jsonutils.NewString(token), "auth", "identity", "token", "id")
	}
	if len(projectId) > 0 {
		body.Add(jsonutils.NewString(projectId), "auth", "scope", "project", "id")
	}
	if len(projectName) > 0 {
		body.Add(jsonutils.NewString("default"), "auth", "scope", "project", "domain", "id")
		body.Add(jsonutils.NewString(projectName), "auth", "scope", "project", "name")
	}
	hdr, rbody, err := this.jsonRequest(this.authUrl, "", "POST", "/auth/tokens", nil, body)
	if err != nil {
		return nil, err
	}
	tokenId := hdr.Get("X-Subject-Token")
	if len(tokenId) == 0 {
		return nil, fmt.Errorf("No X-Subject-Token in header")
	}
	ret, err := UnmarshalV3Token(rbody, tokenId)
	return ret, err
}

func (this *Client) _authV2(uname, passwd, tenantId, tenantName, token string) (TokenCredential, error) {
	body := jsonutils.NewDict()
	if len(uname) > 0 && len(passwd) > 0 {
		body.Add(jsonutils.NewString(uname), "auth", "passwordCredentials", "username")
		body.Add(jsonutils.NewString(passwd), "auth", "passwordCredentials", "password")
	}
	if len(tenantName) > 0 {
		body.Add(jsonutils.NewString(tenantName), "auth", "tenantName")
	}
	if len(tenantId) > 0 {
		body.Add(jsonutils.NewString(tenantId), "auth", "tenantId")
	}
	if len(token) > 0 {
		body.Add(jsonutils.NewString(token), "auth", "token", "id")
	}
	_, rbody, err := this.jsonRequest(this.authUrl, "", "POST", "/tokens", nil, body)
	if err != nil {
		return nil, err
	}
	return UnmarshalV2Token(rbody)
}

func (this *Client) Authenticate(uname, passwd, domainName, tenantName string) (TokenCredential, error) {
	if this.AuthVersion() == "v3" {
		return this._authV3(domainName, uname, passwd, "", tenantName, "")
	}
	return this._authV2(uname, passwd, "", tenantName, "")
}

func UnmarshalV3Token(rbody jsonutils.JSONObject, tokenId string) (cred TokenCredential, err error) {
	cred = &TokenCredentialV3{Id: tokenId}
	err = rbody.Unmarshal(cred)
	if err != nil {
		err = fmt.Errorf("Invalid response when unmarshal V3 Token: %v", err)
	}
	return
}

func UnmarshalV2Token(rbody jsonutils.JSONObject) (cred TokenCredential, err error) {
	access, err := rbody.Get("access")
	if err == nil {
		cred = &TokenCredentialV2{}
		err = access.Unmarshal(cred)
		if err != nil {
			err = fmt.Errorf("Invalid response when unmarshal V2 Token: %s", err)
		}
		return
	}
	err = fmt.Errorf("Invalid response: no access object")
	return
}

func (this *Client) verifyV3(adminToken, token string) (TokenCredential, error) {
	header := http.Header{}
	header.Add("X-Auth-Token", adminToken)
	header.Add("X-Subject-Token", token)
	_, rbody, err := this.jsonRequest(this.authUrl, "", "GET", "/auth/tokens", header, nil)
	if err != nil {
		return nil, err
	}
	return UnmarshalV3Token(rbody, token)
}

func (this *Client) verifyV2(adminToken, token string) (TokenCredential, error) {
	header := http.Header{}
	header.Add("X-Auth-Token", adminToken)
	verifyUrl := fmt.Sprintf("/tokens/%s", token)
	_, rbody, err := this.jsonRequest(this.authUrl, "", "GET", verifyUrl, header, nil)
	if err != nil {
		return nil, err
	}
	return UnmarshalV2Token(rbody)
}

func (this *Client) Verify(adminToken, token string) (cred TokenCredential, err error) {
	if this.AuthVersion() == "v3" {
		return this.verifyV3(adminToken, token)
	}
	return this.verifyV2(adminToken, token)
}

func (this *Client) SetTenant(tenantId, tenantName string, token TokenCredential) (TokenCredential, error) {
	return this.SetProject(tenantId, tenantName, token)
}

func (this *Client) SetProject(tenantId, tenantName string, token TokenCredential) (TokenCredential, error) {
	if this.AuthVersion() == "v3" {
		return this._authV3("", "", "", tenantId, tenantName, token.GetTokenString())
	} else {
		return this._authV2("", "", "", tenantName, token.GetTokenString())
	}
}

func (this *Client) NewSession(region, zone, endpointType string, token TokenCredential, apiVersion string) *ClientSession {
	return &ClientSession{client: this, region: region, zone: zone,
		endpointType: endpointType, token: token,
		apiVersion: apiVersion,
		Header:     http.Header{}}
}

/*
func (this *Client) fetchTenants(Token string) error {
    _, body, err := this.jsonRequest(this.authUrl, Token, "GET", "/tenants", nil, nil)
    if err != nil {
        return fmt.Errorf("Fetch tenant error: %s", err)
    }
    tarr, err := body.GetArray("tenant")
    if err != nil {
        return fmt.Errorf("Invalid response: %s", err)
    }
    for _, t := range tarr {
        id, err := t.GetString("id")
        if err != nil {
            return fmt.Errorf("Invalid tenant: %s", err)
        }
        name, err := t.GetString("name")
        if err != nil {
            return fmt.Errorf("Invalid tenant: %s", err)
        }
        this.tenantsManager.Add(id, name)
    }
    return nil
}

func (this *Client) SetTenant(tenantId, tenantName string) error {
    tenant := this.tenantsManager.GetTenant(tenantId, tenantName)
    if tenant == nil {
        return this.authenticate(tenantId, tenantName)
    }else {
        this.defaultTenant = tenant
        return nil
    }
}

func (this *Client) GetTenants() ([]KeystoneTenant, error) {
    err := this.authenticate("", "")
    return []KeystoneTenant(this.tenantsManager), err
}

func (this *Client) getMatchEndpoint(eplist []Endpoint) (*Endpoint, error) {
    if len(this.region) == 0 {
        if len(eplist) == 1 {
            return &eplist[0], nil
        }else if len(eplist) > 1 {
            return nil, fmt.Errorf("Need to specify OS_REGION_NAME")
        }else {
            return nil, fmt.Errorf("Empty endpoints")
        }
    }else {
        var match, matchZone, matchRegion *Endpoint = nil, nil, nil
        region := this.region
        zone := fmt.Sprintf("%s/%s", this.region, this.zone)
        for _, ep := range eplist {
            switch ep.Region {
                case zone:
                    matchZone = &ep
                case region:
                    matchRegion = &ep
            }
        }
        if matchZone != nil {
            match = matchZone
        }else if matchRegion != nil {
            match = matchRegion
        }
        if match != nil {
            return match, nil
        }else {
            return nil, fmt.Errorf("No match endpoint")
        }
    }
}

func (this *Client) GetEndpoint(service string) (string, error) {
    for _, srv := range this.serviceCatalog {
        if srv.Type == service {
            ep, err := this.getMatchEndpoint(srv.Endpoints)
            if err != nil {
                return "", err
            }else {
                switch this.endpointType {
                    case "adminURL":
                        return ep.AdminURL, nil
                    case "internalURL":
                        return ep.InternalURL, nil
                    default:
                        return ep.PublicURL, nil
                }
            }
        }
    }
    return "", fmt.Errorf("%s not found", service)
}

func (this *Client) RequestService(method string, service string, requrl string, body jsonutils.JSONObject) (jsonutils.JSONObject, error) {
    ep, err := this.GetEndpoint(service)
    if err != nil {
        return nil, err
    }
    Token := this.defaultTenant.Token.Id
    _, rbody, err := this.json_request(ep, Token, method, requrl, nil, body)
    return rbody, err
}

func (this *Client) IsSystemAdmin() bool {
    if this.defaultTenant != nil {
        return this.defaultTenant.isSystemAdmin()
    }
    return false
}

func (this *KeystoneTenant) isSystemAdmin() bool {
    if this.Name != "system" {
        return false
    }
    for _, r := range this.User.Roles {
        if r.Name == "admin" {
            return true
        }
    }
    return false
}
*/
