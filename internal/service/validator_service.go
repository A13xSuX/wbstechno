package service

import (
	"fmt"
	"order-service/internal/database"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

type ValidatorService struct {
	validate *validator.Validate
}

func NewValidatorService() *ValidatorService {
	v := validator.New()

	// Регистрируем кастомные валидации
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	// Регистрируем кастомную валидацию для alphanum с дефисами
	v.RegisterValidation("alphanumdash", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		// Разрешаем буквы, цифры, дефисы и подчеркивания
		matched, _ := regexp.MatchString(`^[a-zA-Z0-9\-_]+$`, value)
		return matched
	})

	return &ValidatorService{validate: v}
}

// ValidateOrder validates order using go-playground/validator
func (vs *ValidatorService) ValidateOrder(order database.Order) error {
	err := vs.validate.Struct(order)
	if err != nil {
		return vs.formatValidationError(err)
	}
	return nil
}

// formatValidationError formats validation errors to user-friendly messages
func (vs *ValidatorService) formatValidationError(err error) error {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var errorMessages []string

		for _, fieldError := range validationErrors {
			var message string

			switch fieldError.Tag() {
			case "required":
				message = fmt.Sprintf("поле '%s' обязательно для заполнения", fieldError.Field())
			case "min":
				message = fmt.Sprintf("поле '%s' должно быть не менее %s", fieldError.Field(), fieldError.Param())
			case "max":
				message = fmt.Sprintf("поле '%s' должно быть не более %s", fieldError.Field(), fieldError.Param())
			case "email":
				message = fmt.Sprintf("поле '%s' должно быть валидным email адресом", fieldError.Field())
			case "e164":
				message = fmt.Sprintf("поле '%s' должно быть в формате E.164 (например: +79161234567)", fieldError.Field())
			case "uuid":
				message = fmt.Sprintf("поле '%s' должно быть в формате UUID", fieldError.Field())
			case "alpha":
				message = fmt.Sprintf("поле '%s' должно содержать только буквы", fieldError.Field())
			case "alphanum":
				message = fmt.Sprintf("поле '%s' должно содержать только буквы и цифры", fieldError.Field())
			case "numeric":
				message = fmt.Sprintf("поле '%s' должно содержать только цифры", fieldError.Field())
			case "uppercase":
				message = fmt.Sprintf("поле '%s' должно быть в верхнем регистре", fieldError.Field())
			default:
				message = fmt.Sprintf("поле '%s' невалидно: %s", fieldError.Field(), fieldError.Tag())
			}

			errorMessages = append(errorMessages, message)
		}

		// Исправлено: используем strings.Join для создания строки
		return fmt.Errorf("%s", strings.Join(errorMessages, "; "))
	}

	return err
}

// ValidateStruct validates any struct using validator
func (vs *ValidatorService) ValidateStruct(s interface{}) error {
	return vs.validate.Struct(s)
}
