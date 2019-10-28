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
	"database/sql"
	"math"
	"net"
	"reflect"
	"regexp"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/sqlchemy"

	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/choices"
)

type ValidatorFunc func(*jsonutils.JSONDict) error

type IValidatorBase interface {
	Validate(data *jsonutils.JSONDict) error
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

func (v *ValidatorIPv4Prefix) Validate(data *jsonutils.JSONDict) error {
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

type ValidatorStringChoices struct {
	Validator
	Choices    choices.Choices
	defaultVal string
	Value      string
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
		return v.Validator.Default(s)
	}
	panic("invalid default for " + v.Key)
}

func (v *ValidatorStringChoices) getValue() interface{} {
	return v.Value
}

func (v *ValidatorStringChoices) Validate(data *jsonutils.JSONDict) error {
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
	Choices    choices.Choices
	defaultVal string
	Value      string
	sep        string
	keepDup    bool
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

func (v *ValidatorStringMultiChoices) Validate(data *jsonutils.JSONDict) error {
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
func (v *ValidatorBool) Validate(data *jsonutils.JSONDict) error {
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
func (v *ValidatorRange) Validate(data *jsonutils.JSONDict) error {
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

func (v *ValidatorModelIdOrName) validate(data *jsonutils.JSONDict) error {
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

	modelManager := db.GetModelManager(v.ModelKeyword)
	if modelManager == nil {
		return newModelManagerError(v.ModelKeyword)
	}
	v.ModelManager = modelManager
	model, err := modelManager.FetchByIdOrName(v, modelIdOrName)
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

func (v *ValidatorModelIdOrName) Validate(data *jsonutils.JSONDict) error {
	err := v.validate(data)
	if err != nil {
		return err
	}
	if v.Model != nil {
		if len(v.modelIdKey) > 0 {
			data.Remove(v.Key)
			data.Set(v.modelIdKey, jsonutils.NewString(v.Model.GetId()))
		}
	}
	return nil
}

func (v *ValidatorModelIdOrName) QueryFilter(q *sqlchemy.SQuery, data *jsonutils.JSONDict) (*sqlchemy.SQuery, error) {
	err := v.validate(data)
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

func (v *ValidatorRegexp) Validate(data *jsonutils.JSONDict) error {
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

func (v *ValidatorStruct) Validate(data *jsonutils.JSONDict) error {
	if err, isSet := v.Validator.validateEx(data); err != nil || !isSet {
		return err
	}
	err := v.value.Unmarshal(v.Value)
	if err != nil {
		return newGeneralError(v.Key, err)
	}
	if valueValidator, ok := v.Value.(IValidatorBase); ok {
		err = valueValidator.Validate(data)
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

func (v *ValidatorIPv4Addr) Validate(data *jsonutils.JSONDict) error {
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
