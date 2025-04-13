package validation

import (
	"encoding/json"
	"errors"
	"github.com/go-playground/locales/fa"
	ut "github.com/go-playground/universal-translator"
	"github.com/parsidev/go-validator/locales"
	"gorm.io/gorm/clause"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
	"sync"
)

type CurrentPasswordChecker func(userID uint64, password string) bool

var currentPasswordChecker CurrentPasswordChecker

type Validation struct {
	validator *validator.Validate
	uni       *ut.UniversalTranslator
	database  *gorm.DB
	trans     ut.Translator
}

var (
	once sync.Once
)

func Init(db *gorm.DB) (v *Validation, err error) {
	once.Do(func() {
		persian := fa.New()
		u := ut.New(persian, persian)
		t, _ := u.GetTranslator("fa")

		v = &Validation{
			database:  db,
			uni:       u,
			trans:     t,
			validator: validator.New(),
		}

		if err = v.validator.RegisterValidation("exists", v.exists); err != nil {
			return
		}

		if err = v.validator.RegisterValidation("nullable", v.nullable); err != nil {
			return
		}

		if err = v.validator.RegisterValidation("uq", v.unique); err != nil {
			return
		}

		if err = v.validator.RegisterValidation("current_password", v.currentPassword); err != nil {
			return
		}

		if err = v.validator.RegisterValidation("mobile", v.mobile); err != nil {
			return
		}

		v.validator.RegisterAlias("string", "alphanumunicode|alphaunicode|ascii")

		if err = locales.RegisterDefaultTranslations(v.validator, v.trans); err != nil {
			return
		}
	})

	if err != nil {
		v = nil
		return nil, err
	}

	return v, nil
}

func (v *Validation) mobile(fl validator.FieldLevel) bool {
	return mobileRegex().MatchString(fl.Field().String())
}

func (v *Validation) exists(fl validator.FieldLevel) bool {
	var (
		field  = fl.Field()
		params = fl.Param()
		table  = params
		column = "id"
		count  = 0
		err    error
	)

	if strings.Contains(params, ";") {
		t := strings.Split(params, ";")
		table = t[0]
		column = t[1]
	}

	err = v.database.
		Table(table).
		Select("CASE WHEN COUNT(*) > 0 THEN 1 ELSE 0 END").
		Where(clause.Eq{Column: column, Value: field.String()}).
		Find(&count).Error

	return err == nil && count == 1
}

func (v *Validation) nullable(fl validator.FieldLevel) bool {
	var (
		field = fl.Field()
	)

	return len(field.String()) >= 0
}

func (v *Validation) currentPassword(fl validator.FieldLevel) bool {
	if currentPasswordChecker == nil {
		return false
	}

	password := fl.Field().String()
	parent := fl.Parent()

	userIDField := parent.FieldByName("UserID")
	if !userIDField.IsValid() || userIDField.Kind() != reflect.Uint {
		return false
	}

	userID := userIDField.Uint()
	return currentPasswordChecker(userID, password)
}

func (v *Validation) unique(fl validator.FieldLevel) bool {
	var (
		field  = fl.Field()
		params = fl.Param()
		table  = params
		column = "id"
		count  = 0
		err    error
	)

	if strings.Contains(params, ";") {
		t := strings.Split(params, ";")
		table = t[0]
		column = t[1]
	}

	err = v.database.
		Table(table).
		Select("CASE WHEN COUNT(*) > 0 THEN 1 ELSE 0 END").
		Where(clause.Eq{Column: column, Value: field.String()}).
		Find(&count).Error

	return errors.Is(err, gorm.ErrRecordNotFound) || count == 0
}

func (v *Validation) checkError(e error) error {
	var (
		invalidValidationError *validator.InvalidValidationError
		errs                   validator.ValidationErrors
		fe                     validator.FieldError
		translated             = make(map[string][]string)
	)

	if e != nil {
		if errors.As(e, &invalidValidationError) {
			return errors.New("something went wrong. Please try again later")
		}

		errors.As(e, &errs)

		for i := 0; i < len(errs); i++ {
			errors.As(errs[i], &fe)
			field := locales.ToSnakeCase(fe.Field())
			translated[field] = append(translated[field], fe.Translate(v.trans))
		}

		jsonErr, _ := json.Marshal(translated)

		return errors.New(string(jsonErr))
	}

	return nil
}

func (v *Validation) Validate(s interface{}) error {
	e := v.validator.Struct(s)
	return v.checkError(e)
}

func (v *Validation) VarValidate(value, rule string) error {
	err := v.validator.Var(value, rule)

	return v.checkError(err)
}

func (v *Validation) RegisterAppDependencies(checker CurrentPasswordChecker) {
	currentPasswordChecker = checker
}

func (v *Validation) RegisterStructValidation(fn StructLevelFunc, types ...interface{}) {
	v.validator.RegisterStructValidation(fn, types...)
}

func (v *Validation) RegisterAlias(alias, tags string) {
	v.validator.RegisterAlias(alias, tags)
}
