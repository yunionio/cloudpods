package models

import (
	"context"
	"fmt"
	"regexp"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLoadbalancerListenerManager struct {
	db.SVirtualResourceBaseManager
}

var LoadbalancerListenerManager *SLoadbalancerListenerManager

func init() {
	LoadbalancerListenerManager = &SLoadbalancerListenerManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SLoadbalancerListener{},
			"loadbalancerlisteners_tbl",
			"loadbalancerlistener",
			"loadbalancerlisteners",
		),
	}
}

type SLoadbalancerTCPListener struct{}
type SLoadbalancerUDPListener struct{}

// TODO sensible default for knobs
type SLoadbalancerHTTPListener struct {
	StickySession              string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`
	StickySessionType          string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`
	StickySessionCookie        string `width:"128" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`
	StickySessionCookieTimeout int    `nullable:"false" list:"user" create:"optional" update:"user"`

	//XForwardedForSLBIP bool `nullable:"false" list:"user" create:"optional"`
	//XForwardedForSLBID bool `nullable:"false" list:"user" create:"optional"`
	XForwardedFor bool `nullable:"false" list:"user" create:"optional" update:"user"`
	Gzip          bool `nullable:"false" list:"user" create:"optional" update:"user"`
}

// TODO
//
//  - CACertificate string
//  - Certificate2Id // multiple certificates for rsa, ecdsa
//  - Use certificate for tcp listener
//  - Customize ciphers?
type SLoadbalancerHTTPSListener struct {
	CertificateId   string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	TLSCipherPolicy string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	EnableHttp2     bool   `create:"optional" list:"user"`
}

type SLoadbalancerListener struct {
	db.SVirtualResourceBase

	LoadbalancerId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional"`
	ListenerType   string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required"`
	ListenerPort   int    `nullable:"false" list:"user" create:"required"`
	BackendGroupId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`

	Bandwidth int    `nullable:"false" list:"user" create:"optional" update:"user"`
	Scheduler string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"required" update:"user"`

	ClientRequestTimeout  int `nullable:"false" list:"user" create:"optional" update:"user"`
	ClientIdleTimeout     int `nullable:"false" list:"user" create:"optional" update:"user"`
	BackendConnectTimeout int `nullable:"false" list:"user" create:"optional" update:"user"`
	BackendIdleTimeout    int `nullable:"false" list:"user" create:"optional" update:"user"`

	AclStatus string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`
	AclType   string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`
	AclId     string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`

	HealthCheck     string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`
	HealthCheckType string `width:"16" charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`

	HealthCheckDomain   string `charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`
	HealthCheckURI      string `charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`
	HealthCheckHttpCode string `charset:"ascii" nullable:"false" list:"user" create:"optional" update:"user"`

	HealthCheckRise     int `nullable:"false" list:"user" create:"optional" update:"user"`
	HealthCheckFall     int `nullable:"false" list:"user" create:"optional" update:"user"`
	HealthCheckTimeout  int `nullable:"false" list:"user" create:"optional" update:"user"`
	HealthCheckInterval int `nullable:"false" list:"user" create:"optional" update:"user"`

	HealthCheckReq string `list:"user" create:"optional" update:"user"`
	HealthCheckExp string `list:"user" create:"optional" update:"user"`

	SLoadbalancerTCPListener
	SLoadbalancerUDPListener
	SLoadbalancerHTTPListener
	SLoadbalancerHTTPSListener
}

func (man *SLoadbalancerListenerManager) checkListenerUniqueness(ctx context.Context, lb *SLoadbalancer, listenerType string, listenerPort int64) error {
	q := man.Query().
		Equals("loadbalancer_id", lb.Id).
		Equals("listener_port", listenerPort)
	switch listenerType {
	case LB_LISTENER_TYPE_TCP, LB_LISTENER_TYPE_HTTP, LB_LISTENER_TYPE_HTTPS:
		q = q.NotEquals("listener_type", LB_LISTENER_TYPE_UDP)
	case LB_LISTENER_TYPE_UDP:
		q = q.Equals("listener_type", LB_LISTENER_TYPE_UDP)
	default:
		return fmt.Errorf("unexpected listener type: %s", listenerType)
	}
	var listener SLoadbalancerListener
	q.First(&listener)
	if len(listener.Id) > 0 {
		return fmt.Errorf("%s listener port %d is already taken by listener %s",
			listenerType, listenerPort, listener.Id)
	}
	return nil
}

