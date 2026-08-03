package validation

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationErrorCollection struct {
	Errors []FieldError `json:"errors"`
}

func (v *ValidationErrorCollection) Error() string {
	var msgs []string
	for _, e := range v.Errors {
		msgs = append(msgs, e.Field+": "+e.Message)
	}
	return strings.Join(msgs, ", ")
}

func getCustomMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required."
	case "email":
		return "Invalid email format."
	case "min":
		return fmt.Sprintf("Must be at least %s characters long.", fe.Param())
	case "gte":
		return fmt.Sprintf("Must be greater than or equal to %s.", fe.Param())
	default:
		return "Invalid value."
	}
}

func ValidateRequest(req interface{}) error {
	validate := validator.New()
	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	err := validate.Struct(req)
	if err == nil {
		return nil
	}

	var valErrs validator.ValidationErrors
	if errors.As(err, &valErrs) {
		out := &ValidationErrorCollection{
			Errors: make([]FieldError, 0, len(valErrs)),
		}

		for _, fe := range valErrs {
			out.Errors = append(out.Errors, FieldError{
				Field:   strings.ToLower(fe.Field()),
				Message: strings.ToLower(getCustomMessage(fe)),
			})
		}

		return out
	}
	return err
}
