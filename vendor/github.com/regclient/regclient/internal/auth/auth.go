// Package auth is used for HTTP authentication
package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type charLU byte

var charLUs [256]charLU

var defaultClientID = "regclient"

// minTokenLife tokens are required to last at least 60 seconds to support older docker clients
var minTokenLife = 60

// tokenBuffer is used to renew a token before it expires to account for time to process requests on the server
var tokenBuffer = time.Second * 5

const (
	isSpace charLU = 1 << iota
	isToken
)

func init() {
	for c := 0; c < 256; c++ {
		charLUs[c] = 0
		if strings.ContainsRune(" \t\r\n", rune(c)) {
			charLUs[c] |= isSpace
		}
		if (rune('a') <= rune(c) && rune(c) <= rune('z')) || (rune('A') <= rune(c) && rune(c) <= rune('Z') || (rune('0') <= rune(c) && rune(c) <= rune('9')) || strings.ContainsRune("-._~+/", rune(c))) {
			charLUs[c] |= isToken
		}
	}
}

// CredsFn is passed to lookup credentials for a given hostname, response is a username and password or empty strings
type CredsFn func(string) Cred

// Cred is returned by the CredsFn
type Cred struct {
	User, Password, Token string
}

// Auth manages authorization requests/responses for http requests
type Auth interface {
	AddScope(host, scope string) error
	HandleResponse(*http.Response) error
	UpdateRequest(*http.Request) error
}

// Challenge is the extracted contents of the WWW-Authenticate header
type Challenge struct {
	authType string
	params   map[string]string
}

// Handler handles a challenge for a host to return an auth header
type Handler interface {
	AddScope(scope string) error
	ProcessChallenge(Challenge) error
	GenerateAuth() (string, error)
}

// HandlerBuild is used to make a new handler for a specific authType and URL
type HandlerBuild func(client *http.Client, clientID, host string, credFn CredsFn, log *logrus.Logger) Handler

// Opts configures options for NewAuth
type Opts func(*auth)

type auth struct {
	httpClient *http.Client
	clientID   string
	credsFn    CredsFn
	hbs        map[string]HandlerBuild       // handler builders based on authType
	hs         map[string]map[string]Handler // handlers based on url and authType
	authTypes  []string
	log        *logrus.Logger
	mu         sync.Mutex
}

// NewAuth creates a new Auth
func NewAuth(opts ...Opts) Auth {
	a := &auth{
		httpClient: &http.Client{},
		clientID:   defaultClientID,
		credsFn:    DefaultCredsFn,
		hbs:        map[string]HandlerBuild{},
		hs:         map[string]map[string]Handler{},
		authTypes:  []string{},
	}
	a.log = &logrus.Logger{
		Out:       os.Stderr,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.WarnLevel,
	}

	for _, opt := range opts {
		opt(a)
	}

	if len(a.authTypes) == 0 {
		a.addDefaultHandlers()
	}

	return a
}

// WithCreds provides a user/pass lookup for a url
func WithCreds(f CredsFn) Opts {
	return func(a *auth) {
		if f != nil {
			a.credsFn = f
		}
	}
}

// WithHTTPClient uses a specific http client with requests
func WithHTTPClient(h *http.Client) Opts {
	return func(a *auth) {
		if h != nil {
			a.httpClient = h
		}
	}
}

// WithClientID uses a client ID with request headers
func WithClientID(clientID string) Opts {
	return func(a *auth) {
		a.clientID = clientID
	}
}

// WithHandler includes a handler for a specific auth type
func WithHandler(authType string, hb HandlerBuild) Opts {
	return func(a *auth) {
		lcat := strings.ToLower(authType)
		a.hbs[lcat] = hb
		a.authTypes = append(a.authTypes, lcat)
	}
}

// WithDefaultHandlers includes a Basic and Bearer handler, this is automatically added with "WithHandler" is not called
func WithDefaultHandlers() Opts {
	return func(a *auth) {
		a.addDefaultHandlers()
	}
}

// WithLog injects a logrus Logger
func WithLog(log *logrus.Logger) Opts {
	return func(a *auth) {
		a.log = log
	}
}

// AddScope extends an existing auth with additional scopes.
// This is used to pre-populate scopes with the Docker convention rather than
// depend on the registry to respond with the correct http status and headers.
func (a *auth) AddScope(host, scope string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	success := false
	if a.hs[host] == nil {
		return ErrNoNewChallenge
	}
	for _, at := range a.authTypes {
		if a.hs[host][at] != nil {
			err := a.hs[host][at].AddScope(scope)
			if err == nil {
				success = true
			} else if err != ErrNoNewChallenge {
				return err
			}
		}
	}
	if !success {
		return ErrNoNewChallenge
	}
	a.log.WithFields(logrus.Fields{
		"host":  host,
		"scope": scope,
	}).Debug("Auth scope added")
	return nil
}

