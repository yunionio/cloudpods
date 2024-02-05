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

package validators

// TODO
//
// email
// uuid
// uri

import (
	"context"
	"database/sql"
	"math"
	"net"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/choices"
)

type ValidatorFunc func(context.Context, *jsonutils.JSONDict) error

type IValidatorBase interface {
	Validate(ctx context.Context, data *jsonutils.JSONDict) error
}

type IValidator interface {
	IValidatorBase
	Optional(bool) IValidator
	getValue() interface{}
	setDefault(data *jsonutils.JSONDict) bool
}

type Validator struct {
	parent     IValidator
	Key        string
	optional   bool
	defaultVal interface{}
	value      jsonutils.JSONObject
}

func (v *Validator) SetParent(parent IValidator) IValidator {
	v.parent = parent
	return parent
}

func (v *Validator) Optional(optional bool) IValidator {
	v.optional = optional
	return v.parent
}

func (v *Validator) Default(defaultVal interface{}) IValidator {
	v.defaultVal = defaultVal
	return v.parent
}

func (v *Validator) getValue() interface{} {
	return nil
}

func (v *Validator) setDefault(data *jsonutils.JSONDict) bool {
	switch v.defaultVal.(type) {
	case string:
		s := v.defaultVal.(string)
		v.value = jsonutils.NewString(s)
		data.Set(v.Key, v.value)
		return true
	case bool:
		b := v.defaultVal.(bool)
		v.value = jsonutils.NewBool(b)
		data.Set(v.Key, v.value)
		return true
	case int, int32, int64, uint, uint32, uint64:
		value := reflect.ValueOf(v.defaultVal)
		value64 := value.Convert(gotypes.Int64Type)
		defaultVal64 := value64.Interface().(int64)
		v.value = jsonutils.NewInt(defaultVal64)
		data.Set(v.Key, v.value)
		return true
	}
	return false
}

func (v *Validator) Validate(data *jsonutils.JSONDict) error {
	err, _ := v.validateEx(data)
	return err
}

func (v *Validator) validateEx(data *jsonutils.JSONDict) (err error, isSet bool) {
	if !data.Contains(v.Key) {
		if v.defaultVal != nil {
			isSet = v.parent.setDefault(data)
			return nil, isSet
		}
		if !v.optional {
			err = newMissingKeyError(v.Key)
			return
		}
		return
	}
	value, err := data.Get(v.Key)
	if err != nil {
		return
	}
	v.value = value
	isSet = true
	return
}

type ValidatorIPv4Prefix struct {
	Validator
	Value netutils.IPV4Prefix
}

func NewIPv4PrefixValidator(key string) *ValidatorIPv4Prefix {
	v := &ValidatorIPv4Prefix{
		Validator: Validator{Key: key},
	}
	v.SetParent(v)
	return v
}

func (v *ValidatorIPv4Prefix) Validate(ctx context.Context, data *jsonutils.JSONDict) error {
	if err, isSet := v.Validator.validateEx(data); err != nil || !isSet {
		return err
	}
	s, err := v.value.GetString()
	if err != nil {
		return newGeneralError(v.Key, err)
	}
	v.Value, err = netutils.NewIPV4Prefix(s)
	if err != nil {
		return err
	}
	data.Set(v.Key, jsonutils.NewString(s))
	return nil
}

type ValidatorIntChoices struct {
	Validator
	choices []int64

	Value int64
}

func NewIntChoicesValidator(key string, choices []int64) *ValidatorIntChoices {
	v := &ValidatorIntChoices{
		Validator: Validator{Key: key},
		choices:   choices,
	}
	v.SetParent(v)
	return v
}

func (v *ValidatorIntChoices) has(i int64) bool {
	for _, c := range v.choices {
		if c == i {
			return true
		}
	}
	return false
}

func (v *ValidatorIntChoices) Default(i int64) IValidator {
	if v.has(i) {
		v.Validator.Default(i)
		return v
	}
	panic("invalid default for " + v.Key)
}

func (v *ValidatorIntChoices) getValue() interface{} {
	return v.Value
}

func (v *ValidatorIntChoices) Validate(ctx context.Context, data *jsonutils.JSONDict) error {
	if err, isSet := v.Validator.validateEx(data); err != nil || !isSet {
		return err
	}
	i, err := v.value.Int()
	if err != nil {
		return newGeneralError(v.Key, err)
	}
	if !v.has(i) {
		return newInvalidIntChoiceError(v.Key, v.choices, i)
	}
	// in case it's stringified from v.value
	data.Set(v.Key, jsonutils.NewInt(i))
	v.Value = i
	return nil
}

