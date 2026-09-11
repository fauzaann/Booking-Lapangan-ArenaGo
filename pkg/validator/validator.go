// Package validator membungkus go-playground/validator untuk memvalidasi
// payload request berdasarkan tag `validate` pada struct DTO.
package validator

import (
	"fmt"
	"reflect"
	"strings"

	playground "github.com/go-playground/validator/v10"
)

// Validator adalah wrapper tipis di atas go-playground/validator.
type Validator struct {
	validate *playground.Validate
}

// New membuat Validator baru. Nama field pada pesan error mengikuti tag
// `json` pada struct, bukan nama field Go, supaya lebih mudah dipahami client.
func New() *Validator {
	v := playground.New()
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	return &Validator{validate: v}
}

// Validate memvalidasi payload berdasarkan tag `validate`.
// Mengembalikan nil jika valid, atau map field -> pesan error jika tidak.
func (v *Validator) Validate(payload interface{}) map[string]string {
	err := v.validate.Struct(payload)
	if err == nil {
		return nil
	}

	validationErrors, ok := err.(playground.ValidationErrors)
	if !ok {
		return map[string]string{"_error": err.Error()}
	}

	fields := make(map[string]string, len(validationErrors))
	for _, fe := range validationErrors {
		fields[fe.Field()] = fmt.Sprintf("failed on '%s' validation", fe.Tag())
	}
	return fields
}
