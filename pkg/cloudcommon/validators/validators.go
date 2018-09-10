package validators

// TODO
//
// email
// uuid
// uri

import (
	"math"
	"net"
	"reflect"
	"regexp"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/pkg/gotypes"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/sqlchemy"
)

type ValidatorFunc func(*jsonutils.JSONDict) error

type IValidatorBase interface {
	Validate(data *jsonutils.JSONDict) error
}

type IValidator interface {
	IValidatorBase
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

type ValidatorStringChoices struct {
	Validator
	Choices    Choices
	defaultVal string
	Value      string
}

func NewStringChoicesValidator(key string, choices Choices) *ValidatorStringChoices {
	v := &ValidatorStringChoices{
		Validator: Validator{Key: key},
		Choices:   choices,
	}
	v.parent = v
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
	Choices    Choices
	defaultVal string
	Value      string
	sep        string
	keepDup    bool
}

func NewStringMultiChoicesValidator(key string, choices Choices) *ValidatorStringMultiChoices {
	v := &ValidatorStringMultiChoices{
		Validator: Validator{Key: key},
		Choices:   choices,
	}
	v.parent = v
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
	v.parent = v
	return v
}

func NewPortValidator(key string) *ValidatorRange {
	return NewRangeValidator(key, 1, 65535)
}

func NewNonNegativeValidator(key string) *ValidatorRange {
	return NewRangeValidator(key, 0, math.MaxInt64)
}

type ValidatorModelIdOrName struct {
	Validator
	ModelKeyword string
	ProjectId    string
	ModelManager db.IModelManager
	Model        db.IModel
	modelIdKey   string
}

func (v *ValidatorModelIdOrName) getValue() interface{} {
	return v.Model
}

func NewModelIdOrNameValidator(key string, modelKeyword string, projectId string) *ValidatorModelIdOrName {
	v := &ValidatorModelIdOrName{
		Validator:    Validator{Key: key},
		ProjectId:    projectId,
		ModelKeyword: modelKeyword,
		modelIdKey:   key + "_id",
	}
	v.parent = v
	return v
}

func (v *ValidatorModelIdOrName) ModelIdKey(modelIdKey string) *ValidatorModelIdOrName {
	v.modelIdKey = modelIdKey
	return v
}

func (v *ValidatorModelIdOrName) validate(data *jsonutils.JSONDict) error {
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
	model, err := modelManager.FetchByIdOrName(v.ProjectId, modelIdOrName)
	if err != nil {
		return newModelNotFoundError(v.ModelKeyword, modelIdOrName, err)
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
	v.parent = v
	return v
}

type ValidatorDomainName struct {
	ValidatorRegexp
}

func NewDomainNameValidator(key string) *ValidatorDomainName {
	v := &ValidatorDomainName{
		ValidatorRegexp: *NewRegexpValidator(key, regutils.DOMAINNAME_REG),
	}
	v.parent = v
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
	v.parent = v
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
			return err
		}
	}
	return nil
}

func NewStructValidator(key string, value interface{}) *ValidatorStruct {
	v := &ValidatorStruct{
		Validator: Validator{Key: key},
		Value:     value,
	}
	v.parent = v
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
	v.parent = v
	return v
}