type ValidatorStringChoices struct {
	Validator
	Choices choices.Choices
	Value   string
}

func NewStringChoicesValidator(key string, choices choices.Choices) *ValidatorStringChoices {
	v := &ValidatorStringChoices{
		Validator: Validator{Key: key},
		Choices:   choices,
	}
	v.SetParent(v)
	return v
}

func (v *ValidatorStringChoices) Default(s string) IValidator {
	if v.Choices.Has(s) {
		v.Validator.Default(s)
		return v
	}
	panic("invalid default for " + v.Key)
}

func (v *ValidatorStringChoices) getValue() interface{} {
	return v.Value
}

func (v *ValidatorStringChoices) Validate(ctx context.Context, data *jsonutils.JSONDict) error {
	if err, isSet := v.Validator.validateEx(data); err != nil || !isSet {
		return err
	}
	s, err := v.value.GetString()
	if err != nil {
		return newGeneralError(v.Key, err)
	}
	if !v.Choices.Has(s) {
		return newInvalidChoiceError(v.Key, v.Choices, s)
	}
	// in case it's stringified from v.value
	data.Set(v.Key, jsonutils.NewString(s))
	v.Value = s
	return nil
}

type ValidatorStringMultiChoices struct {
	Validator
	Choices choices.Choices
	Value   string
	sep     string
	keepDup bool
}

func NewStringMultiChoicesValidator(key string, choices choices.Choices) *ValidatorStringMultiChoices {
	v := &ValidatorStringMultiChoices{
		Validator: Validator{Key: key},
		Choices:   choices,
	}
	v.SetParent(v)
	return v
}

func (v *ValidatorStringMultiChoices) Sep(s string) *ValidatorStringMultiChoices {
	v.sep = s
	return v
}

func (v *ValidatorStringMultiChoices) KeepDup(b bool) *ValidatorStringMultiChoices {
	v.keepDup = b
	return v
}

func (v *ValidatorStringMultiChoices) validateString(s string) (string, bool) {
	choices := strings.Split(s, v.sep)
	j := 0
	for i, choice := range choices {
		if !v.Choices.Has(choice) {
			return "", false
		}
		if !v.keepDup {
			isDup := false
			for k := 0; k < j; k++ {
				if choices[k] == choices[i] {
					isDup = true
				}
			}
			if !isDup {
				choices[j] = choices[i]
				j += 1
			}
		}
	}
	if !v.keepDup {
		choices = choices[:j]
	}
	s = strings.Join(choices, v.sep)
	return s, true
}

func (v *ValidatorStringMultiChoices) Default(s string) IValidator {
	s, ok := v.validateString(s)
	if !ok {
		panic("invalid default for " + v.Key)
	}
	v.Validator.Default(s)
	return v
}

func (v *ValidatorStringMultiChoices) getValue() interface{} {
	return v.Value
}

func (v *ValidatorStringMultiChoices) Validate(ctx context.Context, data *jsonutils.JSONDict) error {
	if err, isSet := v.Validator.validateEx(data); err != nil || !isSet {
		return err
	}
	s, err := v.value.GetString()
	if err != nil {
		return newGeneralError(v.Key, err)
	}
	s, ok := v.validateString(s)
	if !ok {
		return newInvalidChoiceError(v.Key, v.Choices, s)
	}
	// in case it's stringified from v.value
	data.Set(v.Key, jsonutils.NewString(s))
	v.Value = s
	return nil
}

type ValidatorBool struct {
	Validator
	Value bool
}

func (v *ValidatorBool) getValue() interface{} {
	return v.Value
}

func (v *ValidatorBool) Default(i bool) IValidator {
	return v.Validator.Default(i)
}
func (v *ValidatorBool) Validate(ctx context.Context, data *jsonutils.JSONDict) error {
	if err, isSet := v.Validator.validateEx(data); err != nil || !isSet {
		return err
	}
	i, err := v.value.Bool()
	if err != nil {
		return newInvalidTypeError(v.Key, "bool", err)
	}
	data.Set(v.Key, jsonutils.NewBool(i))
	v.Value = i
	return nil
}

func NewBoolValidator(key string) *ValidatorBool {
	v := &ValidatorBool{
		Validator: Validator{Key: key},
	}
	v.SetParent(v)
	return v
}

type ValidatorRange struct {
	Validator
	Lower int64
	Upper int64
	Value int64
}

func (v *ValidatorRange) getValue() interface{} {
	return v.Value
}