// HandleResponse parses the 401 response, extracting the WWW-Authenticate
// header and verifying the requirement is different from what was included in
// the last request
func (a *auth) HandleResponse(resp *http.Response) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	// verify response is an access denied
	if resp.StatusCode != http.StatusUnauthorized {
		return ErrUnsupported
	}

	// extract host and auth header
	host := resp.Request.URL.Host
	cl, err := ParseAuthHeaders(resp.Header.Values("WWW-Authenticate"))
	if err != nil {
		return err
	}
	a.log.WithFields(logrus.Fields{
		"challenge": cl,
	}).Debug("Auth request parsed")
	if len(cl) < 1 {
		return ErrEmptyChallenge
	}
	goodChallenge := false
	// loop over the received challenge(s)
	for _, c := range cl {
		if _, ok := a.hbs[c.authType]; !ok {
			a.log.WithFields(logrus.Fields{
				"authtype": c.authType,
			}).Warn("Unsupported auth type")
			continue
		}
		// setup a handler for the host and auth type
		if _, ok := a.hs[host]; !ok {
			a.hs[host] = map[string]Handler{}
		}
		if _, ok := a.hs[host][c.authType]; !ok {
			h := a.hbs[c.authType](a.httpClient, a.clientID, host, a.credsFn, a.log)
			if h == nil {
				continue
			}
			a.hs[host][c.authType] = h
		}
		// process the challenge with that handler
		err := a.hs[host][c.authType].ProcessChallenge(c)
		if err == nil {
			goodChallenge = true
		} else if err == ErrNoNewChallenge {
			// handle race condition when another request updates the challenge
			// detect that by seeing the current auth header is different
			prevAH := resp.Request.Header.Get("Authorization")
			ah, err := a.hs[host][c.authType].GenerateAuth()
			if err == nil && prevAH != ah {
				goodChallenge = true
			}
		} else {
			return err
		}
	}
	if !goodChallenge {
		return ErrUnauthorized
	}

	return nil
}

// UpdateRequest adds Authorization headers to a request
func (a *auth) UpdateRequest(req *http.Request) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	host := req.URL.Host
	if a.hs[host] == nil {
		return nil
	}
	var err error
	var ah string
	for _, at := range a.authTypes {
		if a.hs[host][at] != nil {
			ah, err = a.hs[host][at].GenerateAuth()
			if err != nil {
				a.log.WithFields(logrus.Fields{
					"err":      err,
					"host":     host,
					"authtype": at,
				}).Debug("Failed to generate auth")
				continue
			}
			req.Header.Set("Authorization", ah)
			break
		}
	}
	if err != nil {
		return err
	}
	return nil
}

func (a *auth) addDefaultHandlers() {
	if _, ok := a.hbs["basic"]; !ok {
		a.hbs["basic"] = NewBasicHandler
		a.authTypes = append(a.authTypes, "basic")
	}
	if _, ok := a.hbs["bearer"]; !ok {
		a.hbs["bearer"] = NewBearerHandler
		a.authTypes = append(a.authTypes, "bearer")
	}
	// jwt is considered experimental, used for some Hub specific API's
	if _, ok := a.hbs["jwt"]; !ok {
		a.hbs["jwt"] = NewJWTHandler
		a.authTypes = append(a.authTypes, "jwt")
	}
}

// DefaultCredsFn is used to return no credentials when auth is not configured with a CredsFn
// This avoids the need to check for nil pointers
func DefaultCredsFn(h string) Cred {
	return Cred{}
}

// ParseAuthHeaders extracts the scheme and realm from WWW-Authenticate headers
func ParseAuthHeaders(ahl []string) ([]Challenge, error) {
	var cl []Challenge
	for _, ah := range ahl {
		c, err := ParseAuthHeader(ah)
		if err != nil {
			return nil, fmt.Errorf("failed to parse challenge header: %s, %w", ah, err)
		}
		cl = append(cl, c...)
	}
	return cl, nil
}

