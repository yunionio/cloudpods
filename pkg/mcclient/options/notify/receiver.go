package notify

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type ReceiverListOptions struct {
	options.BaseListOptions
	UId                 string `help:"user id in keystone"`
	UName               string `help:"user name in keystone"`
	EnabledContactType  string `help:"enabled contact type"`
	VerifiedContactType string `help:"verified contact type"`
}

func (rl *ReceiverListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(rl)
}

type ReceiverCreateOptions struct {
	UID                 string   `help:"user id in keystone"`
	Email               string   `help:"email of receiver"`
	Mobile              string   `help:"mobile of receiver"`
	EnabledContactTypes []string `help:"enabled contact type"`
}

func (rc *ReceiverCreateOptions) Params() (jsonutils.JSONObject, error) {
	return options.StructToParams(rc)
}

type ReceiverOptions struct {
	ID string `help:"Id or Name of receiver"`
}

func (r *ReceiverOptions) GetId() string {
	return r.ID
}

func (r *ReceiverOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type ReceiverUpdateOptions struct {
	ReceiverOptions
	receiverUpdateOptions
}

type receiverUpdateOptions struct {
	Email              string   `help:"email of receiver"`
	Mobile             string   `help:"mobile of receiver"`
	EnabledContactType []string `help:"enabled contact type"`
}

func (ru *ReceiverUpdateOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(ru.receiverUpdateOptions), nil
}

type ReceiverTriggerVerifyOptions struct {
	ReceiverOptions
	receiverTriggerVerifyOptions
}

type receiverTriggerVerifyOptions struct {
	ContactType string `help:"Contact type to trigger verify" choices:"email|mobile"`
}

func (rt *ReceiverTriggerVerifyOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(rt.receiverTriggerVerifyOptions), nil
}

type ReceiverVerifyOptions struct {
	ReceiverOptions
	receiverVerifyOptions
}

type receiverVerifyOptions struct {
	ContactType string `help:"Contact type to trigger verify" choices:"email|mobile"`
	Token       string `help:"Token from verify message sent to you"`
}

func (rv *ReceiverVerifyOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(rv.receiverVerifyOptions), nil
}

type ReceiverIntellijGetOptions struct {
	USERID     string `help:"user id in keystone" json:"user_id"`
	CreateIfNo *bool  `help:"create if receiver with UserId does not exist"`
	SCOPE      string `help:"scope"`
}

func (ri *ReceiverIntellijGetOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(ri), nil
}
