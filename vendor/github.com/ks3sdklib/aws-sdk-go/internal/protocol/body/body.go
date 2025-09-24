package body

import (
	"github.com/ks3sdklib/aws-sdk-go/aws"
	"github.com/ks3sdklib/aws-sdk-go/internal/protocol/rest"
	"github.com/ks3sdklib/aws-sdk-go/internal/protocol/restjson"
	"github.com/ks3sdklib/aws-sdk-go/internal/protocol/restxml"
)

// Build builds the REST component of a service request.
func Build(r *aws.Request) {
	if r.ContentType == "application/json" {
		restjson.Build(r)
	} else {
		restxml.Build(r)
	}
}

// UnmarshalBody unmarshal a response body for the REST protocol.
func UnmarshalBody(r *aws.Request) {
	rest.Unmarshal(r)
	if r.ContentType == "application/json" {
		restjson.Unmarshal(r)
	} else {
		restxml.Unmarshal(r)
	}
}

// UnmarshalMeta unmarshal response headers for the REST protocol.
func UnmarshalMeta(r *aws.Request) {
	rest.UnmarshalMeta(r)
}

// UnmarshalError unmarshal a response error for the REST protocol.
func UnmarshalError(r *aws.Request) {
	restxml.UnmarshalError(r)
}
