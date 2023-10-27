package common

import (
	"fmt"
	"reflect"

	"github.com/go-playground/validator/v10"
)

type SrcForm struct {
	Src string `json:"src" validate:"gte=3,lte=50"`
}

type ValidationError struct {
	asMap    map[string]string
	asString string
}

func (err ValidationError) Error() string {
	return err.asString
}

func (err ValidationError) Map() map[string]string {
	return err.asMap
}

func ValidateStruct(validate *validator.Validate, i interface{}) error {
	err := validate.Struct(i)
	if err == nil {
		return nil
	}
	var validationErr ValidationError
	validationErr.asMap = make(map[string]string)
	for _, e := range err.(validator.ValidationErrors) {
		field := e.StructField()
		errParam := e.Param()
		fieldLen := len(fmt.Sprintf("%v", e.Value()))
		reflection := reflect.TypeOf(i)
		reflectedField, _ := reflection.FieldByName(field)
		errString := fmt.Sprintf("Поле %s", reflectedField.Tag.Get("uaLocal"))
		switch e.Tag() {
		case "gte":
			if fieldLen == 0 {
				errString = fmt.Sprintf("%s обов'язкове", errString)
			} else {
				errString = fmt.Sprintf("%s надто коротке (%d), мінімальна довжина - %s символи(ів)", errString, fieldLen, errParam)
			}
		case "lte":
			errString = fmt.Sprintf(
				"%s надто довге (%d), максимальна довжина - %s символи(ів)",
				errString,
				fieldLen,
				errParam,
			)
		default:
			errString = errString + " невірне"
		}
		validationErr.asMap[field+"Err"] = errString
		if validationErr.asString != "" {
			validationErr.asString = validationErr.asString + "\n" + errString
		} else {
			validationErr.asString = errString
		}
	}
	return validationErr
}
