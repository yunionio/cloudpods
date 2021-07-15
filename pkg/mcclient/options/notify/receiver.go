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
	ProjectDomainFilter bool   `help:"filter receivers who join the project under the domain where the requester is currently located"`
}

func (rl *ReceiverListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(rl)
}

type ReceiverCreateOptions struct {
	UID                 string   `help:"user id in keystone"`
	Email               string   `help:"email of receiver"`
	Mobile              string   `help:"mobile of receiver"`
	MobileAreaCode      string   `help:"area code of mobile"`
	EnabledContactTypes []string `help:"enabled contact type"`
}

func (rc *ReceiverCreateOptions) Params() (jsonutils.JSONObject, error) {
	d := jsonutils.NewDict()
	d.Set("uid", jsonutils.NewString(rc.UID))
	d.Set("email", jsonutils.NewString(rc.Email))
	d.Set("enabled_contact_types", jsonutils.NewStringArray(rc.EnabledContactTypes))
	d.Add(jsonutils.NewString(rc.Mobile), "international_mobile", "mobile")
	d.Add(jsonutils.NewString(rc.MobileAreaCode), "international_mobile", "area_code")
	return d, nil
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
	SreceiverUpdateOptions
}

type SreceiverUpdateOptions struct {
	Email              string   `help:"email of receiver"`
	Mobile             string   `help:"mobile of receiver"`
	MobileAreaCode     string   `help:"area code of mobile"`
	EnabledContactType []string `help:"enabled contact type"`
}

func (ru *ReceiverUpdateOptions) Params() (jsonutils.JSONObject, error) {
	d := jsonutils.NewDict()
	if len(ru.Email) > 0 {
		d.Set("email", jsonutils.NewString(ru.Email))
	}
	d.Set("enabled_contact_types", jsonutils.NewStringArray(ru.EnabledContactType))
	if len(ru.Mobile) > 0 {
		d.Add(jsonutils.NewString(ru.Mobile), "international_mobile", "mobile")
		d.Add(jsonutils.NewString(ru.MobileAreaCode), "international_mobile", "area_code")
	}
	return d, nil
}

type ReceiverTriggerVerifyOptions struct {
	ReceiverOptions
	SreceiverTriggerVerifyOptions
}

type SreceiverTriggerVerifyOptions struct {
	ContactType string `help:"Contact type to trigger verify" choices:"email|mobile"`
}

func (rt *ReceiverTriggerVerifyOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(rt.SreceiverTriggerVerifyOptions), nil
}

type ReceiverVerifyOptions struct {
	ReceiverOptions
	SreceiverVerifyOptions
}

type SreceiverVerifyOptions struct {
	ContactType string `help:"Contact type to trigger verify" choices:"email|mobile"`
	Token       string `help:"Token from verify message sent to you"`
}

func (rv *ReceiverVerifyOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(rv.SreceiverVerifyOptions), nil
}

type ReceiverEnableContactTypeInput struct {
	ReceiverOptions
	SreceiverEnableContactTypeInput
}

func (re *ReceiverEnableContactTypeInput) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(re.SreceiverEnableContactTypeInput), nil
}

type SreceiverEnableContactTypeInput struct {
	EnabledContactTypes []string `help:"Enabled contact types"`
}

type ReceiverIntellijGetOptions struct {
	USERID     string `help:"user id in keystone" json:"user_id"`
	CreateIfNo *bool  `help:"create if receiver with UserId does not exist"`
	SCOPE      string `help:"scope"`
}

func (ri *ReceiverIntellijGetOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(ri), nil
}

type ReceiverGetTypeOptions struct {
	DomainIds []string `help:"View the available notification channels for the domains with these DomainIds"`
	Operation string   `help:"Operation of reduce" choices:"merge|union"`
}

func (rg *ReceiverGetTypeOptions) Params() (jsonutils.JSONObject, error) {
	return jsonutils.Marshal(rg), nil
}