func (man *SLoadbalancerListenerManager) PreDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery) {
	subs := []SLoadbalancerListener{}
	db.FetchModelObjects(man, q, &subs)
	for _, sub := range subs {
		sub.PreDelete(ctx, userCred)
	}
}

func (man *SLoadbalancerListenerManager) ListItemFilter(ctx context.Context, q *sqlchemy.SQuery, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (*sqlchemy.SQuery, error) {
	q, err := man.SVirtualResourceBaseManager.ListItemFilter(ctx, q, userCred, query)
	if err != nil {
		return nil, err
	}
	userProjId := userCred.GetProjectId()
	data := query.(*jsonutils.JSONDict)
	{
		lbV := validators.NewModelIdOrNameValidator("loadbalancer", "loadbalancer", userProjId)
		lbV.Optional(true)
		q, err = lbV.QueryFilter(q, data)
		if err != nil {
			return nil, err
		}
	}
	{
		backendGroupV := validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", userProjId)
		backendGroupV.Optional(true)
		q, err = backendGroupV.QueryFilter(q, data)
		if err != nil {
			return nil, err
		}
	}
	{
		aclV := validators.NewModelIdOrNameValidator("acl", "loadbalanceracl", userProjId)
		aclV.Optional(true)
		q, err = aclV.QueryFilter(q, data)
		if err != nil {
			return nil, err
		}
	}
	return q, nil
}

func (man *SLoadbalancerListenerManager) ValidateCreateData(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	lbV := validators.NewModelIdOrNameValidator("loadbalancer", "loadbalancer", ownerProjId)
	listenerTypeV := validators.NewStringChoicesValidator("listener_type", LB_LISTENER_TYPES)
	listenerPortV := validators.NewPortValidator("listener_port")
	backendGroupV := validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", ownerProjId)
	aclStatusV := validators.NewStringChoicesValidator("acl_status", LB_BOOL_VALUES)
	aclTypeV := validators.NewStringChoicesValidator("acl_type", LB_ACL_TYPES)
	aclV := validators.NewModelIdOrNameValidator("acl", "loadbalanceracl", ownerProjId)
	keyV := map[string]validators.IValidator{
		"status": validators.NewStringChoicesValidator("status", LB_STATUS_SPEC).Default(LB_STATUS_ENABLED),

		"loadbalancer":  lbV,
		"listener_type": listenerTypeV,
		"listener_port": listenerPortV,
		"backend_group": backendGroupV.Optional(true),

		"acl_status": aclStatusV.Default(LB_BOOL_OFF),
		"acl_type":   aclTypeV.Optional(true),
		"acl":        aclV.Optional(true),

		"scheduler": validators.NewStringChoicesValidator("scheduler", LB_SCHEDULER_TYPES),
		"bandwidth": validators.NewRangeValidator("bandwidth", 0, 10000).Optional(true),

		"client_request_timeout":  validators.NewRangeValidator("client_request_timeout", 0, 600).Default(10),
		"client_idle_timeout":     validators.NewRangeValidator("client_idle_timeout", 0, 600).Default(90),
		"backend_connect_timeout": validators.NewRangeValidator("backend_connect_timeout", 0, 180).Default(5),
		"backend_idle_timeout":    validators.NewRangeValidator("backend_idle_timeout", 0, 600).Default(90),

		"sticky_session":                validators.NewStringChoicesValidator("sticky_session", LB_BOOL_VALUES).Default(LB_BOOL_OFF),
		"sticky_session_type":           validators.NewStringChoicesValidator("sticky_session_type", LB_STICKY_SESSION_TYPES).Default(LB_STICKY_SESSION_TYPE_INSERT),
		"sticky_session_cookie":         validators.NewRegexpValidator("sticky_session_cookie", regexp.MustCompile(`\w+`)).Optional(true),
		"sticky_session_cookie_timeout": validators.NewNonNegativeValidator("sticky_session_cookie_timeout").Optional(true),

		"x_forwarded_for": validators.NewBoolValidator("x_forwarded_for").Default(true),
		"gzip":            validators.NewBoolValidator("gzip").Default(false),
	}
	for _, v := range keyV {
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	{
		lb := lbV.Model.(*SLoadbalancer)
		listenerPort := listenerPortV.Value
		listenerType := listenerTypeV.Value
		err := man.checkListenerUniqueness(ctx, lb, listenerType, listenerPort)
		if err != nil {
			// duplicate?
			return nil, err
		}
	}
	{
		if listenerTypeV.Value == LB_LISTENER_TYPE_HTTPS {
			certV := validators.NewModelIdOrNameValidator("certificate", "loadbalancercertificate", ownerProjId)
			tlsCipherPolicyV := validators.NewStringChoicesValidator("tls_cipher_policy", LB_TLS_CIPHER_POLICIES).Default(LB_TLS_CIPHER_POLICY_1_2)
			httpsV := map[string]validators.IValidator{
				"certificate":       certV,
				"tls_cipher_policy": tlsCipherPolicyV,
				"enable_http2":      validators.NewBoolValidator("enable_http2").Default(true),
			}
			for _, v := range httpsV {
				if err := v.Validate(data); err != nil {
					return nil, err
				}
			}
		}
	}
	{
		// health check default depends on input parameters
		checkTypeV := man.checkTypeV(listenerTypeV.Value)
		keyVHealth := map[string]validators.IValidator{
			"health_check":      validators.NewStringChoicesValidator("health_check", LB_BOOL_VALUES).Default(LB_BOOL_ON),
			"health_check_type": checkTypeV,

			"health_check_domain":    validators.NewDomainNameValidator("domain").AllowEmpty(true).Default(""),
			"health_check_path":      validators.NewURLPathValidator("path").Default(""),
			"health_check_http_code": validators.NewStringMultiChoicesValidator("health_check_http_code", LB_HEALTH_CHECK_HTTP_CODES).Sep(",").Default(LB_HEALTH_CHECK_HTTP_CODE_DEFAULT),

			"health_check_rise":     validators.NewRangeValidator("health_check_rise", 1, 1000).Default(3),
			"health_check_fall":     validators.NewRangeValidator("health_check_fall", 1, 1000).Default(3),
			"health_check_timeout":  validators.NewRangeValidator("health_check_timeout", 1, 300).Default(5),
			"health_check_interval": validators.NewRangeValidator("health_check_interval", 1, 1000).Default(5),
		}
		for _, v := range keyVHealth {
			if err := v.Validate(data); err != nil {
				return nil, err
			}
		}
	}
	{
		if aclStatusV.Value == LB_BOOL_ON {
			acl := aclV.Model.(*SLoadbalancerAcl)
			if acl == nil {
				return nil, fmt.Errorf("missing acl")
			}
			if len(aclTypeV.Value) == 0 {
				return nil, fmt.Errorf("missing acl_type")
			}
		}
	}
	return man.SVirtualResourceBaseManager.ValidateCreateData(ctx, userCred, ownerProjId, query, data)
}

func (man *SLoadbalancerListenerManager) checkTypeV(listenerType string) validators.IValidator {
	switch listenerType {
	case LB_LISTENER_TYPE_HTTP, LB_LISTENER_TYPE_HTTPS:
		return validators.NewStringChoicesValidator("health_check_type", LB_HEALTH_CHECK_TYPES_TCP).Default(LB_HEALTH_CHECK_HTTP)
	case LB_LISTENER_TYPE_TCP:
		return validators.NewStringChoicesValidator("health_check_type", LB_HEALTH_CHECK_TYPES_TCP).Default(LB_HEALTH_CHECK_TCP)
	case LB_LISTENER_TYPE_UDP:
		return validators.NewStringChoicesValidator("health_check_type", LB_HEALTH_CHECK_TYPES_UDP).Default(LB_HEALTH_CHECK_UDP)
	}
	// should it happen, panic then
	return nil
}

func (lblis *SLoadbalancerListener) AllowPerformStatus(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) bool {
	return lblis.IsOwner(userCred) || userCred.IsSystemAdmin()
}

func (lblis *SLoadbalancerListener) ValidateUpdateData(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data *jsonutils.JSONDict) (*jsonutils.JSONDict, error) {
	ownerProjId := lblis.GetOwnerProjectId()
	backendGroupV := validators.NewModelIdOrNameValidator("backend_group", "loadbalancerbackendgroup", ownerProjId)
	aclStatusV := validators.NewStringChoicesValidator("acl_status", LB_BOOL_VALUES)
	aclTypeV := validators.NewStringChoicesValidator("acl_type", LB_ACL_TYPES)
	aclV := validators.NewModelIdOrNameValidator("acl", "loadbalanceracl", ownerProjId)
	certV := validators.NewModelIdOrNameValidator("certificate", "loadbalancercertificate", ownerProjId)
	tlsCipherPolicyV := validators.NewStringChoicesValidator("tls_cipher_policy", LB_TLS_CIPHER_POLICIES).Default(LB_TLS_CIPHER_POLICY_1_2)
	keyV := map[string]validators.IValidator{
		"backend_group": backendGroupV,

		"acl_status": aclStatusV,
		"acl_type":   aclTypeV,
		"acl":        aclV,

		"scheduler": validators.NewStringChoicesValidator("scheduler", LB_SCHEDULER_TYPES),
		"bandwidth": validators.NewRangeValidator("bandwidth", 0, 10000),

		"client_request_timeout":  validators.NewRangeValidator("client_request_timeout", 0, 600),
		"client_idle_timeout":     validators.NewRangeValidator("client_idle_timeout", 0, 600),
		"backend_connect_timeout": validators.NewRangeValidator("backend_connect_timeout", 0, 180),
		"backend_idle_timeout":    validators.NewRangeValidator("backend_idle_timeout", 0, 600),

		"sticky_session":                validators.NewStringChoicesValidator("sticky_session", LB_BOOL_VALUES),
		"sticky_session_type":           validators.NewStringChoicesValidator("sticky_session_type", LB_STICKY_SESSION_TYPES),
		"sticky_session_cookie":         validators.NewRegexpValidator("sticky_session_cookie", regexp.MustCompile(`\w+`)),
		"sticky_session_cookie_timeout": validators.NewNonNegativeValidator("sticky_session_cookie_timeout"),

		"health_check":      validators.NewStringChoicesValidator("health_check", LB_BOOL_VALUES),
		"health_check_type": LoadbalancerListenerManager.checkTypeV(lblis.ListenerType),

		"health_check_domain":    validators.NewDomainNameValidator("domain").AllowEmpty(true),
		"health_check_path":      validators.NewURLPathValidator("path"),
		"health_check_http_code": validators.NewStringMultiChoicesValidator("health_check_http_code", LB_HEALTH_CHECK_HTTP_CODES).Sep(","),

		"health_check_rise":     validators.NewRangeValidator("health_check_rise", 1, 1000),
		"health_check_fall":     validators.NewRangeValidator("health_check_fall", 1, 1000),
		"health_check_timeout":  validators.NewRangeValidator("health_check_timeout", 1, 300),
		"health_check_interval": validators.NewRangeValidator("health_check_interval", 1, 1000),

		"x_forwarded_for": validators.NewBoolValidator("x_forwarded_for"),
		"gzip":            validators.NewBoolValidator("gzip"),

		"certificate":       certV,
		"tls_cipher_policy": tlsCipherPolicyV,
		"enable_http2":      validators.NewBoolValidator("enable_http2").Default(true),
	}
	for _, v := range keyV {
		v.Optional(true)
		if err := v.Validate(data); err != nil {
			return nil, err
		}
	}
	return lblis.SVirtualResourceBase.ValidateUpdateData(ctx, userCred, query, data)
}

func (lblis *SLoadbalancerListener) GetCustomizeColumns(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := lblis.SVirtualResourceBase.GetCustomizeColumns(ctx, userCred, query)
	if lblis.BackendGroupId == "" {
		return extra
	}
	lbbg, err := LoadbalancerBackendGroupManager.FetchById(lblis.BackendGroupId)
	if err != nil {
		log.Errorf("loadbalancer listener %s(%s): fetch backend group (%s) error: %s",
			lblis.Name, lblis.Id, lblis.BackendGroupId, err)
		return extra
	}
	extra.Set("backend_group", jsonutils.NewString(lbbg.GetName()))
	return extra
}

func (lblis *SLoadbalancerListener) GetExtraDetails(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) *jsonutils.JSONDict {
	extra := lblis.GetCustomizeColumns(ctx, userCred, query)
	return extra
}

func (lblis *SLoadbalancerListener) PreDelete(ctx context.Context, userCred mcclient.TokenCredential) {
	lblis.SetStatus(userCred, LB_STATUS_DISABLED, "preDelete")
	lblis.DoPendingDelete(ctx, userCred)
	lblis.PreDeleteSubs(ctx, userCred)
}

func (lblis *SLoadbalancerListener) PreDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential) {
	subMan := LoadbalancerListenerRuleManager
	ownerProjId := lblis.GetOwnerProjectId()

	lockman.LockClass(ctx, subMan, ownerProjId)
	defer lockman.ReleaseClass(ctx, subMan, ownerProjId)
	q := subMan.Query().Equals("listener_id", lblis.Id)
	subMan.PreDeleteSubs(ctx, userCred, q)
}

func (lblis *SLoadbalancerListener) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return nil
}