func (v *ValidatorRange) Default(i int64) IValidator {
	if i >= v.Lower && i <= v.Upper {
		return v.Validator.Default(i)
	}
	panic("invalid default for " + v.Key)
}
func (v *ValidatorRange) Validate(ctx context.Context, data *jsonutils.JSONDict) error {
	if err, isSet := v.Validator.validateEx(data); err != nil || !isSet {
		return err
	}
	i, err := v.value.Int()
	if err != nil {
		return newInvalidTypeError(v.Key, "integer", err)
	}
	if i < v.Lower || i > v.Upper {
		return newNotInRangeError(v.Key, i, v.Lower, v.Upper)
	}
	data.Set(v.Key, jsonutils.NewInt(i))
	v.Value = i
	return nil
}

func NewRangeValidator(key string, lower int64, upper int64) *ValidatorRange {
	v := &ValidatorRange{
		Validator: Validator{Key: key},
		Lower:     lower,
		Upper:     upper,
	}
	v.SetParent(v)
	return v
}

func NewPortValidator(key string) *ValidatorRange {
	return NewRangeValidator(key, 1, 65535)
}

func NewVlanIdValidator(key string) *ValidatorRange {
	// The convention is vendor specific
	//
	// 0, 4095: reserved
	// 1: no vlan tagging
	return NewRangeValidator(key, 1, 4094)
}

func NewNonNegativeValidator(key string) *ValidatorRange {
	return NewRangeValidator(key, 0, math.MaxInt64)
}

type ValidatorModelIdOrName struct {
	Validator
	ModelKeyword string
	OwnerId      mcclient.IIdentityProvider
	ModelManager db.IModelManager
	Model        db.IModel

	modelIdKey       string
	noPendingDeleted bool
	allowEmpty       bool
}

func (v *ValidatorModelIdOrName) GetProjectId() string {
	return v.OwnerId.GetProjectId()
}

func (v *ValidatorModelIdOrName) GetUserId() string {
	return v.OwnerId.GetUserId()
}

func (v *ValidatorModelIdOrName) GetTenantId() string {
	return v.OwnerId.GetTenantId()
}

func (v *ValidatorModelIdOrName) GetProjectDomainId() string {
	return v.OwnerId.GetProjectDomainId()
}

func (v *ValidatorModelIdOrName) GetUserName() string {
	return v.OwnerId.GetUserName()
}

func (v *ValidatorModelIdOrName) GetProjectName() string {
	return v.OwnerId.GetProjectName()
}

func (v *ValidatorModelIdOrName) GetTenantName() string {
	return v.OwnerId.GetTenantName()
}

func (v *ValidatorModelIdOrName) GetProjectDomain() string {
	return v.OwnerId.GetProjectDomain()
}

func (v *ValidatorModelIdOrName) GetDomainId() string {
	return v.OwnerId.GetDomainId()
}

func (v *ValidatorModelIdOrName) GetDomainName() string {
	return v.OwnerId.GetDomainName()
}

func (v *ValidatorModelIdOrName) getValue() interface{} {
	return v.Model
}

func NewModelIdOrNameValidator(key string, modelKeyword string, ownerId mcclient.IIdentityProvider) *ValidatorModelIdOrName {
	v := &ValidatorModelIdOrName{
		Validator:        Validator{Key: key},
		OwnerId:          ownerId,
		ModelKeyword:     modelKeyword,
		modelIdKey:       key + "_id",
		noPendingDeleted: true,
	}
	v.SetParent(v)
	return v
}

func (v *ValidatorModelIdOrName) ModelIdKey(modelIdKey string) *ValidatorModelIdOrName {
	v.modelIdKey = modelIdKey
	return v
}

// AllowPendingDeleted allows the to-be-validated id or name to be of a pending deleted model
func (v *ValidatorModelIdOrName) AllowPendingDeleted(b bool) *ValidatorModelIdOrName {
	v.noPendingDeleted = !b
	return v
}

func (v *ValidatorModelIdOrName) AllowEmpty(b bool) *ValidatorModelIdOrName {
	v.allowEmpty = b
	return v
}