// ParseAuthHeader parses a single header line for WWW-Authenticate
// Example values:
// Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:samalba/my-app:pull,push"
// Basic realm="GitHub Package Registry"
func ParseAuthHeader(ah string) ([]Challenge, error) {
	var cl []Challenge
	var c *Challenge
	var eb, atb, kb, vb []byte // eb is element bytes, atb auth type, kb key, vb value
	state := "string"

	for _, b := range []byte(ah) {
		switch state {
		case "string":
			if len(eb) == 0 {
				// beginning of string
				if b == '"' { // TODO: Invalid?
					state = "quoted"
				} else if charLUs[b]&isToken != 0 {
					// read any token
					eb = append(eb, b)
				} else if charLUs[b]&isSpace != 0 {
					// ignore leading whitespace
				} else {
					// unknown leading char
					return nil, ErrParseFailure
				}
			} else {
				if charLUs[b]&isToken != 0 {
					// read any token
					eb = append(eb, b)
				} else if b == '=' && len(atb) > 0 {
					// equals when authtype is defined makes this a key
					kb = eb
					eb = []byte{}
					state = "value"
				} else if charLUs[b]&isSpace != 0 {
					// space ends the element
					atb = eb
					eb = []byte{}
					c = &Challenge{authType: strings.ToLower(string(atb)), params: map[string]string{}}
					cl = append(cl, *c)
				} else {
					// unknown char
					return nil, ErrParseFailure
				}
			}

		case "value":
			if charLUs[b]&isToken != 0 {
				// read any token
				vb = append(vb, b)
			} else if b == '"' && len(vb) == 0 {
				// quoted value
				state = "quoted"
			} else if charLUs[b]&isSpace != 0 || b == ',' {
				// space or comma ends the value
				c.params[strings.ToLower(string(kb))] = string(vb)
				kb = []byte{}
				vb = []byte{}
				if b == ',' {
					state = "string"
				} else {
					state = "endvalue"
				}
			} else {
				// unknown char
				return nil, ErrParseFailure
			}

		case "quoted":
			if b == '"' {
				// end quoted string
				c.params[strings.ToLower(string(kb))] = string(vb)
				kb = []byte{}
				vb = []byte{}
				state = "endvalue"
			} else if b == '\\' {
				state = "escape"
			} else {
				// all other bytes in a quoted string are taken as-is
				vb = append(vb, b)
			}

		case "endvalue":
			if charLUs[b]&isSpace != 0 {
				// ignore leading whitespace
			} else if b == ',' {
				// expect a comma separator, return to start of a string
				state = "string"
			} else {
				// unknown char
				return nil, ErrParseFailure
			}

		case "escape":
			vb = append(vb, b)
			state = "quoted"

		default:
			return nil, ErrParseFailure
		}
	}

	// process any content left at end of string, and handle any unfinished sections
	switch state {
	case "string":
		if len(eb) != 0 {
			atb = eb
			c = &Challenge{authType: strings.ToLower(string(atb)), params: map[string]string{}}
			cl = append(cl, *c)
		}
	case "value":
		if len(vb) != 0 {
			c.params[strings.ToLower(string(kb))] = string(vb)
		}
	case "quoted", "escape":
		return nil, ErrParseFailure
	}

	return cl, nil
}

// BasicHandler supports Basic auth type requests
type BasicHandler struct {
	realm   string
	host    string
	credsFn CredsFn
}

// NewBasicHandler creates a new BasicHandler
func NewBasicHandler(client *http.Client, clientID, host string, credsFn CredsFn, log *logrus.Logger) Handler {
	return &BasicHandler{
		realm:   "",
		host:    host,
		credsFn: credsFn,
	}
}

// AddScope is not valid for BasicHandler
func (b *BasicHandler) AddScope(scope string) error {
	return ErrNoNewChallenge
}

// ProcessChallenge for BasicHandler is a noop
func (b *BasicHandler) ProcessChallenge(c Challenge) error {
	if _, ok := c.params["realm"]; !ok {
		return ErrInvalidChallenge
	}
	if b.realm != c.params["realm"] {
		b.realm = c.params["realm"]
		return nil
	}
	return ErrNoNewChallenge
}

// GenerateAuth for BasicHandler generates base64 encoded user/pass for a host
func (b *BasicHandler) GenerateAuth() (string, error) {
	cred := b.credsFn(b.host)
	if cred.User == "" || cred.Password == "" {
		return "", ErrNotFound
	}
	auth := base64.StdEncoding.EncodeToString([]byte(cred.User + ":" + cred.Password))
	return fmt.Sprintf("Basic %s", auth), nil
}

// BearerHandler supports Bearer auth type requests
type BearerHandler struct {
	client         *http.Client
	clientID       string
	realm, service string
	host           string
	credsFn        CredsFn
	scopes         []string
	token          BearerToken
	log            *logrus.Logger
}

