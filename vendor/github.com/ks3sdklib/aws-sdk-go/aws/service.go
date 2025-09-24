package aws

import (
	"github.com/ks3sdklib/aws-sdk-go/aws/retry"
	"github.com/ks3sdklib/aws-sdk-go/internal/endpoints"
	"net/http"
	"net/http/httputil"
	"regexp"
)

// A Service implements the base service request and response handling
// used by all services.
type Service struct {
	Config        *Config
	Handlers      Handlers
	ManualSend    bool
	ServiceName   string
	APIVersion    string
	Endpoint      string
	SigningName   string
	SigningRegion string
	JSONVersion   string
	TargetPrefix  string
	RetryRule     retry.RetryRule
	ShouldRetry   func(error) bool
	MaxRetries    int
}

var schemeRE = regexp.MustCompile("^([^:]+)://")

// NewService will return a pointer to a new Server object initialized.
func NewService(config *Config) *Service {
	svc := &Service{Config: config}
	svc.Initialize()
	return svc
}

// Initialize initializes the service.
func (s *Service) Initialize() {
	if s.Config == nil {
		s.Config = &Config{}
	}
	if s.Config.HTTPClient == nil {
		s.Config.HTTPClient = http.DefaultClient
	}

	if s.RetryRule == nil {
		s.RetryRule = s.Config.RetryRule
	}

	if s.ShouldRetry == nil {
		s.ShouldRetry = s.Config.ShouldRetry
	}

	s.MaxRetries = s.Config.MaxRetries
	s.Handlers.Validate.PushBack(ValidateEndpointHandler)
	s.Handlers.Build.PushBack(UserAgentHandler)
	s.Handlers.Sign.PushBack(BuildContentLength)
	s.Handlers.Send.PushBack(SendHandler)
	s.Handlers.AfterRetry.PushBack(AfterRetryHandler)
	s.Handlers.ValidateResponse.PushBack(ValidateResponseHandler)
	s.AddDebugHandlers()
	s.buildEndpoint()

	if !s.Config.DisableParamValidation {
		s.Handlers.Validate.PushBack(ValidateParameters)
	}
}

// buildEndpoint builds the endpoint values the service will use to make requests with.
func (s *Service) buildEndpoint() {
	if s.Config.Endpoint != "" {
		s.Endpoint = s.Config.Endpoint
	} else {
		s.Endpoint, s.SigningRegion =
			endpoints.EndpointForRegion(s.ServiceName, s.Config.Region)
	}
	if s.Endpoint != "" && !schemeRE.MatchString(s.Endpoint) {
		scheme := "https"
		if s.Config.DisableSSL {
			scheme = "http"
		}
		s.Endpoint = scheme + "://" + s.Endpoint
	}
}

// AddDebugHandlers injects debug logging handlers into the service to log request
// debug information.
func (s *Service) AddDebugHandlers() {
	if s.Config.LogLevel < Debug {
		return
	}

	s.Handlers.Send.PushFront(func(r *Request) {
		logBody := r.Config.LogHTTPBody
		dumpedBody, _ := httputil.DumpRequestOut(r.HTTPRequest, logBody)
		r.Config.LogDebug("---[ REQUEST ]-----------------------------")
		r.Config.LogDebug("%s", string(dumpedBody))
		r.Config.LogDebug("-----------------------------------------------------")
	})
	s.Handlers.Send.PushBack(func(r *Request) {
		r.Config.LogDebug("---[ RESPONSE ]--------------------------------------")
		if r.HTTPResponse != nil {
			logBody := r.Config.LogHTTPBody
			dumpedBody, _ := httputil.DumpResponse(r.HTTPResponse, logBody)
			r.Config.LogDebug("%s", string(dumpedBody))
		} else if r.Error != nil {
			r.Config.LogDebug("%s", r.Error.Error())
		}
		r.Config.LogDebug("-----------------------------------------------------")
	})
}
