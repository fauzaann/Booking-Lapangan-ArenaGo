package utils

import (
	"errors"
	"fmt"
	"reflect"

	govalidator "github.com/go-playground/validator/v10"
)

// Validator adalah struct yang digunakan untuk memvalidasi input.
type Validator struct {
	validator *govalidator.Validate
}

// NewValidator membuat validator baru beserta rule kustom aplikasi.
func NewValidator() *Validator {
	v := govalidator.New()
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		return fld.Tag.Get("json")
	})

	_ = v.RegisterValidation("clock", validateClock)
	_ = v.RegisterValidation("dateonly", validateDateOnly)
	_ = v.RegisterValidation("notpast", validateNotPastTime)

	return &Validator{validator: v}
}

// Validate mengembalikan nil jika payload valid, atau map error per field.
func (v *Validator) Validate(payload interface{}) map[string]string {
	if err := v.validator.Struct(payload); err != nil {
		var fieldErrors govalidator.ValidationErrors
		if !errors.As(err, &fieldErrors) {
			return map[string]string{"_error": err.Error()}
		}
		result := make(map[string]string, len(fieldErrors))
		for _, fe := range fieldErrors {
			result[fe.Field()] = message(fe)
		}
		return result
	}
	return nil
}

// validateClock memastikan waktu berformat HH:MM
func validateClock(fl govalidator.FieldLevel) bool {
	value := fl.Field().String()
	return ValidClock(value)
}

// validateDateOnly memastikan tanggal berformat YYYY-MM-DD
func validateDateOnly(fl govalidator.FieldLevel) bool {
	value := fl.Field().String()
	if value == "" {
		return true
	}
	_, err := ParseDate(value)
	return err == nil
}

// validateNotPastTime memastikan tanggal tidak lebih lama dari hari ini
func validateNotPastTime(fl govalidator.FieldLevel) bool {
	now := Today()
	value := fl.Field().String()
	date, err := ParseDate(value)
	if err != nil {
		return false
	}
	return date.After(now) || date.Equal(now)
}

// message menerjemahkan error validasi menjadi pesan yang user-friendly.
func message(fe govalidator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s wajib diisi", fe.Field())
	case "email":
		return fmt.Sprintf("%s harus berformat email", fe.Field())
	case "min":
		return fmt.Sprintf("%s minimal %s karakter", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("%s maksimal %s karakter", fe.Field(), fe.Param())
	case "oneof":
		return fmt.Sprintf("%s harus salah satu dari %s", fe.Field(), fe.Param())
	case "clock":
		return fmt.Sprintf("%s harus berformat HH:MM", fe.Field())
	case "dateonly":
		return fmt.Sprintf("%s harus berformat YYYY-MM-DD", fe.Field())
	case "notpast":
		return fmt.Sprintf("%s tidak boleh tanggal lampau", fe.Field())
	default:
		return fmt.Sprintf("%s is invalid (%s)", fe.Field(), fe.Tag())
	}
}