func (v *ValidatorModelIdOrName) validate(ctx context.Context, data *jsonutils.JSONDict) error {
	if !data.Contains(v.Key) && data.Contains(v.modelIdKey) {
		// a hack when validator is used solely for fetching model
		// object.  This can happen when input json data was validated
		// more than once in different places
		j, err := data.Get(v.modelIdKey)
		if err != nil {
			return err
		}
		data.Set(v.Key, j)
		defer func() {
			// restore
			data.Remove(v.Key)
		}()
	}
	if err, isSet := v.Validator.validateEx(data); err != nil || !isSet {
		return err
	}
	modelIdOrName, err := v.value.GetString()
	if err != nil {
		return err
	}
	if modelIdOrName == "" && v.allowEmpty {
		return nil
	}

	modelManager := db.GetModelManager(v.ModelKeyword)
	if modelManager == nil {
		return newModelManagerError(v.ModelKeyword)
	}
	v.ModelManager = modelManager
	model, err := modelManager.FetchByIdOrName(ctx, v, modelIdOrName)
	if err != nil {
		if err == sql.ErrNoRows {
			return newModelNotFoundError(v.ModelKeyword, modelIdOrName, err)
		} else {
			return httperrors.NewGeneralError(err)
		}
	}
	if v.noPendingDeleted {
		if pd, ok := model.(db.IPendingDeletable); ok && pd.GetPendingDeleted() {
			return newModelNotFoundError(v.ModelKeyword, modelIdOrName, nil)
		}
	}
	v.Model = model
	return nil
}

func (v *ValidatorModelIdOrName) Validate(ctx context.Context, data *jsonutils.JSONDict) error {
	err := v.validate(ctx, data)
	if err != nil {
		return err
	}
	var val string
	if v.Model != nil {
		val = v.Model.GetId()
	}
	if v.modelIdKey != "" {
		if data.Contains(v.Key) {
			data.Remove(v.Key)
		}
		if val != "" {
			data.Set(v.modelIdKey, jsonutils.NewString(val))
		}
	}
	return nil
}

func (v *ValidatorModelIdOrName) QueryFilter(ctx context.Context, q *sqlchemy.SQuery, data *jsonutils.JSONDict) (*sqlchemy.SQuery, error) {
	err := v.validate(ctx, data)
	if err != nil {
		if IsModelNotFoundError(err) {
			// hack
			q = q.Equals(v.modelIdKey, "0")
			q = q.Equals(v.modelIdKey, "1")
			return q, nil
		}
		return nil, err
	}
	if v.Model != nil {
		q = q.Equals(v.modelIdKey, v.Model.GetId())
	}
	return q, nil
}

type ValidatorRegexp struct {
	Validator
	Regexp     *regexp.Regexp
	Value      string
	allowEmpty bool
}

func (v *ValidatorRegexp) AllowEmpty(allowEmpty bool) *ValidatorRegexp {
	v.allowEmpty = allowEmpty
	return v
}

func (v *ValidatorRegexp) getValue() interface{} {
	return v.Value
}

func (v *ValidatorRegexp) Validate(ctx context.Context, data *jsonutils.JSONDict) error {
	if err, isSet := v.Validator.validateEx(data); err != nil || !isSet {
		return err
	}
	value, err := v.value.GetString()
	if err != nil {
		return newInvalidTypeError(v.Key, "string", err)
	}
	if v.allowEmpty && len(value) == 0 {
		return nil
	}
	if !v.Regexp.MatchString(value) {
		return newInvalidValueError(v.Key, value)
	}
	v.Value = value
	return nil
}

func NewRegexpValidator(key string, regexp *regexp.Regexp) *ValidatorRegexp {
	v := &ValidatorRegexp{
		Validator: Validator{Key: key},
		Regexp:    regexp,
	}
	v.SetParent(v)
	return v
}

type ValidatorDomainName struct {
	ValidatorRegexp
}

func NewDomainNameValidator(key string) *ValidatorDomainName {
	v := &ValidatorDomainName{
		ValidatorRegexp: *NewRegexpValidator(key, regutils.DOMAINNAME_REG),
	}
	v.SetParent(v)
	return v
}

type ValidatorHostPort struct {
	ValidatorRegexp

	optionalPort bool
	Domain       string
	Port         int
	Value        string
}

var regHostPort *regexp.Regexp

func init() {
	// guard against surprise
	exp := regutils.DOMAINNAME_REG.String()
	if exp != "" && exp[len(exp)-1] == '$' {
		exp = exp[:len(exp)-1]
	}
	exp += "(?::[0-9]{1,5})?"
	regHostPort = regexp.MustCompile(exp)
}

func NewHostPortValidator(key string) *ValidatorHostPort {
	v := &ValidatorHostPort{
		ValidatorRegexp: *NewRegexpValidator(key, regHostPort),
	}
	v.SetParent(v)
	return v
}

func (v *ValidatorHostPort) getValue() interface{} {
	return v.Value
}