// BearerToken is the json response to the Bearer request
type BearerToken struct {
	Token        string    `json:"token"`
	AccessToken  string    `json:"access_token"`
	ExpiresIn    int       `json:"expires_in"`
	IssuedAt     time.Time `json:"issued_at"`
	RefreshToken string    `json:"refresh_token"`
	Scope        string    `json:"scope"`
}

// NewBearerHandler creates a new BearerHandler
func NewBearerHandler(client *http.Client, clientID, host string, credsFn CredsFn, log *logrus.Logger) Handler {
	return &BearerHandler{
		client:   client,
		clientID: clientID,
		host:     host,
		credsFn:  credsFn,
		realm:    "",
		service:  "",
		scopes:   []string{},
		log:      log,
	}
}

// AddScope appends a new scope if it doesn't already exist
func (b *BearerHandler) AddScope(scope string) error {
	if b.scopeExists(scope) {
		if b.token.Token == "" || !b.isExpired() {
			return ErrNoNewChallenge
		}
		return nil
	}
	return b.addScope(scope)
}

func (b *BearerHandler) addScope(scope string) error {
	replaced := false
	for i, cur := range b.scopes {
		// extend an existing scope with more actions
		if strings.HasPrefix(scope, cur+",") {
			b.scopes[i] = scope
			replaced = true
			break
		}
	}
	if !replaced {
		b.scopes = append(b.scopes, scope)
	}
	// delete any scope specific or invalid tokens
	b.token.Token = ""
	b.token.RefreshToken = ""
	return nil
}

// ProcessChallenge handles WWW-Authenticate header for bearer tokens
// Bearer realm="https://auth.docker.io/token",service="registry.docker.io",scope="repository:samalba/my-app:pull,push"
func (b *BearerHandler) ProcessChallenge(c Challenge) error {
	if _, ok := c.params["realm"]; !ok {
		return ErrInvalidChallenge
	}
	if _, ok := c.params["service"]; !ok {
		c.params["service"] = ""
	}
	if _, ok := c.params["scope"]; !ok {
		c.params["scope"] = ""
	}

	existingScope := b.scopeExists(c.params["scope"])

	if b.realm == c.params["realm"] && b.service == c.params["service"] && existingScope && (b.token.Token == "" || !b.isExpired()) {
		return ErrNoNewChallenge
	}

	if b.realm == "" {
		b.realm = c.params["realm"]
	} else if b.realm != c.params["realm"] {
		return ErrInvalidChallenge
	}
	if b.service == "" {
		b.service = c.params["service"]
	} else if b.service != c.params["service"] {
		return ErrInvalidChallenge
	}
	if !existingScope {
		return b.addScope(c.params["scope"])
	}
	return nil
}

// GenerateAuth for BasicHandler generates base64 encoded user/pass for a host
func (b *BearerHandler) GenerateAuth() (string, error) {
	// if unexpired token already exists, return it
	if b.token.Token != "" && !b.isExpired() {
		return fmt.Sprintf("Bearer %s", b.token.Token), nil
	}

	// attempt to post with oauth form, this also uses refresh tokens
	if err := b.tryPost(); err == nil {
		return fmt.Sprintf("Bearer %s", b.token.Token), nil
	} else if err != ErrUnauthorized {
		return "", err
	}

	// attempt a get (with basic auth if user/pass available)
	if err := b.tryGet(); err == nil {
		return fmt.Sprintf("Bearer %s", b.token.Token), nil
	} else if err != ErrUnauthorized {
		return "", err
	}

	return "", ErrUnauthorized
}

// isExpired returns true when token issue date is either 0, token has expired,
// or will expire within buffer time
func (b *BearerHandler) isExpired() bool {
	if b.token.IssuedAt.IsZero() {
		return true
	}
	expireSec := b.token.IssuedAt.Add(time.Duration(b.token.ExpiresIn) * time.Second)
	expireSec = expireSec.Add(tokenBuffer * -1)
	return time.Now().After(expireSec)
}

// tryGet requests a new token with a GET request
func (b *BearerHandler) tryGet() error {
	cred := b.credsFn(b.host)
	req, err := http.NewRequest("GET", b.realm, nil)
	if err != nil {
		return err
	}

	reqParams := req.URL.Query()
	reqParams.Add("client_id", b.clientID)
	reqParams.Add("offline_token", "true")
	if b.service != "" {
		reqParams.Add("service", b.service)
	}

	for _, s := range b.scopes {
		reqParams.Add("scope", s)
	}

	if cred.User != "" && cred.Password != "" {
		reqParams.Add("account", cred.User)
		req.SetBasicAuth(cred.User, cred.Password)
	}

	req.Header.Add("User-Agent", b.clientID)
	req.URL.RawQuery = reqParams.Encode()

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return b.validateResponse(resp)
}

