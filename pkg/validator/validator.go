package validator

import (
	"reflect"
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	validate *validator.Validate
	once     sync.Once
)

func Get() *validator.Validate {
	once.Do(func() {
		validate = validator.New()
		validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := fld.Name
			if jsonTag := fld.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
				name = jsonTag
			}
			return name
		})
	})
	return validate
}

func Struct(s interface{}) error {
	return Get().Struct(s)
}