func (v *ValidatorHostPort) OptionalPort(optionalPort bool) *ValidatorHostPort {
	v.optionalPort = optionalPort
	return v
}

func (v *ValidatorHostPort) Validate(ctx context.Context, data *jsonutils.JSONDict) error {
	err := v.ValidatorRegexp.Validate(ctx, data)
	if err != nil {
		return err
	}
	hostPort := v.ValidatorRegexp.Value
	if hostPort == "" && (v.optional || v.allowEmpty) {
		return nil
	}
	i := strings.IndexRune(hostPort, ':')
	if i < 0 {
		if v.optionalPort {
			v.Value = hostPort
			v.Domain = hostPort
			return nil
		}
		return newInvalidValueError(v.Key, "port missing")
	}
	portStr := hostPort[i+1:]
	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return newInvalidValueError(v.Key, "bad port integer: "+err.Error())
	}
	if port <= 0 {
		return newInvalidValueError(v.Key, "negative port")
	}
	v.Value = hostPort
	v.Domain = hostPort[:i]
	v.Port = int(port)
	return nil
}

type ValidatorURLPath struct {
	ValidatorRegexp
}

// URI Path as defined in https://tools.ietf.org/html/rfc3986#section-3.3
var regexpURLPath = regexp.MustCompile(`^(?:/[a-zA-Z0-9.%$&'()*+,;=!~_-]*)*$`)

func NewURLPathValidator(key string) *ValidatorURLPath {
	v := &ValidatorURLPath{
		ValidatorRegexp: *NewRegexpValidator(key, regexpURLPath),
	}
	v.SetParent(v)
	return v
}

type ValidatorStruct struct {
	Validator
	Value interface{}
}

func (v *ValidatorStruct) getValue() interface{} {
	return v.Value
}

func (v *ValidatorStruct) Validate(ctx context.Context, data *jsonutils.JSONDict) error {
	if err, isSet := v.Validator.validateEx(data); err != nil || !isSet {
		return err
	}
	err := v.value.Unmarshal(v.Value)
	if err != nil {
		return newGeneralError(v.Key, err)
	}
	if valueValidator, ok := v.Value.(IValidatorBase); ok {
		err = valueValidator.Validate(ctx, data)
		if err != nil {
			return newInvalidStructError(v.Key, err)
		}
	}
	data.Set(v.Key, jsonutils.Marshal(v.Value))
	return nil
}

func NewStructValidator(key string, value interface{}) *ValidatorStruct {
	v := &ValidatorStruct{
		Validator: Validator{Key: key},
		Value:     value,
	}
	v.SetParent(v)
	return v
}

type ValidatorIPv4Addr struct {
	Validator
	IP net.IP
}

func (v *ValidatorIPv4Addr) getValue() interface{} {
	return v.IP
}

func (v *ValidatorIPv4Addr) setDefault(data *jsonutils.JSONDict) bool {
	if v.defaultVal == nil {
		return false
	}
	defaultIP, ok := v.defaultVal.(net.IP)
	if !ok {
		return false
	}
	value := jsonutils.NewString(defaultIP.String())
	v.value = value
	data.Set(v.Key, value)
	return true
}

func (v *ValidatorIPv4Addr) Validate(ctx context.Context, data *jsonutils.JSONDict) error {
	if err, isSet := v.Validator.validateEx(data); err != nil || !isSet {
		return err
	}
	s, err := v.value.GetString()
	if err != nil {
		return newInvalidTypeError(v.Key, "string", err)
	}
	ip := net.ParseIP(s).To4()
	if ip == nil {
		return newInvalidValueError(v.Key, s)
	}
	v.IP = ip
	return nil
}

func NewIPv4AddrValidator(key string) *ValidatorIPv4Addr {
	v := &ValidatorIPv4Addr{
		Validator: Validator{Key: key},
	}
	v.SetParent(v)
	return v
}

var ValidateModel = func(ctx context.Context, userCred mcclient.TokenCredential, manager db.IStandaloneModelManager, id *string) (db.IModel, error) {
	if len(*id) == 0 {
		return nil, httperrors.NewMissingParameterError(manager.Keyword() + "_id")
	}

	model, err := manager.FetchByIdOrName(ctx, userCred, *id)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2(manager.Keyword(), *id)
		}
		if errors.Cause(err) == sqlchemy.ErrDuplicateEntry {
			return nil, httperrors.NewDuplicateResourceError(manager.Keyword(), *id)
		}
		return nil, httperrors.NewGeneralError(err)
	}
	*id = model.GetId()
	return model, nil
}