// tryPost requests a new token via a POST request
func (b *BearerHandler) tryPost() error {
	cred := b.credsFn(b.host)
	form := url.Values{}
	if len(b.scopes) > 0 {
		form.Set("scope", strings.Join(b.scopes, " "))
	}
	if b.service != "" {
		form.Set("service", b.service)
	}
	form.Set("client_id", b.clientID)
	if b.token.RefreshToken != "" {
		form.Set("grant_type", "refresh_token")
		form.Set("refresh_token", b.token.RefreshToken)
	} else if cred.Token != "" {
		form.Set("grant_type", "refresh_token")
		form.Set("refresh_token", cred.Token)
	} else if cred.User != "" && cred.Password != "" {
		form.Set("grant_type", "password")
		form.Set("username", cred.User)
		form.Set("password", cred.Password)
	}

	req, err := http.NewRequest("POST", b.realm, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	req.Header.Add("User-Agent", b.clientID)

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return b.validateResponse(resp)
}

// scopeExists check if the scope already exists within the list of scopes
func (b *BearerHandler) scopeExists(search string) bool {
	if search == "" {
		return true
	}
	for _, scope := range b.scopes {
		// allow scopes with additional actions, search for pull should match pull,push
		if scope == search || strings.HasPrefix(scope, search+",") {
			return true
		}
	}
	return false
}

// validateResponse extracts the returned token
func (b *BearerHandler) validateResponse(resp *http.Response) error {
	if resp.StatusCode != 200 {
		return ErrUnauthorized
	}

	// decode response and if successful, update token
	decoder := json.NewDecoder(resp.Body)
	decoded := BearerToken{}
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}
	b.token = decoded

	if b.token.ExpiresIn < minTokenLife {
		b.token.ExpiresIn = minTokenLife
	}

	// If token is already expired, it was sent with a zero value or
	// there may be a clock skew between the client and auth server.
	// Also handle cases of remote time in the future.
	// But if remote time is slightly in the past, leave as is so token
	// expires here before the server.
	if b.isExpired() || b.token.IssuedAt.After(time.Now()) {
		b.token.IssuedAt = time.Now().UTC()
	}

	// AccessToken and Token should be the same and we use Token elsewhere
	if b.token.AccessToken != "" {
		b.token.Token = b.token.AccessToken
	}

	return nil
}

// JWTHubHandler supports JWT auth type requests
type JWTHubHandler struct {
	client   *http.Client
	clientID string
	realm    string
	host     string
	credsFn  CredsFn
	jwt      string
}

type jwtHubPost struct {
	User string `json:"username"`
	Pass string `json:"password"`
}
type jwtHubResp struct {
	Detail       string `json:"detail"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
}

// NewJWTHandler creates a new JWTHandler
func NewJWTHandler(client *http.Client, clientID, host string, credsFn CredsFn, log *logrus.Logger) Handler {
	// JWT handler is only tested against Hub, and the API is Hub specific
	if host == "hub.docker.com" {
		return &JWTHubHandler{
			client:   client,
			clientID: clientID,
			host:     host,
			credsFn:  credsFn,
			realm:    "https://hub.docker.com/v2/users/login",
		}
	}
	return nil
}

// AddScope is not valid for JWTHubHandler
func (j *JWTHubHandler) AddScope(scope string) error {
	return ErrNoNewChallenge
}

// ProcessChallenge handles WWW-Authenticate header for JWT auth on Docker Hub
func (j *JWTHubHandler) ProcessChallenge(c Challenge) error {
	cred := j.credsFn(j.host)
	// use token if provided
	if cred.Token != "" {
		j.jwt = cred.Token
		return nil
	}

	// send a login request to hub
	bodyBytes, err := json.Marshal(jwtHubPost{
		User: cred.User,
		Pass: cred.Password,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", j.realm, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("User-Agent", j.clientID)

	resp, err := j.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 || resp.StatusCode >= 300 {
		return ErrUnauthorized
	}

	var bodyParsed jwtHubResp
	err = json.Unmarshal(body, &bodyParsed)
	if err != nil {
		return err
	}
	j.jwt = bodyParsed.Token

	return nil
}

// GenerateAuth for JWTHubHandler adds JWT header
func (j *JWTHubHandler) GenerateAuth() (string, error) {
	if len(j.jwt) > 0 {
		return fmt.Sprintf("JWT %s", j.jwt), nil
	}
	return "", ErrUnauthorized
}
