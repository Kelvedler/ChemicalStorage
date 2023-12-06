package common

import (
	"fmt"
	"reflect"

	"github.com/go-playground/validator/v10"
)

func NewValidator() (v *validator.Validate) {
	v = validator.New(validator.WithRequiredStructEnabled())
	return v
}

type ValidationError struct {
	asMapLocal map[string]string
	asString   string
}

func (err ValidationError) Error() string {
	return err.asString
}

func (err ValidationError) Map() map[string]string {
	return err.asMapLocal
}

func LocalizeValidationErrors(errs validator.ValidationErrors, targetStruct interface{}) error {
	var validationErr ValidationError
	validationErr.asMapLocal = make(map[string]string)
	for _, err := range errs {
		field := err.StructField()
		errParam := err.Param()
		fieldLen := len(fmt.Sprintf("%v", err.Value()))
		reflection := reflect.TypeOf(targetStruct)
		reflectedField, _ := reflection.FieldByName(field)
		errString := fmt.Sprintf("Field %s", field)
		errStringLocal := fmt.Sprintf("Поле %s", reflectedField.Tag.Get("uaLocal"))
		switch err.Tag() {
		case "gte":
			if fieldLen == 0 {
				errString = fmt.Sprintf("%s is required", errString)
				errStringLocal = fmt.Sprintf("%s обов'язкове", errStringLocal)
			} else {
				errString = fmt.Sprintf("%s is too short (%d), min length - %s", errString, fieldLen, errParam)
				errStringLocal = fmt.Sprintf("%s надто коротке (%d), мінімальна довжина - %s символи(ів)", errStringLocal, fieldLen, errParam)
			}
		case "lte":
			errString = fmt.Sprintf(
				"%s too long (%d), max length - %s",
				errString,
				fieldLen,
				errParam,
			)
			errStringLocal = fmt.Sprintf(
				"%s надто довге (%d), максимальна довжина - %s символи(ів)",
				errStringLocal,
				fieldLen,
				errParam,
			)
		default:
			errString = errString + " invalid"
			errStringLocal = errStringLocal + " невірне"
		}
		validationErr.asMapLocal[field+"Err"] = errStringLocal
		if validationErr.asString != "" {
			validationErr.asString = validationErr.asString + "\n" + errString
		} else {
			validationErr.asString = errString
		}
	}
	return validationErr
}
